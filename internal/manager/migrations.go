package manager

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version string
	Name    string
	SQL     string
}

type MigrationStore interface {
	AppliedMigrationVersions(ctx context.Context) (map[string]bool, error)
	ApplyMigration(ctx context.Context, migration Migration) error
}

type SQLMigrationStore struct {
	DB *sql.DB
}

func BuiltInMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read manager migrations: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read manager migration %q: %w", entry.Name(), err)
		}
		migration, err := migrationFromFile(entry.Name(), string(content))
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func migrationFromFile(name string, sql string) (Migration, error) {
	trimmed := strings.TrimSuffix(name, ".sql")
	parts := strings.SplitN(trimmed, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Migration{}, fmt.Errorf("invalid manager migration filename %q", name)
	}
	return Migration{
		Version: parts[0],
		Name:    parts[1],
		SQL:     sql,
	}, nil
}

func PendingMigrations(migrations []Migration, applied map[string]bool) []Migration {
	pending := make([]Migration, 0, len(migrations))
	for _, migration := range migrations {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}
	return pending
}

func ApplyMigrations(ctx context.Context, store MigrationStore, migrations []Migration) ([]Migration, error) {
	if store == nil {
		return nil, fmt.Errorf("manager migration store is required")
	}
	applied, err := store.AppliedMigrationVersions(ctx)
	if err != nil {
		return nil, err
	}
	pending := PendingMigrations(migrations, applied)
	for _, migration := range pending {
		if err := store.ApplyMigration(ctx, migration); err != nil {
			return nil, err
		}
	}
	return pending, nil
}

func (s SQLMigrationStore) AppliedMigrationVersions(ctx context.Context) (map[string]bool, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS manager_schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return nil, fmt.Errorf("ensure manager schema migrations table: %w", err)
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT version FROM manager_schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("list applied manager migrations: %w", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied manager migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied manager migrations: %w", err)
	}
	return applied, nil
}

func (s SQLMigrationStore) ApplyMigration(ctx context.Context, migration Migration) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin manager migration %s: %w", migration.Version, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, migration.SQL); err != nil {
		return fmt.Errorf("apply manager migration %s: %w", migration.Version, err)
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO manager_schema_migrations (version, name) VALUES ($1, $2)", migration.Version, migration.Name); err != nil {
		return fmt.Errorf("record manager migration %s: %w", migration.Version, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit manager migration %s: %w", migration.Version, err)
	}
	return nil
}
