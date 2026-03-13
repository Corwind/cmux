package pty

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Corwind/cmux/backend/internal/adapters/pty/sandbox"
	"github.com/Corwind/cmux/backend/internal/ports"
	ptylib "github.com/creack/pty/v2"
)

type managedProcess struct {
	handle *ports.PTYHandle
	cmd    *exec.Cmd
}

type Option func(*Manager)

func WithCommand(command string) Option {
	return func(m *Manager) {
		m.command = command
	}
}

func WithFixedArgs(args ...string) Option {
	return func(m *Manager) {
		m.fixedArgs = args
	}
}

func WithSandbox(builder *sandbox.ProfileBuilder) Option {
	return func(m *Manager) {
		m.sandboxBuilder = builder
	}
}

func WithSandboxTemplates(templates ...string) Option {
	return func(m *Manager) {
		m.sandboxTemplates = templates
	}
}

type Manager struct {
	mu               sync.RWMutex
	processes        map[int]*managedProcess
	command          string
	fixedArgs        []string
	sandboxBuilder   *sandbox.ProfileBuilder
	sandboxTemplates []string
}

func NewManager(opts ...Option) *Manager {
	m := &Manager{
		processes: make(map[int]*managedProcess),
		command:   "claude",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Manager) Spawn(_ context.Context, workingDir string, args ...string) (*ports.PTYHandle, error) {
	spawnArgs := args
	if m.fixedArgs != nil {
		spawnArgs = m.fixedArgs
	}

	// Resolve symlinks so sandbox profile and cmd.Dir use the same real path
	// (e.g., /var/folders -> /private/var/folders on macOS)
	resolvedDir, err := filepath.EvalSymlinks(workingDir)
	if err != nil {
		resolvedDir = workingDir
	}

	var cmd *exec.Cmd
	if m.sandboxBuilder != nil {
		sandboxCmd, err := m.buildSandboxCommand(resolvedDir, spawnArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to build sandbox command: %w", err)
		}
		cmd = sandboxCmd
	} else {
		cmd = exec.Command(m.command, spawnArgs...)
	}

	cmd.Dir = resolvedDir
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")

	ptmx, err := ptylib.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	handle := &ports.PTYHandle{
		PTY:  ptmx,
		PID:  cmd.Process.Pid,
		Done: done,
	}

	m.mu.Lock()
	m.processes[handle.PID] = &managedProcess{handle: handle, cmd: cmd}
	m.mu.Unlock()

	return handle, nil
}

func filterEnv(env []string, exclude string) []string {
	prefix := exclude + "="
	var filtered []string
	for _, e := range env {
		if len(e) < len(prefix) || e[:len(prefix)] != prefix {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m *Manager) Resize(pid int, rows, cols uint16) error {
	m.mu.RLock()
	proc, ok := m.processes[pid]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("process %d not found", pid)
	}

	return ptylib.Setsize(proc.handle.PTY, &ptylib.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

func (m *Manager) Kill(pid int) error {
	m.mu.Lock()
	proc, ok := m.processes[pid]
	if ok {
		delete(m.processes, pid)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("process %d not found", pid)
	}

	if err := proc.handle.PTY.Close(); err != nil {
		log.Printf("failed to close PTY for pid %d: %v", pid, err)
	}
	return proc.cmd.Process.Signal(syscall.SIGTERM)
}

func (m *Manager) KillAll() {
	m.mu.Lock()
	procs := make([]*managedProcess, 0, len(m.processes))
	for _, p := range m.processes {
		procs = append(procs, p)
	}
	m.processes = make(map[int]*managedProcess)
	m.mu.Unlock()

	for _, p := range procs {
		if err := p.handle.PTY.Close(); err != nil {
			log.Printf("failed to close PTY for pid %d: %v", p.handle.PID, err)
		}
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	}
}

func (m *Manager) IsAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

func (m *Manager) GetHandle(pid int) (*ports.PTYHandle, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	proc, ok := m.processes[pid]
	if !ok {
		return nil, false
	}
	return proc.handle, true
}

func (m *Manager) buildSandboxCommand(workingDir string, originalArgs []string) (*exec.Cmd, error) {
	cfg := sandbox.ProfileConfig{
		WorkingDir:    workingDir,
		TemplateNames: m.sandboxTemplates,
	}

	profile, err := m.sandboxBuilder.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build sandbox profile: %w", err)
	}

	params := m.sandboxBuilder.Params(cfg)

	// Build sandbox-exec args: -p <profile> -D KEY=VALUE ... <command> <args...>
	sandboxArgs := []string{"-p", profile}
	for key, value := range params {
		sandboxArgs = append(sandboxArgs, "-D", key+"="+value)
	}
	sandboxArgs = append(sandboxArgs, m.command)
	sandboxArgs = append(sandboxArgs, originalArgs...)

	return exec.Command("sandbox-exec", sandboxArgs...), nil
}
