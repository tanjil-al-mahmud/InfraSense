package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// TestMigrationApplication tests that migrations apply successfully on a clean database
func TestMigrationApplication(t *testing.T) {
	// Skip if no test database available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	// Apply migrations
	err := db.RunMigrations("migrations")
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify schema version
	version, dirty, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	if dirty {
		t.Error("Schema is in dirty state after successful migration")
	}

	if version == 0 {
		t.Error("Schema version is 0 after applying migrations")
	}

	t.Logf("Successfully applied migrations to version %d", version)
}

// TestMigrationRollback tests that failed migrations are rolled back automatically
func TestMigrationRollback(t *testing.T) {
	// Skip if no test database available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	// Create a temporary migration directory with a failing migration
	tempDir := t.TempDir()

	// Copy existing migrations
	copyMigrations(t, "migrations", tempDir)

	// Add a failing migration
	failingMigrationUp := filepath.Join(tempDir, "010_failing_migration.up.sql")
	failingMigrationDown := filepath.Join(tempDir, "010_failing_migration.down.sql")

	// Create a migration that will fail (invalid SQL)
	if err := os.WriteFile(failingMigrationUp, []byte("INVALID SQL SYNTAX HERE;"), 0644); err != nil {
		t.Fatalf("Failed to create failing migration: %v", err)
	}

	if err := os.WriteFile(failingMigrationDown, []byte("-- Rollback"), 0644); err != nil {
		t.Fatalf("Failed to create failing migration down: %v", err)
	}

	// Apply migrations - should fail and rollback
	err := db.RunMigrations(tempDir)
	if err == nil {
		t.Fatal("Expected migration to fail, but it succeeded")
	}

	t.Logf("Migration failed as expected: %v", err)

	// Verify database is not in dirty state after rollback
	version, dirty, err := getSchemaVersion(db)
	if err != nil && err != migrate.ErrNilVersion {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	if dirty {
		t.Error("Schema is in dirty state after automatic rollback")
	}

	t.Logf("Schema successfully rolled back to version %d", version)
}

// TestSchemaVersionValidation tests schema version compatibility checking
func TestSchemaVersionValidation(t *testing.T) {
	db := &DB{}

	tests := []struct {
		name          string
		version       uint
		expectError   bool
		errorContains string
	}{
		{
			name:        "Empty database (version 0) is valid",
			version:     0,
			expectError: false,
		},
		{
			name:        "Current version is valid",
			version:     5,
			expectError: false,
		},
		{
			name:        "Maximum supported version is valid",
			version:     9,
			expectError: false,
		},
		{
			name:          "Future version is invalid",
			version:       100,
			expectError:   true,
			errorContains: "newer than supported version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.validateSchemaVersion(tt.version)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for version %d, but got none", tt.version)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for version %d, but got: %v", tt.version, err)
				}
			}
		})
	}
}

// TestDirtyStateRecovery tests that the system can recover from a dirty database state
func TestDirtyStateRecovery(t *testing.T) {
	// Skip if no test database available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	// First apply migrations normally
	err := db.RunMigrations("migrations")
	if err != nil {
		t.Fatalf("Failed to apply initial migrations: %v", err)
	}

	// Manually set database to dirty state
	driver, err := postgres.WithInstance(db.conn, &postgres.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		t.Fatalf("Failed to create migration instance: %v", err)
	}

	currentVersion, _, err := m.Version()
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}

	// Force dirty state
	if err := m.Force(int(currentVersion)); err != nil {
		t.Fatalf("Failed to force version: %v", err)
	}

	// Manually mark as dirty in schema_migrations table
	_, err = db.conn.Exec("UPDATE schema_migrations SET dirty = true WHERE version = $1", currentVersion)
	if err != nil {
		t.Fatalf("Failed to set dirty state: %v", err)
	}

	// Verify dirty state
	version, dirty, err := getSchemaVersion(db)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	if !dirty {
		t.Fatal("Failed to set database to dirty state")
	}

	t.Logf("Database is in dirty state at version %d", version)

	// Now run migrations again - should recover from dirty state
	err = db.RunMigrations("migrations")
	if err != nil {
		t.Fatalf("Failed to recover from dirty state: %v", err)
	}

	// Verify database is clean now
	version, dirty, err = getSchemaVersion(db)
	if err != nil {
		t.Fatalf("Failed to get schema version after recovery: %v", err)
	}

	if dirty {
		t.Error("Database is still in dirty state after recovery")
	}

	t.Logf("Successfully recovered from dirty state, current version: %d", version)
}

// Helper functions

func setupTestDB(t *testing.T) *DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL environment variable not set")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Drop all tables to start fresh
	_, err = conn.Exec(`
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO public;
	`)
	if err != nil {
		t.Fatalf("Failed to reset test database: %v", err)
	}

	return &DB{conn: conn}
}

func getSchemaVersion(db *DB) (uint, bool, error) {
	driver, err := postgres.WithInstance(db.conn, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return m.Version()
}

func copyMigrations(t *testing.T, src, dst string) {
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", srcPath, err)
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			t.Fatalf("Failed to write migration file %s: %v", dstPath, err)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
