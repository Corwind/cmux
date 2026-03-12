package http

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

type WebSocketHandler struct {
	service *app.SessionService
}

func NewWebSocketHandler(service *app.SessionService) *WebSocketHandler {
	return &WebSocketHandler{service: service}
}

type resizeMessage struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	handle, err := h.service.GetPTYHandle(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:5173"},
	})
	if err != nil {
		log.Printf("websocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()

	// PTY -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := handle.PTY.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error: %v", err)
				}
				conn.Write(ctx, websocket.MessageText, []byte(`{"type":"status","status":"stopped"}`))
				conn.Close(websocket.StatusNormalClosure, "process exited")
				return
			}
			if err := conn.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
				log.Printf("websocket write error: %v", err)
				return
			}
		}
	}()

	// WebSocket -> PTY
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("websocket read error: %v", err)
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
				_ = h.service.ResizePTY(session.PID, msg.Rows, msg.Cols)
			}
		}
	}
}
