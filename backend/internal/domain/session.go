package domain

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type SessionStatus string

const (
	StatusRunning SessionStatus = "running"
	StatusStopped SessionStatus = "stopped"
)

type Session struct {
	ID              string
	Name            string
	WorkingDir      string
	Status          SessionStatus
	PID             int
	ClaudeSessionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewSession(name, workingDir string) (Session, error) {
	if workingDir == "" {
		return Session{}, fmt.Errorf("working directory cannot be empty")
	}

	if name == "" {
		name = filepath.Base(workingDir)
	}

	now := time.Now()
	return Session{
		ID:              uuid.New().String(),
		Name:            name,
		WorkingDir:      workingDir,
		Status:          StatusStopped,
		ClaudeSessionID: uuid.New().String(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}
