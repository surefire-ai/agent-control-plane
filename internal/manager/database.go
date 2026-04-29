package manager

import (
	"context"
	"database/sql"
	"fmt"
)

type Database struct {
	DB *sql.DB
}

func OpenDatabase(ctx context.Context, config Config) (*Database, error) {
	config = config.normalized()
	if config.DatabaseURL == "" {
		return nil, nil
	}
	db, err := sql.Open(config.DatabaseDriver, config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open manager database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping manager database: %w", err)
	}
	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	if d == nil || d.DB == nil {
		return nil
	}
	return d.DB.Close()
}
