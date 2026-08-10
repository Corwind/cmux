package harness

import (
	"bytes"
	"errors"
	"strings"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

// ClaudeType identifies the Claude Code harness.
const ClaudeType Type = "claude"

// ClaudeHarness implements Harness for the Claude Code CLI.
type ClaudeHarness struct {
	model                string
	notificationsEnabled bool
}

// NewClaudeHarness constructs a ClaudeHarness configured with the model from
// cfg, if any. cfg.NotificationsEnabled is expected to already be the
// effective value (root NotificationConfig.Enabled AND this harness's own
// setting) — ClaudeHarness itself doesn't know about the root switch.
func NewClaudeHarness(cfg domain.ClaudeConfig) *ClaudeHarness {
	return &ClaudeHarness{model: cfg.Model, notificationsEnabled: cfg.NotificationsEnabled}
}

// Type returns ClaudeType.
func (h *ClaudeHarness) Type() Type {
	return ClaudeType
}

// BinaryName returns the name of the Claude Code executable.
func (h *ClaudeHarness) BinaryName() string {
	return "claude"
}

// HasResumeSupport reports that Claude Code supports resuming sessions.
func (h *ClaudeHarness) HasResumeSupport() bool {
	return true
}

// HasSkipPermissionsSupport reports that Claude Code supports
// --dangerously-skip-permissions.
func (h *ClaudeHarness) HasSkipPermissionsSupport() bool {
	return true
}

// HasModelSelection reports that Claude Code supports --model.
func (h *ClaudeHarness) HasModelSelection() bool {
	return true
}

// HasNotificationSupport reports whether Claude Code notifications are
// enabled for this harness instance (see NewClaudeHarness) — Claude Code is
// always technically capable of emitting parseable notification sequences,
// but this gates whether cmux acts on them at all.
func (h *ClaudeHarness) HasNotificationSupport() bool {
	return h.notificationsEnabled
}

// HasSandboxPathGrants reports that Claude Code needs extra sandbox path
// grants.
func (h *ClaudeHarness) HasSandboxPathGrants() bool {
	return true
}

// BuildSpawnArgs translates intent into Claude Code CLI arguments, mirroring
// the exact argument construction previously duplicated across
// SessionService.CreateSession, provisionWorktree, and ResumeSession.
func (h *ClaudeHarness) BuildSpawnArgs(intent SpawnIntent) []string {
	var args []string
	if intent.Resume {
		args = append(args, "--resume", intent.SessionID)
	} else {
		args = append(args, "--session-id", intent.SessionID)
	}
	if intent.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	if h.model != "" {
		args = append(args, "--model", h.model)
	}
	return args
}

// EnvOverrides strips CLAUDECODE, the variable Claude Code itself sets to
// detect nested sessions. The other currently-stripped environment variables
// and the terminal-spoofing force-set block live in pty/manager.go because
// they are generic PTY presentation concerns, not Claude-specific, and
// deliberately stay there rather than moving here.
func (h *ClaudeHarness) EnvOverrides() (stripKeys []string, setVars map[string]string) {
	return []string{"CLAUDECODE"}, nil
}

// NeedsOpenURLWrapper reports that Claude Code needs the `open` URL-handoff
// wrapper available in the sandbox.
func (h *ClaudeHarness) NeedsOpenURLWrapper() bool {
	return true
}

// SandboxPathGrants returns the Claude-specific write grants. A read-only
// grant for ~/.config also exists in the sandbox profile builder, but it is
// generic (not Claude-specific) and stays there; only the write grant for
// .config is Claude-specific and belongs here.
func (h *ClaudeHarness) SandboxPathGrants() []PathGrant {
	return []PathGrant{
		{Path: "$HOME/.claude.json", Recursive: false, Write: true},
		{Path: "$HOME/.claude", Recursive: true, Write: true},
		{Path: "$HOME/.config", Recursive: true, Write: true},
	}
}

// ParseNotification scans a PTY data chunk for notification sequences:
//   - OSC 777: \x1b]777;notify;title;body\x07 (used by Claude Code)
//   - OSC 9:   \x1b]9;message\x07 (ConEmu / Ghostty format)
//   - Standalone BEL \x07 (terminal_bell channel)
//
// Returns the notification result and whether a notification was found.
// Handles BEL as OSC terminator vs. standalone correctly.
func (h *ClaudeHarness) ParseNotification(data []byte) (NotificationResult, bool) {
	// OSC 777: \x1b]777;notify;title;body\x07
	if idx := bytes.Index(data, []byte("\x1b]777;")); idx >= 0 {
		rest := data[idx+6:]
		if end := oscEnd(rest); end >= 0 {
			parts := strings.SplitN(string(rest[:end]), ";", 3)
			msg := string(rest[:end])
			if len(parts) >= 3 {
				msg = parts[2]
			} else if len(parts) > 0 {
				msg = parts[len(parts)-1]
			}
			return NotificationResult{Message: msg, EventType: classifyNotification(msg)}, true
		}
	}

	// OSC 9: \x1b]9;message\x07
	if idx := bytes.Index(data, []byte("\x1b]9;")); idx >= 0 {
		rest := data[idx+4:]
		if end := oscEnd(rest); end >= 0 {
			msg := string(rest[:end])
			return NotificationResult{Message: msg, EventType: classifyNotification(msg)}, true
		}
	}

	// Standalone BEL — only when not used as an OSC terminator.
	if hasStandaloneBell(data) {
		return NotificationResult{Message: "Claude needs your attention", EventType: "waiting_input"}, true
	}

	return NotificationResult{}, false
}

// oscEnd returns the index of the first BEL (0x07) or ESC-backslash in data,
// which terminates an OSC sequence. Returns -1 if none found.
func oscEnd(data []byte) int {
	for i, b := range data {
		switch {
		case b == 0x07:
			return i
		case b == 0x1b && i+1 < len(data) && data[i+1] == '\\':
			return i
		}
	}
	return -1
}

// hasStandaloneBell returns true when data contains a BEL byte that is not
// being used as an OSC sequence terminator (i.e. no \x1b] precedes it within
// the last 256 bytes).
func hasStandaloneBell(data []byte) bool {
	for i, b := range data {
		if b != 0x07 {
			continue
		}
		start := 0
		if i > 256 {
			start = i - 256
		}
		if !bytes.Contains(data[start:i], []byte{0x1b, ']'}) {
			return true
		}
	}
	return false
}

// HasExternalSessionIDMinting reports that Claude Code does not mint its own
// session ID — it always accepts the one cmux supplies via --session-id.
func (h *ClaudeHarness) HasExternalSessionIDMinting() bool {
	return false
}

// DiscoverSessionID is not supported: Claude Code already dictates its own
// session ID via --session-id and never needs post-spawn discovery.
func (h *ClaudeHarness) DiscoverSessionID(workingDir string, notBefore time.Time) (string, error) {
	return "", errors.New("claude harness does not support external session id discovery")
}

// classifyNotification maps a notification message to an event type.
func classifyNotification(msg string) string {
	lower := strings.ToLower(msg)
	for _, kw := range []string{"permission", "waiting", "input", "attention", "approve"} {
		if strings.Contains(lower, kw) {
			return "waiting_input"
		}
	}
	for _, kw := range []string{"done", "complete", "finished"} {
		if strings.Contains(lower, kw) {
			return "task_complete"
		}
	}
	return "generic"
}
