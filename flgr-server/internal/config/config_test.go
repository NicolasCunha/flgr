package config

import "testing"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FLGR_DB_PATH", "./data/flgr.db")
	t.Setenv("FLGR_ENCRYPTION_KEY", "test-encryption-key")
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FLGR_HTTP_PORT", "")
	t.Setenv("FLGR_SESSION_COOKIE_SECURE", "")
	t.Setenv("FLGR_KAFKA_BROKERS", "")
	t.Setenv("FLGR_LOG_LEVEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "8080")
	}
	if cfg.SessionCookieSecure != true {
		t.Errorf("SessionCookieSecure = %v, want true", cfg.SessionCookieSecure)
	}
	if cfg.KafkaBrokers != "" {
		t.Errorf("KafkaBrokers = %q, want empty", cfg.KafkaBrokers)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_Overrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FLGR_HTTP_PORT", "9090")
	t.Setenv("FLGR_SESSION_COOKIE_SECURE", "false")
	t.Setenv("FLGR_KAFKA_BROKERS", "localhost:9092")
	t.Setenv("FLGR_LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.HTTPPort != "9090" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "9090")
	}
	if cfg.SessionCookieSecure != false {
		t.Errorf("SessionCookieSecure = %v, want false", cfg.SessionCookieSecure)
	}
	if cfg.KafkaBrokers != "localhost:9092" {
		t.Errorf("KafkaBrokers = %q, want %q", cfg.KafkaBrokers, "localhost:9092")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoad_InvalidBooleanFallsBackToDefault(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FLGR_SESSION_COOKIE_SECURE", "not-a-bool")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.SessionCookieSecure != true {
		t.Errorf("SessionCookieSecure = %v, want true (default) on invalid input", cfg.SessionCookieSecure)
	}
}

func TestLoad_MissingDBPath(t *testing.T) {
	t.Setenv("FLGR_DB_PATH", "")
	t.Setenv("FLGR_ENCRYPTION_KEY", "test-encryption-key")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing FLGR_DB_PATH, got nil")
	}
}

func TestLoad_MissingEncryptionKey(t *testing.T) {
	t.Setenv("FLGR_DB_PATH", "./data/flgr.db")
	t.Setenv("FLGR_ENCRYPTION_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing FLGR_ENCRYPTION_KEY, got nil")
	}
}
