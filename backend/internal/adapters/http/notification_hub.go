package http

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
)

// sessionNotificationMsg is the JSON payload broadcast to notification subscribers.
type sessionNotificationMsg struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	Message     string `json:"message"`
	EventType   string `json:"event_type"`
}

// notificationHub fans out session notifications to all subscribed WebSocket clients.
type notificationHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newNotificationHub() *notificationHub {
	return &notificationHub{
		clients: make(map[chan []byte]struct{}),
	}
}

// subscribe returns a channel that receives broadcast messages and a cleanup function.
func (h *notificationHub) subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		close(ch)
	}
}

// broadcast sends a notification to all current subscribers, dropping slow ones.
func (h *notificationHub) broadcast(n sessionNotificationMsg) {
	data, err := json.Marshal(n)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
		}
	}
}

// parseOscNotification scans a PTY data chunk for notification sequences:
//   - OSC 777: \x1b]777;notify;title;body\x07 (used by Claude Code)
//   - OSC 9:   \x1b]9;message\x07 (ConEmu / Ghostty format)
//   - Standalone BEL \x07 (terminal_bell channel)
//
// Returns the human-readable message, the event type, and whether a notification
// was found. Handles BEL as OSC terminator vs. standalone correctly.
func parseOscNotification(data []byte) (message, eventType string, ok bool) {
	// OSC 777: \x1b]777;notify;title;body\x07
	if idx := bytes.Index(data, []byte("\x1b]777;")); idx >= 0 {
		rest := data[idx+6:]
		if end := oscEnd(rest); end >= 0 {
			parts := strings.SplitN(string(rest[:end]), ";", 3)
			msg := string(rest[:end])
			if len(parts) >= 3 {
				msg = parts[2]
			} else if len(parts) > 0 {
				msg = parts[len(parts)-1]
			}
			return msg, classifyNotification(msg), true
		}
	}

	// OSC 9: \x1b]9;message\x07
	if idx := bytes.Index(data, []byte("\x1b]9;")); idx >= 0 {
		rest := data[idx+4:]
		if end := oscEnd(rest); end >= 0 {
			msg := string(rest[:end])
			return msg, classifyNotification(msg), true
		}
	}

	// Standalone BEL — only when not used as an OSC terminator.
	if hasStandaloneBell(data) {
		return "Claude needs your attention", "waiting_input", true
	}

	return "", "", false
}

// oscEnd returns the index of the first BEL (0x07) or ESC-backslash in data,
// which terminates an OSC sequence. Returns -1 if none found.
func oscEnd(data []byte) int {
	for i, b := range data {
		switch {
		case b == 0x07:
			return i
		case b == 0x1b && i+1 < len(data) && data[i+1] == '\\':
			return i
		}
	}
	return -1
}

// hasStandaloneBell returns true when data contains a BEL byte that is not
// being used as an OSC sequence terminator (i.e. no \x1b] precedes it within
// the last 256 bytes).
func hasStandaloneBell(data []byte) bool {
	for i, b := range data {
		if b != 0x07 {
			continue
		}
		start := 0
		if i > 256 {
			start = i - 256
		}
		if !bytes.Contains(data[start:i], []byte{0x1b, ']'}) {
			return true
		}
	}
	return false
}

// classifyNotification maps a notification message to an event type.
func classifyNotification(msg string) string {
	lower := strings.ToLower(msg)
	for _, kw := range []string{"permission", "waiting", "input", "attention", "approve"} {
		if strings.Contains(lower, kw) {
			return "waiting_input"
		}
	}
	for _, kw := range []string{"done", "complete", "finished"} {
		if strings.Contains(lower, kw) {
			return "task_complete"
		}
	}
	return "generic"
}
