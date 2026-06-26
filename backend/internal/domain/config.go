package domain

type Config struct {
	Server  ServerConfig
	Sandbox SandboxConfig
	Shell   ShellConfig
	Git     GitConfig
	Claude  ClaudeConfig
	Env     map[string]string
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
	Model string
}
