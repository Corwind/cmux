package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Unset all env vars that could affect the result
	for _, key := range []string{"CMUX_CONFIG_PATH", "CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES", "CMUX_CLAUDE_MODEL", "CMUX_CODEX_MODEL", "CMUX_PI_MODEL", "CMUX_NOTIFICATIONS_ENABLED", "CMUX_CLAUDE_NOTIFICATIONS_ENABLED", "CMUX_CODEX_NOTIFICATIONS_ENABLED"} {
		t.Setenv(key, "")
	}
	// Point config path to a non-existent file so no TOML is loaded
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "2689" {
		t.Errorf("expected default port '2689', got %q", cfg.Server.Port)
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

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES", "CMUX_CLAUDE_MODEL", "CMUX_CODEX_MODEL", "CMUX_PI_MODEL", "CMUX_NOTIFICATIONS_ENABLED", "CMUX_CLAUDE_NOTIFICATIONS_ENABLED", "CMUX_CODEX_NOTIFICATIONS_ENABLED"} {
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
	if cfg.Claude.Model != "" {
		t.Errorf("expected empty claude model by default, got %q", cfg.Claude.Model)
	}
	if cfg.Codex.Model != "" {
		t.Errorf("expected empty codex model by default, got %q", cfg.Codex.Model)
	}
	if cfg.Codex.Home != "" {
		t.Errorf("expected empty codex home by default, got %q", cfg.Codex.Home)
	}
	if cfg.Pi.Model != "" {
		t.Errorf("expected empty pi model by default, got %q", cfg.Pi.Model)
	}
	if cfg.Pi.Home != "" {
		t.Errorf("expected empty pi home by default, got %q", cfg.Pi.Home)
	}
	if !cfg.Notifications.Enabled {
		t.Error("expected notifications enabled by default")
	}
	if !cfg.Claude.NotificationsEnabled {
		t.Error("expected claude notifications enabled by default")
	}
	if !cfg.Codex.NotificationsEnabled {
		t.Error("expected codex notifications enabled by default")
	}
}

func TestLoadClaudeModelFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[claude]
model = "claude-opus-4-8"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES", "CMUX_CLAUDE_MODEL"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Claude.Model != "claude-opus-4-8" {
		t.Errorf("expected claude model 'claude-opus-4-8', got %q", cfg.Claude.Model)
	}
}

func TestLoadClaudeModelFromEnvVar(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CLAUDE_MODEL", "claude-haiku-4-5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Claude.Model != "claude-haiku-4-5" {
		t.Errorf("expected claude model 'claude-haiku-4-5', got %q", cfg.Claude.Model)
	}
}

func TestLoadClaudeModelTOMLOverridesEnvVar(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[claude]
model = "claude-sonnet-4-6"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CLAUDE_MODEL", "claude-haiku-4-5")
	t.Setenv("CMUX_CONFIG_PATH", configFile)
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Claude.Model != "claude-sonnet-4-6" {
		t.Errorf("expected TOML claude model 'claude-sonnet-4-6' to override env var, got %q", cfg.Claude.Model)
	}
}

func TestLoadHarnessesDefaults(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Harnesses) != 1 || cfg.Harnesses[0] != "claude" {
		t.Errorf("expected default harnesses ['claude'], got %v", cfg.Harnesses)
	}
	if cfg.Claude.SectionName != "Claude Code" {
		t.Errorf("expected default claude section_name 'Claude Code', got %q", cfg.Claude.SectionName)
	}
	if cfg.Codex.SectionName != "Codex" {
		t.Errorf("expected default codex section_name 'Codex', got %q", cfg.Codex.SectionName)
	}
	if cfg.Pi.SectionName != "Pi" {
		t.Errorf("expected default pi section_name 'Pi', got %q", cfg.Pi.SectionName)
	}
	if !cfg.Notifications.Enabled {
		t.Error("expected notifications enabled by default")
	}
	if !cfg.Claude.NotificationsEnabled {
		t.Error("expected claude notifications enabled by default")
	}
	if !cfg.Codex.NotificationsEnabled {
		t.Error("expected codex notifications enabled by default")
	}
}

func TestLoadHarnessesFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
harnesses = ["claude", "codex"]

[claude]
section_name = "Claude Code"

[codex]
section_name = "Codex"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Harnesses) != 2 || cfg.Harnesses[0] != "claude" || cfg.Harnesses[1] != "codex" {
		t.Errorf("expected harnesses ['claude', 'codex'], got %v", cfg.Harnesses)
	}
	if cfg.Claude.SectionName != "Claude Code" {
		t.Errorf("expected claude section_name 'Claude Code', got %q", cfg.Claude.SectionName)
	}
}

func TestLoadClaudeSectionNameOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[claude]
section_name = "My Claude"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Claude.SectionName != "My Claude" {
		t.Errorf("expected claude section_name 'My Claude', got %q", cfg.Claude.SectionName)
	}
	// harnesses should still default since not set in TOML
	if len(cfg.Harnesses) != 1 || cfg.Harnesses[0] != "claude" {
		t.Errorf("expected default harnesses ['claude'], got %v", cfg.Harnesses)
	}
}

func TestLoadCodexModelFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[codex]
model = "gpt-5-codex"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES", "CMUX_CODEX_MODEL"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.Model != "gpt-5-codex" {
		t.Errorf("expected codex model 'gpt-5-codex', got %q", cfg.Codex.Model)
	}
}

func TestLoadCodexModelFromEnvVar(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CODEX_MODEL", "gpt-5-codex-mini")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.Model != "gpt-5-codex-mini" {
		t.Errorf("expected codex model 'gpt-5-codex-mini', got %q", cfg.Codex.Model)
	}
}

func TestLoadCodexModelTOMLOverridesEnvVar(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[codex]
model = "gpt-5-codex"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CODEX_MODEL", "gpt-5-codex-mini")
	t.Setenv("CMUX_CONFIG_PATH", configFile)
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.Model != "gpt-5-codex" {
		t.Errorf("expected TOML codex model 'gpt-5-codex' to override env var, got %q", cfg.Codex.Model)
	}
}

func TestLoadCodexSectionNameOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[codex]
section_name = "My Codex"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.SectionName != "My Codex" {
		t.Errorf("expected codex section_name 'My Codex', got %q", cfg.Codex.SectionName)
	}
	// harnesses should still default since not set in TOML
	if len(cfg.Harnesses) != 1 || cfg.Harnesses[0] != "claude" {
		t.Errorf("expected default harnesses ['claude'], got %v", cfg.Harnesses)
	}
}

func TestLoadCodexHomeFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[codex]
home = "/tmp/custom-codex-home"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.Home != "/tmp/custom-codex-home" {
		t.Errorf("expected codex home '/tmp/custom-codex-home', got %q", cfg.Codex.Home)
	}
}

func TestLoadPiModelFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[pi]
model = "anthropic/claude-sonnet-5"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES", "CMUX_PI_MODEL"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Pi.Model != "anthropic/claude-sonnet-5" {
		t.Errorf("expected pi model 'anthropic/claude-sonnet-5', got %q", cfg.Pi.Model)
	}
}

func TestLoadPiModelFromEnvVar(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}
	t.Setenv("CMUX_PI_MODEL", "openai/gpt-5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Pi.Model != "openai/gpt-5" {
		t.Errorf("expected pi model 'openai/gpt-5', got %q", cfg.Pi.Model)
	}
}

func TestLoadPiModelTOMLOverridesEnvVar(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[pi]
model = "anthropic/claude-sonnet-5"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_PI_MODEL", "openai/gpt-5")
	t.Setenv("CMUX_CONFIG_PATH", configFile)
	for _, key := range []string{"CMUX_PORT", "CMUX_DB_PATH", "CMUX_SANDBOX_TEMPLATE_DIR", "CMUX_SANDBOX_TEMPLATES"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Pi.Model != "anthropic/claude-sonnet-5" {
		t.Errorf("expected TOML pi model 'anthropic/claude-sonnet-5' to override env var, got %q", cfg.Pi.Model)
	}
}

func TestLoadPiSectionNameOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[pi]
section_name = "My Pi"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Pi.SectionName != "My Pi" {
		t.Errorf("expected pi section_name 'My Pi', got %q", cfg.Pi.SectionName)
	}
	// harnesses should still default since not set in TOML
	if len(cfg.Harnesses) != 1 || cfg.Harnesses[0] != "claude" {
		t.Errorf("expected default harnesses ['claude'], got %v", cfg.Harnesses)
	}
}

func TestLoadPiHomeFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[pi]
home = "/tmp/custom-pi-home"
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Pi.Home != "/tmp/custom-pi-home" {
		t.Errorf("expected pi home '/tmp/custom-pi-home', got %q", cfg.Pi.Home)
	}
}

func TestLoadNotificationsDisabledFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[notifications]
enabled = false
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_NOTIFICATIONS_ENABLED", "")
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Notifications.Enabled {
		t.Error("expected notifications disabled via TOML")
	}
	// Per-harness defaults are untouched — the root switch composes with
	// them at harness-construction time, not by mutating these.
	if !cfg.Claude.NotificationsEnabled {
		t.Error("expected claude notifications_enabled to keep its own default of true")
	}
}

func TestLoadNotificationsEnabledFromEnvVar(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	t.Setenv("CMUX_NOTIFICATIONS_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Notifications.Enabled {
		t.Error("expected notifications disabled via CMUX_NOTIFICATIONS_ENABLED=false")
	}
}

func TestLoadNotificationsTOMLOverridesEnvVar(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[notifications]
enabled = true
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_NOTIFICATIONS_ENABLED", "false")
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !cfg.Notifications.Enabled {
		t.Error("expected TOML enabled=true to override env var CMUX_NOTIFICATIONS_ENABLED=false")
	}
}

func TestLoadClaudeNotificationsDisabledFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[claude]
notifications_enabled = false
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CLAUDE_NOTIFICATIONS_ENABLED", "")
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Claude.NotificationsEnabled {
		t.Error("expected claude notifications disabled via TOML")
	}
	if !cfg.Notifications.Enabled {
		t.Error("expected root notifications.enabled to keep its default of true")
	}
}

func TestLoadCodexNotificationsDisabledFromTOML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")

	tomlContent := `
[codex]
notifications_enabled = false
`
	if err := os.WriteFile(configFile, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CMUX_CODEX_NOTIFICATIONS_ENABLED", "")
	t.Setenv("CMUX_CONFIG_PATH", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.NotificationsEnabled {
		t.Error("expected codex notifications disabled via TOML")
	}
}

func TestLoadCodexNotificationsEnabledFromEnvVar(t *testing.T) {
	t.Setenv("CMUX_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.toml"))
	t.Setenv("CMUX_CODEX_NOTIFICATIONS_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Codex.NotificationsEnabled {
		t.Error("expected codex notifications disabled via CMUX_CODEX_NOTIFICATIONS_ENABLED=false")
	}
}
