package manager

import (
	"os"
	"strings"
)

const (
	defaultAddr = ":8090"
)

type Config struct {
	Addr           string
	DatabaseDriver string
	DatabaseURL    string
	Mode           string
}

func ConfigFromEnv() Config {
	return Config{
		Addr:           envOrDefault("MANAGER_ADDR", defaultAddr),
		DatabaseDriver: envOrDefault("MANAGER_DATABASE_DRIVER", "postgres"),
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

func (c Config) normalized() Config {
	if strings.TrimSpace(c.Addr) == "" {
		c.Addr = defaultAddr
	}
	if strings.TrimSpace(c.Mode) == "" {
		c.Mode = "standalone"
	}
	if strings.TrimSpace(c.DatabaseDriver) == "" {
		c.DatabaseDriver = "postgres"
	}
	c.DatabaseDriver = strings.TrimSpace(c.DatabaseDriver)
	c.DatabaseURL = strings.TrimSpace(c.DatabaseURL)
	return c
}
