package client

import (
	"context"
	"net/http"
	"net/url"
)

// The WIF endpoints (service accounts, federation issuers, federation rules)
// reject admin API keys: every method here requires an org:admin OAuth token.

const (
	serviceAccountsPath = "/v1/organizations/service_accounts"
	federationIssuers   = "/v1/organizations/federation_issuers"
	federationRulesPath = "/v1/organizations/federation_rules"
)

// --- Service accounts ---

func (c *Client) CreateServiceAccount(ctx context.Context, req CreateServiceAccountRequest) (*ServiceAccount, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var sa ServiceAccount
	if err := c.do(ctx, http.MethodPost, serviceAccountsPath, nil, req, &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

func (c *Client) GetServiceAccount(ctx context.Context, id string) (*ServiceAccount, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var sa ServiceAccount
	if err := c.do(ctx, http.MethodGet, serviceAccountsPath+"/"+id, nil, nil, &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

func (c *Client) UpdateServiceAccount(ctx context.Context, id string, req UpdateServiceAccountRequest) (*ServiceAccount, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var sa ServiceAccount
	if err := c.do(ctx, http.MethodPost, serviceAccountsPath+"/"+id, nil, req, &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

// ArchiveServiceAccount soft-deletes a service account. Returns 400 while a
// live federation rule still references it.
func (c *Client) ArchiveServiceAccount(ctx context.Context, id string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, serviceAccountsPath+"/"+id+"/archive", nil, nil, nil)
}

func (c *Client) ListServiceAccounts(ctx context.Context, includeArchived bool) ([]ServiceAccount, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	q := url.Values{}
	if includeArchived {
		q.Set("include_archived", "true")
	}
	return listAllCursor[ServiceAccount](ctx, c, serviceAccountsPath, q)
}

// Explicit workspace memberships. Every service account is implicitly a
// member of the org's default workspace.

func (c *Client) AddServiceAccountWorkspace(ctx context.Context, serviceAccountID, workspaceID string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	body := map[string]string{"workspace_id": workspaceID}
	return c.do(ctx, http.MethodPost, serviceAccountsPath+"/"+serviceAccountID+"/workspaces", nil, body, nil)
}

func (c *Client) RemoveServiceAccountWorkspace(ctx context.Context, serviceAccountID, workspaceID string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, serviceAccountsPath+"/"+serviceAccountID+"/workspaces/"+workspaceID, nil, nil, nil)
}

// --- Federation issuers ---

func (c *Client) CreateFederationIssuer(ctx context.Context, req FederationIssuerRequest) (*FederationIssuer, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var iss FederationIssuer
	if err := c.do(ctx, http.MethodPost, federationIssuers, nil, req, &iss); err != nil {
		return nil, err
	}
	return &iss, nil
}

func (c *Client) GetFederationIssuer(ctx context.Context, id string) (*FederationIssuer, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var iss FederationIssuer
	if err := c.do(ctx, http.MethodGet, federationIssuers+"/"+id, nil, nil, &iss); err != nil {
		return nil, err
	}
	return &iss, nil
}

func (c *Client) UpdateFederationIssuer(ctx context.Context, id string, req FederationIssuerRequest) (*FederationIssuer, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var iss FederationIssuer
	if err := c.do(ctx, http.MethodPost, federationIssuers+"/"+id, nil, req, &iss); err != nil {
		return nil, err
	}
	return &iss, nil
}

func (c *Client) ArchiveFederationIssuer(ctx context.Context, id string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, federationIssuers+"/"+id+"/archive", nil, nil, nil)
}

func (c *Client) ListFederationIssuers(ctx context.Context, includeArchived bool) ([]FederationIssuer, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	q := url.Values{}
	if includeArchived {
		q.Set("include_archived", "true")
	}
	return listAllCursor[FederationIssuer](ctx, c, federationIssuers, q)
}

// --- Federation rules ---

func (c *Client) CreateFederationRule(ctx context.Context, req CreateFederationRuleRequest) (*FederationRule, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var rule FederationRule
	if err := c.do(ctx, http.MethodPost, federationRulesPath, nil, req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) GetFederationRule(ctx context.Context, id string) (*FederationRule, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var rule FederationRule
	if err := c.do(ctx, http.MethodGet, federationRulesPath+"/"+id, nil, nil, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) UpdateFederationRule(ctx context.Context, id string, req UpdateFederationRuleRequest) (*FederationRule, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	var rule FederationRule
	if err := c.do(ctx, http.MethodPost, federationRulesPath+"/"+id, nil, req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (c *Client) ArchiveFederationRule(ctx context.Context, id string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, federationRulesPath+"/"+id+"/archive", nil, nil, nil)
}

func (c *Client) ListFederationRules(ctx context.Context, issuerID string, includeArchived bool) ([]FederationRule, error) {
	if err := c.requireOAuth(); err != nil {
		return nil, err
	}
	q := url.Values{}
	if issuerID != "" {
		q.Set("issuer_id", issuerID)
	}
	if includeArchived {
		q.Set("include_archived", "true")
	}
	return listAllCursor[FederationRule](ctx, c, federationRulesPath, q)
}

// Rule ↔ workspace enablement sub-resource.

func (c *Client) AddFederationRuleWorkspace(ctx context.Context, ruleID, workspaceID string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	body := map[string]string{"workspace_id": workspaceID}
	return c.do(ctx, http.MethodPost, federationRulesPath+"/"+ruleID+"/workspaces", nil, body, nil)
}

func (c *Client) RemoveFederationRuleWorkspace(ctx context.Context, ruleID, workspaceID string) error {
	if err := c.requireOAuth(); err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, federationRulesPath+"/"+ruleID+"/workspaces/"+workspaceID, nil, nil, nil)
}
