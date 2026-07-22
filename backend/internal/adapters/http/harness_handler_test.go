package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
	"github.com/Corwind/cmux/backend/internal/harness"
)

func TestHarnessHandler_List(t *testing.T) {
	registry := harness.NewRegistry()
	registry.Register(harness.NewClaudeHarness(domain.ClaudeConfig{}), "Claude Code")

	handler := NewHarnessHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/api/harnesses", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []harnessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 harness, got %d", len(resp))
	}
	if resp[0].Type != "claude" {
		t.Errorf("expected type 'claude', got %q", resp[0].Type)
	}
	if resp[0].SectionName != "Claude Code" {
		t.Errorf("expected section_name 'Claude Code', got %q", resp[0].SectionName)
	}
	if !resp[0].IsDefault {
		t.Errorf("expected first entry to be default")
	}
}

func TestHarnessHandler_List_OnlyFirstIsDefault(t *testing.T) {
	registry := harness.NewRegistry()
	registry.Register(harness.NewClaudeHarness(domain.ClaudeConfig{}), "Claude Code")
	registry.Register(&fakeHarness{harnessType: "codex"}, "Codex")

	handler := NewHarnessHandler(registry)

	req := httptest.NewRequest(http.MethodGet, "/api/harnesses", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	var resp []harnessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 harnesses, got %d", len(resp))
	}
	if !resp[0].IsDefault {
		t.Errorf("expected first entry (claude) to be default")
	}
	if resp[1].IsDefault {
		t.Errorf("expected second entry (codex) to not be default")
	}
	if resp[1].Type != "codex" || resp[1].SectionName != "Codex" {
		t.Errorf("unexpected second entry: %+v", resp[1])
	}
}

// fakeHarness is a minimal harness.Harness stand-in used only to exercise
// multi-entry registry behavior in tests.
type fakeHarness struct {
	harnessType string
}

func (f *fakeHarness) Type() harness.Type                          { return harness.Type(f.harnessType) }
func (f *fakeHarness) BinaryName() string                          { return "true" }
func (f *fakeHarness) BuildSpawnArgs(harness.SpawnIntent) []string { return nil }
func (f *fakeHarness) HasResumeSupport() bool                      { return false }
func (f *fakeHarness) HasSkipPermissionsSupport() bool             { return false }
func (f *fakeHarness) HasModelSelection() bool                     { return false }
func (f *fakeHarness) HasNotificationSupport() bool                { return false }
func (f *fakeHarness) HasSandboxPathGrants() bool                  { return false }
func (f *fakeHarness) EnvOverrides() ([]string, map[string]string) { return nil, nil }
func (f *fakeHarness) NeedsOpenURLWrapper() bool                   { return false }
func (f *fakeHarness) SandboxPathGrants() []harness.PathGrant      { return nil }
func (f *fakeHarness) ParseNotification([]byte) (harness.NotificationResult, bool) {
	return harness.NotificationResult{}, false
}
func (f *fakeHarness) HasExternalSessionIDMinting() bool { return false }
func (f *fakeHarness) DiscoverSessionID(workingDir string, notBefore time.Time) (string, error) {
	return "", nil
}
