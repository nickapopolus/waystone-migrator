package migrator

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Migrator struct {
	db             *sql.DB
	migrationsDir  string
	seedDir        string
	migrationTable string
	seedTable      string
	migrations     []Migration
	seeds          []Seed
}

func NewMigrator(db *sql.DB, opts ...Option) (*Migrator, error) {
	migrator := &Migrator{
		db:             db,
		migrationsDir:  "./migrations",
		migrationTable: "waystone_migrations",
		seedDir:        "./seeds",
		seedTable:      "waystone_seeds",
	}

	for _, opt := range opts {
		opt(migrator)
	}
	migrationsQuery := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	    version INTEGER PRIMARY KEY,
	    filename VARCHAR(255) NOT NULL,
	    applied_at TIMESTAMP WITH TIME ZONE
	)`, migrator.migrationTable)

	_, err := db.Exec(migrationsQuery)
	if err != nil {
		return nil, fmt.Errorf("Could Not Initialize Migrations Table %w", err)
	}

	seedQuery := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	    version INTEGER PRIMARY KEY,
	    filename VARCHAR(255) NOT NULL,
	    applied_at TIMESTAMP WITH TIME ZONE
	)`, migrator.seedTable)

	_, err = db.Exec(seedQuery)
	if err != nil {
		return nil, fmt.Errorf("Could Not Initialize Seeds Table %w", err)
	}

	return migrator, nil
}

type Option func(*Migrator)

func WithMigrationsDir(dir string) Option {
	return func(m *Migrator) {
		m.migrationsDir = dir
	}
}

func WithMigrationsTable(table string) Option {
	return func(m *Migrator) {
		m.migrationTable = table
	}
}

func WithSeedsDir(dir string) Option {
	return func(m *Migrator) {
		m.seedDir = dir
	}
}

func WithSeedsTable(table string) Option {
	return func(m *Migrator) {
		m.seedTable = table
	}
}

func (m *Migrator) Up() error {
	err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("Could not load migrations %w", err)
	}
	appliedMigrations, err := m.fetchAppliedMigrations()
	if err != nil {
		return fmt.Errorf("Could not fetch applied migrations %w", err)
	}
	err = m.validateMigrations(m.migrations)
	if err != nil {
		return fmt.Errorf("Could not validate migrations %w", err)
	}
	var pendingMigrations []Migration
	for _, migration := range m.migrations {
		if !appliedMigrations[migration.version] {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}
	if len(pendingMigrations) == 0 {
		fmt.Println("No migrations to apply")
		return nil
	}

	for _, migration := range pendingMigrations {
		err = m.executeMigration(migration, false)
		if err != nil {
			return fmt.Errorf("Could not execute migration %d: %w", migration.version, err)
		}
	}

	return nil
}

func (m *Migrator) Down(targetVersion int) error {
	err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("Could not load migrations %w", err)
	}
	appliedMigrations, err := m.fetchAppliedMigrations()
	if err != nil {
		return fmt.Errorf("Could not fetch applied migrations %w", err)
	}
	err = m.validateMigrations(m.migrations)
	if err != nil {
		return fmt.Errorf("Could not validate migrations %w", err)
	}
	var pendingMigrations []Migration
	for _, migration := range m.migrations {
		if appliedMigrations[migration.version] && migration.version > targetVersion {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}
	if len(pendingMigrations) == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	sort.Slice(pendingMigrations, func(i, j int) bool {
		return pendingMigrations[i].version > pendingMigrations[j].version
	})

	for _, migration := range pendingMigrations {
		err = m.executeMigration(migration, true)
		if err != nil {
			return fmt.Errorf("Could not execute rollback %d: %w", migration.version, err)
		}
	}

	return nil
}

func (m *Migrator) loadMigrations() error {
	migrations := []Migration{}
	fmt.Println("Loading migrations...")

	//appliedMigrations, err := m.fetchAppliedMigrations()
	//if err != nil {
	//	return err
	//}

	migFiles, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return err
	}

	for _, migFile := range migFiles {
		if migFile.IsDir() {
			continue
		}
		if !strings.HasSuffix(migFile.Name(), ".sql") {
			message := fmt.Sprintf("%s is not a sql file. Migration files must be sql", migFile.Name())
			return errors.New(message)
		}
		split := strings.Split(migFile.Name(), "_")
		if len(split) < 2 {
			message := fmt.Sprintf("%s is not formatted properly. use {ver_no}_{comment}.sql", migFile.Name())
			return errors.New(message)
		}
		version, err := strconv.Atoi(split[0])
		if err != nil {
			message := fmt.Sprintf("%s is not a valid version number", split[0])
			return errors.New(message)
		}
		up, down, err := m.parseFile(migFile)
		if err != nil {
			message := fmt.Sprintf("is not a valid version number %w", err)
			return errors.New(message)
		}
		migration := Migration{
			version:  version,
			name:     split[1],
			filename: migFile.Name(),
			up:       up,
			down:     down,
		}

		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	m.migrations = migrations

	return nil
}

func (m *Migrator) validateMigrations(migrations []Migration) error {
	for i, migration := range migrations {
		expectedVersion := i + 1
		if migration.version != expectedVersion {
			if migration.version < expectedVersion {
				return fmt.Errorf("Duplicate migration version. Expected %d, got %d", migration.version, expectedVersion)
			} else {
				return fmt.Errorf("Migration Gap detected. Expected %d, got %d", expectedVersion, migration.version)
			}
		}
	}
	return nil
}

func (m *Migrator) fetchAppliedMigrations() (map[int]bool, error) {
	appliedMigrations := map[int]bool{}

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY version ASC", m.migrationTable)
	result, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer result.Close()
	for result.Next() {
		migration := Migration{}
		err := result.Scan(&migration.version, &migration.filename)
		if err != nil {
			return nil, err
		}
		appliedMigrations[migration.version] = true
	}
	if err := result.Err(); err != nil {
		return nil, err
	}
	return appliedMigrations, nil
}

func (m *Migrator) parseFile(file os.DirEntry) (string, string, error) {
	fullPath := filepath.Join(m.migrationsDir, file.Name())

	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", err
	}

	stringContents := string(contents)

	splits := strings.SplitN(stringContents, "-- +down", 2)

	up := strings.TrimSpace(splits[0])
	var down string
	if len(splits) > 1 {
		down = strings.TrimSpace(splits[1])
	}

	return up, down, nil
}

func (m *Migrator) splitSQL(sql string) []string {
	commands := strings.Split(sql, ";")
	var statements []string
	for _, command := range commands {
		fmt.Println(command)
	}
	for _, command := range commands {
		trimmedCommand := strings.TrimSpace(command)
		if trimmedCommand == "" || strings.HasPrefix(trimmedCommand, "--") {
			continue
		}
		if !strings.HasSuffix(trimmedCommand, ";") {
			trimmedCommand = trimmedCommand + ";"
		}
		statements = append(statements, trimmedCommand)
	}
	return statements
}

func (m *Migrator) executeMigration(migration Migration, isDown bool) error {
	fmt.Printf("Executing migration: %d %s\n", migration.version, migration.name)
	var statements []string
	if isDown {
		statements = m.splitSQL(migration.down)
	} else {
		statements = m.splitSQL(migration.up)
	}

	transaction, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	for i, statement := range statements {
		line := i + 1
		fmt.Printf("Executing statement: %d/%d\n", line, len(statements))
		if _, err := transaction.Exec(statement); err != nil {
			return fmt.Errorf("Error executing statement: %d/%d for migration %d %w\n", i, len(statements), migration.version, err)
		}
	}
	if isDown {
		query := fmt.Sprintf("DELETE FROM %s WHERE VERSION = %d;", m.migrationTable, migration.version)
		_, err = transaction.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to record down migrations: %w", err)
		}
	} else {
		query := fmt.Sprintf("INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, now());", m.migrationTable)
		_, err = transaction.Exec(query, migration.version, migration.name)
		if err != nil {
			return fmt.Errorf("failed to record up migrations: %w", err)
		}
	}
	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}
	fmt.Printf("✅ Migration %d: %s succeeded!\n", migration.version, migration.name)

	return nil
}

func (m *Migrator) loadSeeds() error {
	m.seeds = []Seed{}
	return nil
}
