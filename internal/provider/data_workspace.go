package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

// --- claudeplatform_workspace (single) ---

var _ datasource.DataSource = (*workspaceDataSource)(nil)

func NewWorkspaceDataSource() datasource.DataSource { return &workspaceDataSource{} }

type workspaceDataSource struct {
	client *client.Client
}

type workspaceDataModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DisplayColor types.String `tfsdk:"display_color"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *workspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (d *workspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a workspace by id or by name (exactly one).",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Optional: true, Computed: true, Description: "Workspace ID (wrkspc_...)."},
			"name":          schema.StringAttribute{Optional: true, Computed: true, Description: "Workspace name (exact match)."},
			"display_color": schema.StringAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *workspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		d.client = c
	}
}

func (d *workspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config workspaceDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull()
	hasName := !config.Name.IsNull()
	if hasID == hasName {
		resp.Diagnostics.AddError("Invalid lookup", "Exactly one of id or name must be set.")
		return
	}

	var ws *client.Workspace
	if hasID {
		found, err := d.client.GetWorkspace(ctx, config.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading workspace", err.Error())
			return
		}
		ws = found
	} else {
		all, err := d.client.ListWorkspaces(ctx, false)
		if err != nil {
			resp.Diagnostics.AddError("Error listing workspaces", err.Error())
			return
		}
		for i := range all {
			if all[i].Name == config.Name.ValueString() {
				ws = &all[i]
				break
			}
		}
		if ws == nil {
			resp.Diagnostics.AddError("Workspace not found",
				fmt.Sprintf("No active workspace named %q.", config.Name.ValueString()))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceDataModel{
		ID:           types.StringValue(ws.ID),
		Name:         types.StringValue(ws.Name),
		DisplayColor: types.StringValue(ws.DisplayColor),
		CreatedAt:    types.StringValue(ws.CreatedAt),
	})...)
}

// --- claudeplatform_workspaces (list) ---

var _ datasource.DataSource = (*workspacesDataSource)(nil)

func NewWorkspacesDataSource() datasource.DataSource { return &workspacesDataSource{} }

type workspacesDataSource struct {
	client *client.Client
}

type workspacesDataModel struct {
	IncludeArchived types.Bool           `tfsdk:"include_archived"`
	Workspaces      []workspaceDataModel `tfsdk:"workspaces"`
}

func (d *workspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspaces"
}

func (d *workspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All workspaces in the organization.",
		Attributes: map[string]schema.Attribute{
			"include_archived": schema.BoolAttribute{Optional: true, Description: "Include archived workspaces."},
			"workspaces": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true},
						"name":          schema.StringAttribute{Computed: true},
						"display_color": schema.StringAttribute{Computed: true},
						"created_at":    schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *workspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		d.client = c
	}
}

func (d *workspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config workspacesDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := d.client.ListWorkspaces(ctx, config.IncludeArchived.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error listing workspaces", err.Error())
		return
	}

	config.Workspaces = make([]workspaceDataModel, 0, len(all))
	for _, ws := range all {
		config.Workspaces = append(config.Workspaces, workspaceDataModel{
			ID:           types.StringValue(ws.ID),
			Name:         types.StringValue(ws.Name),
			DisplayColor: types.StringValue(ws.DisplayColor),
			CreatedAt:    types.StringValue(ws.CreatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
