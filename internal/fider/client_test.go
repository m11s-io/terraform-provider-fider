package fider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOAuthConfigNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/_api/admin/oauth/github" {
			t.Fatalf("request = %s %s, want GET /_api/admin/oauth/github", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tenant-key" {
			t.Errorf("Authorization = %q, want bearer tenant key", got)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{URL: server.URL + "/", APIKey: "tenant-key"})
	cfg, err := client.GetOAuthConfig(context.Background(), "github")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Fatalf("GetOAuthConfig() = %#v, want nil for 404", cfg)
	}
}

func TestSaveOAuthConfigSendsTenantPayload(t *testing.T) {
	want := OAuthConfig{Provider: "github", DisplayName: "GitHub", Status: 2, ClientID: "client"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/_api/admin/oauth" {
			t.Fatalf("request = %s %s, want POST /_api/admin/oauth", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tenant-key" {
			t.Errorf("Authorization = %q, want bearer tenant key", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var got OAuthConfig
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("request body = %#v, want %#v", got, want)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{URL: server.URL, APIKey: "tenant-key"})
	if err := client.SaveOAuthConfig(context.Background(), want); err != nil {
		t.Fatal(err)
	}
}

func TestGetBootstrapTenantEscapesSubdomainAndUsesBootstrapToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/_api/internal/bootstrap/tenant/team%2Fblue" {
			t.Errorf("request = %s %s, want escaped tenant path", r.Method, r.URL.EscapedPath())
		}
		if got := r.URL.Query().Get("adminEmail"); got != "admin+test@example.com" {
			t.Errorf("adminEmail = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer bootstrap-key" {
			t.Errorf("Authorization = %q, want bootstrap token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"name":"Blue Team","subdomain":"team/blue"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{BootstrapURL: server.URL, BootstrapToken: "bootstrap-key"})
	state, err := client.GetBootstrapTenant(context.Background(), "team/blue", "admin+test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil || state.ID != 7 || state.Subdomain != "team/blue" {
		t.Fatalf("GetBootstrapTenant() = %#v", state)
	}
}
