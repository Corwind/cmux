package http

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
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
	service *app.SessionService
	mu      sync.Mutex
	bridges map[string]*ptyBridge
}

func NewWebSocketHandler(service *app.SessionService) *WebSocketHandler {
	return &WebSocketHandler{
		service: service,
		bridges: make(map[string]*ptyBridge),
	}
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

	// Start a single PTY reader goroutine for this session
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := handle.PTY.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error for session %s: %v", sessionID, err)
				}
				if conn := bridge.getConn(); conn != nil {
					conn.Write(context.Background(), websocket.MessageText, []byte(`{"type":"status","status":"stopped"}`))
					conn.Close(websocket.StatusNormalClosure, "process exited")
				}
				h.mu.Lock()
				delete(h.bridges, sessionID)
				h.mu.Unlock()
				return
			}

			if conn := bridge.getConn(); conn != nil {
				data := make([]byte, n)
				copy(data, buf[:n])
				if err := conn.Write(context.Background(), websocket.MessageBinary, data); err != nil {
					log.Printf("websocket write error: %v", err)
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
		OriginPatterns:  []string{"localhost:5173"},
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	conn.SetReadLimit(128 * 1024) // 128KB read limit

	log.Printf("new WebSocket connection for session %s", sessionID)
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
				log.Printf("PTY write error: %v", err)
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
