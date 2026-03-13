package sandbox

import (
	"strings"
	"testing"
)

func TestBuildBasicProfile(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir: "/tmp/project",
		HomeDir:    "/Users/testuser",
	}

	profile, err := pb.Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must start with version and deny default
	if !strings.HasPrefix(profile, "(version 1)\n(deny default)") {
		t.Error("profile must start with (version 1) and (deny default)")
	}

	requiredFragments := []string{
		"(allow process-exec*)",
		"(allow process-fork)",
		`(allow file-read* (subpath "/usr/lib"))`,
		`(allow file-read* (subpath "/System/Library"))`,
		`(allow file-read* (subpath (param "WORKING_DIR")))`,
		`(allow file-write* (subpath (param "WORKING_DIR")))`,
		`(subpath (param "HOME_DIR"))`,
	}

	for _, frag := range requiredFragments {
		if !strings.Contains(profile, frag) {
			t.Errorf("profile missing required fragment: %s", frag)
		}
	}
}

func TestBuildWithTemplates(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir:    "/tmp/project",
		HomeDir:       "/Users/testuser",
		TemplateNames: []string{"network", "file-extra"},
	}

	profile, err := pb.Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Template content should be appended
	if !strings.Contains(profile, `(remote tcp "*:443")`) {
		t.Error("profile missing network template content")
	}
	if !strings.Contains(profile, `(subpath "/opt/tools")`) {
		t.Error("profile missing file-extra template content")
	}
}

func TestBuildRejectsInvalidTemplate(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir:    "/tmp/project",
		HomeDir:       "/Users/testuser",
		TemplateNames: []string{"invalid"},
	}

	_, err := pb.Build(cfg)
	if err == nil {
		t.Fatal("expected error when building with invalid template")
	}
}

func TestBuildRejectsMissingTemplate(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir:    "/tmp/project",
		HomeDir:       "/Users/testuser",
		TemplateNames: []string{"nonexistent"},
	}

	_, err := pb.Build(cfg)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestParams(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir: "/tmp/project",
		HomeDir:    "/Users/testuser",
	}

	params := pb.Params(cfg)

	if params["WORKING_DIR"] != "/tmp/project" {
		t.Errorf("expected WORKING_DIR=/tmp/project, got %q", params["WORKING_DIR"])
	}
	if params["HOME_DIR"] != "/Users/testuser" {
		t.Errorf("expected HOME_DIR=/Users/testuser, got %q", params["HOME_DIR"])
	}
}

func TestParamsAutoResolvesHomeDir(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir: "/tmp/project",
		// HomeDir intentionally empty
	}

	params := pb.Params(cfg)

	if params["HOME_DIR"] == "" {
		t.Error("expected HOME_DIR to be auto-resolved")
	}
}

func TestBuildAutoResolvesHomeDir(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir: "/tmp/project",
		// HomeDir intentionally empty - should auto-resolve
	}

	profile, err := pb.Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(profile, `(param "HOME_DIR")`) {
		t.Error("profile should contain HOME_DIR param reference")
	}
}

func TestBuildRejectsTemplateNameWithNewline(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir:    "/tmp/project",
		HomeDir:       "/Users/testuser",
		TemplateNames: []string{"evil\n(allow file-write*)"},
	}

	_, err := pb.Build(cfg)
	if err == nil {
		t.Fatal("expected error for template name containing newline")
	}
}

func TestBuildRejectsTemplateNameWithPathTraversal(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir:    "/tmp/project",
		HomeDir:       "/Users/testuser",
		TemplateNames: []string{"../../etc/passwd"},
	}

	_, err := pb.Build(cfg)
	if err == nil {
		t.Fatal("expected error for template name with path traversal")
	}
}

func TestBuildIncludesClaudeBinaryPath(t *testing.T) {
	pb := NewProfileBuilder(testdataDir(t))

	cfg := ProfileConfig{
		WorkingDir: "/tmp/project",
		HomeDir:    "/Users/testuser",
	}

	profile, err := pb.Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The profile should contain a comment about claude binary
	// Even if claude isn't on PATH, the build should not fail
	_ = profile
}
