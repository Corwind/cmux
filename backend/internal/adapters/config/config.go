package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Corwind/cmux/backend/internal/domain"
)

type tomlConfig struct {
	Server        tomlServer        `toml:"server"`
	Sandbox       tomlSandbox       `toml:"sandbox"`
	Shell         tomlShell         `toml:"shell"`
	Git           tomlGit           `toml:"git"`
	Claude        tomlClaude        `toml:"claude"`
	Codex         tomlCodex         `toml:"codex"`
	Pi            tomlPi            `toml:"pi"`
	Notifications tomlNotifications `toml:"notifications"`
	Env           map[string]string `toml:"env"`
	Harnesses     []string          `toml:"harnesses"`
}

type tomlClaude struct {
	Model       string `toml:"model"`
	SectionName string `toml:"section_name"`
	// Pointer so an explicit `notifications_enabled = false` in the file is
	// distinguishable from the key being absent — a plain bool can't tell
	// "unset" from "false", and the override pattern below needs to know.
	NotificationsEnabled *bool `toml:"notifications_enabled"`
}

type tomlCodex struct {
	Model       string `toml:"model"`
	SectionName string `toml:"section_name"`
	Home        string `toml:"home"`
	// See tomlClaude.NotificationsEnabled for why this is a pointer.
	NotificationsEnabled *bool `toml:"notifications_enabled"`
}

type tomlPi struct {
	Model       string `toml:"model"`
	SectionName string `toml:"section_name"`
	Home        string `toml:"home"`
}

type tomlNotifications struct {
	// See tomlClaude.NotificationsEnabled for why this is a pointer.
	Enabled *bool `toml:"enabled"`
}

type tomlGit struct {
	WorktreesDir string `toml:"worktrees_dir"`
}

type tomlServer struct {
	Port   string `toml:"port"`
	DBPath string `toml:"db_path"`
}

type tomlSandbox struct {
	TemplateDir string   `toml:"template_dir"`
	Templates   []string `toml:"templates"`
}

type tomlShell struct {
	Path      string   `toml:"path"`
	InitFiles []string `toml:"init_files"`
}

// Load reads the cmux configuration from a TOML file and environment variables.
// Precedence (highest wins): config file > env var > default.
func Load() (domain.Config, error) {
	cfg := defaults()

	// Determine config file path
	configPath := os.Getenv("CMUX_CONFIG_PATH")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return domain.Config{}, fmt.Errorf("get home dir: %w", err)
		}
		configPath = filepath.Join(home, ".cmux", "config.toml")
	}

	// Apply env var overrides onto defaults
	applyEnvVars(&cfg)

	// Try to load config file on top
	if err := loadFile(configPath, &cfg); err != nil {
		return domain.Config{}, err
	}

	// Expand ~ in all path fields
	expandPaths(&cfg)

	return cfg, nil
}

func defaults() domain.Config {
	return domain.Config{
		Server: domain.ServerConfig{
			Port:   "2689",
			DBPath: "db/cmux.db",
		},
		Sandbox: domain.SandboxConfig{
			TemplateDir: "sandbox-profiles",
		},
		Git: domain.GitConfig{
			WorktreesDir: "~/.cmux/worktrees",
		},
		Claude: domain.ClaudeConfig{
			SectionName:          "Claude Code",
			NotificationsEnabled: true,
		},
		Codex: domain.CodexConfig{
			SectionName:          "Codex",
			NotificationsEnabled: true,
		},
		Pi: domain.PiConfig{
			SectionName: "Pi",
		},
		Notifications: domain.NotificationConfig{
			Enabled: true,
		},
		Harnesses: []string{"claude"},
	}
}

// applyBoolEnvVar sets *target from the named env var if it's set and parses
// as a bool (accepting strconv.ParseBool's usual forms: "true"/"false",
// "1"/"0", etc.); a missing or unparsable value leaves *target untouched.
func applyBoolEnvVar(name string, target *bool) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return
	}
	*target = b
}

func applyEnvVars(cfg *domain.Config) {
	if v := os.Getenv("CMUX_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("CMUX_DB_PATH"); v != "" {
		cfg.Server.DBPath = v
	}
	if v := os.Getenv("CMUX_SANDBOX_TEMPLATE_DIR"); v != "" {
		cfg.Sandbox.TemplateDir = v
	}
	if v := os.Getenv("CMUX_SANDBOX_TEMPLATES"); v != "" {
		cfg.Sandbox.Templates = strings.Split(v, ",")
	}
	applyBoolEnvVar("CMUX_NOTIFICATIONS_ENABLED", &cfg.Notifications.Enabled)
	applyBoolEnvVar("CMUX_CLAUDE_NOTIFICATIONS_ENABLED", &cfg.Claude.NotificationsEnabled)
	applyBoolEnvVar("CMUX_CODEX_NOTIFICATIONS_ENABLED", &cfg.Codex.NotificationsEnabled)
	if v := os.Getenv("CMUX_CLAUDE_MODEL"); v != "" {
		cfg.Claude.Model = v
	}
	if v := os.Getenv("CMUX_CODEX_MODEL"); v != "" {
		cfg.Codex.Model = v
	}
	if v := os.Getenv("CMUX_PI_MODEL"); v != "" {
		cfg.Pi.Model = v
	}
}

func loadFile(path string, cfg *domain.Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // missing config file is fine
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}

	var tc tomlConfig
	if err := toml.Unmarshal(data, &tc); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}

	// Config file values override env/defaults
	if tc.Server.Port != "" {
		cfg.Server.Port = tc.Server.Port
	}
	if tc.Server.DBPath != "" {
		cfg.Server.DBPath = tc.Server.DBPath
	}
	if tc.Sandbox.TemplateDir != "" {
		cfg.Sandbox.TemplateDir = tc.Sandbox.TemplateDir
	}
	if len(tc.Sandbox.Templates) > 0 {
		cfg.Sandbox.Templates = tc.Sandbox.Templates
	}
	if tc.Shell.Path != "" {
		cfg.Shell.Path = tc.Shell.Path
	}
	if len(tc.Shell.InitFiles) > 0 {
		cfg.Shell.InitFiles = tc.Shell.InitFiles
	}
	if len(tc.Env) > 0 {
		cfg.Env = tc.Env
	}
	if tc.Git.WorktreesDir != "" {
		cfg.Git.WorktreesDir = tc.Git.WorktreesDir
	}
	if tc.Claude.Model != "" {
		cfg.Claude.Model = tc.Claude.Model
	}
	if tc.Claude.SectionName != "" {
		cfg.Claude.SectionName = tc.Claude.SectionName
	}
	if tc.Claude.NotificationsEnabled != nil {
		cfg.Claude.NotificationsEnabled = *tc.Claude.NotificationsEnabled
	}
	if tc.Codex.Model != "" {
		cfg.Codex.Model = tc.Codex.Model
	}
	if tc.Codex.SectionName != "" {
		cfg.Codex.SectionName = tc.Codex.SectionName
	}
	if tc.Codex.Home != "" {
		cfg.Codex.Home = tc.Codex.Home
	}
	if tc.Codex.NotificationsEnabled != nil {
		cfg.Codex.NotificationsEnabled = *tc.Codex.NotificationsEnabled
	}
	if tc.Pi.Model != "" {
		cfg.Pi.Model = tc.Pi.Model
	}
	if tc.Pi.SectionName != "" {
		cfg.Pi.SectionName = tc.Pi.SectionName
	}
	if tc.Pi.Home != "" {
		cfg.Pi.Home = tc.Pi.Home
	}
	if tc.Notifications.Enabled != nil {
		cfg.Notifications.Enabled = *tc.Notifications.Enabled
	}
	if len(tc.Harnesses) > 0 {
		cfg.Harnesses = tc.Harnesses
	}

	return nil
}

func expandPaths(cfg *domain.Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	expandTilde := func(p string) string {
		if strings.HasPrefix(p, "~/") {
			return filepath.Join(home, p[2:])
		}
		if p == "~" {
			return home
		}
		return p
	}

	cfg.Server.DBPath = expandTilde(cfg.Server.DBPath)
	cfg.Sandbox.TemplateDir = expandTilde(cfg.Sandbox.TemplateDir)
	cfg.Shell.Path = expandTilde(cfg.Shell.Path)

	for i, f := range cfg.Shell.InitFiles {
		cfg.Shell.InitFiles[i] = expandTilde(f)
	}

	cfg.Git.WorktreesDir = expandTilde(cfg.Git.WorktreesDir)
}
