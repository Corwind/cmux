package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Corwind/cmux/backend/internal/domain"
)

type tomlConfig struct {
	Server  tomlServer  `toml:"server"`
	Sandbox tomlSandbox `toml:"sandbox"`
	Shell   tomlShell   `toml:"shell"`
	Env     map[string]string `toml:"env"`
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
	}
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
}
