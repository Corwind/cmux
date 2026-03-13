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
	Spawn(ctx context.Context, workingDir string, args ...string) (*PTYHandle, error)
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
