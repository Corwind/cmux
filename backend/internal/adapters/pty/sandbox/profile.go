package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// System read paths
	b.WriteString("\n;; system read paths\n")
	for _, rule := range systemReadPaths() {
		b.WriteString(rule + "\n")
	}

	// Claude binary path
	if binRules := claudeBinaryRules(); len(binRules) > 0 {
		b.WriteString("\n;; claude binary\n")
		for _, rule := range binRules {
			b.WriteString(rule + "\n")
		}
	}

	// Working directory
	b.WriteString("\n;; working directory\n")
	b.WriteString(`(allow file-read* (subpath (param "WORKING_DIR")))` + "\n")
	b.WriteString(`(allow file-write* (subpath (param "WORKING_DIR")))` + "\n")

	// Home directory for config
	b.WriteString("\n;; home directory config\n")
	b.WriteString(`(allow file-read* (subpath (param "HOME_DIR")))` + "\n")

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
	}
}

func systemReadPaths() []string {
	return []string{
		`(allow file-read* (subpath "/usr/lib"))`,
		`(allow file-read* (subpath "/usr/share"))`,
		`(allow file-read* (subpath "/System/Library"))`,
		`(allow file-read* (subpath "/usr/bin"))`,
		`(allow file-read* (subpath "/bin"))`,
		`(allow file-read* (literal "/dev/ptmx"))`,
		`(allow file-read* (regex #"/dev/tty.*"))`,
		`(allow file-read* (literal "/dev/null"))`,
		`(allow file-read* (literal "/dev/urandom"))`,
		`(allow file-read-metadata)`,
	}
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

func claudeBinaryRules() []string {
	path, err := exec.LookPath("claude")
	if err != nil {
		return nil
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}

	var rules []string
	dir := filepath.Dir(resolved)
	rules = append(rules, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, dir))

	// If the binary is in a nested path, also allow the parent
	parent := filepath.Dir(dir)
	if parent != dir {
		rules = append(rules, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, parent))
	}

	return rules
}
