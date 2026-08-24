package harness

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

// PiType identifies the Pi harness.
const PiType Type = "pi"

// PiHarness implements Harness for the pi CLI (github.com/earendil-works/pi-mono).
type PiHarness struct {
	model  string
	piHome string
}

// NewPiHarness constructs a PiHarness configured with the model and config
// directory from cfg. If cfg.Home is empty, it defaults to $HOME/.pi/agent
// (best-effort; degrades to an empty string if the home directory cannot be
// determined, rather than panicking).
func NewPiHarness(cfg domain.PiConfig) *PiHarness {
	piHome := cfg.Home
	if piHome == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			piHome = filepath.Join(homeDir, ".pi", "agent")
		}
	}
	return &PiHarness{model: cfg.Model, piHome: piHome}
}

// Type returns PiType.
func (h *PiHarness) Type() Type {
	return PiType
}

// BinaryName returns the name of the pi executable.
func (h *PiHarness) BinaryName() string {
	return "pi"
}

// HasResumeSupport reports that pi supports resuming sessions. Unlike
// Claude/Codex, pi uses the same --session-id flag for both starting a new
// session and resuming an existing one (see BuildSpawnArgs), so this only
// governs whether SessionService is willing to ask for a resume at all.
func (h *PiHarness) HasResumeSupport() bool {
	return true
}

// HasSkipPermissionsSupport reports that pi has nothing to bypass: per its
// own security model, pi has no built-in sandbox and no per-tool-call
// approval gate — built-in tools always run with the pi process's own
// permissions (see security.md, "No Built-in Sandbox"). SpawnIntent.SkipPermissions
// is therefore silently ignored in BuildSpawnArgs rather than mapped to a flag.
func (h *PiHarness) HasSkipPermissionsSupport() bool {
	return false
}

// HasModelSelection reports that pi supports --model.
func (h *PiHarness) HasModelSelection() bool {
	return true
}

// HasNotificationSupport reports that pi has no built-in OSC 9/777 or
// terminal-bell notification protocol to detect — notifications in pi are
// only available to TypeScript extensions via ctx.ui.notify, which never
// reaches cmux's PTY output stream.
func (h *PiHarness) HasNotificationSupport() bool {
	return false
}

// HasSandboxPathGrants reports that pi needs an extra sandbox path grant.
func (h *PiHarness) HasSandboxPathGrants() bool {
	return true
}

// HasExternalSessionIDMinting reports that pi does not mint its own session
// ID — --session-id accepts cmux's own UUID directly, creating a new session
// under that exact ID if none exists yet, or reusing it if one does.
func (h *PiHarness) HasExternalSessionIDMinting() bool {
	return false
}

// BuildSpawnArgs translates intent into pi CLI arguments. Both new and
// resumed sessions use the same --session-id flag; there is no separate
// "resume" flag/subcommand like Claude's --resume or Codex's `resume`.
func (h *PiHarness) BuildSpawnArgs(intent SpawnIntent) []string {
	args := []string{"--session-id", intent.SessionID}
	if h.model != "" {
		args = append(args, "--model", h.model)
	}
	return args
}

// EnvOverrides sets PI_CODING_AGENT_DIR when a custom Home was configured —
// pi has no CLI flag for its config directory, only this env var.
func (h *PiHarness) EnvOverrides() (stripKeys []string, setVars map[string]string) {
	if h.piHome == "" {
		return nil, nil
	}
	return nil, map[string]string{"PI_CODING_AGENT_DIR": h.piHome}
}

// NeedsOpenURLWrapper reports that pi does not need the `open` URL-handoff
// wrapper. Unconfirmed against a real OAuth login flow (`/login`) — pi's own
// docs don't state whether that flow auto-opens a browser; flip this to true
// if that turns out to be needed.
func (h *PiHarness) NeedsOpenURLWrapper() bool {
	return false
}

// SandboxPathGrants returns the pi-specific write grant for its config/
// session-state directory.
func (h *PiHarness) SandboxPathGrants() []PathGrant {
	return []PathGrant{
		{Path: "$HOME/.pi/agent", Recursive: true, Write: true},
	}
}

// ParseNotification is never called in practice: HasNotificationSupport
// always returns false for PiHarness. Implemented to satisfy the Harness
// interface.
func (h *PiHarness) ParseNotification(data []byte) (NotificationResult, bool) {
	return NotificationResult{}, false
}

// DiscoverSessionID is not supported: pi already accepts cmux's own session
// ID directly via --session-id and never needs post-spawn discovery.
func (h *PiHarness) DiscoverSessionID(workingDir string, notBefore time.Time) (string, error) {
	return "", errors.New("pi harness does not support external session id discovery")
}
