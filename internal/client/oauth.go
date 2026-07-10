package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TokenExchangeInput carries the RFC 7523 jwt-bearer grant parameters for
// POST /v1/oauth/token.
type TokenExchangeInput struct {
	Assertion        string // the OIDC JWT from the workload's identity provider
	FederationRuleID string // fdrl_...
	OrganizationID   string // org UUID
	ServiceAccountID string // svac_...
	WorkspaceID      string // wrkspc_... or "default"; required when the rule spans multiple workspaces
}

type TokenResponse struct {
	AccessToken string `json:"access_token"` // sk-ant-oat01-...
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

// ExchangeToken exchanges a workload OIDC JWT for a short-lived Anthropic
// bearer token. It is a standalone function (not a Client method) because it
// runs before any credential exists.
func ExchangeToken(ctx context.Context, httpClient *http.Client, baseURL string, in TokenExchangeInput) (*TokenResponse, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	body := map[string]string{
		"grant_type":         "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"assertion":          in.Assertion,
		"federation_rule_id": in.FederationRuleID,
		"organization_id":    in.OrganizationID,
		"service_account_id": in.ServiceAccountID,
	}
	if in.WorkspaceID != "" {
		body["workspace_id"] = in.WorkspaceID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/oauth/token", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var env errorEnvelope
		apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
		if json.Unmarshal(respBody, &env) == nil && env.Error.Message != "" {
			apiErr.ErrType = env.Error.Type
			apiErr.Message = env.Error.Message
			apiErr.RequestID = env.RequestID
		}
		return nil, fmt.Errorf("token exchange failed (note: invalid_grant causes are logged server-side only; "+
			"check the Console authentication history page): %w", apiErr)
	}

	var tok TokenResponse
	if err := json.Unmarshal(respBody, &tok); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("token exchange returned an empty access_token")
	}
	return &tok, nil
}
