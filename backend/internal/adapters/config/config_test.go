package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Unset all env vars that could affect the result
	for _, key := range []string{"CMUX_CONFIG_PATH", "CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	// Point config path to a non-existent file so no TOML is loaded
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "3001" {
		t.Errorf("expected default port '3001', got %q", cfg.Server.Port)
	}
	if cfg.Server.DBPath != "db/cmux.db" {
		t.Errorf("expected default db_path 'db/cmux.db', got %q", cfg.Server.DBPath)
	}
	if cfg.Sandbox.TemplateDir != "sandbox-profiles" {
		t.Errorf("expected default template_dir 'sandbox-profiles', got %q", cfg.Sandbox.TemplateDir)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Point config to non-existent file
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	t.Setenv("CMUX_PORT", "8080")
	t.Setenv("CMUX_DB_PATH", "/tmp/test.db")
	t.Setenv("CMUX_SANDBOX_TEMPLATE_DIR", "/tmp/templates")
	t.Setenv("CMUX_SANDBOX_TEMPLATES", "a,b,c")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("expected port '8080', got %q", cfg.Server.Port)
	}
	if cfg.Server.DBPath != "/tmp/test.db" {
		t.Errorf("expected db_path '/tmp/test.db', got %q", cfg.Server.DBPath)
	}
	if cfg.Sandbox.TemplateDir != "/tmp/templates" {
		t.Errorf("expected template_dir '/tmp/templates', got %q", cfg.Sandbox.TemplateDir)
	}
	if len(cfg.Sandbox.Templates) != 3 || cfg.Sandbox.Templates[0] != "a" {
		t.Errorf("expected templates [a,b,c], got %v", cfg.Sandbox.Templates)
	}
}

func TestLoadWithTOMLFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[server]
port = "9090"
db_path = "/data/cmux.db"

[sandbox]
template_dir = "/etc/sandbox"
templates = ["net", "fs"]

[shell]
path = "/bin/bash"
init_files = ["/etc/profile"]

[env]
FOO = "bar"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Unset env vars so they don't interfere
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected port '9090', got %q", cfg.Server.Port)
	}
	if cfg.Server.DBPath != "/data/cmux.db" {
		t.Errorf("expected db_path '/data/cmux.db', got %q", cfg.Server.DBPath)
	}
	if cfg.Sandbox.TemplateDir != "/etc/sandbox" {
		t.Errorf("expected template_dir '/etc/sandbox', got %q", cfg.Sandbox.TemplateDir)
	}
	if len(cfg.Sandbox.Templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(cfg.Sandbox.Templates))
	}
	if cfg.Shell.Path != "/bin/bash" {
		t.Errorf("expected shell path '/bin/bash', got %q", cfg.Shell.Path)
	}
	if len(cfg.Shell.InitFiles) != 1 || cfg.Shell.InitFiles[0] != "/etc/profile" {
		t.Errorf("expected init_files [/etc/profile], got %v", cfg.Shell.InitFiles)
	}
	if cfg.Env["FOO"] != "bar" {
		t.Errorf("expected env FOO=bar, got %v", cfg.Env)
	}
}

func TestLoadTOMLOverridesEnvVars(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[server]
port = "7070"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Set env var to a different value — TOML should win
	t.Setenv("CMUX_PORT", "5555")
	t.Setenv("CMUX_CONFIG_PATH", configFile)
	// Clear others
	for _, key := range []string{"CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "7070" {
		t.Errorf("expected TOML port '7070' to override env var '5555', got %q", cfg.Server.Port)
	}
}

func TestLoadTildeExpansion(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[server]
db_path = "~/data/cmux.db"

[sandbox]
template_dir = "~/sandbox"

[shell]
path = "~/bin/zsh"
init_files = ["~/.zshrc", "~/.profile"]
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	home, _ := os.UserHomeDir()

	if cfg.Server.DBPath != filepath.Join(home, "data/cmux.db") {
		t.Errorf("expected expanded db_path, got %q", cfg.Server.DBPath)
	}
	if cfg.Sandbox.TemplateDir != filepath.Join(home, "sandbox") {
		t.Errorf("expected expanded template_dir, got %q", cfg.Sandbox.TemplateDir)
	}
	if cfg.Shell.Path != filepath.Join(home, "bin/zsh") {
		t.Errorf("expected expanded shell path, got %q", cfg.Shell.Path)
	}
	if len(cfg.Shell.InitFiles) != 2 || cfg.Shell.InitFiles[0] != filepath.Join(home, ".zshrc") {
		t.Errorf("expected expanded init_files, got %v", cfg.Shell.InitFiles)
	}
}

func TestLoadMissingConfigFileNotError(t *testing.T) {
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "does_not_exist.toml"))

	_, err := Load()
	if err != nil {
		t.Fatalf("expected no error for missing config file, got: %v", err)
	}
}

func TestLoadMalformedTOMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(configFile, []byte("this is not valid toml {{{}}}"), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for malformed TOML, got nil")
	}
}

func TestLoadPartialTOMLKeepsDefaults(t *testing.T) {
	// A config that only sets port should keep all other defaults intact
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[server]
port = "4000"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "4000" {
		t.Errorf("expected port '4000', got %q", cfg.Server.Port)
	}
	// Other fields should still have defaults
	if cfg.Server.DBPath != "db/cmux.db" {
		t.Errorf("expected default db_path 'db/cmux.db', got %q", cfg.Server.DBPath)
	}
	if cfg.Sandbox.TemplateDir != "sandbox-profiles" {
		t.Errorf("expected default template_dir 'sandbox-profiles', got %q", cfg.Sandbox.TemplateDir)
	}
}
