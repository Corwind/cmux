package harness

import (
	"reflect"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func TestPiHarness_Type(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if h.Type() != PiType {
		t.Fatalf("Type() = %q, want %q", h.Type(), PiType)
	}
}

func TestPiHarness_BinaryName(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if got := h.BinaryName(); got != "pi" {
		t.Fatalf("BinaryName() = %q, want %q", got, "pi")
	}
}

func TestPiHarness_BuildSpawnArgs_New(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPiHarness_BuildSpawnArgs_Resume(t *testing.T) {
	// pi uses the same --session-id flag regardless of Resume: it creates
	// the session if missing, or reuses it if found — unlike Claude
	// (--session-id vs --resume) or Codex ("resume" subcommand).
	h := NewPiHarness(domain.PiConfig{})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: true})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPiHarness_BuildSpawnArgs_SkipPermissionsIgnored(t *testing.T) {
	// pi has no per-tool approval gate to bypass, so SkipPermissions must
	// never translate into a flag.
	h := NewPiHarness(domain.PiConfig{})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", SkipPermissions: true})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v — SkipPermissions must be ignored", got, want)
	}
}

func TestPiHarness_BuildSpawnArgs_WithModel(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{Model: "anthropic/claude-sonnet-5"})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1"})
	want := []string{"--session-id", "sess-1", "--model", "anthropic/claude-sonnet-5"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPiHarness_BuildSpawnArgs_WithoutModel(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{Model: ""})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1"})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPiHarness_HasSkipPermissionsSupport(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if h.HasSkipPermissionsSupport() {
		t.Fatal("HasSkipPermissionsSupport() = true, want false")
	}
}

func TestPiHarness_HasNotificationSupport(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if h.HasNotificationSupport() {
		t.Fatal("HasNotificationSupport() = true, want false")
	}
}

func TestPiHarness_ParseNotification_NeverMatches(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	_, ok := h.ParseNotification([]byte("\x1b]9;anything\x07"))
	if ok {
		t.Fatal("expected ParseNotification to never match for PiHarness")
	}
}

func TestPiHarness_HasResumeSupport(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if !h.HasResumeSupport() {
		t.Fatal("HasResumeSupport() = false, want true")
	}
}

func TestPiHarness_HasModelSelection(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if !h.HasModelSelection() {
		t.Fatal("HasModelSelection() = false, want true")
	}
}

func TestPiHarness_HasSandboxPathGrants(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if !h.HasSandboxPathGrants() {
		t.Fatal("HasSandboxPathGrants() = false, want true")
	}
}

func TestPiHarness_SandboxPathGrants(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	got := h.SandboxPathGrants()
	want := []PathGrant{
		{Path: "$HOME/.pi/agent", Recursive: true, Write: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPiHarness_HasExternalSessionIDMinting(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if h.HasExternalSessionIDMinting() {
		t.Fatal("HasExternalSessionIDMinting() = true, want false")
	}
}

func TestPiHarness_DiscoverSessionID_NotSupported(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	_, err := h.DiscoverSessionID("/some/dir", time.Time{})
	if err == nil {
		t.Fatal("expected DiscoverSessionID to return an error")
	}
}

func TestPiHarness_NeedsOpenURLWrapper(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{})
	if h.NeedsOpenURLWrapper() {
		t.Fatal("NeedsOpenURLWrapper() = true, want false")
	}
}

func TestPiHarness_EnvOverrides_DefaultHomeStillSet(t *testing.T) {
	// With Home unset, NewPiHarness best-effort-resolves it to
	// $HOME/.pi/agent, so PI_CODING_AGENT_DIR is still set to that resolved
	// default rather than left unset — only os.UserHomeDir() failing (not
	// forceable in a unit test) would leave piHome empty.
	h := NewPiHarness(domain.PiConfig{})
	strip, set := h.EnvOverrides()
	if strip != nil {
		t.Fatalf("expected nil strip keys, got %v", strip)
	}
	if set["PI_CODING_AGENT_DIR"] == "" {
		t.Fatal("expected PI_CODING_AGENT_DIR to be set to the resolved default home")
	}
}

func TestPiHarness_EnvOverrides_CustomHome(t *testing.T) {
	h := NewPiHarness(domain.PiConfig{Home: "/tmp/custom-pi-home"})
	strip, set := h.EnvOverrides()
	if strip != nil {
		t.Fatalf("expected nil strip keys, got %v", strip)
	}
	if set["PI_CODING_AGENT_DIR"] != "/tmp/custom-pi-home" {
		t.Fatalf("expected PI_CODING_AGENT_DIR=/tmp/custom-pi-home, got %v", set)
	}
}
