package harness

import (
	"errors"
	"testing"
	"time"
)

// stubHarness is a minimal Harness implementation used only to exercise
// Registry behavior independent of any concrete harness.
type stubHarness struct {
	t Type
}

func (s stubHarness) Type() Type                                  { return s.t }
func (s stubHarness) BinaryName() string                          { return string(s.t) }
func (s stubHarness) BuildSpawnArgs(SpawnIntent) []string         { return nil }
func (s stubHarness) HasResumeSupport() bool                      { return false }
func (s stubHarness) HasSkipPermissionsSupport() bool             { return false }
func (s stubHarness) HasModelSelection() bool                     { return false }
func (s stubHarness) HasNotificationSupport() bool                { return false }
func (s stubHarness) HasSandboxPathGrants() bool                  { return false }
func (s stubHarness) EnvOverrides() ([]string, map[string]string) { return nil, nil }
func (s stubHarness) NeedsOpenURLWrapper() bool                   { return false }
func (s stubHarness) SandboxPathGrants() []PathGrant              { return nil }
func (s stubHarness) ParseNotification(data []byte) (NotificationResult, bool) {
	return NotificationResult{}, false
}
func (s stubHarness) HasExternalSessionIDMinting() bool { return false }
func (s stubHarness) DiscoverSessionID(workingDir string, notBefore time.Time) (string, error) {
	return "", errors.New("not supported")
}

func TestRegistry_Default_EmptyReturnsNil(t *testing.T) {
	r := NewRegistry()

	if got := r.Default(); got != nil {
		t.Fatalf("Default() = %v, want nil", got)
	}
}

func TestRegistry_Default_FirstRegisteredWins(t *testing.T) {
	r := NewRegistry()
	first := stubHarness{t: Type("first")}
	second := stubHarness{t: Type("second")}

	r.Register(first, "First")
	r.Register(second, "Second")

	if got := r.Default(); got != first {
		t.Fatalf("Default() = %v, want %v", got, first)
	}
}

func TestRegistry_Get_UnregisteredReturnsNotOK(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get(Type("missing"))
	if ok {
		t.Fatalf("Get(missing) ok = true, want false")
	}
}

func TestRegistry_Get_RegisteredReturnsOK(t *testing.T) {
	r := NewRegistry()
	h := stubHarness{t: Type("claude")}
	r.Register(h, "Claude Code")

	got, ok := r.Get(Type("claude"))
	if !ok || got != h {
		t.Fatalf("Get(claude) = (%v, %v), want (%v, true)", got, ok, h)
	}
}

func TestRegistry_SectionName_FallsBackToRawType(t *testing.T) {
	r := NewRegistry()

	if got := r.SectionName(Type("codex")); got != "codex" {
		t.Fatalf("SectionName(codex) = %q, want %q", got, "codex")
	}
}

func TestRegistry_SectionName_UsesRegisteredName(t *testing.T) {
	r := NewRegistry()
	r.Register(stubHarness{t: Type("claude")}, "Claude Code")

	if got := r.SectionName(Type("claude")); got != "Claude Code" {
		t.Fatalf("SectionName(claude) = %q, want %q", got, "Claude Code")
	}
}

func TestRegistry_SectionName_EmptyFallsBackToRawType(t *testing.T) {
	r := NewRegistry()
	r.Register(stubHarness{t: Type("claude")}, "")

	if got := r.SectionName(Type("claude")); got != "claude" {
		t.Fatalf("SectionName(claude) = %q, want %q", got, "claude")
	}
}

func TestRegistry_All_ReturnsRegistrationOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(stubHarness{t: Type("claude")}, "Claude Code")
	r.Register(stubHarness{t: Type("codex")}, "Codex")

	got := r.All()
	want := []Type{Type("claude"), Type("codex")}
	if len(got) != len(want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("All() = %v, want %v", got, want)
		}
	}
}
