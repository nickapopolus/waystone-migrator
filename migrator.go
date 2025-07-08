package migrator

import "database/sql"

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

	return nil
}

func (m *Migrator) LoadSeeds() error {
	m.seeds = make([]Seed, 0)
	return nil
}
