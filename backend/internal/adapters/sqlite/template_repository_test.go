package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	_ "modernc.org/sqlite"
)

func setupTestTemplateRepo(t *testing.T) *TemplateRepository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if _, err := db.Exec(createTemplatesTable); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewTemplateRepository(db)
}

func makeTemplate(name string) domain.SandboxTemplate {
	now := time.Now()
	return domain.SandboxTemplate{
		ID:        "test-tmpl-" + name,
		Name:      name,
		Content:   "(allow file-write* (subpath \"/tmp\"))",
		IsDefault: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestTemplateRepository_CreateAndGet(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()
	tmpl := makeTemplate("test1")

	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.Get(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != tmpl.ID {
		t.Errorf("expected ID %q, got %q", tmpl.ID, got.ID)
	}
	if got.Name != tmpl.Name {
		t.Errorf("expected Name %q, got %q", tmpl.Name, got.Name)
	}
	if got.Content != tmpl.Content {
		t.Errorf("expected Content %q, got %q", tmpl.Content, got.Content)
	}
	if got.IsDefault != tmpl.IsDefault {
		t.Errorf("expected IsDefault %v, got %v", tmpl.IsDefault, got.IsDefault)
	}
}

func TestTemplateRepository_GetNotFound(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestTemplateRepository_List(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	t1 := makeTemplate("first")
	t2 := makeTemplate("second")
	if err := repo.Create(ctx, t1); err != nil {
		t.Fatalf("Create t1 failed: %v", err)
	}
	if err := repo.Create(ctx, t2); err != nil {
		t.Fatalf("Create t2 failed: %v", err)
	}

	templates, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
}

func TestTemplateRepository_ListEmpty(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	templates, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}

func TestTemplateRepository_Update(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()
	tmpl := makeTemplate("update-me")
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tmpl.Name = "updated-name"
	tmpl.Content = "(allow file-read* (subpath \"/opt\"))"
	if err := repo.Update(ctx, tmpl); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.Get(ctx, tmpl.ID)
	if got.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got %q", got.Name)
	}
	if got.Content != tmpl.Content {
		t.Errorf("expected updated content, got %q", got.Content)
	}
}

func TestTemplateRepository_Delete(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()
	tmpl := makeTemplate("delete-me")
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, tmpl.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.Get(ctx, tmpl.ID)
	if err == nil {
		t.Fatal("expected error after deleting template")
	}
}

func TestTemplateRepository_SetAndGetDefault(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	t1 := makeTemplate("tmpl1")
	t2 := makeTemplate("tmpl2")
	if err := repo.Create(ctx, t1); err != nil {
		t.Fatalf("Create t1 failed: %v", err)
	}
	if err := repo.Create(ctx, t2); err != nil {
		t.Fatalf("Create t2 failed: %v", err)
	}

	if err := repo.SetDefault(ctx, t1.ID); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	def, err := repo.GetDefault(ctx)
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}
	if def.ID != t1.ID {
		t.Errorf("expected default ID %q, got %q", t1.ID, def.ID)
	}

	// Setting a new default should clear the old one
	if err := repo.SetDefault(ctx, t2.ID); err != nil {
		t.Fatalf("SetDefault t2 failed: %v", err)
	}

	def, err = repo.GetDefault(ctx)
	if err != nil {
		t.Fatalf("GetDefault after switch failed: %v", err)
	}
	if def.ID != t2.ID {
		t.Errorf("expected default ID %q, got %q", t2.ID, def.ID)
	}

	// Verify old default is cleared
	got, _ := repo.Get(ctx, t1.ID)
	if got.IsDefault {
		t.Error("expected t1 to no longer be default")
	}
}

func TestTemplateRepository_GetDefaultNotSet(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	_, err := repo.GetDefault(ctx)
	if err == nil {
		t.Fatal("expected error when no default is set")
	}
}

func TestTemplateRepository_ClearDefault(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	tmpl := makeTemplate("clearable")
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.SetDefault(ctx, tmpl.ID); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	if err := repo.ClearDefault(ctx); err != nil {
		t.Fatalf("ClearDefault failed: %v", err)
	}

	_, err := repo.GetDefault(ctx)
	if err == nil {
		t.Fatal("expected error after clearing default")
	}
}

func TestTemplateRepository_ListOrderByCreatedAtDesc(t *testing.T) {
	repo := setupTestTemplateRepo(t)
	ctx := context.Background()

	t1 := makeTemplate("older")
	t1.CreatedAt = time.Now().Add(-time.Hour)
	t2 := makeTemplate("newer")
	t2.CreatedAt = time.Now()

	if err := repo.Create(ctx, t1); err != nil {
		t.Fatalf("Create t1 failed: %v", err)
	}
	if err := repo.Create(ctx, t2); err != nil {
		t.Fatalf("Create t2 failed: %v", err)
	}

	templates, _ := repo.List(ctx)
	if templates[0].Name != "newer" {
		t.Errorf("expected newest template first, got %q", templates[0].Name)
	}
}
