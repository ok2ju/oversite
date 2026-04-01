package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ok2ju/oversite/backend/internal/auth"
)

func TestFaceitClient_ExchangeCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected content-type application/x-www-form-urlencoded, got %q", ct)
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "client-id" || pass != "client-secret" {
			t.Errorf("unexpected basic auth: %q:%q", user, pass)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type authorization_code, got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-code" {
			t.Errorf("expected code test-code, got %q", r.FormValue("code"))
		}
		if r.FormValue("code_verifier") != "test-verifier" {
			t.Errorf("expected code_verifier test-verifier, got %q", r.FormValue("code_verifier"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-123",
			"refresh_token": "refresh-456",
			"expires_in":    3600,
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://localhost/callback",
		TokenURL:     server.URL,
	}
	client := auth.NewFaceitClient(cfg)

	tokens, err := client.ExchangeCode(context.Background(), "test-code", "test-verifier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.AccessToken != "access-123" {
		t.Errorf("expected access token 'access-123', got %q", tokens.AccessToken)
	}
	if tokens.RefreshToken != "refresh-456" {
		t.Errorf("expected refresh token 'refresh-456', got %q", tokens.RefreshToken)
	}
}

func TestFaceitClient_ExchangeCode_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{TokenURL: server.URL}
	client := auth.NewFaceitClient(cfg)

	_, err := client.ExchangeCode(context.Background(), "bad-code", "verifier")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestFaceitClient_ExchangeCode_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{TokenURL: server.URL}
	client := auth.NewFaceitClient(cfg)

	_, err := client.ExchangeCode(context.Background(), "code", "verifier")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestFaceitClient_GetUserInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer my-token" {
			t.Errorf("expected Authorization 'Bearer my-token', got %q", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"guid":     "player-id-123",
			"nickname": "testplayer",
			"avatar":   "https://example.com/avatar.png",
			"country":  "US",
		})
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{UserInfoURL: server.URL}
	client := auth.NewFaceitClient(cfg)

	info, err := client.GetUserInfo(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PlayerID != "player-id-123" {
		t.Errorf("expected PlayerID 'player-id-123', got %q", info.PlayerID)
	}
	if info.Nickname != "testplayer" {
		t.Errorf("expected Nickname 'testplayer', got %q", info.Nickname)
	}
}

func TestFaceitClient_GetUserInfo_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{UserInfoURL: server.URL}
	client := auth.NewFaceitClient(cfg)

	_, err := client.GetUserInfo(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestFaceitClient_GetUserInfo_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	cfg := auth.FaceitOAuthConfig{UserInfoURL: server.URL}
	client := auth.NewFaceitClient(cfg)

	_, err := client.GetUserInfo(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
