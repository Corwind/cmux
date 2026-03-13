package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata")
}

func TestLoadTemplate(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	t.Run("loads valid template", func(t *testing.T) {
		tmpl, err := pb.LoadTemplate("network")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl.Name != "network" {
			t.Errorf("expected name 'network', got %q", tmpl.Name)
		}
		if len(tmpl.Content) == 0 {
			t.Error("expected non-empty content")
		}
	})

	t.Run("rejects template with version directive", func(t *testing.T) {
		_, err := pb.LoadTemplate("invalid")
		if err == nil {
			t.Fatal("expected error for template containing version directive")
		}
	})

	t.Run("returns error for missing template", func(t *testing.T) {
		_, err := pb.LoadTemplate("nonexistent")
		if err == nil {
			t.Fatal("expected error for missing template")
		}
	})
}

func TestListTemplates(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	templates, err := pb.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// testdata has network.sbpl, file-extra.sbpl, invalid.sbpl
	if len(templates) < 3 {
		t.Errorf("expected at least 3 templates, got %d: %v", len(templates), templates)
	}

	found := map[string]bool{}
	for _, name := range templates {
		found[name] = true
	}
	for _, want := range []string{"network", "file-extra", "invalid"} {
		if !found[want] {
			t.Errorf("expected template %q in list", want)
		}
	}
}

func TestListTemplatesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	pb := NewProfileBuilder(dir)

	templates, err := pb.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}

func TestListTemplatesMissingDir(t *testing.T) {
	pb := NewProfileBuilder(filepath.Join(os.TempDir(), "nonexistent-sandbox-dir"))

	_, err := pb.ListTemplates()
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}
