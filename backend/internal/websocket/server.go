package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/ok2ju/oversite/backend/internal/auth"
	customMw "github.com/ok2ju/oversite/backend/internal/middleware"
)

// SessionStore is the minimal interface the WS server needs for auth.
// Satisfied by auth.RedisSessionStore.
type SessionStore interface {
	Get(ctx context.Context, token string) (*auth.SessionData, error)
}

// Server handles WebSocket upgrades and manages the hub.
type Server struct {
	hub      *Hub
	sessions SessionStore
	upgrader websocket.Upgrader
}

// NewServer creates a new WebSocket server.
func NewServer(hub *Hub, sessions SessionStore) *Server {
	return &Server{
		hub:      hub,
		sessions: sessions,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: restrict origins in production
				return true
			},
		},
	}
}

// HandleUpgrade validates the session cookie and upgrades the connection.
// Auth is checked BEFORE the WebSocket upgrade so invalid sessions get a
// standard HTTP 401 JSON response.
func (s *Server) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "id")
	if boardID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing board id"})
		return
	}

	// Validate session before upgrade.
	cookie, err := r.Cookie("session_token")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	session, err := s.sessions.Get(r.Context(), cookie.Value)
	if err != nil {
		if !errors.Is(err, auth.ErrSessionNotFound) {
			slog.Error("getting session in ws upgrade", "error", err, "request_id", middleware.GetReqID(r.Context()))
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Upgrade to WebSocket.
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err, "request_id", middleware.GetReqID(r.Context()))
		return
	}

	client := NewClient(s.hub, conn, boardID, session.UserID)
	s.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// Router creates a chi router for the WebSocket server with health endpoints.
func (s *Server) Router(health HealthHandler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMw.StructuredLogger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	r.Get("/ws/strat/{id}", s.HandleUpgrade)

	return r
}

// HealthHandler is the interface for health check endpoints.
type HealthHandler interface {
	Healthz(w http.ResponseWriter, r *http.Request)
	Readyz(w http.ResponseWriter, r *http.Request)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
