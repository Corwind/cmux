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
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")

	// Base process permissions
	b.WriteString("\n;; base permissions\n")
	for _, rule := range basePermissions() {
		b.WriteString(rule + "\n")
	}

	// Allow all reads - the sandbox primarily restricts file writes
	b.WriteString("\n;; read access (unrestricted - sandbox focuses on write containment)\n")
	b.WriteString("(allow file-read*)\n")
	b.WriteString("(allow file-read-metadata)\n")

	// Device write access for PTY/stdout/stderr
	b.WriteString("\n;; device write access\n")
	b.WriteString(`(allow file-write* (subpath "/dev"))` + "\n")

	// Working directory write access
	b.WriteString("\n;; working directory\n")
	b.WriteString(`(allow file-write* (subpath (param "WORKING_DIR")))` + "\n")

	// Home directory write access for .claude config
	b.WriteString("\n;; home directory config\n")
	b.WriteString(`(allow file-write* (subpath (param "HOME_DIR")))` + "\n")

	// Template fragments
	for _, name := range cfg.TemplateNames {
		if err := validateTemplateName(name); err != nil {
			return "", fmt.Errorf("build profile: %w", err)
		}
		tmpl, err := pb.LoadTemplate(name)
		if err != nil {
			return "", fmt.Errorf("build profile: %w", err)
		}
		b.WriteString("\n;; template: " + name + "\n")
		b.WriteString(tmpl.Content + "\n")
	}

	return b.String(), nil
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

func basePermissions() []string {
	return []string{
		"(allow process-exec*)",
		"(allow process-fork)",
		"(allow pseudo-tty)",
		"(allow sysctl-read)",
		"(allow mach-lookup)",
		"(allow network-outbound)",
		"(allow system-socket)",
		"(allow signal)",
		"(allow file-ioctl)",
	}
}

// BuildWithContent assembles a complete SBPL profile string from the base rules,
// working directory permissions, and raw template content strings (instead of loading from files).
func (pb *ProfileBuilder) BuildWithContent(cfg ProfileConfig, templateContents []string) (string, error) {
	var b strings.Builder

	b.WriteString("(version 1)\n")
	b.WriteString("(deny default)\n")

	b.WriteString("\n;; base permissions\n")
	for _, rule := range basePermissions() {
		b.WriteString(rule + "\n")
	}

	b.WriteString("\n;; read access (unrestricted - sandbox focuses on write containment)\n")
	b.WriteString("(allow file-read*)\n")
	b.WriteString("(allow file-read-metadata)\n")

	b.WriteString("\n;; device write access\n")
	b.WriteString(`(allow file-write* (subpath "/dev"))` + "\n")

	b.WriteString("\n;; working directory\n")
	b.WriteString(`(allow file-write* (subpath (param "WORKING_DIR")))` + "\n")

	b.WriteString("\n;; home directory config\n")
	b.WriteString(`(allow file-write* (subpath (param "HOME_DIR")))` + "\n")

	for i, content := range templateContents {
		if err := validateTemplate(content); err != nil {
			return "", fmt.Errorf("build profile: template content %d: %w", i, err)
		}
		b.WriteString("\n;; inline template\n")
		b.WriteString(strings.TrimSpace(content) + "\n")
	}

	return b.String(), nil
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

