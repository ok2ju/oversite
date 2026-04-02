package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ok2ju/oversite/backend/internal/auth"
)

// AuthHandler handles OAuth login, callback, logout, and session routes.
type AuthHandler struct {
	oauth    *auth.OAuthService
	sessions auth.SessionStore
	secure   bool // whether to set Secure flag on cookies
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(oauth *auth.OAuthService, sessions auth.SessionStore, secure bool) *AuthHandler {
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

	sessionToken, err := h.sessions.Create(r.Context(), &auth.SessionData{
		UserID:       user.ID.String(),
		FaceitID:     user.FaceitID,
		Nickname:     user.Nickname,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
	if err != nil {
		slog.Error("creating session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

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

// HandleLogout clears the session and cookie.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		_ = h.sessions.Delete(r.Context(), cookie.Value)
	}

	// Clear the cookie regardless
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleMe returns the current user's public session info.
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.sessions.Get(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		slog.Error("getting session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"user_id":   session.UserID,
		"faceit_id": session.FaceitID,
		"nickname":  session.Nickname,
	})
}
