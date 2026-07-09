package client

import (
	"context"
	"net/http"
	"net/url"
)

const workspacesPath = "/v1/organizations/workspaces"

func (c *Client) GetOrganization(ctx context.Context) (*Organization, error) {
	var org Organization
	if err := c.do(ctx, http.MethodGet, "/v1/organizations/me", nil, nil, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

func (c *Client) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*Workspace, error) {
	var ws Workspace
	if err := c.do(ctx, http.MethodPost, workspacesPath, nil, req, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

func (c *Client) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	var ws Workspace
	if err := c.do(ctx, http.MethodGet, workspacesPath+"/"+id, nil, nil, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, id string, req UpdateWorkspaceRequest) (*Workspace, error) {
	var ws Workspace
	if err := c.do(ctx, http.MethodPost, workspacesPath+"/"+id, nil, req, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

func (c *Client) ArchiveWorkspace(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, workspacesPath+"/"+id+"/archive", nil, nil, nil)
}

func (c *Client) ListWorkspaces(ctx context.Context, includeArchived bool) ([]Workspace, error) {
	q := url.Values{}
	if includeArchived {
		q.Set("include_archived", "true")
	}
	return listAll[Workspace](ctx, c, workspacesPath, q)
}

// --- Workspace members ---

func (c *Client) AddWorkspaceMember(ctx context.Context, workspaceID, userID, role string) (*WorkspaceMember, error) {
	body := map[string]string{"user_id": userID, "workspace_role": role}
	var m WorkspaceMember
	if err := c.do(ctx, http.MethodPost, workspacesPath+"/"+workspaceID+"/members", nil, body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) GetWorkspaceMember(ctx context.Context, workspaceID, userID string) (*WorkspaceMember, error) {
	var m WorkspaceMember
	if err := c.do(ctx, http.MethodGet, workspacesPath+"/"+workspaceID+"/members/"+userID, nil, nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) UpdateWorkspaceMember(ctx context.Context, workspaceID, userID, role string) (*WorkspaceMember, error) {
	body := map[string]string{"workspace_role": role}
	var m WorkspaceMember
	if err := c.do(ctx, http.MethodPost, workspacesPath+"/"+workspaceID+"/members/"+userID, nil, body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) DeleteWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	return c.do(ctx, http.MethodDelete, workspacesPath+"/"+workspaceID+"/members/"+userID, nil, nil, nil)
}

func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	return listAll[WorkspaceMember](ctx, c, workspacesPath+"/"+workspaceID+"/members", nil)
}
