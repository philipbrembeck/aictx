package copilot

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

// --- helpers ---

func formResp(w http.ResponseWriter, kv url.Values) {
	w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	fmt.Fprint(w, kv.Encode())
}

// --- RunDeviceFlow ---

func TestRunDeviceFlow_Success(t *testing.T) {
	// Mock device code endpoint.
	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formResp(w, url.Values{
			"device_code":      {"dev-code-123"},
			"user_code":        {"ABCD-1234"},
			"verification_uri": {"https://github.com/login/device"},
			"interval":         {"0"},
			"expires_in":       {"900"},
		})
	}))
	defer deviceSrv.Close()

	// Mock token endpoint: first poll returns pending, second returns token.
	pollCount := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		if pollCount == 1 {
			formResp(w, url.Values{"error": {"authorization_pending"}})
			return
		}
		formResp(w, url.Values{"access_token": {"gho_testtoken123"}})
	}))
	defer tokenSrv.Close()

	// Mock user endpoint.
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"login": "testuser"})
	}))
	defer userSrv.Close()

	// Override package vars.
	origDevice, origToken, origUser := DeviceCodeURL, TokenURL, UserURL
	origTTL, origInterval := DeviceCodeTTL, PollInterval
	DeviceCodeURL = deviceSrv.URL
	TokenURL = tokenSrv.URL
	UserURL = userSrv.URL
	DeviceCodeTTL = 30 * time.Second
	PollInterval = 0
	t.Cleanup(func() {
		DeviceCodeURL, TokenURL, UserURL = origDevice, origToken, origUser
		DeviceCodeTTL, PollInterval = origTTL, origInterval
	})

	token, err := RunDeviceFlow(context.Background())
	if err != nil {
		t.Fatalf("RunDeviceFlow() error: %v", err)
	}
	if token.Token != "gho_testtoken123" {
		t.Errorf("Token = %q, want gho_testtoken123", token.Token)
	}
	if token.Username != "testuser" {
		t.Errorf("Username = %q, want testuser", token.Username)
	}
	if pollCount < 2 {
		t.Errorf("pollCount = %d, want >= 2", pollCount)
	}
}

func TestRunDeviceFlow_Timeout(t *testing.T) {
	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formResp(w, url.Values{
			"device_code":      {"dev-code-123"},
			"user_code":        {"XXXX-9999"},
			"verification_uri": {"https://github.com/login/device"},
			"interval":         {"0"},
			"expires_in":       {"900"},
		})
	}))
	defer deviceSrv.Close()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formResp(w, url.Values{"error": {"authorization_pending"}})
	}))
	defer tokenSrv.Close()

	origDevice, origToken := DeviceCodeURL, TokenURL
	origTTL, origInterval := DeviceCodeTTL, PollInterval
	DeviceCodeURL = deviceSrv.URL
	TokenURL = tokenSrv.URL
	DeviceCodeTTL = 50 * time.Millisecond
	PollInterval = 0
	t.Cleanup(func() {
		DeviceCodeURL, TokenURL = origDevice, origToken
		DeviceCodeTTL, PollInterval = origTTL, origInterval
	})

	_, err := RunDeviceFlow(context.Background())
	if err == nil {
		t.Fatal("RunDeviceFlow() expected error on timeout, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want 'timed out'", err.Error())
	}
}

func TestRunDeviceFlow_AccessDenied(t *testing.T) {
	deviceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formResp(w, url.Values{
			"device_code":      {"dev-code-123"},
			"user_code":        {"ZZZZ-0000"},
			"verification_uri": {"https://github.com/login/device"},
			"interval":         {"0"},
			"expires_in":       {"900"},
		})
	}))
	defer deviceSrv.Close()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		formResp(w, url.Values{"error": {"access_denied"}})
	}))
	defer tokenSrv.Close()

	origDevice, origToken := DeviceCodeURL, TokenURL
	origTTL, origInterval := DeviceCodeTTL, PollInterval
	DeviceCodeURL = deviceSrv.URL
	TokenURL = tokenSrv.URL
	DeviceCodeTTL = 30 * time.Second
	PollInterval = 0
	t.Cleanup(func() {
		DeviceCodeURL, TokenURL = origDevice, origToken
		DeviceCodeTTL, PollInterval = origTTL, origInterval
	})

	_, err := RunDeviceFlow(context.Background())
	if err == nil {
		t.Fatal("RunDeviceFlow() expected error on access_denied, got nil")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error = %q, want 'access denied'", err.Error())
	}
}

// --- ExchangeToken ---

func TestExchangeToken_Success(t *testing.T) {
	expiresAt := time.Now().Add(30 * time.Minute).Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token gho_oauthtoken" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      "tid=copilot_api_token",
			"expires_at": expiresAt,
		})
	}))
	defer srv.Close()

	origURL := CopilotTokenURL
	CopilotTokenURL = srv.URL
	t.Cleanup(func() { CopilotTokenURL = origURL })

	ct, err := ExchangeToken("gho_oauthtoken")
	if err != nil {
		t.Fatalf("ExchangeToken() error: %v", err)
	}
	if ct.Token != "tid=copilot_api_token" {
		t.Errorf("Token = %q, want tid=copilot_api_token", ct.Token)
	}
	if ct.ExpiresAt.Unix() != expiresAt {
		t.Errorf("ExpiresAt = %v, want %v", ct.ExpiresAt.Unix(), expiresAt)
	}
}

func TestExchangeToken_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no subscription", http.StatusForbidden)
	}))
	defer srv.Close()

	origURL := CopilotTokenURL
	CopilotTokenURL = srv.URL
	t.Cleanup(func() { CopilotTokenURL = origURL })

	_, err := ExchangeToken("gho_badtoken")
	if err == nil {
		t.Fatal("ExchangeToken() expected error on 403, got nil")
	}
	if err != ErrNoCopilotSubscription {
		t.Errorf("error = %v, want ErrNoCopilotSubscription", err)
	}
}

// --- ListModels ---

func TestListModels_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o", "name": "GPT-4o", "capabilities": map[string]string{"type": "chat"}},
				{"id": "gpt-4o-mini", "name": "GPT-4o mini", "capabilities": map[string]string{"type": "chat"}},
				{"id": "text-embedding-3-large", "name": "Embedding", "capabilities": map[string]string{"type": "embeddings"}},
			},
		})
	}))
	defer srv.Close()

	origURL := ModelsURL
	ModelsURL = srv.URL
	t.Cleanup(func() { ModelsURL = origURL })

	models, err := ListModels("tid=test")
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	// Embedding model should be filtered out.
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	// Should be sorted: gpt-4o before gpt-4o-mini.
	if models[0].ID != "gpt-4o" || models[1].ID != "gpt-4o-mini" {
		t.Errorf("models = %v", models)
	}
	for _, m := range models {
		if !m.IsChat {
			t.Errorf("model %q has IsChat=false", m.ID)
		}
	}
}

func TestListModels_EmptyResponse_ReturnsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer srv.Close()

	origURL := ModelsURL
	ModelsURL = srv.URL
	t.Cleanup(func() { ModelsURL = origURL })

	models, err := ListModels("tid=test")
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("ListModels() returned empty slice, want fallback list")
	}
	// Should contain gpt-4o at minimum.
	found := false
	for _, m := range models {
		if m.ID == "gpt-4o" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("fallback list missing gpt-4o: %v", models)
	}
}
