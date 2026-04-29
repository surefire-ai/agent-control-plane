package manager

import (
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
