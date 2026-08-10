package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func TestCodexHarness_Type(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	if h.Type() != CodexType {
		t.Fatalf("Type() = %q, want %q", h.Type(), CodexType)
	}
}

func TestCodexHarness_BinaryName(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	if got := h.BinaryName(); got != "codex" {
		t.Fatalf("BinaryName() = %q, want %q", got, "codex")
	}
}

func TestCodexHarness_BuildSpawnArgs_New(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: true})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_BuildSpawnArgs_Resume(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: true})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: true})
	want := []string{
		"resume", "sess-1",
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_BuildSpawnArgs_SkipPermissions(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: true})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false, SkipPermissions: true})
	want := []string{
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
		"--dangerously-bypass-approvals-and-sandbox",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_BuildSpawnArgs_WithModel(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{Model: "gpt-5-codex", NotificationsEnabled: true})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
		"-m", "gpt-5-codex",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_BuildSpawnArgs_WithoutModel(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{Model: "", NotificationsEnabled: true})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{
		"-c", "tui.notification_method=osc9",
		"-c", `tui.notifications=["approval-requested"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_BuildSpawnArgs_NotificationsDisabled(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: false})
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false, SkipPermissions: true})
	want := []string{"--dangerously-bypass-approvals-and-sandbox"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v — expected no -c notification flags when disabled", got, want)
	}
}

func TestCodexHarness_HasNotificationSupport_ReflectsConfig(t *testing.T) {
	if h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: true}); !h.HasNotificationSupport() {
		t.Error("expected HasNotificationSupport() true when NotificationsEnabled: true")
	}
	if h := NewCodexHarness(domain.CodexConfig{NotificationsEnabled: false}); h.HasNotificationSupport() {
		t.Error("expected HasNotificationSupport() false when NotificationsEnabled: false")
	}
}

func TestCodexHarness_ParseNotification_Osc9(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	data := []byte("\x1b]9;Codex task complete\x07")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if res.Message != "Codex task complete" {
		t.Fatalf("unexpected message: %q", res.Message)
	}
	if res.EventType != "task_complete" {
		t.Fatalf("expected task_complete, got %q", res.EventType)
	}
}

func TestCodexHarness_ParseNotification_StandaloneBell(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	data := []byte("some output\x07more output")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected BEL to be detected")
	}
	if res.Message == "" {
		t.Fatal("expected non-empty message")
	}
	if res.EventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", res.EventType)
	}
}

func TestCodexHarness_ParseNotification_NoMatch(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	data := []byte("hello world\r\nsome terminal output\r\n")
	_, ok := h.ParseNotification(data)
	if ok {
		t.Fatal("expected no notification in plain output")
	}
}

func TestCodexHarness_ParseNotification_Osc777DoesNotMatch(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	data := []byte("\x1b]777;notify;Claude Code;Claude needs your permission\x07")
	_, ok := h.ParseNotification(data)
	if ok {
		t.Fatal("expected OSC 777 to NOT be detected by Codex harness")
	}
}

func TestCodexHarness_SandboxPathGrants(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	got := h.SandboxPathGrants()
	want := []PathGrant{
		{Path: "$HOME/.codex", Recursive: true, Write: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCodexHarness_EnvOverrides(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	strip, set := h.EnvOverrides()
	if strip != nil || set != nil {
		t.Fatalf("EnvOverrides() = (%v, %v), want (nil, nil)", strip, set)
	}
}

func TestCodexHarness_NeedsOpenURLWrapper(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	if h.NeedsOpenURLWrapper() {
		t.Fatal("NeedsOpenURLWrapper() = true, want false")
	}
}

func TestCodexHarness_HasExternalSessionIDMinting(t *testing.T) {
	h := NewCodexHarness(domain.CodexConfig{})
	if !h.HasExternalSessionIDMinting() {
		t.Fatal("HasExternalSessionIDMinting() = false, want true")
	}
}

// writeRollout writes a fabricated rollout-*.jsonl file with the given
// session id / cwd as the first line's payload, and sets its mtime.
func writeRollout(t *testing.T, dir, name, sessionID, cwd string, mtime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	line := struct {
		Payload struct {
			SessionID string `json:"session_id"`
			Cwd       string `json:"cwd"`
		} `json:"payload"`
	}{}
	line.Payload.SessionID = sessionID
	line.Payload.Cwd = cwd
	b, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	content := string(b) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return path
}

func TestCodexHarness_DiscoverSessionID_MatchingCwdNewestMtimeWins(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions", "2026", "01", "01")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	h := NewCodexHarness(domain.CodexConfig{Home: tmp})

	notBefore := time.Now().Add(-time.Hour)
	cwd := "/work/dir"

	writeRollout(t, sessionsDir, "rollout-1.jsonl", "session-old", cwd, notBefore.Add(1*time.Minute))
	writeRollout(t, sessionsDir, "rollout-2.jsonl", "session-newest", cwd, notBefore.Add(5*time.Minute))
	writeRollout(t, sessionsDir, "rollout-3.jsonl", "session-other-cwd", "/other/dir", notBefore.Add(10*time.Minute))

	got, err := h.DiscoverSessionID(cwd, notBefore)
	if err != nil {
		t.Fatalf("DiscoverSessionID: %v", err)
	}
	if got != "session-newest" {
		t.Fatalf("got %q, want %q", got, "session-newest")
	}
}

func TestCodexHarness_DiscoverSessionID_ExcludesBeforeNotBefore(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions", "2026", "01", "01")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	h := NewCodexHarness(domain.CodexConfig{Home: tmp})

	notBefore := time.Now()
	cwd := "/work/dir"

	writeRollout(t, sessionsDir, "rollout-old.jsonl", "session-too-old", cwd, notBefore.Add(-time.Minute))
	writeRollout(t, sessionsDir, "rollout-new.jsonl", "session-valid", cwd, notBefore.Add(time.Minute))

	got, err := h.DiscoverSessionID(cwd, notBefore)
	if err != nil {
		t.Fatalf("DiscoverSessionID: %v", err)
	}
	if got != "session-valid" {
		t.Fatalf("got %q, want %q", got, "session-valid")
	}
}

func TestCodexHarness_DiscoverSessionID_MismatchedCwdNotPicked(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions", "2026", "01", "01")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	h := NewCodexHarness(domain.CodexConfig{Home: tmp})

	notBefore := time.Now().Add(-time.Hour)
	cwd := "/work/dir"

	writeRollout(t, sessionsDir, "rollout-1.jsonl", "session-a", cwd, notBefore.Add(1*time.Minute))
	writeRollout(t, sessionsDir, "rollout-2.jsonl", "session-b", cwd, notBefore.Add(2*time.Minute))
	// Newest overall, but mismatched cwd — must not be picked.
	writeRollout(t, sessionsDir, "rollout-3.jsonl", "session-newest-wrong-cwd", "/other/dir", notBefore.Add(10*time.Minute))

	got, err := h.DiscoverSessionID(cwd, notBefore)
	if err != nil {
		t.Fatalf("DiscoverSessionID: %v", err)
	}
	if got != "session-b" {
		t.Fatalf("got %q, want %q", got, "session-b")
	}
}

func TestCodexHarness_DiscoverSessionID_NoCandidatesReturnsError(t *testing.T) {
	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, "sessions", "2026", "01", "01")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	h := NewCodexHarness(domain.CodexConfig{Home: tmp})

	notBefore := time.Now().Add(-time.Hour)
	writeRollout(t, sessionsDir, "rollout-1.jsonl", "session-a", "/other/dir", notBefore.Add(time.Minute))

	_, err := h.DiscoverSessionID("/work/dir", notBefore)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
