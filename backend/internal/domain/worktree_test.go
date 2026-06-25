package domain

import (
	"testing"
)

func TestWorktreeStatus_Constants(t *testing.T) {
	if WorktreeStatusReady != WorktreeStatus("ready") {
		t.Errorf("expected WorktreeStatusReady to be 'ready', got %q", WorktreeStatusReady)
	}
	if WorktreeStatusDeleting != WorktreeStatus("deleting") {
		t.Errorf("expected WorktreeStatusDeleting to be 'deleting', got %q", WorktreeStatusDeleting)
	}
}

func TestManagedWorktree_StatusDefault(t *testing.T) {
	var wt ManagedWorktree
	if wt.Status != "" {
		t.Errorf("expected zero-value Status to be empty, got %q", wt.Status)
	}
}

func TestManagedWorktree_StatusCanBeSet(t *testing.T) {
	wt := ManagedWorktree{Status: WorktreeStatusDeleting}
	if wt.Status != WorktreeStatusDeleting {
		t.Errorf("expected Status %q, got %q", WorktreeStatusDeleting, wt.Status)
	}
}
