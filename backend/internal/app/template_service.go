package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/ports"
)

type TemplateService struct {
	repo ports.TemplateRepository
}

func NewTemplateService(repo ports.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) CreateTemplate(ctx context.Context, name, content string) (domain.SandboxTemplate, error) {
	tmpl, err := domain.NewSandboxTemplate(name, content)
	if err != nil {
		return domain.SandboxTemplate{}, fmt.Errorf("invalid template: %w", err)
	}

	if err := s.repo.Create(ctx, tmpl); err != nil {
		return domain.SandboxTemplate{}, fmt.Errorf("failed to store template: %w", err)
	}

	return tmpl, nil
}

func (s *TemplateService) GetTemplate(ctx context.Context, id string) (domain.SandboxTemplate, error) {
	return s.repo.Get(ctx, id)
}

func (s *TemplateService) ListTemplates(ctx context.Context) ([]domain.SandboxTemplate, error) {
	return s.repo.List(ctx)
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, id, name, content string) (domain.SandboxTemplate, error) {
	tmpl, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.SandboxTemplate{}, err
	}

	if name != "" {
		tmpl.Name = name
	}
	if content != "" {
		if err := domain.ValidateTemplateContent(content); err != nil {
			return domain.SandboxTemplate{}, fmt.Errorf("invalid content: %w", err)
		}
		tmpl.Content = content
	}

	tmpl.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tmpl); err != nil {
		return domain.SandboxTemplate{}, fmt.Errorf("failed to update template: %w", err)
	}

	return tmpl, nil
}

func (s *TemplateService) DeleteTemplate(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) SetDefault(ctx context.Context, id string) error {
	// Verify the template exists
	if _, err := s.repo.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.SetDefault(ctx, id)
}

func (s *TemplateService) ClearDefault(ctx context.Context) error {
	return s.repo.ClearDefault(ctx)
}

func (s *TemplateService) GetDefault(ctx context.Context) (domain.SandboxTemplate, error) {
	return s.repo.GetDefault(ctx)
}

func (s *TemplateService) ImportTemplate(ctx context.Context, name, content string) (domain.SandboxTemplate, error) {
	return s.CreateTemplate(ctx, name, content)
}

type TemplateExport struct {
	Name    string
	Content string
}

func (s *TemplateService) ExportTemplate(ctx context.Context, id string) (TemplateExport, error) {
	tmpl, err := s.repo.Get(ctx, id)
	if err != nil {
		return TemplateExport{}, err
	}
	return TemplateExport{
		Name:    tmpl.Name,
		Content: tmpl.Content,
	}, nil
}
