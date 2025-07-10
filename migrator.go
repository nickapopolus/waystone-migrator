package migrator

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func (m *Migrator) LoadMigrations() error {
	m.migrations = make([]Migration, 0)
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
			message := fmt.Sprintf("is not a valid version number")
			return errors.New(message)
		}
		migration := Migration{
			version:  version,
			name:     split[1],
			filename: migFile.Name(),
			up:       up,
			down:     down,
		}
		fmt.Println("Mig file up", migration.up)
		m.migrations = append(m.migrations, migration)
	}

	return nil
}

func (m *Migrator) LoadSeeds() error {
	m.seeds = make([]Seed, 0)
	return nil
}

func (m *Migrator) fetchAppliedMigrations() ([]Migration, error) {
	appliedMigrations := []Migration{}

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
		appliedMigrations = append(appliedMigrations, migration)
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
