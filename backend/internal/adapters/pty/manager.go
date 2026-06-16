package pty

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/Corwind/cmux/backend/internal/adapters/pty/sandbox"
	"github.com/Corwind/cmux/backend/internal/ports"
	ptylib "github.com/creack/pty/v2"
)

type managedProcess struct {
	handle     *ports.PTYHandle
	cmd        *exec.Cmd
	wrapperDir string
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

func WithEnv(env []string) Option {
	return func(m *Manager) {
		m.baseEnv = env
	}
}

func WithEnvResolver(fn func() []string) Option {
	return func(m *Manager) {
		m.envResolver = fn
	}
}

type Manager struct {
	mu               sync.RWMutex
	processes        map[int]*managedProcess
	command          string
	fixedArgs        []string
	baseEnv          []string
	envResolver      func() []string
	sandboxBuilder   *sandbox.ProfileBuilder
	sandboxTemplates []string
	sandboxContent   []string
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
	var env []string
	if m.envResolver != nil {
		env = m.envResolver()
	} else if m.baseEnv != nil {
		env = m.baseEnv
	} else {
		env = os.Environ()
	}
	env = filterEnv(env, "CLAUDECODE")
	env = filterEnv(env, "TERM_PROGRAM")
	env = filterEnv(env, "TERM_PROGRAM_VERSION")
	env = filterEnv(env, "GHOSTTY_RESOURCES_DIR")
	env = append(env, "TERM=xterm-ghostty")
	env = append(env, "TERM_PROGRAM=ghostty")
	env = append(env, "TERM_PROGRAM_VERSION=1.0.0")
	env = append(env, "GHOSTTY_RESOURCES_DIR=/Applications/Ghostty.app/Contents/Resources")
	env = append(env, "LANG=en_US.UTF-8")

	// Install a per-session `open` wrapper so sandboxed Claude can open URLs
	// in the host browser via POST /api/open (Apple Events are blocked inside
	// sandbox-exec, so the real /usr/bin/open can't communicate with the browser).
	wrapperDir, wrapErr := installOpenWrapper(env)
	if wrapErr != nil {
		log.Printf("failed to install open wrapper: %v", wrapErr)
	} else {
		env = prependToPath(env, wrapperDir)
	}

	cmd.Env = env

	ptmx, err := ptylib.Start(cmd)
	if err != nil {
		if wrapperDir != "" {
			_ = os.RemoveAll(wrapperDir)
		}
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
	m.processes[handle.PID] = &managedProcess{handle: handle, cmd: cmd, wrapperDir: wrapperDir}
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

	if proc.wrapperDir != "" {
		_ = os.RemoveAll(proc.wrapperDir)
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
		if p.wrapperDir != "" {
			_ = os.RemoveAll(p.wrapperDir)
		}
		if err := p.handle.PTY.Close(); err != nil {
			log.Printf("failed to close PTY for pid %d: %v", p.handle.PID, err)
		}
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	}
}

// installOpenWrapper writes a tiny shell script named `open` into a temp
// directory and returns the directory path. The script proxies URL-open
// requests to POST /api/open on the cmux backend, which runs outside the
// sandbox and can invoke the real macOS `open` binary.
func installOpenWrapper(env []string) (string, error) {
	port := "2689"
	for _, e := range env {
		if strings.HasPrefix(e, "CMUX_PORT=") {
			port = e[len("CMUX_PORT="):]
			break
		}
	}

	dir, err := os.MkdirTemp("", "cmux-open-*")
	if err != nil {
		return "", err
	}

	script := "#!/bin/sh\n" +
		`curl -sf -X POST "http://127.0.0.1:` + port + `/api/open" --data-urlencode "url=$1" >/dev/null 2>&1` + "\n"

	path := filepath.Join(dir, "open")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// prependToPath inserts dir at the front of the PATH entry in env.
func prependToPath(env []string, dir string) []string {
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + dir + ":" + e[len("PATH="):]
			return env
		}
	}
	return append(env, "PATH="+dir+":/usr/bin:/bin")
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

// SetSandboxContent sets raw SBPL content strings to use for the next spawn.
// The content is cleared after use.
func (m *Manager) SetSandboxContent(contents []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sandboxContent = contents
}

func (m *Manager) buildSandboxCommand(workingDir string, originalArgs []string) (*exec.Cmd, error) {
	cfg := sandbox.ProfileConfig{
		WorkingDir:    workingDir,
		TemplateNames: m.sandboxTemplates,
	}

	// Detect git worktrees: if workingDir/.git is a file (gitlink), the shared
	// object DB lives in the main repo outside our sandbox. Grant r/w access to
	// the git common dir so git commands work inside the session.
	gitPath := filepath.Join(workingDir, ".git")
	if fi, err := os.Stat(gitPath); err == nil && !fi.IsDir() {
		if commonDir, err := resolveGitCommonDir(workingDir); err == nil && commonDir != "" {
			cfg.ExtraWritePaths = append(cfg.ExtraWritePaths, commonDir)
		}
	}

	var profile string
	var err error

	// Use sandboxContent if set, otherwise use template names from files
	m.mu.Lock()
	content := m.sandboxContent
	m.sandboxContent = nil
	m.mu.Unlock()

	if len(content) > 0 {
		profile, err = m.sandboxBuilder.BuildWithContent(cfg, content)
	} else {
		profile, err = m.sandboxBuilder.Build(cfg)
	}
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

// resolveGitCommonDir runs `git rev-parse --git-common-dir` inside a worktree
// and returns the symlink-resolved absolute path of the shared git dir.
func resolveGitCommonDir(worktreePath string) (string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == ".git" {
		return "", fmt.Errorf("not a linked worktree")
	}
	// The path may be relative to the repo; make it absolute
	if !filepath.IsAbs(raw) {
		raw = filepath.Join(worktreePath, raw)
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return raw, nil
	}
	return resolved, nil
}
