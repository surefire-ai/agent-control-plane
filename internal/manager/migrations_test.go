package manager

import "testing"

func TestBuiltInMigrations(t *testing.T) {
	migrations, err := BuiltInMigrations()
	if err != nil {
		t.Fatalf("BuiltInMigrations returned error: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least one migration")
	}
	first := migrations[0]
	if first.Version != "0001" {
		t.Fatalf("expected first migration version 0001, got %#v", first)
	}
	if first.Name != "manager_core" {
		t.Fatalf("expected first migration name manager_core, got %#v", first)
	}
	if first.SQL == "" {
		t.Fatalf("expected migration SQL, got %#v", first)
	}
}

func TestMigrationFromFileRejectsInvalidName(t *testing.T) {
	if _, err := migrationFromFile("invalid.sql", "SELECT 1"); err == nil {
		t.Fatal("expected invalid migration filename to fail")
	}
}
