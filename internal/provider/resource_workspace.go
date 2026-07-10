package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var (
	_ resource.Resource                = (*workspaceResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceResource)(nil)
)

func NewWorkspaceResource() resource.Resource { return &workspaceResource{} }

type workspaceResource struct {
	client *client.Client
}

type workspaceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Tags         types.Map    `tfsdk:"tags"`
	DisplayColor types.String `tfsdk:"display_color"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (r *workspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Claude platform workspace. Destroy archives the workspace (soft delete); " +
			"the Admin API has no hard delete.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Workspace ID (wrkspc_...).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable workspace name.",
			},
			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "User-defined tags. Keys may not begin with \"anthropic\".",
			},
			"display_color": schema.StringAttribute{
				Computed:    true,
				Description: "Hex color representing the workspace in the Console.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "RFC 3339 creation timestamp.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *workspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		r.client = c
	}
}

func (r *workspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, diags := tagsFromMap(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ws, err := r.client.CreateWorkspace(ctx, client.CreateWorkspaceRequest{
		Name: plan.Name.ValueString(),
		Tags: tags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating workspace", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceToModel(ctx, ws, plan.Tags))...)
}

func (r *workspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ws, err := r.client.GetWorkspace(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading workspace", err.Error())
		return
	}
	if ws.ArchivedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceToModel(ctx, ws, state.Tags))...)
}

func (r *workspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWorkspaceRequest{Name: plan.Name.ValueString()}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags, diags := tagsFromMap(ctx, plan.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Tags = &tags
	}

	ws, err := r.client.UpdateWorkspace(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating workspace", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceToModel(ctx, ws, plan.Tags))...)
}

func (r *workspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ArchiveWorkspace(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error archiving workspace", err.Error())
	}
}

func (r *workspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func workspaceToModel(ctx context.Context, ws *client.Workspace, priorTags types.Map) workspaceModel {
	m := workspaceModel{
		ID:           types.StringValue(ws.ID),
		Name:         types.StringValue(ws.Name),
		DisplayColor: types.StringValue(ws.DisplayColor),
		CreatedAt:    types.StringValue(ws.CreatedAt),
		Tags:         types.MapNull(types.StringType),
	}
	// Preserve null tags when the API returns an empty map so an omitted
	// attribute doesn't produce perpetual drift.
	if len(ws.Tags) > 0 || (!priorTags.IsNull() && !priorTags.IsUnknown()) {
		tagVals, _ := types.MapValueFrom(ctx, types.StringType, ws.Tags)
		m.Tags = tagVals
	}
	return m
}

func tagsFromMap(ctx context.Context, m types.Map) (map[string]string, diagList) {
	tags := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	diags := m.ElementsAs(ctx, &tags, false)
	return tags, diags
}
