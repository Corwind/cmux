//go:build darwin

package pty

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/adapters/pty/sandbox"
)

func sandboxTestdataDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("sandbox", "testdata")
}

func TestSpawnWithSandbox(t *testing.T) {
	builder := sandbox.NewProfileBuilder(sandboxTestdataDir(t))
	m := NewManager(
		WithCommand("sh"),
		WithSandbox(builder),
	)

	ctx := context.Background()
	handle, err := m.Spawn(ctx, os.TempDir())
	if err != nil {
		t.Fatalf("Spawn with sandbox failed: %v", err)
	}
	defer func() { _ = m.Kill(handle.PID) }()

	if handle.PID <= 0 {
		t.Fatalf("expected positive PID, got %d", handle.PID)
	}
}

func TestSandboxDeniesFileWriteOutsideWorkDir(t *testing.T) {
	builder := sandbox.NewProfileBuilder(sandboxTestdataDir(t))
	workDir := t.TempDir()

	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("failed to resolve workdir: %v", err)
	}

	// Write a marker inside workdir (should succeed), then attempt to write
	// outside workdir (should be denied by sandbox). Use resolved paths.
	deniedFile := "/private/sandbox-test-denied-" + filepath.Base(resolvedWorkDir)
	markerFile := filepath.Join(resolvedWorkDir, "marker")
	script := "touch " + markerFile + "; touch " + deniedFile + " 2>/dev/null; exit 0"

	m := NewManager(
		WithCommand("sh"),
		WithFixedArgs("-c", script),
		WithSandbox(builder),
	)

	ctx := context.Background()
	handle, err := m.Spawn(ctx, workDir)
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// Drain PTY output so the process doesn't block
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := handle.PTY.Read(buf); err != nil {
				return
			}
		}
	}()

	select {
	case <-handle.Done:
	case <-time.After(5 * time.Second):
		_ = m.Kill(handle.PID)
		t.Fatal("process did not exit in time")
	}

	// Marker file should exist (write inside workdir allowed)
	if _, err := os.Stat(markerFile); err != nil {
		t.Fatalf("marker file missing (workdir write should succeed): %v", err)
	}

	// Denied file should NOT exist (write outside workdir blocked)
	if _, err := os.Stat(deniedFile); err == nil {
		os.Remove(deniedFile)
		t.Fatal("expected sandbox to deny file write outside working directory")
	}
}

func TestSandboxAllowsFileWriteInWorkDir(t *testing.T) {
	builder := sandbox.NewProfileBuilder(sandboxTestdataDir(t))
	workDir := t.TempDir()

	// Resolve symlinks so the touch command uses the same real path
	// that the sandbox profile allows (e.g., /var -> /private/var on macOS)
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("failed to resolve workdir symlinks: %v", err)
	}

	testFile := filepath.Join(resolvedWorkDir, "sandbox-test-allowed")
	m := NewManager(
		WithCommand("sh"),
		WithFixedArgs("-c", "touch "+testFile),
		WithSandbox(builder),
	)

	ctx := context.Background()
	handle, err := m.Spawn(ctx, workDir)
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// Drain PTY output so the process doesn't block
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := handle.PTY.Read(buf); err != nil {
				return
			}
		}
	}()

	select {
	case <-handle.Done:
	case <-time.After(5 * time.Second):
		_ = m.Kill(handle.PID)
		t.Fatal("process did not exit in time")
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("expected sandbox to allow file write in working directory: %v", err)
	}
}

