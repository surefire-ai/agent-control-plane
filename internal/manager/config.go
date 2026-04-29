package manager

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultAddr = ":8090"
)

type Config struct {
	Addr           string
	AutoMigrate    bool
	DatabaseDriver string
	DatabaseURL    string
	Mode           string
}

func ConfigFromEnv() Config {
	return Config{
		Addr:           envOrDefault("MANAGER_ADDR", defaultAddr),
		AutoMigrate:    envBoolOrDefault("MANAGER_MIGRATE_ON_START", false),
		DatabaseDriver: envOrDefault("MANAGER_DATABASE_DRIVER", "pgx"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("MANAGER_DATABASE_URL")),
		Mode:           envOrDefault("MANAGER_MODE", "standalone"),
	}
}

func envOrDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBoolOrDefault(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (c Config) normalized() Config {
	if strings.TrimSpace(c.Addr) == "" {
		c.Addr = defaultAddr
	}
	if strings.TrimSpace(c.Mode) == "" {
		c.Mode = "standalone"
	}
	if strings.TrimSpace(c.DatabaseDriver) == "" {
		c.DatabaseDriver = "pgx"
	}
	c.DatabaseDriver = strings.TrimSpace(c.DatabaseDriver)
	c.DatabaseURL = strings.TrimSpace(c.DatabaseURL)
	return c
}
