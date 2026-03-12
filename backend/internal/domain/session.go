package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionStatus string

const (
	StatusRunning SessionStatus = "running"
	StatusStopped SessionStatus = "stopped"
)

type Session struct {
	ID         string
	Name       string
	WorkingDir string
	Status     SessionStatus
	PID        int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewSession(name, workingDir string) (Session, error) {
	if name == "" {
		return Session{}, fmt.Errorf("session name cannot be empty")
	}
	if workingDir == "" {
		return Session{}, fmt.Errorf("working directory cannot be empty")
	}

	now := time.Now()
	return Session{
		ID:         uuid.New().String(),
		Name:       name,
		WorkingDir: workingDir,
		Status:     StatusStopped,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
