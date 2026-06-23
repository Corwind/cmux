package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
)

// ptyBridge manages a single PTY reader goroutine per session
// and fans out output to the current WebSocket connection.
type ptyBridge struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (b *ptyBridge) setConn(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.conn = conn
}

func (b *ptyBridge) getConn() *websocket.Conn {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.conn
}

type WebSocketHandler struct {
	service        *app.SessionService
	hub            *notificationHub
	mu             sync.Mutex
	bridges        map[string]*ptyBridge
	originPatterns []string
}

type WebSocketOption func(*WebSocketHandler)

func WithOriginPatterns(patterns []string) WebSocketOption {
	return func(h *WebSocketHandler) {
		h.originPatterns = patterns
	}
}

func NewWebSocketHandler(service *app.SessionService, opts ...WebSocketOption) *WebSocketHandler {
	h := &WebSocketHandler{
		service:        service,
		hub:            newNotificationHub(),
		bridges:        make(map[string]*ptyBridge),
		originPatterns: []string{"*"},
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// SetService wires the SessionService into an already-constructed handler.
// This is used in main.go to break the circular dependency between hub
// initialisation and SessionService construction:
//
//  1. Construct WebSocketHandler (service=nil) — hub is ready.
//  2. Construct SessionService with hub as broadcaster.
//  3. Call SetService to give the handler its service.
func (h *WebSocketHandler) SetService(svc *app.SessionService) {
	h.service = svc
}

type resizeMessage struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func (h *WebSocketHandler) getBridge(sessionID string, handle *ports.PTYHandle) *ptyBridge {
	h.mu.Lock()
	defer h.mu.Unlock()

	bridge, ok := h.bridges[sessionID]
	if ok {
		return bridge
	}

	bridge = &ptyBridge{}
	h.bridges[sessionID] = bridge

	// Start a single PTY reader goroutine for this session.
	// This goroutine runs for the entire lifetime of the session regardless of
	// whether a WebSocket client is currently connected, so notification detection
	// works for background sessions too.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := handle.PTY.Read(buf)
			if err != nil {
				if err != io.EOF {
					slog.Error("PTY read error", "session_id", sessionID, "err", err)
				}
				if conn := bridge.getConn(); conn != nil {
					_ = conn.Write(context.Background(), websocket.MessageText, []byte(`{"type":"status","status":"stopped"}`))
					_ = conn.Close(websocket.StatusNormalClosure, "process exited")
				}
				h.mu.Lock()
				delete(h.bridges, sessionID)
				h.mu.Unlock()
				return
			}

			data := make([]byte, n)
			copy(data, buf[:n])

			if msg, eventType, ok := parseOscNotification(data); ok {
				name := sessionID
				if session, sErr := h.service.GetSession(context.Background(), sessionID); sErr == nil {
					name = session.Name
				}
				h.hub.broadcast(sessionNotificationMsg{
					SessionID:   sessionID,
					SessionName: name,
					Message:     msg,
					EventType:   eventType,
				})
			}

			if conn := bridge.getConn(); conn != nil {
				if err := conn.Write(context.Background(), websocket.MessageBinary, data); err != nil {
					slog.Error("websocket write error", "err", err)
				}
			}
		}
	}()

	return bridge
}

func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	handle, err := h.service.GetPTYHandle(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:  h.originPatterns,
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		slog.Error("websocket accept error", "err", err)
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	conn.SetReadLimit(128 * 1024) // 128KB read limit

	slog.Info("new WebSocket connection", "session_id", sessionID)
	bridge := h.getBridge(sessionID, handle)
	bridge.setConn(conn)

	ctx := r.Context()
	firstResize := true

	// WebSocket -> PTY (reads from browser, writes to PTY)
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			bridge.setConn(nil)
			return
		}

		switch msgType {
		case websocket.MessageBinary:
			if _, err := handle.PTY.Write(data); err != nil {
				slog.Error("PTY write error", "err", err)
				return
			}
		case websocket.MessageText:
			var msg resizeMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type == "resize" {
				session, _ := h.service.GetSession(ctx, sessionID)
				if firstResize {
					// On first resize (reconnect), nudge the size to force a full redraw.
					// Small delay ensures the bridge goroutine is ready to write.
					// Set size to cols-1, then back to real size — this triggers SIGWINCH twice,
					// making claude repaint its TUI.
					firstResize = false
					go func() {
						time.Sleep(100 * time.Millisecond)
						_ = h.service.ResizePTY(session.PID, msg.Rows, msg.Cols-1)
						time.Sleep(50 * time.Millisecond)
						_ = h.service.ResizePTY(session.PID, msg.Rows, msg.Cols)
					}()
				} else {
					_ = h.service.ResizePTY(session.PID, msg.Rows, msg.Cols)
				}
			}
		}
	}
}

// Hub returns the underlying notification hub as an app.SessionEventBroadcaster so
// it can be injected into SessionService via app.WithBroadcaster.
func (h *WebSocketHandler) Hub() app.SessionEventBroadcaster { return h.hub }

// HandleNotifications streams session notification events to a WebSocket client.
// A single global connection per browser tab is enough; the hub fans out to all subscribers.
func (h *WebSocketHandler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns:  h.originPatterns,
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		slog.Error("notification websocket accept error", "err", err)
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	ch, unsub := h.hub.subscribe()
	defer unsub()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				return
			}
		}
	}
}
