// Package copilot handles GitHub Copilot OAuth Device Flow authentication
// and Copilot API token exchange.
package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Package-level URL vars so tests can override them with httptest servers.
var (
	DeviceCodeURL   = "https://github.com/login/device/code"
	TokenURL        = "https://github.com/login/oauth/access_token"
	CopilotTokenURL = "https://api.github.com/copilot_internal/v2/token"
	ModelsURL       = "https://api.githubcopilot.com/models"
	UserURL         = "https://api.github.com/user"
)

// Package-level timing vars so tests can override them.
var (
	DeviceCodeTTL = 15 * time.Minute
	PollInterval  = 5 * time.Second
)

const (
	clientID           = "Iv1.b507a08c87ecfe98"
	CopilotAPIEndpoint = "https://api.githubcopilot.com"
)

// ErrNoCopilotSubscription is returned when the user's GitHub account does not
// have an active Copilot subscription.
var ErrNoCopilotSubscription = errors.New("no active GitHub Copilot subscription")

// OAuthToken holds the result of a completed Device Flow.
type OAuthToken struct {
	Token    string
	Username string
}

// CopilotToken is a short-lived API token (~30 min) used to authenticate
// requests to the GitHub Copilot API.
type CopilotToken struct {
	Token     string
	ExpiresAt time.Time
}

// Model represents a model available via the Copilot API.
type Model struct {
	ID     string
	Name   string
	IsChat bool
}

// RequiredHeaders returns the headers the Copilot API requires on every request.
func RequiredHeaders() map[string]string {
	return map[string]string{
		"Editor-Version":         "vscode/1.85.0",
		"Editor-Plugin-Version":  "copilot-chat/0.12.0",
		"Copilot-Integration-Id": "vscode-chat",
		"OpenAI-Intent":          "conversation-panel",
	}
}

// RunDeviceFlow performs the GitHub OAuth 2.0 Device Flow and returns the
// resulting OAuth token with the authenticated username. The caller's ctx
// can be used to cancel early.
func RunDeviceFlow(ctx context.Context) (*OAuthToken, error) {
	// Step 1: Request device and user codes.
	resp, err := http.PostForm(DeviceCodeURL, url.Values{
		"client_id": {clientID},
		"scope":     {"read:user"},
	})
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("reading device code response: %w", err)
	}

	params, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}

	deviceCode := params.Get("device_code")
	userCode := params.Get("user_code")
	verificationURI := params.Get("verification_uri")
	if deviceCode == "" || userCode == "" {
		return nil, fmt.Errorf("invalid device code response: %s", string(body))
	}

	// Use the interval from the response if provided; fall back to the package var.
	interval := PollInterval
	if ivStr := params.Get("interval"); ivStr != "" {
		var ivSec int
		if _, parseErr := fmt.Sscanf(ivStr, "%d", &ivSec); parseErr == nil && ivSec > 0 {
			interval = time.Duration(ivSec) * time.Second
		}
	}

	fmt.Printf("  Open %s and enter: %s\n", verificationURI, userCode)
	fmt.Printf("  Waiting for authorization")

	// Step 2: Poll for access token.
	deadline := time.Now().Add(DeviceCodeTTL)
	for {
		if time.Now().After(deadline) {
			fmt.Println()
			return nil, fmt.Errorf("authorization timed out — please try again")
		}

		select {
		case <-ctx.Done():
			fmt.Println()
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		fmt.Printf(".")

		tokenResp, err := http.PostForm(TokenURL, url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			return nil, fmt.Errorf("polling for token: %w", err)
		}
		pollBody, _ := io.ReadAll(tokenResp.Body)
		tokenResp.Body.Close()

		pollParams, err := url.ParseQuery(string(pollBody))
		if err != nil {
			return nil, fmt.Errorf("parsing token response: %w", err)
		}

		if accessToken := pollParams.Get("access_token"); accessToken != "" {
			fmt.Println()
			username, err := GetUsername(accessToken)
			if err != nil {
				username = "unknown"
			}
			return &OAuthToken{Token: accessToken, Username: username}, nil
		}

		switch pollParams.Get("error") {
		case "authorization_pending":
			// Keep polling.
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			fmt.Println()
			return nil, fmt.Errorf("device code expired — please try again")
		case "access_denied":
			fmt.Println()
			return nil, fmt.Errorf("access denied by user")
		default:
			if e := pollParams.Get("error"); e != "" {
				fmt.Println()
				return nil, fmt.Errorf("authorization error: %s", e)
			}
			// Empty response: keep polling.
		}
	}
}

// ExchangeToken exchanges a GitHub OAuth token for a short-lived Copilot API
// token. Returns ErrNoCopilotSubscription on 401/403.
func ExchangeToken(oauthToken string) (*CopilotToken, error) {
	req, err := http.NewRequest(http.MethodGet, CopilotTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building token exchange request: %w", err)
	}
	req.Header.Set("Authorization", "token "+oauthToken)
	for k, v := range RequiredHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchanging copilot token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrNoCopilotSubscription
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing copilot token response: %w", err)
	}
	if result.Token == "" {
		return nil, fmt.Errorf("copilot token response missing token field")
	}

	return &CopilotToken{
		Token:     result.Token,
		ExpiresAt: time.Unix(result.ExpiresAt, 0),
	}, nil
}

// modelsResponse mirrors the OpenAI-compatible /models response from the Copilot API.
type modelsResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Capabilities struct {
			Type string `json:"type"`
		} `json:"capabilities"`
	} `json:"data"`
}

// fallbackModels is returned when the API returns nothing useful.
var fallbackModels = []Model{
	{ID: "gpt-4o", Name: "gpt-4o", IsChat: true},
	{ID: "gpt-4o-mini", Name: "gpt-4o-mini", IsChat: true},
}

// ListModels fetches the models available via the Copilot API, filtered to
// chat-capable models. Returns a hardcoded fallback if the API response is empty.
func ListModels(copilotToken string) ([]Model, error) {
	req, err := http.NewRequest(http.MethodGet, ModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+copilotToken)
	for k, v := range RequiredHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackModels, nil
	}

	var mr modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return fallbackModels, nil
	}

	var models []Model
	for _, m := range mr.Data {
		// Include "chat" type models, and also models with no type specified
		// (to be inclusive of future API changes).
		if m.Capabilities.Type == "chat" || m.Capabilities.Type == "" {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			models = append(models, Model{ID: m.ID, Name: name, IsChat: true})
		}
	}

	if len(models) == 0 {
		return fallbackModels, nil
	}

	// Sort alphabetically by ID for consistent display.
	for i := 1; i < len(models); i++ {
		for j := i; j > 0 && models[j].ID < models[j-1].ID; j-- {
			models[j], models[j-1] = models[j-1], models[j]
		}
	}

	return models, nil
}

// GetUsername fetches the GitHub login name for the given OAuth token.
func GetUsername(oauthToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, UserURL, nil)
	if err != nil {
		return "", fmt.Errorf("building user request: %w", err)
	}
	req.Header.Set("Authorization", "token "+oauthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching user: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing user response: %w", err)
	}

	return result.Login, nil
}
