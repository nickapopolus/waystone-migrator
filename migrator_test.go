package migrator

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewMigrator(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrator, err := NewMigrator(db)
	if err != nil {
		t.Fatalf("Test Instantiantion: Could not instatniate the migrator: %v", err)
	}

	testMigratorDir := WithMigrationsDir("alpha_migration")
	testMigratorDir(migrator)
	if migrator.migrationsDir != "alpha_migration" {
		t.Errorf("WithMigrationsDir did not change the migrator dir")
	}
	testSeedDir := WithSeedsDir("alpha_seed")
	testSeedDir(migrator)
	if migrator.seedDir != "alpha_seed" {
		t.Errorf("WithSeedsDir did not change theseed dir")
	}
	testMigrationTable := WithMigrationsTable("alpha_migration")
	testMigrationTable(migrator)
	if migrator.migrationTable != "alpha_migration" {
		t.Errorf("WithMigrationsTable did not change the migrator table")
	}
	testSeedTable := WithSeedsTable("alpha_seed")
	testSeedTable(migrator)
	if migrator.seedTable != "alpha_seed" {
		t.Errorf("WithSeedsTable did not change the migrator table")
	}
}

func TestMigrateUpandDown(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrator, err := NewMigrator(db, WithMigrationsDir("testdatabase/migrations"))
	if err != nil {
		t.Fatalf("Testing Up: Could not instatniate the migrator: %v", err)
	}
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Testing Up: Could not migrate up: %v", err)
	}
	err = migrator.Down(000)
	if err != nil {
		t.Fatalf("Testing Down: Could not migrate down: %v", err)
	}

}
