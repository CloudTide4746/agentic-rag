package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDoesNotRequireDotEnv(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
app:
  name: test
jwt:
  secret: test-secret
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.App.Name != "test" {
		t.Fatalf("app name = %q, want test", cfg.App.Name)
	}
}

func TestLoadConfigUsesEnvironmentForDatabasePassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "test-db-password")
	t.Setenv("JWT_SECRET", "test-jwt-secret")

	configPath, err := filepath.Abs("config.yaml")
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Database.Password != "test-db-password" {
		t.Fatalf("database password = %q, want value from DB_PASSWORD", cfg.Database.Password)
	}
	if cfg.Database.Password == "${DB_PASSWORD}" {
		t.Fatal("database password remained an unexpanded placeholder")
	}
}

func TestLoadConfigReadsRedisMinIdleConns(t *testing.T) {
	configPath, err := filepath.Abs("config.yaml")
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Redis.MinIdleConns != 5 {
		t.Fatalf("Redis.MinIdleConns = %d, want 5", cfg.Redis.MinIdleConns)
	}
}
