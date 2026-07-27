package harness

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

// CodexType identifies the Codex harness.
const CodexType Type = "codex"

// CodexHarness implements Harness for the Codex CLI.
type CodexHarness struct {
	model     string
	codexHome string
}

// NewCodexHarness constructs a CodexHarness configured with the model and
// codex home directory from cfg. If cfg.Home is empty, it defaults to
// $HOME/.codex (best-effort; degrades to an empty string if the home
// directory cannot be determined, rather than panicking).
func NewCodexHarness(cfg domain.CodexConfig) *CodexHarness {
	codexHome := cfg.Home
	if codexHome == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			codexHome = filepath.Join(homeDir, ".codex")
		}
	}
	return &CodexHarness{model: cfg.Model, codexHome: codexHome}
}

// Type returns CodexType.
func (h *CodexHarness) Type() Type {
	return CodexType
}

// BinaryName returns the name of the Codex executable.
func (h *CodexHarness) BinaryName() string {
	return "codex"
}

// HasResumeSupport reports that Codex supports resuming sessions.
func (h *CodexHarness) HasResumeSupport() bool {
	return true
}

// HasSkipPermissionsSupport reports that Codex supports
// --dangerously-bypass-approvals-and-sandbox.
func (h *CodexHarness) HasSkipPermissionsSupport() bool {
	return true
}

// HasModelSelection reports that Codex supports -m.
func (h *CodexHarness) HasModelSelection() bool {
	return true
}

// HasNotificationSupport reports that Codex emits parseable notification
// sequences.
func (h *CodexHarness) HasNotificationSupport() bool {
	return true
}

// HasSandboxPathGrants reports that Codex needs extra sandbox path grants.
func (h *CodexHarness) HasSandboxPathGrants() bool {
	return true
}

// HasExternalSessionIDMinting reports that Codex always mints its own
// session ID, requiring callers to discover it after spawn via
// DiscoverSessionID.
func (h *CodexHarness) HasExternalSessionIDMinting() bool {
	return true
}

// BuildSpawnArgs translates intent into Codex CLI arguments.
func (h *CodexHarness) BuildSpawnArgs(intent SpawnIntent) []string {
	var args []string
	if intent.Resume {
		args = append(args, "resume", intent.SessionID)
	}
	// Only ask Codex to notify on approval-requested — cmux's own dispatch
	// (websocket_handler.go) also filters to attention-needed events only,
	// but there's no reason to have Codex emit and cmux scan OSC 9 pings for
	// agent-turn-complete when they'd be dropped anyway.
	args = append(args,
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
	)
	if intent.SkipPermissions {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	if h.model != "" {
		args = append(args, "-m", h.model)
	}
	return args
}

// EnvOverrides reports that Codex needs no environment stripping or
// overrides.
func (h *CodexHarness) EnvOverrides() (stripKeys []string, setVars map[string]string) {
	return nil, nil
}

// NeedsOpenURLWrapper reports that Codex does not need the `open` URL-handoff
// wrapper.
func (h *CodexHarness) NeedsOpenURLWrapper() bool {
	return false
}

// SandboxPathGrants returns the Codex-specific write grant for its config/
// session-state directory.
func (h *CodexHarness) SandboxPathGrants() []PathGrant {
	return []PathGrant{
		{Path: "$HOME/.codex", Recursive: true, Write: true},
	}
}

// ParseNotification scans a PTY data chunk for notification sequences:
//   - OSC 9:   \x1b]9;message\x07 (ConEmu / Ghostty format)
//   - Standalone BEL \x07 (terminal_bell channel)
//
// Unlike ClaudeHarness, Codex does not emit OSC 777, so it is not checked
// here. Returns the notification result and whether a notification was
// found.
func (h *CodexHarness) ParseNotification(data []byte) (NotificationResult, bool) {
	if idx := bytes.Index(data, []byte("\x1b]9;")); idx >= 0 {
		rest := data[idx+4:]
		if end := oscEnd(rest); end >= 0 {
			msg := string(rest[:end])
			return NotificationResult{Message: msg, EventType: classifyNotification(msg)}, true
		}
	}

	if hasStandaloneBell(data) {
		return NotificationResult{Message: "Codex needs your attention", EventType: "waiting_input"}, true
	}

	return NotificationResult{}, false
}

// rolloutFirstLine models the first line of a Codex rollout-*.jsonl file,
// which contains the session ID and working directory for that session.
type rolloutFirstLine struct {
	Payload struct {
		SessionID string `json:"session_id"`
		Cwd       string `json:"cwd"`
	} `json:"payload"`
}

// DiscoverSessionID inspects Codex's on-disk rollout files under
// <codexHome>/sessions to find the session ID Codex minted for the most
// recently started session rooted at workingDir, ignoring anything started
// before notBefore.
func (h *CodexHarness) DiscoverSessionID(workingDir string, notBefore time.Time) (string, error) {
	sessionsDir := filepath.Join(h.codexHome, "sessions")

	var bestSessionID string
	var bestModTime time.Time
	found := false

	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Best-effort: skip entries we can't stat/read rather than aborting
			// the whole walk.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().Before(notBefore) {
			return nil
		}

		sessionID, cwd, ok := readRolloutFirstLine(path)
		if !ok || cwd != workingDir {
			return nil
		}

		if !found || info.ModTime().After(bestModTime) {
			bestSessionID = sessionID
			bestModTime = info.ModTime()
			found = true
		}
		return nil
	})

	if !found {
		return "", fmt.Errorf("no codex session found for working dir %q", workingDir)
	}
	return bestSessionID, nil
}

// readRolloutFirstLine reads only the first line of the rollout file at path
// and extracts the session ID and cwd from it.
func readRolloutFirstLine(path string) (sessionID, cwd string, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", "", false
	}

	var line rolloutFirstLine
	if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
		return "", "", false
	}
	return line.Payload.SessionID, line.Payload.Cwd, true
}
