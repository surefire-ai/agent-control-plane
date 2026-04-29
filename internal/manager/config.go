package manager

import (
	"os"
	"strings"
)

const (
	defaultAddr = ":8090"
)

type Config struct {
	Addr        string
	DatabaseURL string
	Mode        string
}

func ConfigFromEnv() Config {
	return Config{
		Addr:        envOrDefault("MANAGER_ADDR", defaultAddr),
		DatabaseURL: strings.TrimSpace(os.Getenv("MANAGER_DATABASE_URL")),
		Mode:        envOrDefault("MANAGER_MODE", "standalone"),
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
	c.DatabaseURL = strings.TrimSpace(c.DatabaseURL)
	return c
}
