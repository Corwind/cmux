package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Corwind/cmux/backend/internal/domain"
)

type mockTemplateRepository struct {
	mu        sync.Mutex
	templates map[string]domain.SandboxTemplate
}

func newMockTemplateRepo() *mockTemplateRepository {
	return &mockTemplateRepository{
		templates: make(map[string]domain.SandboxTemplate),
	}
}

func (m *mockTemplateRepository) Create(_ context.Context, tmpl domain.SandboxTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[tmpl.ID] = tmpl
	return nil
}

func (m *mockTemplateRepository) Get(_ context.Context, id string) (domain.SandboxTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl, ok := m.templates[id]
	if !ok {
		return domain.SandboxTemplate{}, fmt.Errorf("template not found: %s", id)
	}
	return tmpl, nil
}

func (m *mockTemplateRepository) List(_ context.Context) ([]domain.SandboxTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.SandboxTemplate
	for _, t := range m.templates {
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTemplateRepository) Update(_ context.Context, tmpl domain.SandboxTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.templates[tmpl.ID]; !ok {
		return fmt.Errorf("template not found: %s", tmpl.ID)
	}
	m.templates[tmpl.ID] = tmpl
	return nil
}

func (m *mockTemplateRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.templates, id)
	return nil
}

func (m *mockTemplateRepository) GetDefault(_ context.Context) (domain.SandboxTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.templates {
		if t.IsDefault {
			return t, nil
		}
	}
	return domain.SandboxTemplate{}, fmt.Errorf("no default template set")
}

func (m *mockTemplateRepository) SetDefault(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, t := range m.templates {
		t.IsDefault = (k == id)
		m.templates[k] = t
	}
	return nil
}

func (m *mockTemplateRepository) ClearDefault(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, t := range m.templates {
		t.IsDefault = false
		m.templates[k] = t
	}
	return nil
}

func TestTemplateService_CreateTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	tmpl, err := svc.CreateTemplate(ctx, "test-tmpl", "(allow file-read* (subpath \"/opt\"))")
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}
	if tmpl.Name != "test-tmpl" {
		t.Errorf("expected name 'test-tmpl', got %q", tmpl.Name)
	}
	if tmpl.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestTemplateService_CreateTemplateEmptyName(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	_, err := svc.CreateTemplate(ctx, "", "(allow file-read*)")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestTemplateService_CreateTemplateInvalidContent(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	_, err := svc.CreateTemplate(ctx, "bad", "(version 1)\n(allow file-read*)")
	if err == nil {
		t.Fatal("expected error for content with version directive")
	}
}

func TestTemplateService_GetTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTemplate(ctx, "findme", "(allow file-read*)")
	got, err := svc.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
}

func TestTemplateService_ListTemplates(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	if _, err := svc.CreateTemplate(ctx, "t1", "(allow file-read*)"); err != nil {
		t.Fatalf("CreateTemplate t1 failed: %v", err)
	}
	if _, err := svc.CreateTemplate(ctx, "t2", "(allow file-write*)"); err != nil {
		t.Fatalf("CreateTemplate t2 failed: %v", err)
	}

	list, err := svc.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 templates, got %d", len(list))
	}
}

func TestTemplateService_UpdateTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTemplate(ctx, "original", "(allow file-read*)")
	updated, err := svc.UpdateTemplate(ctx, created.ID, "renamed", "(allow file-write*)")
	if err != nil {
		t.Fatalf("UpdateTemplate failed: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("expected name 'renamed', got %q", updated.Name)
	}
	if updated.Content != "(allow file-write*)" {
		t.Errorf("expected updated content, got %q", updated.Content)
	}
}

func TestTemplateService_UpdateTemplateInvalidContent(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTemplate(ctx, "original", "(allow file-read*)")
	_, err := svc.UpdateTemplate(ctx, created.ID, "", "(deny default)")
	if err == nil {
		t.Fatal("expected error for invalid content update")
	}
}

func TestTemplateService_DeleteTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	created, _ := svc.CreateTemplate(ctx, "deleteme", "(allow file-read*)")
	if err := svc.DeleteTemplate(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTemplate failed: %v", err)
	}
	_, err := svc.GetTemplate(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestTemplateService_SetAndGetDefault(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	t1, _ := svc.CreateTemplate(ctx, "t1", "(allow file-read*)")
	if err := svc.SetDefault(ctx, t1.ID); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	def, err := svc.GetDefault(ctx)
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}
	if def.ID != t1.ID {
		t.Errorf("expected default ID %q, got %q", t1.ID, def.ID)
	}
}

func TestTemplateService_ClearDefault(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	t1, _ := svc.CreateTemplate(ctx, "t1", "(allow file-read*)")
	if err := svc.SetDefault(ctx, t1.ID); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}
	if err := svc.ClearDefault(ctx); err != nil {
		t.Fatalf("ClearDefault failed: %v", err)
	}

	_, err := svc.GetDefault(ctx)
	if err == nil {
		t.Fatal("expected error after clearing default")
	}
}

func TestTemplateService_SetDefaultNonexistent(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	err := svc.SetDefault(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestTemplateService_ImportTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	tmpl, err := svc.ImportTemplate(ctx, "imported", "(allow file-read* (subpath \"/opt\"))")
	if err != nil {
		t.Fatalf("ImportTemplate failed: %v", err)
	}
	if tmpl.Name != "imported" {
		t.Errorf("expected name 'imported', got %q", tmpl.Name)
	}
}

func TestTemplateService_ExportTemplate(t *testing.T) {
	repo := newMockTemplateRepo()
	svc := NewTemplateService(repo)
	ctx := context.Background()

	content := "(allow file-read* (subpath \"/opt\"))"
	created, _ := svc.CreateTemplate(ctx, "exportme", content)
	exported, err := svc.ExportTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("ExportTemplate failed: %v", err)
	}
	if exported.Name != "exportme" {
		t.Errorf("expected name 'exportme', got %q", exported.Name)
	}
	if exported.Content != content {
		t.Errorf("expected content %q, got %q", content, exported.Content)
	}
}
