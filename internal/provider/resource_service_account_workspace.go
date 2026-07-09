package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var (
	_ resource.Resource                = (*serviceAccountWorkspaceResource)(nil)
	_ resource.ResourceWithImportState = (*serviceAccountWorkspaceResource)(nil)
)

func NewServiceAccountWorkspaceResource() resource.Resource {
	return &serviceAccountWorkspaceResource{}
}

type serviceAccountWorkspaceResource struct {
	client *client.Client
}

type serviceAccountWorkspaceModel struct {
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
}

func (r *serviceAccountWorkspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_workspace"
}

func (r *serviceAccountWorkspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Explicit membership of a service account in a workspace. Required before a " +
			"federation rule can target the service account in that workspace (every service account " +
			"is implicitly a member of the organization's default workspace only). Requires an " +
			"org:admin OAuth token. Import with \"service_account_id/workspace_id\".",
		Attributes: map[string]schema.Attribute{
			"service_account_id": schema.StringAttribute{
				Required:      true,
				Description:   "Service account ID (svac_...).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_id": schema.StringAttribute{
				Required:      true,
				Description:   "Workspace ID (wrkspc_...).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *serviceAccountWorkspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		r.client = c
	}
}

func (r *serviceAccountWorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceAccountWorkspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AddServiceAccountWorkspace(ctx, plan.ServiceAccountID.ValueString(), plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error adding service account to workspace", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *serviceAccountWorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceAccountWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, err := r.client.ListServiceAccountWorkspaces(ctx, state.ServiceAccountID.ValueString())
	if err != nil {
		if client.IsNotFound(err) { // service account gone (archived/removed)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error listing service account workspaces", err.Error())
		return
	}

	for _, id := range ids {
		if id == state.WorkspaceID.ValueString() {
			resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *serviceAccountWorkspaceResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both attributes force replacement.
	resp.Diagnostics.AddError("Service account workspace memberships cannot be updated",
		"Changes to service_account_id or workspace_id require replacement.")
}

func (r *serviceAccountWorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountWorkspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.RemoveServiceAccountWorkspace(ctx, state.ServiceAccountID.ValueString(), state.WorkspaceID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error removing service account from workspace", err.Error())
	}
}

func (r *serviceAccountWorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected \"service_account_id/workspace_id\", got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[1])...)
}
