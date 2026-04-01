package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ok2ju/oversite/backend/internal/auth"
)

// AuthHandler handles OAuth login and callback routes.
type AuthHandler struct {
	oauth    *auth.OAuthService
	sessions auth.StateStore
	secure   bool // whether to set Secure flag on cookies
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(oauth *auth.OAuthService, sessions auth.StateStore, secure bool) *AuthHandler {
	return &AuthHandler{
		oauth:    oauth,
		sessions: sessions,
		secure:   secure,
	}
}

// HandleLogin initiates the Faceit OAuth flow by redirecting the user.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	url, err := h.oauth.AuthorizationURL(r.Context())
	if err != nil {
		slog.Error("generating authorization URL", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// HandleCallback processes the OAuth callback from Faceit.
func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "missing code or state parameter", http.StatusBadRequest)
		return
	}

	user, tokens, err := h.oauth.HandleCallback(r.Context(), code, state)
	if err != nil {
		slog.Error("handling OAuth callback", "error", err)
		http.Error(w, "authentication failed", http.StatusBadRequest)
		return
	}

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		slog.Error("generating session token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Store session data
	sessionData, _ := json.Marshal(map[string]interface{}{
		"user_id":       user.ID.String(),
		"faceit_id":     user.FaceitID,
		"nickname":      user.Nickname,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})

	if err := h.sessions.Create(r.Context(), "session:"+sessionToken, sessionData, 7*24*time.Hour); err != nil {
		slog.Error("creating session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
