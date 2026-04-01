package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// FaceitClient is the real HTTP client for Faceit OAuth API.
type FaceitClient struct {
	httpClient *http.Client
	config     FaceitOAuthConfig
}

// NewFaceitClient creates a new FaceitClient.
func NewFaceitClient(cfg FaceitOAuthConfig) *FaceitClient {
	return &FaceitClient{
		httpClient: &http.Client{},
		config:     cfg,
	}
}

// ExchangeCode exchanges an authorization code for tokens via Faceit's token endpoint.
func (c *FaceitClient) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.config.RedirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.config.ClientID, c.config.ClientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, body)
	}

	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	return &tokens, nil
}

// GetUserInfo fetches the authenticated user's info from Faceit.
func (c *FaceitClient) GetUserInfo(ctx context.Context, accessToken string) (*FaceitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed (status %d): %s", resp.StatusCode, body)
	}

	var info FaceitUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding userinfo response: %w", err)
	}

	return &info, nil
}
