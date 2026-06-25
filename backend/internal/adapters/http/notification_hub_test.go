package http

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNotificationHub_BroadcastWorktreeDeleted_Delivered(t *testing.T) {
	hub := newNotificationHub()

	ch, unsub := hub.subscribe()
	defer unsub()

	hub.BroadcastWorktreeDeleted("wt-1", "")

	select {
	case data := <-ch:
		var msg worktreeDeletedMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		if msg.Type != "worktree_deleted" {
			t.Errorf("expected type %q, got %q", "worktree_deleted", msg.Type)
		}
		if msg.WorktreeID != "wt-1" {
			t.Errorf("expected worktree_id %q, got %q", "wt-1", msg.WorktreeID)
		}
	default:
		t.Fatal("expected data to be delivered to subscriber")
	}
}

func TestNotificationHub_BroadcastWorktreeDeleted_ErrorField(t *testing.T) {
	hub := newNotificationHub()

	ch, unsub := hub.subscribe()
	defer unsub()

	hub.BroadcastWorktreeDeleted("wt-2", "git remove failed")

	select {
	case data := <-ch:
		var msg worktreeDeletedMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		if msg.Error != "git remove failed" {
			t.Errorf("expected error %q, got %q", "git remove failed", msg.Error)
		}
		// With a non-empty error the field must be present in the raw JSON.
		if !strings.Contains(string(data), "\"error\"") {
			t.Errorf("expected error field in payload, got %s", data)
		}
	default:
		t.Fatal("expected data to be delivered to subscriber")
	}
}

func TestNotificationHub_BroadcastWorktreeDeleted_NoSubscribers(t *testing.T) {
	hub := newNotificationHub()

	// Must not panic with zero subscribers.
	hub.BroadcastWorktreeDeleted("wt-3", "")
}

func TestParseOscNotification_Osc777(t *testing.T) {
	data := []byte("\x1b]777;notify;Claude Code;Claude needs your permission\x07")
	msg, eventType, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if msg != "Claude needs your permission" {
		t.Fatalf("unexpected message: %q", msg)
	}
	if eventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", eventType)
	}
}

func TestParseOscNotification_Osc777TaskComplete(t *testing.T) {
	data := []byte("\x1b]777;notify;Claude Code;Task complete\x07")
	msg, eventType, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if msg != "Task complete" {
		t.Fatalf("unexpected message: %q", msg)
	}
	if eventType != "task_complete" {
		t.Fatalf("expected task_complete, got %q", eventType)
	}
}

func TestParseOscNotification_Osc9(t *testing.T) {
	data := []byte("\x1b]9;Claude needs your attention\x07")
	msg, eventType, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected")
	}
	if msg != "Claude needs your attention" {
		t.Fatalf("unexpected message: %q", msg)
	}
	if eventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", eventType)
	}
}

func TestParseOscNotification_StandaloneBell(t *testing.T) {
	data := []byte("some output\x07more output")
	msg, eventType, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected BEL to be detected")
	}
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	if eventType != "waiting_input" {
		t.Fatalf("expected waiting_input, got %q", eventType)
	}
}

func TestParseOscNotification_BellAsOscTerminator(t *testing.T) {
	// BEL used as OSC terminator — should NOT fire as standalone bell
	// but should fire as OSC 777
	data := []byte("\x1b]777;notify;Claude Code;Claude needs your permission\x07")
	_, _, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected OSC 777 to be detected")
	}
}

func TestParseOscNotification_NoNotification(t *testing.T) {
	data := []byte("hello world\r\nsome terminal output\r\n")
	_, _, ok := parseOscNotification(data)
	if ok {
		t.Fatal("expected no notification in plain output")
	}
}

func TestParseOscNotification_Osc777WithStringTerminator(t *testing.T) {
	// ST = ESC \ instead of BEL
	data := []byte("\x1b]777;notify;Claude Code;Task done\x1b\\")
	msg, eventType, ok := parseOscNotification(data)
	if !ok {
		t.Fatal("expected notification to be detected with ST terminator")
	}
	if msg != "Task done" {
		t.Fatalf("unexpected message: %q", msg)
	}
	if eventType != "task_complete" {
		t.Fatalf("expected task_complete, got %q", eventType)
	}
}

func TestClassifyNotification(t *testing.T) {
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

func TestNotificationHub_BroadcastToMultipleSubscribers(t *testing.T) {
	hub := newNotificationHub()

	ch1, unsub1 := hub.subscribe()
	ch2, unsub2 := hub.subscribe()
	defer unsub1()
	defer unsub2()

	hub.broadcast(sessionNotificationMsg{
		SessionID:   "s1",
		SessionName: "Session 1",
		Message:     "Claude needs your permission",
		EventType:   "waiting_input",
	})

	select {
	case data := <-ch1:
		if len(data) == 0 {
			t.Fatal("expected non-empty data on ch1")
		}
	default:
		t.Fatal("expected data on ch1")
	}

	select {
	case data := <-ch2:
		if len(data) == 0 {
			t.Fatal("expected non-empty data on ch2")
		}
	default:
		t.Fatal("expected data on ch2")
	}
}

func TestNotificationHub_UnsubscribeStopsReceiving(t *testing.T) {
	hub := newNotificationHub()

	ch, unsub := hub.subscribe()
	unsub()

	// Channel should be closed after unsubscribe
	_, open := <-ch
	if open {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestNotificationHub_DropsSlowSubscribers(t *testing.T) {
	hub := newNotificationHub()

	ch, unsub := hub.subscribe()
	defer unsub()

	// Vary session IDs to bypass the per-session debounce.
	for i := 0; i < 20; i++ {
		hub.broadcast(sessionNotificationMsg{
			SessionID: fmt.Sprintf("s%d", i),
			EventType: "waiting_input",
		})
	}

	// Should not block; slow subscriber just drops messages
	count := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			count++
		default:
			// No more buffered messages
			if count == 0 {
				t.Fatal("expected at least some messages to be received")
			}
			return
		}
	}
}

func TestNotificationHub_FilterDropsGeneric(t *testing.T) {
	hub := newNotificationHub()
	ch, unsub := hub.subscribe()
	defer unsub()

	hub.broadcast(sessionNotificationMsg{SessionID: "s1", EventType: "generic"})

	select {
	case <-ch:
		t.Fatal("generic notification should be dropped")
	default:
	}
}

func TestNotificationHub_FilterDropsTaskComplete(t *testing.T) {
	hub := newNotificationHub()
	ch, unsub := hub.subscribe()
	defer unsub()

	hub.broadcast(sessionNotificationMsg{SessionID: "s1", EventType: "task_complete"})

	select {
	case <-ch:
		t.Fatal("task_complete notification should be dropped")
	default:
	}
}

func TestNotificationHub_DebounceDeduplicates(t *testing.T) {
	hub := newNotificationHub()
	ch, unsub := hub.subscribe()
	defer unsub()

	// First broadcast fires.
	hub.broadcast(sessionNotificationMsg{SessionID: "s1", EventType: "waiting_input"})
	// Second broadcast within the debounce window is suppressed.
	hub.broadcast(sessionNotificationMsg{SessionID: "s1", EventType: "waiting_input"})

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			if count != 1 {
				t.Fatalf("expected exactly 1 notification, got %d", count)
			}
			return
		}
	}
}

func TestBroadcastSessionStatus_SendsToAllSubscribers(t *testing.T) {
	hub := newNotificationHub()

	ch1, unsub1 := hub.subscribe()
	ch2, unsub2 := hub.subscribe()
	defer unsub1()
	defer unsub2()

	hub.BroadcastSessionStatus("s1", "My Session", "running", "")

	for _, ch := range []<-chan []byte{ch1, ch2} {
		select {
		case data := <-ch:
			if len(data) == 0 {
				t.Fatal("expected non-empty data from subscriber")
			}
		default:
			t.Fatal("expected data to be delivered to subscriber")
		}
	}
}

func TestBroadcastSessionStatus_TypeFieldIsSessionStatus(t *testing.T) {
	hub := newNotificationHub()

	ch, unsub := hub.subscribe()
	defer unsub()

	hub.BroadcastSessionStatus("s1", "My Session", "failed", "worktree creation failed")

	select {
	case data := <-ch:
		var msg sessionStatusMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		if msg.Type != "session_status" {
			t.Errorf("expected type %q, got %q", "session_status", msg.Type)
		}
		if msg.SessionID != "s1" {
			t.Errorf("expected session_id %q, got %q", "s1", msg.SessionID)
		}
		if msg.SessionName != "My Session" {
			t.Errorf("expected session_name %q, got %q", "My Session", msg.SessionName)
		}
		if msg.Status != "failed" {
			t.Errorf("expected status %q, got %q", "failed", msg.Status)
		}
		if msg.Error != "worktree creation failed" {
			t.Errorf("expected error %q, got %q", "worktree creation failed", msg.Error)
		}
	default:
		t.Fatal("expected data to be delivered to subscriber")
	}
}
