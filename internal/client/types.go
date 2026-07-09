package client

import "encoding/json"

// --- Organization ---

type Organization struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// --- Users / invites / workspace members ---

type User struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	AddedAt string `json:"added_at"`
}

type Invite struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	InvitedAt string `json:"invited_at"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
}

type WorkspaceMember struct {
	Type          string `json:"type"`
	UserID        string `json:"user_id"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceRole string `json:"workspace_role"`
}

// --- Workspaces ---

type Workspace struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	CreatedAt    string            `json:"created_at"`
	ArchivedAt   *string           `json:"archived_at"`
	DisplayColor string            `json:"display_color"`
	Tags         map[string]string `json:"tags"`
}

type CreateWorkspaceRequest struct {
	Name string            `json:"name"`
	Tags map[string]string `json:"tags,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name string             `json:"name,omitempty"`
	Tags *map[string]string `json:"tags,omitempty"`
}

// --- API keys ---

type APIKey struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	WorkspaceID    string `json:"workspace_id"`
	CreatedAt      string `json:"created_at"`
	PartialKeyHint string `json:"partial_key_hint"`
	Status         string `json:"status"`
	CreatedBy      struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"created_by"`
}

type UpdateAPIKeyRequest struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// --- Service accounts ---

type ServiceAccount struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	OrganizationRole string  `json:"organization_role"`
	CreatedAt        string  `json:"created_at"`
	ArchivedAt       *string `json:"archived_at"`
}

type CreateServiceAccountRequest struct {
	Name             string `json:"name"`
	OrganizationRole string `json:"organization_role"`
}

type UpdateServiceAccountRequest struct {
	Name string `json:"name,omitempty"`
}

// --- Federation issuers ---

// JWKS is the discriminated union controlling how Anthropic obtains the
// issuer's signing keys: discovery | explicit_url | inline.
type JWKS struct {
	Type          string            `json:"type"`
	DiscoveryBase string            `json:"discovery_base,omitempty"`
	URL           string            `json:"url,omitempty"`
	Keys          []json.RawMessage `json:"keys,omitempty"`
}

type FederationIssuer struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	IssuerURL  string  `json:"issuer_url"`
	JWKS       *JWKS   `json:"jwks"`
	CACertPEM  string  `json:"ca_cert_pem,omitempty"`
	CreatedAt  string  `json:"created_at"`
	ArchivedAt *string `json:"archived_at"`
}

type FederationIssuerRequest struct {
	Name      string `json:"name,omitempty"`
	IssuerURL string `json:"issuer_url,omitempty"`
	JWKS      *JWKS  `json:"jwks,omitempty"`
	CACertPEM string `json:"ca_cert_pem,omitempty"`
}

// --- Federation rules ---

type RuleMatch struct {
	SubjectPrefix string            `json:"subject_prefix,omitempty"`
	Audience      string            `json:"audience,omitempty"`
	Claims        map[string]string `json:"claims,omitempty"`
	Condition     string            `json:"condition,omitempty"`
}

type RuleTarget struct {
	Type             string `json:"type"`
	ServiceAccountID string `json:"service_account_id"`
}

type FederationRule struct {
	ID                     string      `json:"id"`
	Type                   string      `json:"type"`
	Name                   string      `json:"name"`
	IssuerID               string      `json:"issuer_id"`
	Match                  *RuleMatch  `json:"match"`
	Target                 *RuleTarget `json:"target"`
	WorkspaceID            string      `json:"workspace_id,omitempty"`
	AppliesToAllWorkspaces bool        `json:"applies_to_all_workspaces,omitempty"`
	OAuthScope             string      `json:"oauth_scope"`
	TokenLifetimeSeconds   int64       `json:"token_lifetime_seconds"`
	CreatedAt              string      `json:"created_at"`
	ArchivedAt             *string     `json:"archived_at"`
}

type CreateFederationRuleRequest struct {
	Name                   string      `json:"name"`
	IssuerID               string      `json:"issuer_id"`
	Match                  *RuleMatch  `json:"match"`
	Target                 *RuleTarget `json:"target"`
	WorkspaceID            string      `json:"workspace_id,omitempty"`
	AppliesToAllWorkspaces bool        `json:"applies_to_all_workspaces,omitempty"`
	OAuthScope             string      `json:"oauth_scope,omitempty"`
	TokenLifetimeSeconds   int64       `json:"token_lifetime_seconds,omitempty"`
}

type UpdateFederationRuleRequest struct {
	Name                 string     `json:"name,omitempty"`
	Match                *RuleMatch `json:"match,omitempty"`
	OAuthScope           string     `json:"oauth_scope,omitempty"`
	TokenLifetimeSeconds int64      `json:"token_lifetime_seconds,omitempty"`
}
