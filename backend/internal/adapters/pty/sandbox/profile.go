package sandbox

import (
	"fmt"
	"os"
	"strings"
)

// PathGrant describes a filesystem path grant to add to the sandbox profile,
// in addition to the base rules. Path may contain the literal substring
// "$HOME", which is substituted with the actual home directory (via the
// same "HOME_DIR" param mechanism used elsewhere in this file).
type PathGrant struct {
	Path      string
	Recursive bool
	Write     bool
}

// ProfileConfig holds the parameters for building a sandbox profile.
type ProfileConfig struct {
	WorkingDir      string
	TemplateNames   []string
	HomeDir         string
	ExtraWritePaths []string
	PathGrants      []PathGrant
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
	return buildProfile(templateFragments, cfg.ExtraWritePaths, cfg.PathGrants), nil
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
func buildProfile(templateFragments []string, extraWritePaths []string, pathGrants []PathGrant) string {
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
	b.WriteString("(allow ipc-posix-shm*)\n")
	b.WriteString("(allow user-preference-read)\n")
	// Allow `open https://...` to hand URLs off to the browser via Launch Services
	b.WriteString("(allow appleevent-send)\n")

	// Network (Claude Code needs to call Anthropic API)
	b.WriteString("\n;; network\n")
	b.WriteString("(allow network-outbound)\n")
	b.WriteString("(allow network-inbound)\n")
	b.WriteString("(allow network-bind)\n")
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

	// Home directory — only specific subdirs needed by Claude Code and its toolchain
	b.WriteString("\n;; home directory (selective read-only)\n")
	b.WriteString(`(allow file-read* (subpath (string-append (param "HOME_DIR") "/.config")))` + "\n")
	b.WriteString(`(allow file-read* (subpath (string-append (param "HOME_DIR") "/.local")))` + "\n")
	b.WriteString(`(allow file-read* (subpath (string-append (param "HOME_DIR") "/.npm")))` + "\n")
	b.WriteString(`(allow file-read* (subpath (string-append (param "HOME_DIR") "/.nvm")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.gitconfig")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.profile")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.bashrc")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.zshrc")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.zshenv")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.zprofile")))` + "\n")
	b.WriteString(`(allow file-read* (literal (string-append (param "HOME_DIR") "/.CFUserTextEncoding")))` + "\n")
	b.WriteString(`(allow file-read* (subpath (string-append (param "HOME_DIR") "/Library/Keychains")))` + "\n")

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

	// Harness-specific path grants (e.g. Claude Code config dirs)
	if len(pathGrants) > 0 {
		b.WriteString("\n;; harness path grants\n")
		for _, g := range pathGrants {
			path := g.Path
			if strings.HasPrefix(path, "$HOME") {
				path = `(string-append (param "HOME_DIR") "` + strings.TrimPrefix(path, "$HOME") + `")`
			} else {
				path = `"` + path + `"`
			}
			kind := "literal"
			if g.Recursive {
				kind = "subpath"
			}
			b.WriteString(`(allow file-read* (` + kind + " " + path + `))` + "\n")
			if g.Write {
				b.WriteString(`(allow file-write* (` + kind + " " + path + `))` + "\n")
			}
		}
	}

	// Extra write paths (e.g. shared git common dir for worktrees)
	if len(extraWritePaths) > 0 {
		b.WriteString("\n;; extra write paths (e.g. git common dir for worktrees)\n")
		for _, p := range extraWritePaths {
			escaped := strings.ReplaceAll(p, `"`, `\"`)
			b.WriteString(`(allow file-read* (subpath "` + escaped + `"))` + "\n")
			b.WriteString(`(allow file-write* (subpath "` + escaped + `"))` + "\n")
		}
	}

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
	return buildProfile(templateFragments, cfg.ExtraWritePaths, cfg.PathGrants), nil
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
