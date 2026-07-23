package fider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// BootstrapTenantRequest is the payload for the internal bootstrap endpoint.
type BootstrapTenantRequest struct {
	Name                string `json:"name"`
	Subdomain           string `json:"subdomain"`
	CNAME               string `json:"cname"`
	AdminName           string `json:"adminName"`
	AdminEmail          string `json:"adminEmail"`
	Locale              string `json:"locale,omitempty"`
	Invitation          string `json:"invitation,omitempty"`
	WelcomeHeader       string `json:"welcomeHeader,omitempty"`
	WelcomeMessage      string `json:"welcomeMessage,omitempty"`
	IsPrivate           bool   `json:"isPrivate"`
	IsEmailAuthAllowed  bool   `json:"isEmailAuthAllowed"`
	IsFeedEnabled       bool   `json:"isFeedEnabled"`
	IsModerationEnabled bool   `json:"isModerationEnabled"`
	RegenerateAPIKey    bool   `json:"regenerateApiKey"`
}

// TenantState is the response returned by the bootstrap endpoints.
type TenantState struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Subdomain   string `json:"subdomain"`
	Status      int    `json:"status"`
	CNAME       string `json:"cname"`
	PublicURL   string `json:"publicUrl"`
	InternalURL string `json:"internalUrl"`
	AdminUserID int    `json:"adminUserId"`
	APIKey      string `json:"apiKey"`
}

// ClientConfig configures a Fider API client. Tenant-scoped operations use
// URL+APIKey; bootstrap operations use BootstrapURL+BootstrapToken. Either or
// both modes may be configured.
type ClientConfig struct {
	URL            string
	APIKey         string
	BootstrapURL   string
	BootstrapToken string
}

// Client is an HTTP client for the Fider admin and bootstrap APIs.
type Client struct {
	baseURL        string
	apiKey         string
	bootstrapURL   string
	bootstrapToken string
	httpClient     *http.Client
}

// NewClient creates a new Fider API client.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		baseURL:        strings.TrimRight(cfg.URL, "/"),
		apiKey:         cfg.APIKey,
		bootstrapURL:   strings.TrimRight(cfg.BootstrapURL, "/"),
		bootstrapToken: cfg.BootstrapToken,
		httpClient:     &http.Client{},
	}
}

// HasTenantAPI reports whether tenant-scoped (api_key) operations are configured.
func (c *Client) HasTenantAPI() bool {
	return c.baseURL != "" && c.apiKey != ""
}

// HasBootstrap reports whether bootstrap-mode operations are configured.
func (c *Client) HasBootstrap() bool {
	return c.bootstrapURL != "" && c.bootstrapToken != ""
}

func (c *Client) do(ctx context.Context, baseURL, token, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

// GetOAuthConfig fetches a custom OAuth config by provider slug (tenant mode).
// Returns nil if not found (404).
func (c *Client) GetOAuthConfig(ctx context.Context, provider string) (*OAuthConfig, error) {
	resp, err := c.do(ctx, c.baseURL, c.apiKey, http.MethodGet, "/_api/admin/oauth/"+provider, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeOAuthConfig(resp)
}

// SaveOAuthConfig creates or updates a custom OAuth config (tenant mode).
func (c *Client) SaveOAuthConfig(ctx context.Context, cfg OAuthConfig) error {
	resp, err := c.do(ctx, c.baseURL, c.apiKey, http.MethodPost, "/_api/admin/oauth", cfg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return expectOK(resp)
}

// TenantSettings represents the general settings for a Fider tenant.
type TenantSettings struct {
	Title          string `json:"title"`
	Invitation     string `json:"invitation"`
	WelcomeMessage string `json:"welcomeMessage"`
	WelcomeHeader  string `json:"welcomeHeader"`
	CNAME          string `json:"cname"`
	Locale         string `json:"locale"`
}

// UpdateTenantSettings updates the general settings for the current tenant (tenant mode).
// Fider has no GET endpoint for settings; state is the source of truth.
func (c *Client) UpdateTenantSettings(ctx context.Context, s TenantSettings) error {
	resp, err := c.do(ctx, c.baseURL, c.apiKey, http.MethodPost, "/_api/admin/settings/general", s)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return expectOK(resp)
}

// DisableOAuthConfig sets the OAuth config status to disabled (tenant mode).
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

// --- Bootstrap-mode operations ---

const bootstrapBasePath = "/_api/internal/bootstrap/tenant"

// BootstrapTenant creates or reconciles a tenant via the internal bootstrap API.
func (c *Client) BootstrapTenant(ctx context.Context, req BootstrapTenantRequest) (*TenantState, error) {
	resp, err := c.do(ctx, c.bootstrapURL, c.bootstrapToken, http.MethodPost, bootstrapBasePath, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeTenantState(resp)
}

// GetBootstrapTenant reads the current state of a tenant by subdomain. The admin
// email is passed so the response can include the admin user id and API key.
// Returns nil when the tenant does not exist (404).
func (c *Client) GetBootstrapTenant(ctx context.Context, subdomain, adminEmail string) (*TenantState, error) {
	path := bootstrapBasePath + "/" + url.PathEscape(subdomain)
	if adminEmail != "" {
		path += "?adminEmail=" + url.QueryEscape(adminEmail)
	}
	resp, err := c.do(ctx, c.bootstrapURL, c.bootstrapToken, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return decodeTenantState(resp)
}

// LockTenant sets a tenant to the locked (read-only) status.
func (c *Client) LockTenant(ctx context.Context, subdomain string) error {
	path := bootstrapBasePath + "/" + url.PathEscape(subdomain) + "/lock"
	resp, err := c.do(ctx, c.bootstrapURL, c.bootstrapToken, http.MethodPost, path, struct{}{})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return expectOK(resp)
}

// SaveOAuthConfigForTenant creates or updates a tenant's custom OAuth config
// through the bootstrap API, addressed by subdomain.
func (c *Client) SaveOAuthConfigForTenant(ctx context.Context, subdomain string, cfg OAuthConfig) error {
	path := bootstrapBasePath + "/" + url.PathEscape(subdomain) + "/oauth"
	resp, err := c.do(ctx, c.bootstrapURL, c.bootstrapToken, http.MethodPost, path, cfg)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return expectOK(resp)
}

// GetOAuthConfigForTenant reads a tenant's custom OAuth config through the
// bootstrap API. Returns nil if not found (404).
func (c *Client) GetOAuthConfigForTenant(ctx context.Context, subdomain, provider string) (*OAuthConfig, error) {
	path := bootstrapBasePath + "/" + url.PathEscape(subdomain) + "/oauth/" + url.PathEscape(provider)
	resp, err := c.do(ctx, c.bootstrapURL, c.bootstrapToken, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeOAuthConfig(resp)
}

func decodeTenantState(resp *http.Response) (*TenantState, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var state TenantState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &state, nil
}

func decodeOAuthConfig(resp *http.Response) (*OAuthConfig, error) {
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

func expectOK(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
