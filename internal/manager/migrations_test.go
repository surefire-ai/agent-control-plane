package manager

import (
	"context"
	"errors"
	"testing"
)

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

func TestPendingMigrations(t *testing.T) {
	migrations := []Migration{
		{Version: "0001", Name: "core"},
		{Version: "0002", Name: "audit"},
	}
	pending := PendingMigrations(migrations, map[string]bool{"0001": true})

	if len(pending) != 1 || pending[0].Version != "0002" {
		t.Fatalf("expected only second migration to be pending, got %#v", pending)
	}
}

func TestApplyMigrations(t *testing.T) {
	store := &fakeMigrationStore{
		applied: map[string]bool{"0001": true},
	}
	migrations := []Migration{
		{Version: "0001", Name: "core", SQL: "SELECT 1"},
		{Version: "0002", Name: "audit", SQL: "SELECT 2"},
	}

	applied, err := ApplyMigrations(context.Background(), store, migrations)
	if err != nil {
		t.Fatalf("ApplyMigrations returned error: %v", err)
	}
	if len(applied) != 1 || applied[0].Version != "0002" {
		t.Fatalf("expected only second migration to be applied, got %#v", applied)
	}
	if len(store.appliedOrder) != 1 || store.appliedOrder[0] != "0002" {
		t.Fatalf("expected fake store to apply second migration, got %#v", store.appliedOrder)
	}
}

func TestApplyMigrationsPropagatesStoreFailure(t *testing.T) {
	store := &fakeMigrationStore{
		applied:  map[string]bool{},
		applyErr: errors.New("boom"),
	}
	_, err := ApplyMigrations(context.Background(), store, []Migration{{Version: "0001", Name: "core", SQL: "SELECT 1"}})
	if err == nil {
		t.Fatal("expected store failure")
	}
}

type fakeMigrationStore struct {
	applied      map[string]bool
	appliedOrder []string
	applyErr     error
}

func (s *fakeMigrationStore) AppliedMigrationVersions(ctx context.Context) (map[string]bool, error) {
	return s.applied, nil
}

func (s *fakeMigrationStore) ApplyMigration(ctx context.Context, migration Migration) error {
	if s.applyErr != nil {
		return s.applyErr
	}
	s.appliedOrder = append(s.appliedOrder, migration.Version)
	return nil
}
