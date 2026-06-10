package fider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OAuthConfig represents a custom OAuth provider configuration in Fider.
type OAuthConfig struct {
	ID                int    `json:"id"`
	Provider          string `json:"provider"`
	DisplayName       string `json:"displayName"`
	Status            int    `json:"status"`
	ClientID          string `json:"clientID"`
	ClientSecret      string `json:"clientSecret"`
	AuthorizeURL      string `json:"authorizeURL"`
	TokenURL          string `json:"tokenURL"`
	ProfileURL        string `json:"profileURL"`
	Scope             string `json:"scope"`
	IsTrusted         bool   `json:"isTrusted"`
	JSONUserIDPath    string `json:"jsonUserIDPath"`
	JSONUserNamePath  string `json:"jsonUserNamePath"`
	JSONUserEmailPath string `json:"jsonUserEmailPath"`
	JSONUserRolesPath string `json:"jsonUserRolesPath"`
	AllowedRoles      string `json:"allowedRoles"`
}

// Client is an HTTP client for the Fider admin API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Fider API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

// GetOAuthConfig fetches a custom OAuth config by provider slug.
// Returns nil if not found (404).
func (c *Client) GetOAuthConfig(ctx context.Context, provider string) (*OAuthConfig, error) {
	resp, err := c.do(ctx, http.MethodGet, "/_api/admin/oauth/"+provider, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var cfg OAuthConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &cfg, nil
}

// SaveOAuthConfig creates or updates a custom OAuth config.
func (c *Client) SaveOAuthConfig(ctx context.Context, cfg OAuthConfig) error {
	resp, err := c.do(ctx, http.MethodPost, "/_api/admin/oauth", cfg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DisableOAuthConfig sets the OAuth config status to disabled.
// Fider has no delete endpoint for custom OAuth configs.
func (c *Client) DisableOAuthConfig(ctx context.Context, provider string) error {
	existing, err := c.GetOAuthConfig(ctx, provider)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	existing.Status = 1 // OAuthConfigDisabled
	return c.SaveOAuthConfig(ctx, *existing)
}
