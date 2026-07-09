package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var (
	_ resource.Resource                = (*organizationInviteResource)(nil)
	_ resource.ResourceWithImportState = (*organizationInviteResource)(nil)
)

func NewOrganizationInviteResource() resource.Resource { return &organizationInviteResource{} }

type organizationInviteResource struct {
	client *client.Client
}

type organizationInviteModel struct {
	ID        types.String `tfsdk:"id"`
	Email     types.String `tfsdk:"email"`
	Role      types.String `tfsdk:"role"`
	Status    types.String `tfsdk:"status"`
	InvitedAt types.String `tfsdk:"invited_at"`
	ExpiresAt types.String `tfsdk:"expires_at"`
}

func (r *organizationInviteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_invite"
}

func (r *organizationInviteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An invitation to join the organization. Invites expire server-side after 21 days; " +
			"an expired or accepted invite is not recreated automatically (status is tracked as an attribute).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email": schema.StringAttribute{
				Required:      true,
				Description:   "Email address to invite.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				Required:      true,
				Description:   "Organization role granted on acceptance.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("user", "claude_code_user", "developer", "billing", "admin"),
				},
			},
			"status":     schema.StringAttribute{Computed: true, Description: "Invite status (e.g. pending, accepted, expired)."},
			"invited_at": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"expires_at": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *organizationInviteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		r.client = c
	}
}

func (r *organizationInviteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationInviteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inv, err := r.client.CreateInvite(ctx, plan.Email.ValueString(), plan.Role.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating invite", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, inviteToModel(inv))...)
}

func (r *organizationInviteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationInviteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inv, err := r.client.GetInvite(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading invite", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, inviteToModel(inv))...)
}

func (r *organizationInviteResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All configurable attributes force replacement.
	resp.Diagnostics.AddError("Invites cannot be updated", "email and role changes require replacement.")
}

func (r *organizationInviteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationInviteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInvite(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting invite", err.Error())
	}
}

func (r *organizationInviteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func inviteToModel(inv *client.Invite) organizationInviteModel {
	return organizationInviteModel{
		ID:        types.StringValue(inv.ID),
		Email:     types.StringValue(inv.Email),
		Role:      types.StringValue(inv.Role),
		Status:    types.StringValue(inv.Status),
		InvitedAt: types.StringValue(inv.InvitedAt),
		ExpiresAt: types.StringValue(inv.ExpiresAt),
	}
}
