package pty

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/ports"
)

func newTestManager() *Manager {
	return NewManager(WithCommand("sh"))
}

func TestSpawn(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	if handle.PID <= 0 {
		t.Fatalf("expected positive PID, got %d", handle.PID)
	}
	if handle.PTY == nil {
		t.Fatal("expected non-nil PTY file")
	}
	if handle.Done == nil {
		t.Fatal("expected non-nil Done channel")
	}
}

func TestGetHandle(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	got, ok := m.GetHandle(handle.PID)
	if !ok {
		t.Fatal("expected to find handle")
	}
	if got.PID != handle.PID {
		t.Fatalf("expected PID %d, got %d", handle.PID, got.PID)
	}

	_, ok = m.GetHandle(999999)
	if ok {
		t.Fatal("expected not to find handle for non-existent PID")
	}
}

func TestIsAlive(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	if !m.IsAlive(handle.PID) {
		t.Fatal("expected process to be alive")
	}

	if err := m.Kill(handle.PID); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
	// Wait a bit for the process to die
	time.Sleep(100 * time.Millisecond)

	if m.IsAlive(handle.PID) {
		t.Fatal("expected process to be dead after kill")
	}
}

func TestReadWrite(t *testing.T) {
	m := NewManager(WithCommand("cat"))
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	msg := "hello\n"
	_, err = handle.PTY.Write([]byte(msg))
	if err != nil {
		t.Fatalf("PTY write failed: %v", err)
	}

	buf := make([]byte, 256)
	// Set a read deadline so the test doesn't hang
	_ = handle.PTY.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := handle.PTY.Read(buf)
	if err != nil {
		t.Fatalf("PTY read failed: %v", err)
	}
	// cat echoes back through the PTY, so we should see our input
	output := string(buf[:n])
	if len(output) == 0 {
		t.Fatal("expected non-empty output from PTY read")
	}
}

func TestResize(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	err = m.Resize(handle.PID, 40, 120)
	if err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	err = m.Resize(999999, 40, 120)
	if err == nil {
		t.Fatal("expected error resizing non-existent process")
	}
}

func TestKill(t *testing.T) {
	m := newTestManager()
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	err = m.Kill(handle.PID)
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Process should be removed from internal map
	_, ok := m.GetHandle(handle.PID)
	if ok {
		t.Fatal("expected handle to be removed after kill")
	}

	// Killing again should return an error
	err = m.Kill(handle.PID)
	if err == nil {
		t.Fatal("expected error killing already-killed process")
	}
}

func TestWithCommandOptionChangesSpawnedProcess(t *testing.T) {
	// Verify that WithCommand controls which binary runs.
	// "echo hello" exits immediately and produces output — if the wrong
	// command ran, we'd see different output or a spawn failure.
	m := NewManager(WithCommand("sh"), WithFixedArgs("-c", "echo hello_from_custom_cmd"))
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn with custom command failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	output := readAllPTY(t, handle)
	if !contains(output, "hello_from_custom_cmd") {
		t.Fatalf("expected 'hello_from_custom_cmd' in output, got: %s", output)
	}
}

func TestWithEnvOption(t *testing.T) {
	env := []string{"FOO=bar", "BAZ=qux"}
	m := NewManager(WithEnv(env))
	if m.baseEnv == nil {
		t.Fatal("expected baseEnv to be set")
	}
	if len(m.baseEnv) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(m.baseEnv))
	}
	if m.baseEnv[0] != "FOO=bar" {
		t.Fatalf("expected FOO=bar, got %q", m.baseEnv[0])
	}
}

func TestWithEnvSpawnUsesBaseEnv(t *testing.T) {
	// Use "sh" with env command to print environment
	baseEnv := []string{"CMUX_TEST_VAR=hello", "PATH=" + os.Getenv("PATH")}
	m := NewManager(WithCommand("sh"), WithFixedArgs("-c", "env"), WithEnv(baseEnv))
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	output := readAllPTY(t, handle)
	if !contains(output, "CMUX_TEST_VAR=hello") {
		t.Fatalf("expected CMUX_TEST_VAR=hello in output, got: %s", output)
	}
	// TERM and LANG should always be appended
	if !contains(output, "TERM=xterm-256color") {
		t.Fatalf("expected TERM=xterm-256color in output, got: %s", output)
	}
}

func TestWithEnvFiltersCLAUDECODE(t *testing.T) {
	baseEnv := []string{"CLAUDECODE=secret", "KEEP=yes", "PATH=" + os.Getenv("PATH")}
	m := NewManager(WithCommand("sh"), WithFixedArgs("-c", "env"), WithEnv(baseEnv))
	ctx := context.Background()

	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	output := readAllPTY(t, handle)
	if contains(output, "CLAUDECODE=secret") {
		t.Fatalf("expected CLAUDECODE to be filtered, but found in output: %s", output)
	}
	if !contains(output, "KEEP=yes") {
		t.Fatalf("expected KEEP=yes in output, got: %s", output)
	}
}

func TestWithEnvResolverOption(t *testing.T) {
	resolver := func() []string { return []string{"DYNAMIC=value"} }
	m := NewManager(WithEnvResolver(resolver))
	if m.envResolver == nil {
		t.Fatal("expected envResolver to be set")
	}
	got := m.envResolver()
	if len(got) != 1 || got[0] != "DYNAMIC=value" {
		t.Fatalf("expected [DYNAMIC=value], got %v", got)
	}
}

func TestWithEnvResolverTakesPrecedenceOverWithEnv(t *testing.T) {
	resolver := func() []string { return []string{"RESOLVED=new"} }
	m := NewManager(
		WithEnv([]string{"STATIC=old"}),
		WithEnvResolver(resolver),
	)
	if m.baseEnv == nil {
		t.Fatal("expected baseEnv to be set")
	}
	if len(m.baseEnv) != 1 || m.baseEnv[0] != "STATIC=old" {
		t.Fatalf("expected baseEnv [STATIC=old], got %v", m.baseEnv)
	}
	if m.envResolver == nil {
		t.Fatal("expected envResolver to be set")
	}
	got := m.envResolver()
	if len(got) != 1 || got[0] != "RESOLVED=new" {
		t.Fatalf("expected resolver to return [RESOLVED=new], got %v", got)
	}
}

func TestWithEnvResolverCalledEachTime(t *testing.T) {
	counter := 0
	resolver := func() []string {
		counter++
		return []string{fmt.Sprintf("CALL=%d", counter)}
	}
	m := NewManager(WithEnvResolver(resolver))

	first := m.envResolver()
	second := m.envResolver()

	if len(first) != 1 || first[0] != "CALL=1" {
		t.Fatalf("expected first call [CALL=1], got %v", first)
	}
	if len(second) != 1 || second[0] != "CALL=2" {
		t.Fatalf("expected second call [CALL=2], got %v", second)
	}
}

func readAllPTY(t *testing.T, handle *ports.PTYHandle) string {
	t.Helper()

	// Read PTY output concurrently — must drain before process exits and
	// the kernel closes the PTY slave side.
	ch := make(chan string, 1)
	go func() {
		var output strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := handle.PTY.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				ch <- output.String()
				return
			}
		}
	}()

	select {
	case data := <-ch:
		return data
	case <-time.After(5 * time.Second):
		t.Fatal("timed out reading PTY output")
		return ""
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}

