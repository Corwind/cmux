package pty

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"

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

type Manager struct {
	mu        sync.RWMutex
	processes map[int]*managedProcess
	command   string
	fixedArgs []string
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
	cmd := exec.Command(m.command, spawnArgs...)
	cmd.Dir = workingDir
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

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
