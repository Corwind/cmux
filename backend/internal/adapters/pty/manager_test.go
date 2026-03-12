package pty

import (
	"context"
	"os"
	"testing"
	"time"
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
	defer m.Kill(handle.PID)

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
	defer m.Kill(handle.PID)

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

	m.Kill(handle.PID)
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
	defer m.Kill(handle.PID)

	msg := "hello\n"
	_, err = handle.PTY.Write([]byte(msg))
	if err != nil {
		t.Fatalf("PTY write failed: %v", err)
	}

	buf := make([]byte, 256)
	// Set a read deadline so the test doesn't hang
	handle.PTY.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	defer m.Kill(handle.PID)

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

func TestDefaultCommand(t *testing.T) {
	m := NewManager()
	if m.command != "claude" {
		t.Fatalf("expected default command 'claude', got %q", m.command)
	}
}

func TestWithCommandOption(t *testing.T) {
	m := NewManager(WithCommand("echo"))
	if m.command != "echo" {
		t.Fatalf("expected command 'echo', got %q", m.command)
	}
}
