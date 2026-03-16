package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

type SessionService struct {
	repo           ports.SessionRepository
	processManager ports.ProcessManager
	templateRepo   ports.TemplateRepository
	mu             sync.RWMutex
}

func NewSessionService(repo ports.SessionRepository, pm ports.ProcessManager, templateRepo ports.TemplateRepository) *SessionService {
	return &SessionService{
		repo:           repo,
		processManager: pm,
		templateRepo:   templateRepo,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, name, workingDir, templateID string, skipPermissions bool) (domain.Session, error) {
	session, err := domain.NewSession(name, workingDir)
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid session: %w", err)
	}

	// Resolve sandbox template content
	s.applySandboxContent(ctx, templateID)

	args := []string{"--session-id", session.ClaudeSessionID}
	if skipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	handle, err := s.processManager.Spawn(ctx, workingDir, args...)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to spawn process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	session.TemplateID = templateID

	if err := s.repo.Create(ctx, session); err != nil {
		_ = s.processManager.Kill(handle.PID)
		return domain.Session{}, fmt.Errorf("failed to store session: %w", err)
	}

	go s.watchProcess(session.ID, handle)

	return session, nil
}

func (s *SessionService) applySandboxContent(ctx context.Context, templateID string) {
	if s.templateRepo == nil {
		return
	}

	provider, ok := s.processManager.(ports.SandboxContentProvider)
	if !ok {
		return
	}

	var tmpl domain.SandboxTemplate
	var err error

	if templateID != "" {
		tmpl, err = s.templateRepo.Get(ctx, templateID)
	} else {
		tmpl, err = s.templateRepo.GetDefault(ctx)
	}

	if err != nil {
		if templateID != "" {
			log.Printf("failed to resolve template %s: %v", templateID, err)
		}
		return
	}

	provider.SetSandboxContent([]string{tmpl.Content})
}

func (s *SessionService) ResumeSession(ctx context.Context, id string) (domain.Session, error) {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if session.Status == domain.StatusRunning && s.processManager.IsAlive(session.PID) {
		return session, nil
	}

	// Reapply the sandbox template that was used when the session was created
	s.applySandboxContent(ctx, session.TemplateID)

	handle, err := s.processManager.Spawn(ctx, session.WorkingDir, "--resume", session.ClaudeSessionID)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to resume process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning
	if err := s.repo.Update(ctx, session); err != nil {
		_ = s.processManager.Kill(handle.PID)
		return domain.Session{}, fmt.Errorf("failed to update session: %w", err)
	}

	go s.watchProcess(session.ID, handle)

	return session, nil
}

func (s *SessionService) GetSession(ctx context.Context, id string) (domain.Session, error) {
	return s.repo.Get(ctx, id)
}

func (s *SessionService) ListSessions(ctx context.Context) ([]domain.Session, error) {
	sessions, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range sessions {
		if sessions[i].Status == domain.StatusRunning && !s.processManager.IsAlive(sessions[i].PID) {
			sessions[i].Status = domain.StatusStopped
			_ = s.repo.Update(ctx, sessions[i])
		}
	}

	return sessions, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, id string) error {
	session, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status == domain.StatusRunning {
		_ = s.processManager.Kill(session.PID)
	}

	return s.repo.Delete(ctx, id)
}

func (s *SessionService) GetPTYHandle(sessionID string) (*ports.PTYHandle, error) {
	session, err := s.repo.Get(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.StatusRunning {
		return nil, fmt.Errorf("session %s is not running", sessionID)
	}

	handle, ok := s.processManager.GetHandle(session.PID)
	if !ok {
		return nil, fmt.Errorf("no PTY handle for session %s", sessionID)
	}
	return handle, nil
}

func (s *SessionService) ResizePTY(pid int, rows, cols uint16) error {
	return s.processManager.Resize(pid, rows, cols)
}

func (s *SessionService) watchProcess(sessionID string, handle *ports.PTYHandle) {
	<-handle.Done
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	session, err := s.repo.Get(ctx, sessionID)
	if err != nil {
		return
	}
	session.Status = domain.StatusStopped
	_ = s.repo.Update(ctx, session)
}
