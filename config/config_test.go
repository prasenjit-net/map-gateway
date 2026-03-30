package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prasenjit-net/mcp-gateway/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want :8080", cfg.ListenAddr)
	}
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q, want ./data", cfg.DataDir)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.MaxResponseBytes != 1048576 {
		t.Errorf("MaxResponseBytes = %d, want 1048576", cfg.MaxResponseBytes)
	}
	if cfg.OpenAIModel != "gpt-4o" {
		t.Errorf("OpenAIModel = %q, want gpt-4o", cfg.OpenAIModel)
	}
	if cfg.AdminSessionTTL != 24 {
		t.Errorf("AdminSessionTTL = %d, want 24", cfg.AdminSessionTTL)
	}
	if len(cfg.CORS.AllowedOrigins) != 0 {
		t.Errorf("CORS.AllowedOrigins = %v, want empty", cfg.CORS.AllowedOrigins)
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		t.Error("CORS.AllowedMethods should not be empty")
	}
	if len(cfg.CORS.AllowedHeaders) == 0 {
		t.Error("CORS.AllowedHeaders should not be empty")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := config.Load("/tmp/does-not-exist-mcp-gateway-test.toml")
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	// Should return defaults.
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want :8080", cfg.ListenAddr)
	}
}

func TestLoadTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
listen_addr = ":9090"
data_dir = "/tmp/test-data"
log_level = "debug"
max_response_bytes = 2097152
admin_password = "secret123"
admin_session_ttl_hours = 48

[cors]
allowed_origins = ["https://example.com", "https://app.example.com"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.DataDir != "/tmp/test-data" {
		t.Errorf("DataDir = %q", cfg.DataDir)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
	if cfg.MaxResponseBytes != 2097152 {
		t.Errorf("MaxResponseBytes = %d", cfg.MaxResponseBytes)
	}
	if cfg.AdminPassword != "secret123" {
		t.Errorf("AdminPassword = %q", cfg.AdminPassword)
	}
	if cfg.AdminSessionTTL != 48 {
		t.Errorf("AdminSessionTTL = %d", cfg.AdminSessionTTL)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Errorf("CORS.AllowedOrigins = %v, want 2 entries", cfg.CORS.AllowedOrigins)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":7070")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("GATEWAY_SECRET", "mysecret")
	t.Setenv("ADMIN_PASSWORD", "adminpass")
	t.Setenv("MAX_RESPONSE_BYTES", "512000")

	cfg, err := config.Load("/tmp/does-not-exist.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":7070" {
		t.Errorf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
	if cfg.GatewaySecret != "mysecret" {
		t.Errorf("GatewaySecret = %q", cfg.GatewaySecret)
	}
	if cfg.AdminPassword != "adminpass" {
		t.Errorf("AdminPassword = %q", cfg.AdminPassword)
	}
	if cfg.MaxResponseBytes != 512000 {
		t.Errorf("MaxResponseBytes = %d", cfg.MaxResponseBytes)
	}
}

func TestCORSDefaultsAppliedAfterLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	// Config without [cors] section: methods/headers should get defaults.
	if err := os.WriteFile(path, []byte(`listen_addr = ":8080"`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		t.Error("AllowedMethods should have defaults")
	}
	if len(cfg.CORS.AllowedHeaders) == 0 {
		t.Error("AllowedHeaders should have defaults")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestTLSDefaultPaths(t *testing.T) {
	cfg, err := config.Load("/tmp/no-file.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLS.CertFile == "" {
		t.Error("TLS.CertFile should have a default")
	}
	if cfg.TLS.KeyFile == "" {
		t.Error("TLS.KeyFile should have a default")
	}
}
