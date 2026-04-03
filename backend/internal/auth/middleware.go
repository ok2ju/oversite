package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type contextKey string

// UserIDKey is the context key for the authenticated user's ID.
const UserIDKey contextKey = "userID"

// UserIDFromContext extracts the authenticated user's ID from the request context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(UserIDKey).(string)
	return uid, ok
}

// RequireAuth returns a chi-compatible middleware that validates the session_token
// cookie. On valid session it injects the userID into the request context and
// refreshes the session TTL (sliding expiration). Missing or invalid sessions
// receive a 401 JSON response.
func RequireAuth(sessions SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err != nil {
				unauthorized(w)
				return
			}

			session, err := sessions.Get(r.Context(), cookie.Value)
			if err != nil {
				if !errors.Is(err, ErrSessionNotFound) {
					slog.Error("getting session in auth middleware", "error", err)
				}
				unauthorized(w)
				return
			}

			// Sliding expiration — refresh is best-effort; failure does not block the request.
			if err := sessions.Refresh(r.Context(), cookie.Value); err != nil {
				slog.Warn("refreshing session", "error", err)
			}

			ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
