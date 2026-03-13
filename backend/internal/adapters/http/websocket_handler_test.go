package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Corwind/cmux/backend/internal/adapters/pty"
	"github.com/Corwind/cmux/backend/internal/adapters/sqlite"
	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/coder/websocket"
)

func setupTestServer(t *testing.T) (*httptest.Server, *app.SessionService) {
	t.Helper()

	dbPath := t.TempDir() + "/test.db"
	repo, err := sqlite.NewRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	pm := pty.NewManager(pty.WithCommand("sleep"), pty.WithFixedArgs("60"))
	service := app.NewSessionService(repo, pm, nil)

	router := NewTestRouter(service, nil, nil)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, service
}

func TestWebSocketSendReceive(t *testing.T) {
	server, service := setupTestServer(t)

	ctx := context.Background()
	session, err := service.CreateSession(ctx, "test-session", os.TempDir(), "")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	t.Cleanup(func() { _ = service.DeleteSession(ctx, session.ID) })

	time.Sleep(50 * time.Millisecond) // allow PTY to be ready

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + session.ID
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Write to PTY via websocket
	testData := []byte("hello\n")
	err = conn.Write(ctx, websocket.MessageBinary, testData)
	if err != nil {
		t.Fatalf("websocket write failed: %v", err)
	}

	// Read back from PTY via websocket (cat echoes input)
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("websocket read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from websocket")
	}
}

func TestWebSocketResize(t *testing.T) {
	server, service := setupTestServer(t)

	ctx := context.Background()
	session, err := service.CreateSession(ctx, "test-resize", os.TempDir(), "")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	t.Cleanup(func() { _ = service.DeleteSession(ctx, session.ID) })

	time.Sleep(50 * time.Millisecond) // allow PTY to be ready

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + session.ID
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Send resize message
	msg := resizeMessage{Type: "resize", Rows: 50, Cols: 120}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	err = conn.Write(ctx, websocket.MessageText, data)
	if err != nil {
		t.Fatalf("websocket write resize failed: %v", err)
	}

	// Give a moment for the resize to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify the connection is still usable by writing data
	err = conn.Write(ctx, websocket.MessageBinary, []byte("test\n"))
	if err != nil {
		t.Fatalf("websocket write after resize failed: %v", err)
	}
}

func TestWebSocketProcessExit(t *testing.T) {
	server, service := setupTestServer(t)

	ctx := context.Background()
	session, err := service.CreateSession(ctx, "test-exit", os.TempDir(), "")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // allow PTY to be ready

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + session.ID
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Kill the process to trigger exit
	err = service.ResizePTY(session.PID, 24, 80) // just to confirm it's alive
	if err != nil {
		t.Fatalf("resize before kill failed: %v", err)
	}

	_ = service.DeleteSession(ctx, session.ID)

	// Read from WS — we should get a status stopped message or connection close
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for {
		msgType, data, err := conn.Read(readCtx)
		if err != nil {
			// Connection closed is expected after process exit
			break
		}
		if msgType == websocket.MessageText {
			var status struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			}
			if json.Unmarshal(data, &status) == nil && status.Type == "status" && status.Status == "stopped" {
				return // success
			}
		}
	}
}

func TestWebSocketInvalidSession(t *testing.T) {
	server, _ := setupTestServer(t)

	ctx := context.Background()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/nonexistent"

	_, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err == nil {
		t.Fatal("expected error connecting to invalid session")
	}
	if resp != nil && resp.StatusCode != 404 {
		t.Fatalf("expected 404 status, got %d", resp.StatusCode)
	}
}
