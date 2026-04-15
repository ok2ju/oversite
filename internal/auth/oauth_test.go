package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testOAuthConfig(tokenURL string) OAuthConfig {
	return OAuthConfig{
		ClientID:        "test-client-id",
		AuthURL:         "https://cdn.faceit.com/widgets/sso/index.html",
		TokenURL:        tokenURL,
		RedirectURIBase: "http://127.0.0.1",
	}
}

func TestStartLoopbackFlow_FullFlow(t *testing.T) {
	// Fake token endpoint that returns valid tokens.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token endpoint method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}

		// Verify all required form params are present.
		requiredParams := []string{"grant_type", "code", "code_verifier", "redirect_uri", "client_id"}
		for _, p := range requiredParams {
			if r.FormValue(p) == "" {
				t.Errorf("missing form param %q", p)
			}
		}

		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("code = %q, want test-auth-code", r.FormValue("code"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want test-client-id", r.FormValue("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-456",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer tokenServer.Close()

	cfg := testOAuthConfig(tokenServer.URL)

	// The openBrowser mock: parse the auth URL to find the redirect_uri,
	// then simulate the callback by hitting it with ?code=test-auth-code.
	openBrowser := func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return fmt.Errorf("parsing auth URL: %w", err)
		}

		// Verify auth URL has all required params.
		q := parsed.Query()
		expectedParams := map[string]string{
			"response_type":         "code",
			"client_id":             "test-client-id",
			"code_challenge_method": "S256",
			"scope":                 "openid profile email membership",
		}
		for k, want := range expectedParams {
			if got := q.Get(k); got != want {
				t.Errorf("auth URL param %s = %q, want %q", k, got, want)
			}
		}
		if q.Get("code_challenge") == "" {
			t.Error("auth URL missing code_challenge param")
		}
		if q.Get("redirect_uri") == "" {
			t.Error("auth URL missing redirect_uri param")
		}

		// Simulate the OAuth provider redirecting back with a code.
		redirectURI := q.Get("redirect_uri")
		go func() {
			callbackURL := redirectURI + "?code=test-auth-code"
			resp, err := http.Get(callbackURL)
			if err != nil {
				t.Errorf("callback GET: %v", err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("callback status = %d, want 200", resp.StatusCode)
			}
		}()

		return nil
	}

	tokens, err := StartLoopbackFlow(context.Background(), cfg, openBrowser)
	if err != nil {
		t.Fatalf("StartLoopbackFlow: %v", err)
	}

	if tokens.AccessToken != "access-token-123" {
		t.Errorf("AccessToken = %q, want %q", tokens.AccessToken, "access-token-123")
	}
	if tokens.RefreshToken != "refresh-token-456" {
		t.Errorf("RefreshToken = %q, want %q", tokens.RefreshToken, "refresh-token-456")
	}
	if tokens.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", tokens.ExpiresIn)
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", tokens.TokenType, "Bearer")
	}
}

func TestStartLoopbackFlow_ContextCancellation(t *testing.T) {
	cfg := testOAuthConfig("http://127.0.0.1:0/token")

	// Create a context that is already cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	openBrowser := func(authURL string) error {
		// Don't simulate a callback — let the context cancellation take effect.
		return nil
	}

	_, err := StartLoopbackFlow(ctx, cfg, openBrowser)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("error = %q, want it to mention timeout or cancellation", err)
	}
}

func TestStartLoopbackFlow_TokenEndpointError(t *testing.T) {
	// Token endpoint that returns a 400 error.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"code expired"}`)
	}))
	defer tokenServer.Close()

	cfg := testOAuthConfig(tokenServer.URL)

	openBrowser := func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirectURI := parsed.Query().Get("redirect_uri")
		go func() {
			resp, err := http.Get(redirectURI + "?code=bad-code")
			if err != nil {
				t.Errorf("callback GET: %v", err)
				return
			}
			resp.Body.Close()
		}()
		return nil
	}

	_, err := StartLoopbackFlow(context.Background(), cfg, openBrowser)
	if err == nil {
		t.Fatal("expected error for 400 token response, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %q, want it to mention status 400", err)
	}
}

func TestStartLoopbackFlow_BrowserOpenError(t *testing.T) {
	cfg := testOAuthConfig("http://127.0.0.1:0/token")

	openBrowser := func(authURL string) error {
		return fmt.Errorf("browser not found")
	}

	_, err := StartLoopbackFlow(context.Background(), cfg, openBrowser)
	if err == nil {
		t.Fatal("expected error when browser fails to open, got nil")
	}
	if !strings.Contains(err.Error(), "opening browser") {
		t.Errorf("error = %q, want it to mention opening browser", err)
	}
}

func TestStartLoopbackFlow_CallbackNoCode(t *testing.T) {
	cfg := testOAuthConfig("http://127.0.0.1:0/token")

	openBrowser := func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirectURI := parsed.Query().Get("redirect_uri")
		go func() {
			// Simulate callback with error instead of code.
			resp, err := http.Get(redirectURI + "?error=access_denied")
			if err != nil {
				t.Errorf("callback GET: %v", err)
				return
			}
			resp.Body.Close()
		}()
		return nil
	}

	_, err := StartLoopbackFlow(context.Background(), cfg, openBrowser)
	if err == nil {
		t.Fatal("expected error for callback without code, got nil")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("error = %q, want it to mention access_denied", err)
	}
}

func TestExchangeCode_FormParams(t *testing.T) {
	var gotForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "tok",
			RefreshToken: "ref",
			ExpiresIn:    1800,
			TokenType:    "Bearer",
		})
	}))
	defer tokenServer.Close()

	cfg := testOAuthConfig(tokenServer.URL)

	resp, err := exchangeCode(context.Background(), cfg, "my-code", "my-verifier", "http://127.0.0.1:12345/callback")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}

	tests := []struct {
		param string
		want  string
	}{
		{"grant_type", "authorization_code"},
		{"code", "my-code"},
		{"code_verifier", "my-verifier"},
		{"redirect_uri", "http://127.0.0.1:12345/callback"},
		{"client_id", "test-client-id"},
	}
	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			if got := gotForm.Get(tt.param); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.param, got, tt.want)
			}
		})
	}

	if resp.AccessToken != "tok" {
		t.Errorf("AccessToken = %q, want %q", resp.AccessToken, "tok")
	}
}

func TestExchangeCode_BadJSON(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json")
	}))
	defer tokenServer.Close()

	cfg := testOAuthConfig(tokenServer.URL)
	_, err := exchangeCode(context.Background(), cfg, "code", "verifier", "http://127.0.0.1:1/callback")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing token response") {
		t.Errorf("error = %q, want it to mention parsing", err)
	}
}

func TestStartLoopbackFlow_Timeout(t *testing.T) {
	cfg := testOAuthConfig("http://127.0.0.1:0/token")

	// Use a very short timeout context to trigger the timeout path.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	openBrowser := func(authURL string) error {
		// Don't hit the callback — let it time out.
		return nil
	}

	_, err := StartLoopbackFlow(ctx, cfg, openBrowser)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to mention timed out", err)
	}
}
