package ports

import (
	"context"
	"os"
)

type PTYHandle struct {
	PTY  *os.File
	PID  int
	Done <-chan error
}

type ProcessManager interface {
	// Spawn starts a new process. harnessType identifies which harness
	// implementation's env overrides, sandbox path grants, and open-URL-wrapper
	// requirement should be applied (implementations resolve it via their own
	// harness registry); an empty string falls back to the implementation's
	// default harness.
	Spawn(ctx context.Context, workingDir string, harnessType string, args ...string) (*PTYHandle, error)
	Resize(pid int, rows, cols uint16) error
	Kill(pid int) error
	KillAll()
	IsAlive(pid int) bool
	GetHandle(pid int) (*PTYHandle, bool)
}

// SandboxContentProvider allows setting raw SBPL content strings
// to be used when building the sandbox profile for the next spawn.
type SandboxContentProvider interface {
	SetSandboxContent(contents []string)
}
