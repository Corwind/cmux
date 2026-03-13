package sandbox

import (
	"fmt"
	"os"
	"strings"
)

// ProfileConfig holds the parameters for building a sandbox profile.
type ProfileConfig struct {
	WorkingDir    string
	TemplateNames []string
	HomeDir       string
}

// ProfileBuilder assembles SBPL sandbox profiles from a base set of rules
// and optional template fragments.
type ProfileBuilder struct {
	templateDir string
}

// NewProfileBuilder creates a ProfileBuilder that loads templates from templateDir.
func NewProfileBuilder(templateDir string) *ProfileBuilder {
	return &ProfileBuilder{
		templateDir: templateDir,
	}
}

// Build assembles a complete SBPL profile string from the base rules,
// working directory permissions, and any requested template fragments.
func (pb *ProfileBuilder) Build(cfg ProfileConfig) (string, error) {
	var templateFragments []string
	for _, name := range cfg.TemplateNames {
		if err := validateTemplateName(name); err != nil {
			return "", fmt.Errorf("build profile: %w", err)
		}
		tmpl, err := pb.LoadTemplate(name)
		if err != nil {
			return "", fmt.Errorf("build profile: %w", err)
		}
		templateFragments = append(templateFragments, ";; template: "+name+"\n"+tmpl.Content)
	}
	return buildProfile(templateFragments), nil
}

// Params returns the parameter map for sandbox-exec -D flags.
func (pb *ProfileBuilder) Params(cfg ProfileConfig) map[string]string {
	homeDir := cfg.HomeDir
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}

	return map[string]string{
		"WORKING_DIR": cfg.WorkingDir,
		"HOME_DIR":    homeDir,
	}
}

// buildProfile assembles the complete SBPL profile with deny-by-default,
// minimal system access, working directory read/write, and template fragments.
func buildProfile(templateFragments []string) string {
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")

	// Process execution fundamentals
	b.WriteString("\n;; process execution\n")
	b.WriteString("(allow process-exec*)\n")
	b.WriteString("(allow process-fork)\n")
	b.WriteString("(allow signal)\n")

	// PTY support
	b.WriteString("\n;; PTY support\n")
	b.WriteString("(allow pseudo-tty)\n")
	b.WriteString("(allow file-ioctl)\n")

	// IPC and system
	b.WriteString("\n;; IPC and system\n")
	b.WriteString("(allow mach-lookup)\n")
	b.WriteString("(allow sysctl-read)\n")

	// Network (Claude Code needs to call Anthropic API)
	b.WriteString("\n;; network\n")
	b.WriteString("(allow network-outbound)\n")
	b.WriteString("(allow system-socket)\n")

	// File metadata everywhere (needed for path resolution, stat, etc.)
	b.WriteString("\n;; file metadata (needed for path resolution)\n")
	b.WriteString("(allow file-read-metadata)\n")

	// Root directory (needed for readdir by Node.js path resolution)
	b.WriteString("\n;; root directory listing\n")
	b.WriteString(`(allow file-read* (literal "/"))` + "\n")

	// System paths (read-only)
	b.WriteString("\n;; system paths (read-only)\n")
	b.WriteString(`(allow file-read* (subpath "/usr"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/System"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/Library"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/bin"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/sbin"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/opt"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/private"))` + "\n")
	b.WriteString(`(allow file-read* (subpath "/dev"))` + "\n")

	// Home directory (read-only — write restricted to specific subdirs below)
	b.WriteString("\n;; home directory (read-only)\n")
	b.WriteString(`(allow file-read* (subpath (param "HOME_DIR")))` + "\n")

	// Working directory (read)
	b.WriteString("\n;; working directory (read)\n")
	b.WriteString(`(allow file-read* (subpath (param "WORKING_DIR")))` + "\n")

	// Device files (PTY, null, random)
	b.WriteString("\n;; device files (read/write)\n")
	b.WriteString(`(allow file-write* (subpath "/dev"))` + "\n")

	// Temp directories (write) — needed for Node.js, build tools, etc.
	b.WriteString("\n;; temp directories (write)\n")
	b.WriteString(`(allow file-write* (subpath "/tmp"))` + "\n")
	b.WriteString(`(allow file-write* (subpath "/private/tmp"))` + "\n")
	b.WriteString(`(allow file-write* (subpath "/private/var/folders"))` + "\n")

	// Working directory (write)
	b.WriteString("\n;; working directory (write)\n")
	b.WriteString(`(allow file-write* (subpath (param "WORKING_DIR")))` + "\n")

	// Claude Code config — ~/.claude and ~/.config (write)
	b.WriteString("\n;; claude config (write)\n")
	b.WriteString(`(allow file-write* (subpath (string-append (param "HOME_DIR") "/.claude")))` + "\n")
	b.WriteString(`(allow file-write* (subpath (string-append (param "HOME_DIR") "/.config")))` + "\n")

	// Template fragments (additional access rules)
	for _, fragment := range templateFragments {
		b.WriteString("\n" + fragment + "\n")
	}

	return b.String()
}

// BuildWithContent assembles a complete SBPL profile string from the base rules,
// working directory permissions, and raw template content strings (instead of loading from files).
func (pb *ProfileBuilder) BuildWithContent(cfg ProfileConfig, templateContents []string) (string, error) {
	var templateFragments []string
	for i, content := range templateContents {
		if err := validateTemplate(content); err != nil {
			return "", fmt.Errorf("build profile: template content %d: %w", i, err)
		}
		templateFragments = append(templateFragments, ";; inline template\n"+strings.TrimSpace(content))
	}
	return buildProfile(templateFragments), nil
}

// validateTemplateName ensures the template name is safe to embed in a profile comment.
// It rejects names containing newlines or other characters that could inject SBPL directives.
func validateTemplateName(name string) error {
	if strings.ContainsAny(name, "\n\r") {
		return fmt.Errorf("template name %q contains invalid characters", name)
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("template name %q contains path traversal characters", name)
	}
	return nil
}

