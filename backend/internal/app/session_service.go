package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

type SessionService struct {
	repo           ports.SessionRepository
	processManager ports.ProcessManager
	mu             sync.RWMutex
}

func NewSessionService(repo ports.SessionRepository, pm ports.ProcessManager) *SessionService {
	return &SessionService{
		repo:           repo,
		processManager: pm,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, name, workingDir string) (domain.Session, error) {
	session, err := domain.NewSession(name, workingDir)
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid session: %w", err)
	}

	handle, err := s.processManager.Spawn(ctx, workingDir)
	if err != nil {
		return domain.Session{}, fmt.Errorf("failed to spawn process: %w", err)
	}

	session.PID = handle.PID
	session.Status = domain.StatusRunning

	if err := s.repo.Create(ctx, session); err != nil {
		_ = s.processManager.Kill(handle.PID)
		return domain.Session{}, fmt.Errorf("failed to store session: %w", err)
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
