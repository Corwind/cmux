package harness

import (
	"reflect"
	"testing"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func TestClaudeHarness_HasNotificationSupport_ReflectsConfig(t *testing.T) {
	if h := NewClaudeHarness(domain.ClaudeConfig{NotificationsEnabled: true}); !h.HasNotificationSupport() {
		t.Error("expected HasNotificationSupport() true when NotificationsEnabled: true")
	}
	if h := NewClaudeHarness(domain.ClaudeConfig{NotificationsEnabled: false}); h.HasNotificationSupport() {
		t.Error("expected HasNotificationSupport() false when NotificationsEnabled: false")
	}
}

func TestClaudeHarness_BuildSpawnArgs_Create(t *testing.T) {
	h := &ClaudeHarness{}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_Resume(t *testing.T) {
	h := &ClaudeHarness{}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: true})
	want := []string{"--resume", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_CreateWithSkipPermissions(t *testing.T) {
	h := &ClaudeHarness{}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false, SkipPermissions: true})
	want := []string{"--session-id", "sess-1", "--dangerously-skip-permissions"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_ResumeWithoutSkipPermissions(t *testing.T) {
	h := &ClaudeHarness{}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: true, SkipPermissions: false})
	want := []string{"--resume", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_WithModel(t *testing.T) {
	h := &ClaudeHarness{model: "claude-opus-4"}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{"--session-id", "sess-1", "--model", "claude-opus-4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_WithoutModel(t *testing.T) {
	h := &ClaudeHarness{model: ""}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-1", Resume: false})
	want := []string{"--session-id", "sess-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClaudeHarness_BuildSpawnArgs_ResumeSkipPermissionsAndModel(t *testing.T) {
	h := &ClaudeHarness{model: "claude-sonnet-5"}
	got := h.BuildSpawnArgs(SpawnIntent{SessionID: "sess-9", Resume: true, SkipPermissions: true})
	want := []string{"--resume", "sess-9", "--dangerously-skip-permissions", "--model", "claude-sonnet-5"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// The following tests port the notification-parsing test cases from
// notification_hub_test.go to ClaudeHarness.ParseNotification, locking in
// behavior parity with the original parseOscNotification implementation.

func TestClaudeHarness_ParseNotification_Osc777(t *testing.T) {
	h := &ClaudeHarness{}
	data := []byte("\x1b]777;notify;Claude Code;Claude needs your permission\x07")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if res.Message != "Claude needs your permission" {
		t.Fatalf("unexpected message: %q", res.Message)
	}
	if res.EventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", res.EventType)
	}
}

func TestClaudeHarness_ParseNotification_Osc777TaskComplete(t *testing.T) {
	h := &ClaudeHarness{}
	data := []byte("\x1b]777;notify;Claude Code;Task complete\x07")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if res.Message != "Task complete" {
		t.Fatalf("unexpected message: %q", res.Message)
	}
	if res.EventType != "task_complete" {
		t.Fatalf("expected task_complete, got %q", res.EventType)
	}
}

func TestClaudeHarness_ParseNotification_Osc9(t *testing.T) {
	h := &ClaudeHarness{}
	data := []byte("\x1b]9;Claude needs your attention\x07")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if res.Message != "Claude needs your attention" {
		t.Fatalf("unexpected message: %q", res.Message)
	}
	if res.EventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", res.EventType)
	}
}

func TestClaudeHarness_ParseNotification_StandaloneBell(t *testing.T) {
	h := &ClaudeHarness{}
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

func TestClaudeHarness_ParseNotification_BellAsOscTerminator(t *testing.T) {
	// BEL used as OSC terminator — should NOT fire as standalone bell
	// but should fire as OSC 777.
	h := &ClaudeHarness{}
	data := []byte("\x1b]777;notify;Claude Code;Claude needs your permission\x07")
	_, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected OSC 777 to be detected")
	}
}

func TestClaudeHarness_ParseNotification_NoNotification(t *testing.T) {
	h := &ClaudeHarness{}
	data := []byte("hello world\r\nsome terminal output\r\n")
	_, ok := h.ParseNotification(data)
	if ok {
		t.Fatal("expected no notification in plain output")
	}
}

func TestClaudeHarness_ParseNotification_Osc777WithStringTerminator(t *testing.T) {
	// ST = ESC \ instead of BEL
	h := &ClaudeHarness{}
	data := []byte("\x1b]777;notify;Claude Code;Task done\x1b\\")
	res, ok := h.ParseNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected with ST terminator")
	}
	if res.Message != "Task done" {
		t.Fatalf("unexpected message: %q", res.Message)
	}
	if res.EventType != "task_complete" {
		t.Fatalf("expected task_complete, got %q", res.EventType)
	}
}

func TestClaudeHarness_ClassifyNotification(t *testing.T) {
	cases := []struct {
		msg      string
		expected string
	}{
		{"Claude needs your permission", "waiting_input"},
		{"Waiting for input", "waiting_input"},
		{"Task complete", "task_complete"},
		{"Done!", "task_complete"},
		{"Hello", "generic"},
	}
	for _, c := range cases {
		got := classifyNotification(c.msg)
		if got != c.expected {
			t.Errorf("classifyNotification(%q) = %q, want %q", c.msg, got, c.expected)
		}
	}
}
