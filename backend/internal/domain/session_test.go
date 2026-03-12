package domain

import (
	"testing"
)

func TestNewSession_Valid(t *testing.T) {
	s, err := NewSession("my-session", "/tmp")
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
	s, err := NewSession("", "/home/user/my-project")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.Name != "my-project" {
		t.Errorf("expected name 'my-project', got %q", s.Name)
	}
}

func TestNewSession_EmptyWorkingDir(t *testing.T) {
	_, err := NewSession("my-session", "")
	if err == nil {
		t.Fatal("expected error for empty working directory")
	}
}

func TestNewSession_UniqueIDs(t *testing.T) {
	s1, _ := NewSession("a", "/tmp")
	s2, _ := NewSession("b", "/tmp")
	if s1.ID == s2.ID {
		t.Error("expected unique IDs for different sessions")
	}
}
