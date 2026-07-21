package http

import (
	"encoding/json"
	"sync"
	"time"
)

// sessionNotificationMsg is the JSON payload broadcast to notification subscribers.
type sessionNotificationMsg struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	Message     string `json:"message"`
	EventType   string `json:"event_type"`
}

// sessionStatusMsg is the JSON payload pushed when a session's lifecycle status
// changes (e.g. provisioning → running, provisioning → failed).
type sessionStatusMsg struct {
	Type        string `json:"type"`
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// worktreeDeletedMsg is the JSON payload pushed when an async worktree deletion
// finishes (success or git error). The frontend invalidates its worktrees query
// on receipt and surfaces the error string when present.
type worktreeDeletedMsg struct {
	Type       string `json:"type"`
	WorktreeID string `json:"worktree_id"`
	Error      string `json:"error,omitempty"`
}

const notificationDebounce = 30 * time.Second

// debounceKey identifies a (session, eventType) pair for rate-limiting.
type debounceKey struct {
	sessionID string
	eventType string
}

// notificationHub fans out session notifications to all subscribed WebSocket clients.
type notificationHub struct {
	mu           sync.RWMutex
	clients      map[chan []byte]struct{}
	debounceMu   sync.Mutex
	lastNotified map[debounceKey]time.Time
}

func newNotificationHub() *notificationHub {
	return &notificationHub{
		clients:      make(map[chan []byte]struct{}),
		lastNotified: make(map[debounceKey]time.Time),
	}
}

// shouldNotify returns true when the (session, eventType) pair hasn't fired
// within the debounce window, and records the current time if so.
// Only waiting_input and task_complete pass through; generic is always dropped.
func (h *notificationHub) shouldNotify(sessionID, eventType string) bool {
	if eventType != "waiting_input" {
		return false
	}
	key := debounceKey{sessionID, eventType}
	now := time.Now()
	h.debounceMu.Lock()
	defer h.debounceMu.Unlock()
	if last, ok := h.lastNotified[key]; ok && now.Sub(last) < notificationDebounce {
		return false
	}
	h.lastNotified[key] = now
	return true
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
// Notifications are filtered to waiting_input only and debounced per session
// to avoid badge flicker from rapid OSC sequences.
func (h *notificationHub) broadcast(n sessionNotificationMsg) {
	if !h.shouldNotify(n.SessionID, n.EventType) {
		return
	}

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

// BroadcastSessionStatus implements app.SessionEventBroadcaster. It serialises a
// sessionStatusMsg and delivers it to every current subscriber without debouncing —
// provisioning and failure events are always low-frequency and must never be dropped.
func (h *notificationHub) BroadcastSessionStatus(sessionID, sessionName, status, errMsg string) {
	data, err := json.Marshal(sessionStatusMsg{
		Type:        "session_status",
		SessionID:   sessionID,
		SessionName: sessionName,
		Status:      status,
		Error:       errMsg,
	})
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

// BroadcastWorktreeDeleted implements app.SessionEventBroadcaster. It serialises a
// worktreeDeletedMsg and delivers it to every current subscriber without debouncing —
// deletion completions are low-frequency and must never be dropped.
func (h *notificationHub) BroadcastWorktreeDeleted(worktreeID, errMsg string) {
	data, err := json.Marshal(worktreeDeletedMsg{
		Type:       "worktree_deleted",
		WorktreeID: worktreeID,
		Error:      errMsg,
	})
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
