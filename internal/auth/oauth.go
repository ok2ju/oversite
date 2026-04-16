package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig holds the OAuth 2.0 configuration for the desktop loopback flow.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string // injected at build time via ldflags
	AuthURL      string
	TokenURL     string
	RelayURL     string // HTTPS relay page (e.g. "https://ok2ju.github.io/oversite/oauth/callback")
}

// TokenResponse represents the token response from the OAuth provider.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// BrowserOpener is a function that opens a URL in the user's default browser.
// It is a function type (not an interface) for easy testing.
type BrowserOpener func(url string) error

// flowTimeout is the maximum time to wait for the OAuth callback.
const flowTimeout = 120 * time.Second

// successHTML is served to the user's browser after a successful callback.
const successHTML = "<html><body><h1>Authentication successful!</h1><p>You can close this tab.</p></body></html>"

// StartLoopbackFlow runs the full desktop OAuth 2.0 + PKCE loopback flow:
//  1. Generate PKCE verifier + challenge
//  2. Listen on a random loopback port
//  3. Open the authorization URL in the user's browser
//  4. Wait for the /callback with the authorization code
//  5. Exchange the code for tokens
//  6. Shut down the local server
func StartLoopbackFlow(ctx context.Context, cfg OAuthConfig, openBrowser BrowserOpener) (*TokenResponse, error) {
	// 1. PKCE
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generating PKCE verifier: %w", err)
	}
	challenge := GenerateCodeChallenge(verifier)

	// 2. Listen on random loopback port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting loopback listener: %w", err)
	}
	defer listener.Close() //nolint:errcheck

	port := listener.Addr().(*net.TCPAddr).Port

	// The redirect_uri registered with Faceit is the HTTPS relay page.
	// The relay page reads the loopback port from the state parameter
	// and forwards the authorization code to http://127.0.0.1:{port}/callback.
	redirectURI := cfg.RelayURL

	// 3. Build authorization URL
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirectURI},
		"state":                 {fmt.Sprintf("%d", port)},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"scope":                 {"openid profile email"},
		"redirect_popup":        {"true"},
	}
	authURL := cfg.AuthURL + "?" + params.Encode()

	// Channel to receive the authorization code from the callback handler.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// 5. Set up callback handler
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no code in callback"
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth callback error: %s", errMsg)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, successHTML)
		codeCh <- code
	})

	server := &http.Server{Handler: mux}

	// Start serving in background
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("loopback server error: %w", serveErr)
		}
	}()
	defer server.Close() //nolint:errcheck

	// 4. Open browser
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("opening browser: %w", err)
	}

	// 6. Wait for callback with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, flowTimeout)
	defer cancel()

	var code string
	select {
	case code = <-codeCh:
		// Got the authorization code
	case cbErr := <-errCh:
		return nil, cbErr
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("oauth flow timed out: %w", timeoutCtx.Err())
	}

	// 7. Exchange code for tokens
	return exchangeCode(timeoutCtx, cfg, code, verifier, redirectURI)
}

// exchangeCode exchanges an authorization code for tokens via POST to the
// token endpoint. Uses Basic Auth (client_id:client_secret) for confidential clients.
func exchangeCode(ctx context.Context, cfg OAuthConfig, code, verifier, redirectURI string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &tokenResp, nil
}
