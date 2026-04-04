package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorillaWs "github.com/gorilla/websocket"

	"github.com/ok2ju/oversite/backend/internal/auth"
	ws "github.com/ok2ju/oversite/backend/internal/websocket"
)

// --- Mock session store ---

type mockSessionStore struct {
	sessions map[string]*auth.SessionData
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*auth.SessionData)}
}

func (m *mockSessionStore) Get(_ context.Context, token string) (*auth.SessionData, error) {
	d, ok := m.sessions[token]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return d, nil
}

// --- Mock health handler ---

type mockHealthHandler struct{}

func (h *mockHealthHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *mockHealthHandler) Readyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// --- Helpers ---

func setupServer(t *testing.T, sessions *mockSessionStore) (*httptest.Server, *ws.Hub) {
	t.Helper()
	hub := ws.NewHub()
	go hub.Run()
	server := ws.NewServer(hub, sessions)
	router := server.Router(&mockHealthHandler{})
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)
	return ts, hub
}

func dialWS(t *testing.T, serverURL, boardID, sessionToken string) *gorillaWs.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/strat/" + boardID
	header := http.Header{}
	header.Set("Cookie", "session_token="+sessionToken)

	conn, resp, err := gorillaWs.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial failed with status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.WriteMessage(gorillaWs.CloseMessage,
			gorillaWs.FormatCloseMessage(gorillaWs.CloseNormalClosure, ""))
		_ = conn.Close()
	})
	return conn
}

// waitForClients polls until the expected number of clients appears in the room.
func waitForClients(t *testing.T, hub *ws.Hub, boardID string, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if hub.ClientsInRoom(boardID) == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d clients in room %s (got %d)", want, boardID, hub.ClientsInRoom(boardID))
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// --- Tests ---

func TestHandleUpgrade_NoCookie_Returns401(t *testing.T) {
	sessions := newMockSessionStore()
	hub := ws.NewHub()
	go hub.Run()
	server := ws.NewServer(hub, sessions)
	router := server.Router(&mockHealthHandler{})
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Attempt WebSocket dial without cookie — should fail.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/strat/board-1"
	_, resp, err := gorillaWs.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail, but it succeeded")
	}
	if resp == nil {
		t.Fatal("expected HTTP response, got nil")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleUpgrade_InvalidSession_Returns401(t *testing.T) {
	sessions := newMockSessionStore()
	hub := ws.NewHub()
	go hub.Run()
	server := ws.NewServer(hub, sessions)
	router := server.Router(&mockHealthHandler{})
	ts := httptest.NewServer(router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/strat/board-1"
	header := http.Header{}
	header.Set("Cookie", "session_token=invalid-token")

	_, resp, err := gorillaWs.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected dial to fail, but it succeeded")
	}
	if resp == nil {
		t.Fatal("expected HTTP response, got nil")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleUpgrade_ValidSession_Upgrades(t *testing.T) {
	sessions := newMockSessionStore()
	sessions.sessions["valid-token"] = &auth.SessionData{
		UserID:   "user-1",
		FaceitID: "faceit-1",
		Nickname: "player1",
	}

	ts, hub := setupServer(t, sessions)
	conn := dialWS(t, ts.URL, "board-1", "valid-token")
	_ = conn

	waitForClients(t, hub, "board-1", 1)
	if got := hub.ClientsInRoom("board-1"); got != 1 {
		t.Errorf("ClientsInRoom = %d, want 1", got)
	}
}

func TestTwoClients_ExchangeMessages(t *testing.T) {
	sessions := newMockSessionStore()
	sessions.sessions["token-a"] = &auth.SessionData{UserID: "user-a"}
	sessions.sessions["token-b"] = &auth.SessionData{UserID: "user-b"}

	ts, hub := setupServer(t, sessions)
	connA := dialWS(t, ts.URL, "board-1", "token-a")
	connB := dialWS(t, ts.URL, "board-1", "token-b")
	waitForClients(t, hub, "board-1", 2)

	// A sends, B receives.
	msgA := []byte{0x01, 0x02, 0x03}
	if err := connA.WriteMessage(gorillaWs.BinaryMessage, msgA); err != nil {
		t.Fatalf("A write: %v", err)
	}

	_ = connB.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, data, err := connB.ReadMessage()
	if err != nil {
		t.Fatalf("B read: %v", err)
	}
	if msgType != gorillaWs.BinaryMessage {
		t.Errorf("B msgType = %d, want BinaryMessage(%d)", msgType, gorillaWs.BinaryMessage)
	}
	if string(data) != string(msgA) {
		t.Errorf("B got %v, want %v", data, msgA)
	}

	// B sends, A receives.
	msgB := []byte{0x04, 0x05}
	if err := connB.WriteMessage(gorillaWs.BinaryMessage, msgB); err != nil {
		t.Fatalf("B write: %v", err)
	}

	_ = connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, data, err = connA.ReadMessage()
	if err != nil {
		t.Fatalf("A read: %v", err)
	}
	if msgType != gorillaWs.BinaryMessage {
		t.Errorf("A msgType = %d, want BinaryMessage(%d)", msgType, gorillaWs.BinaryMessage)
	}
	if string(data) != string(msgB) {
		t.Errorf("A got %v, want %v", data, msgB)
	}
}

func TestDisconnect_OtherClientContinues(t *testing.T) {
	sessions := newMockSessionStore()
	sessions.sessions["token-a"] = &auth.SessionData{UserID: "user-a"}
	sessions.sessions["token-b"] = &auth.SessionData{UserID: "user-b"}
	sessions.sessions["token-c"] = &auth.SessionData{UserID: "user-c"}

	ts, hub := setupServer(t, sessions)
	connA := dialWS(t, ts.URL, "board-1", "token-a")
	connB := dialWS(t, ts.URL, "board-1", "token-b")
	connC := dialWS(t, ts.URL, "board-1", "token-c")
	waitForClients(t, hub, "board-1", 3)

	// Disconnect A.
	_ = connA.WriteMessage(gorillaWs.CloseMessage,
		gorillaWs.FormatCloseMessage(gorillaWs.CloseNormalClosure, ""))
	_ = connA.Close()

	waitForClients(t, hub, "board-1", 2)

	// B sends, C receives — connection still works.
	msg := []byte{0xFF}
	if err := connB.WriteMessage(gorillaWs.BinaryMessage, msg); err != nil {
		t.Fatalf("B write: %v", err)
	}

	_ = connC.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := connC.ReadMessage()
	if err != nil {
		t.Fatalf("C read: %v", err)
	}
	if string(data) != string(msg) {
		t.Errorf("C got %v, want %v", data, msg)
	}
}

func TestDifferentRooms_NoLeakage(t *testing.T) {
	sessions := newMockSessionStore()
	sessions.sessions["token-1"] = &auth.SessionData{UserID: "user-1"}
	sessions.sessions["token-2"] = &auth.SessionData{UserID: "user-2"}

	ts, hub := setupServer(t, sessions)
	conn1 := dialWS(t, ts.URL, "board-1", "token-1")
	conn2 := dialWS(t, ts.URL, "board-2", "token-2")
	waitForClients(t, hub, "board-1", 1)
	waitForClients(t, hub, "board-2", 1)

	// Client in room-1 sends.
	msg := []byte{0xDE, 0xAD}
	if err := conn1.WriteMessage(gorillaWs.BinaryMessage, msg); err != nil {
		t.Fatalf("conn1 write: %v", err)
	}

	// Client in room-2 should NOT receive.
	_ = conn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err := conn2.ReadMessage()
	if err == nil {
		t.Error("conn2 in room-2 should NOT have received a message from room-1")
	}
}
