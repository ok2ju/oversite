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
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AuthURL:      "https://cdn.faceit.com/widgets/sso/index.html",
		TokenURL:     tokenURL,
		RelayURL:     "https://example.com/oauth/callback",
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

		// Verify Basic Auth is used (confidential client).
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("token request missing Basic Auth")
		}
		if user != "test-client-id" {
			t.Errorf("Basic Auth user = %q, want test-client-id", user)
		}
		if pass != "test-client-secret" {
			t.Errorf("Basic Auth pass = %q, want test-client-secret", pass)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}

		// Verify required form params (client_id no longer sent as form param).
		requiredParams := []string{"grant_type", "code", "code_verifier", "redirect_uri"}
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
		if r.FormValue("redirect_uri") != "https://example.com/oauth/callback" {
			t.Errorf("redirect_uri = %q, want relay URL", r.FormValue("redirect_uri"))
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

	// The openBrowser mock: parse the auth URL, read the state param (loopback port),
	// then simulate the relay page by hitting the loopback server with ?code=.
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
			"redirect_uri":          "https://example.com/oauth/callback",
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
		if q.Get("state") == "" {
			t.Error("auth URL missing state param (loopback port)")
		}

		// Simulate the relay page: read port from state, redirect to loopback.
		port := q.Get("state")
		go func() {
			callbackURL := fmt.Sprintf("http://127.0.0.1:%s/callback?code=test-auth-code", port)
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
		port := parsed.Query().Get("state")
		go func() {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/callback?code=bad-code", port))
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
		port := parsed.Query().Get("state")
		go func() {
			// Simulate callback with error instead of code.
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/callback?error=access_denied", port))
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
	var gotBasicUser, gotBasicPass string
	var gotBasicOK bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBasicUser, gotBasicPass, gotBasicOK = r.BasicAuth()
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

	resp, err := exchangeCode(context.Background(), cfg, "my-code", "my-verifier", "https://example.com/oauth/callback")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}

	// Verify Basic Auth credentials.
	if !gotBasicOK {
		t.Error("expected Basic Auth on token request")
	}
	if gotBasicUser != "test-client-id" {
		t.Errorf("Basic Auth user = %q, want test-client-id", gotBasicUser)
	}
	if gotBasicPass != "test-client-secret" {
		t.Errorf("Basic Auth pass = %q, want test-client-secret", gotBasicPass)
	}

	// Verify form params (client_id is no longer a form param).
	tests := []struct {
		param string
		want  string
	}{
		{"grant_type", "authorization_code"},
		{"code", "my-code"},
		{"code_verifier", "my-verifier"},
		{"redirect_uri", "https://example.com/oauth/callback"},
	}
	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			if got := gotForm.Get(tt.param); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.param, got, tt.want)
			}
		})
	}

	// client_id should NOT be in form body (sent via Basic Auth instead).
	if got := gotForm.Get("client_id"); got != "" {
		t.Errorf("client_id should not be in form body, got %q", got)
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
	_, err := exchangeCode(context.Background(), cfg, "code", "verifier", "https://example.com/oauth/callback")
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
