package client

import (
	"context"
	"net/http"
	"net/url"
)

const (
	usersPath   = "/v1/organizations/users"
	invitesPath = "/v1/organizations/invites"
	apiKeysPath = "/v1/organizations/api_keys"
)

// --- Organization members ---

func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var u User
	if err := c.do(ctx, http.MethodGet, usersPath+"/"+id, nil, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUsers lists organization members, optionally filtered by exact email.
func (c *Client) ListUsers(ctx context.Context, email string) ([]User, error) {
	q := url.Values{}
	if email != "" {
		q.Set("email", email)
	}
	return listAll[User](ctx, c, usersPath, q)
}

func (c *Client) UpdateUserRole(ctx context.Context, id, role string) (*User, error) {
	var u User
	if err := c.do(ctx, http.MethodPost, usersPath+"/"+id, nil, map[string]string{"role": role}, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *Client) RemoveUser(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, usersPath+"/"+id, nil, nil, nil)
}

// --- Invites ---

func (c *Client) CreateInvite(ctx context.Context, email, role string) (*Invite, error) {
	body := map[string]string{"email": email, "role": role}
	var inv Invite
	if err := c.do(ctx, http.MethodPost, invitesPath, nil, body, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (c *Client) GetInvite(ctx context.Context, id string) (*Invite, error) {
	var inv Invite
	if err := c.do(ctx, http.MethodGet, invitesPath+"/"+id, nil, nil, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (c *Client) DeleteInvite(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, invitesPath+"/"+id, nil, nil, nil)
}

func (c *Client) ListInvites(ctx context.Context) ([]Invite, error) {
	return listAll[Invite](ctx, c, invitesPath, nil)
}

// --- API keys (create is Console-only; the API can list/read/update) ---

func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	var k APIKey
	if err := c.do(ctx, http.MethodGet, apiKeysPath+"/"+id, nil, nil, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (c *Client) UpdateAPIKey(ctx context.Context, id string, req UpdateAPIKeyRequest) (*APIKey, error) {
	var k APIKey
	if err := c.do(ctx, http.MethodPost, apiKeysPath+"/"+id, nil, req, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (c *Client) ListAPIKeys(ctx context.Context, status, workspaceID string) ([]APIKey, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if workspaceID != "" {
		q.Set("workspace_id", workspaceID)
	}
	return listAll[APIKey](ctx, c, apiKeysPath, q)
}
