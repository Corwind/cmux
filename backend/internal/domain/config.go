package domain

type Config struct {
	Server        ServerConfig
	Sandbox       SandboxConfig
	Shell         ShellConfig
	Git           GitConfig
	Claude        ClaudeConfig
	Codex         CodexConfig
	Notifications NotificationConfig
	Env           map[string]string
	Harnesses     []string
}

// NotificationConfig is the root switch for system notifications. When
// Enabled is false, no harness ever surfaces a notification regardless of
// its own ClaudeConfig.NotificationsEnabled / CodexConfig.NotificationsEnabled
// setting.
type NotificationConfig struct {
	Enabled bool
}

type ServerConfig struct {
	Port   string
	DBPath string
}

type SandboxConfig struct {
	TemplateDir string
	Templates   []string
}

type ShellConfig struct {
	Path      string
	InitFiles []string
}

type GitConfig struct {
	WorktreesDir string
}

type ClaudeConfig struct {
	Model       string
	SectionName string
	// NotificationsEnabled only takes effect when NotificationConfig.Enabled
	// is also true — this is the per-harness half of that AND, not an
	// independent switch.
	NotificationsEnabled bool
}

type CodexConfig struct {
	Model       string
	SectionName string
	Home        string // overrides $HOME/.codex; empty means use the default
	// NotificationsEnabled only takes effect when NotificationConfig.Enabled
	// is also true — this is the per-harness half of that AND, not an
	// independent switch.
	NotificationsEnabled bool
}
