package domain

import (
	"testing"
)

func TestNewSession_Valid(t *testing.T) {
	s, err := NewSession("my-session", "/tmp", "claude")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.Name != "my-session" {
		t.Errorf("expected name 'my-session', got %q", s.Name)
	}
	if s.WorkingDir != "/tmp" {
		t.Errorf("expected working dir '/tmp', got %q", s.WorkingDir)
	}
	if s.Status != StatusStopped {
		t.Errorf("expected status %q, got %q", StatusStopped, s.Status)
	}
	if s.ID == "" {
		t.Error("expected non-empty ID")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestNewSession_EmptyName_DefaultsToDirectoryBasename(t *testing.T) {
	s, err := NewSession("", "/home/user/my-project", "claude")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", s.Name)
	}
}

func TestNewSession_EmptyWorkingDir(t *testing.T) {
	_, err := NewSession("my-session", "", "claude")
	if err == nil {
		t.Fatal("expected error for empty working directory")
	}
}

func TestNewSession_UniqueIDs(t *testing.T) {
	s1, _ := NewSession("a", "/tmp", "claude")
	s2, _ := NewSession("b", "/tmp", "claude")
	if s1.ID == s2.ID {
		t.Error("expected unique IDs for different sessions")
	}
}

func TestNewSession_SetsHarnessType(t *testing.T) {
	s, err := NewSession("my-session", "/tmp", "claude")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.HarnessType != "claude" {
		t.Errorf("expected harness type 'claude', got %q", s.HarnessType)
	}
}

func TestSessionStatus_ProvisioningConstant(t *testing.T) {
	if StatusProvisioning != SessionStatus("provisioning") {
		t.Errorf("expected StatusProvisioning to be 'provisioning', got %q", StatusProvisioning)
	}
}

func TestSessionStatus_FailedConstant(t *testing.T) {
	if StatusFailed != SessionStatus("failed") {
		t.Errorf("expected StatusFailed to be 'failed', got %q", StatusFailed)
	}
}

func TestSession_ErrorFieldDefaultsEmpty(t *testing.T) {
	s, err := NewSession("test", "/tmp", "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Error != "" {
		t.Errorf("expected empty Error field, got %q", s.Error)
	}
}

func TestSession_ErrorFieldCanBeSet(t *testing.T) {
	s, _ := NewSession("test", "/tmp", "claude")
	s.Error = "worktree creation failed: git error"
	if s.Error != "worktree creation failed: git error" {
		t.Errorf("expected error message to be set, got %q", s.Error)
	}
}

func TestSession_ProvisioningStatus(t *testing.T) {
	s, _ := NewSession("test", "/tmp", "claude")
	s.Status = StatusProvisioning
	if s.Status != StatusProvisioning {
		t.Errorf("expected status %q, got %q", StatusProvisioning, s.Status)
	}
}

func TestSession_FailedStatusWithError(t *testing.T) {
	s, _ := NewSession("test", "/tmp", "claude")
	s.Status = StatusFailed
	s.Error = "something went wrong"
	if s.Status != StatusFailed {
		t.Errorf("expected status %q, got %q", StatusFailed, s.Status)
	}
	if s.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %q", s.Error)
	}
}
