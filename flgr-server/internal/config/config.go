// Package config loads flgr-server's configuration from FLGR_* environment
// variables, per docs/architecture/adr/0010-application-configuration-and-secrets.md.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds flgr-server's runtime configuration.
type Config struct {
	HTTPPort            string
	DBPath              string
	SessionCookieSecure bool
	KafkaBrokers        string
	LogLevel            string
	EncryptionKey       string
}

// Load reads FLGR_* environment variables into a Config, failing on any
// missing required value. A local .env file is loaded first if present
// (docs/architecture/adr/0012-local-development-environment.md); this is a
// no-op in production, where no .env file exists.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		HTTPPort:            envOrDefault("FLGR_HTTP_PORT", "8080"),
		DBPath:              os.Getenv("FLGR_DB_PATH"),
		SessionCookieSecure: envBoolOrDefault("FLGR_SESSION_COOKIE_SECURE", true),
		KafkaBrokers:        os.Getenv("FLGR_KAFKA_BROKERS"),
		LogLevel:            envOrDefault("FLGR_LOG_LEVEL", "info"),
		EncryptionKey:       os.Getenv("FLGR_ENCRYPTION_KEY"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DBPath == "" {
		return fmt.Errorf("FLGR_DB_PATH is required")
	}
	if c.EncryptionKey == "" {
		return fmt.Errorf("FLGR_ENCRYPTION_KEY is required")
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBoolOrDefault(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
