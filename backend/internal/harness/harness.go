// Package harness defines the strategy interface used to abstract over
// coding-agent CLI backends (e.g. Claude Code). This file intentionally
// contains no concrete harness identity — concrete Type constants (such as
// ClaudeType) live in each harness's own file (e.g. claude.go), not here.
package harness

// Type identifies a concrete harness implementation (e.g. "claude"). Concrete
// values are defined alongside their implementation, not in this file.
type Type string

// SpawnIntent describes what a caller wants when spawning (or resuming) a
// harness process. It is translated by a Harness implementation into the
// concrete CLI arguments for that harness's binary.
type SpawnIntent struct {
	// SessionID is the cmux-minted UUID passed to the harness as its own
	// resume handle (e.g. via --session-id / --resume). Harnesses do not mint
	// their own session identifiers; cmux always supplies one.
	SessionID string

	// Resume indicates the harness process should resume an existing session
	// rather than start a new one. Callers must only set this true when the
	// harness's HasResumeSupport() returns true.
	Resume bool

	// SkipPermissions indicates the harness should run without interactive
	// permission prompts. Callers must only set this true when the harness's
	// HasSkipPermissionsSupport() returns true.
	SkipPermissions bool

	// Model selects the model the harness should use. It exists for
	// interface generality across harnesses; a given implementation may
	// ignore it in favor of a baked-in model (as ClaudeHarness does).
	Model string
}

// PathGrant describes a filesystem path a harness needs access to inside the
// sandbox profile.
type PathGrant struct {
	// Path may contain the literal substring "$HOME", which is substituted
	// with the actual home directory at sandbox-profile build time.
	Path string

	// Recursive selects a subpath/dir-tree grant rather than a literal/
	// single-file grant.
	Recursive bool

	// Write indicates the grant permits writes (in addition to reads).
	Write bool
}

// NotificationResult is the outcome of parsing a chunk of PTY output for a
// harness-specific notification sequence.
type NotificationResult struct {
	// Message is the human-readable notification text.
	Message string

	// EventType is one of "waiting_input", "task_complete", or "generic".
	EventType string
}

// Harness is a strategy for a single coding-agent CLI backend. Implementations
// expose feature-discovery methods (HasX) that callers must consult before
// relying on the corresponding behavior.
//
// EnvOverrides and NeedsOpenURLWrapper are deliberately unconditional (not
// gated by a paired HasX) because every harness has some answer for them,
// even if it's a no-op. SandboxPathGrants and ParseNotification, by contrast,
// are only valid to call when the corresponding HasSandboxPathGrants /
// HasNotificationSupport method returns true.
type Harness interface {
	// Type returns the harness's identity.
	Type() Type

	// BinaryName returns the name of the executable to spawn.
	BinaryName() string

	// BuildSpawnArgs translates a SpawnIntent into concrete CLI arguments.
	BuildSpawnArgs(intent SpawnIntent) []string

	// HasResumeSupport reports whether the harness supports resuming a prior
	// session.
	HasResumeSupport() bool

	// HasSkipPermissionsSupport reports whether the harness supports running
	// without interactive permission prompts.
	HasSkipPermissionsSupport() bool

	// HasModelSelection reports whether the harness supports selecting a
	// model.
	HasModelSelection() bool

	// HasNotificationSupport reports whether the harness emits notification
	// sequences that ParseNotification can parse.
	HasNotificationSupport() bool

	// HasSandboxPathGrants reports whether the harness needs extra sandbox
	// path grants beyond the generic ones.
	HasSandboxPathGrants() bool

	// EnvOverrides returns environment variable keys to strip from the
	// spawned process's environment, and variables to set.
	EnvOverrides() (stripKeys []string, setVars map[string]string)

	// NeedsOpenURLWrapper reports whether the harness requires an `open`
	// URL-handoff wrapper to be available in the sandbox.
	NeedsOpenURLWrapper() bool

	// SandboxPathGrants returns the harness-specific sandbox path grants.
	// Only valid to call when HasSandboxPathGrants returns true.
	SandboxPathGrants() []PathGrant

	// ParseNotification scans a chunk of PTY output for a harness-specific
	// notification sequence. Only valid to call when HasNotificationSupport
	// returns true.
	ParseNotification(data []byte) (NotificationResult, bool)
}
