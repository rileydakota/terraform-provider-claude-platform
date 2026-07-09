package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var (
	_ resource.Resource                   = (*federationRuleResource)(nil)
	_ resource.ResourceWithImportState    = (*federationRuleResource)(nil)
	_ resource.ResourceWithValidateConfig = (*federationRuleResource)(nil)
)

func NewFederationRuleResource() resource.Resource { return &federationRuleResource{} }

type federationRuleResource struct {
	client *client.Client
}

type federationRuleModel struct {
	ID                     types.String    `tfsdk:"id"`
	Name                   types.String    `tfsdk:"name"`
	IssuerID               types.String    `tfsdk:"issuer_id"`
	ServiceAccountID       types.String    `tfsdk:"service_account_id"`
	WorkspaceID            types.String    `tfsdk:"workspace_id"`
	AppliesToAllWorkspaces types.Bool      `tfsdk:"applies_to_all_workspaces"`
	OAuthScope             types.String    `tfsdk:"oauth_scope"`
	TokenLifetimeSeconds   types.Int64     `tfsdk:"token_lifetime_seconds"`
	Match                  *ruleMatchModel `tfsdk:"match"`
	CreatedAt              types.String    `tfsdk:"created_at"`
}

type ruleMatchModel struct {
	SubjectPrefix types.String `tfsdk:"subject_prefix"`
	Audience      types.String `tfsdk:"audience"`
	Claims        types.Map    `tfsdk:"claims"`
	Condition     types.String `tfsdk:"condition"`
}

func (r *federationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_rule"
}

func (r *federationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Workload Identity Federation rule (fdrl_...): binds an issuer to a service " +
			"account so matching JWTs can mint short-lived Anthropic tokens. Requires an org:admin OAuth " +
			"token. OAuth callers can only manage rules scoped workspace:developer or workspace:inference; " +
			"org:admin and workspace:manage_tunnels rules are Console-only by design. Destroy archives the rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name, lowercase alphanumeric plus hyphens, unique per organization.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(wifNameRegexp, "must match ^[a-z0-9-]+$ (1-255 chars)"),
				},
			},
			"issuer_id": schema.StringAttribute{
				Required:      true,
				Description:   "Federation issuer ID (fdis_...).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_account_id": schema.StringAttribute{
				Required:      true,
				Description:   "Target service account ID (svac_...). Must be a member of the rule's workspace.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workspace_id": schema.StringAttribute{
				Optional: true,
				Description: "Workspace the rule is enabled in at creation. Exactly one of workspace_id " +
					"or applies_to_all_workspaces is required.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"applies_to_all_workspaces": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable the rule in every workspace in the organization.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"oauth_scope": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("workspace:developer"),
				Description: "Scope granted to minted tokens. API-manageable values: workspace:developer, " +
					"workspace:inference. (org:admin / workspace:manage_tunnels rules must be created in the Console.)",
				Validators: []validator.String{
					stringvalidator.OneOf("workspace:developer", "workspace:inference"),
				},
			},
			"token_lifetime_seconds": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Lifetime of minted tokens, 60-86400 seconds. Server default 3600.",
				Validators:    []validator.Int64{int64validator.Between(60, 86400)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"match": schema.SingleNestedBlock{
				Description: "Conditions an incoming JWT must satisfy (AND semantics). At least one of " +
					"subject_prefix, claims, or condition must be set; audience alone is rejected.",
				Attributes: map[string]schema.Attribute{
					"subject_prefix": schema.StringAttribute{
						Optional: true,
						Description: "Exact match against the JWT sub claim; a trailing * makes it a prefix " +
							"match. Case-sensitive. Pin CI rules to a protected ref — a broad wildcard like " +
							"repo:org/repo:* also matches fork-triggered pull_request runs.",
					},
					"audience": schema.StringAttribute{
						Optional:    true,
						Description: "The JWT aud claim must contain this exact string.",
					},
					"claims": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Top-level claim name to required exact string value. For nested/typed claims use condition.",
					},
					"condition": schema.StringAttribute{
						Optional:    true,
						Description: "CEL expression over the decoded claim set (variable: claims). This is a security boundary — prefer static matchers when they express the constraint.",
					},
				},
			},
		},
	}
}

func (r *federationRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config federationRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown values (references to other resources) can't be validated until
	// apply — only enforce the xor when both sides are known.
	if !config.WorkspaceID.IsUnknown() && !config.AppliesToAllWorkspaces.IsUnknown() {
		hasWorkspace := !config.WorkspaceID.IsNull()
		allWorkspaces := config.AppliesToAllWorkspaces.ValueBool()
		if hasWorkspace && allWorkspaces {
			resp.Diagnostics.AddAttributeError(path.Root("workspace_id"),
				"Conflicting workspace configuration",
				"Set either workspace_id or applies_to_all_workspaces = true, not both.")
		}
		if !hasWorkspace && !allWorkspaces {
			resp.Diagnostics.AddAttributeError(path.Root("workspace_id"),
				"Missing workspace configuration",
				"One of workspace_id or applies_to_all_workspaces = true is required.")
		}
	}

	if config.Match != nil {
		m := config.Match
		hasAnchor := (!m.SubjectPrefix.IsNull() && !m.SubjectPrefix.IsUnknown()) ||
			(!m.Claims.IsNull() && !m.Claims.IsUnknown()) ||
			(!m.Condition.IsNull() && !m.Condition.IsUnknown())
		if !hasAnchor {
			resp.Diagnostics.AddAttributeError(path.Root("match"),
				"Insufficient match conditions",
				"At least one of subject_prefix, claims, or condition must be set; a match block with "+
					"only audience (or nothing) would accept every token from the issuer and is rejected by the API.")
		}
	}
}

func (r *federationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		r.client = c
	}
}

func (m *federationRuleModel) matchToAPI(ctx context.Context) (*client.RuleMatch, diagList) {
	if m.Match == nil {
		return nil, nil
	}
	claims, diags := tagsFromMap(ctx, m.Match.Claims)
	if diags.HasError() {
		return nil, diags
	}
	return &client.RuleMatch{
		SubjectPrefix: m.Match.SubjectPrefix.ValueString(),
		Audience:      m.Match.Audience.ValueString(),
		Claims:        claims,
		Condition:     m.Match.Condition.ValueString(),
	}, diags
}

func (r *federationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan federationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	match, diags := plan.matchToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.CreateFederationRuleRequest{
		Name:     plan.Name.ValueString(),
		IssuerID: plan.IssuerID.ValueString(),
		Match:    match,
		Target: &client.RuleTarget{
			Type:             "service_account",
			ServiceAccountID: plan.ServiceAccountID.ValueString(),
		},
		WorkspaceID:            plan.WorkspaceID.ValueString(),
		AppliesToAllWorkspaces: plan.AppliesToAllWorkspaces.ValueBool(),
		OAuthScope:             plan.OAuthScope.ValueString(),
	}
	if !plan.TokenLifetimeSeconds.IsNull() && !plan.TokenLifetimeSeconds.IsUnknown() {
		apiReq.TokenLifetimeSeconds = plan.TokenLifetimeSeconds.ValueInt64()
	}

	rule, err := r.client.CreateFederationRule(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating federation rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ruleToModel(ctx, rule, &plan))...)
}

func (r *federationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state federationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetFederationRule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading federation rule", err.Error())
		return
	}
	if rule.ArchivedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ruleToModel(ctx, rule, &state))...)
}

func (r *federationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan federationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	match, diags := plan.matchToAPI(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.UpdateFederationRuleRequest{
		Name:       plan.Name.ValueString(),
		Match:      match,
		OAuthScope: plan.OAuthScope.ValueString(),
	}
	if !plan.TokenLifetimeSeconds.IsNull() && !plan.TokenLifetimeSeconds.IsUnknown() {
		apiReq.TokenLifetimeSeconds = plan.TokenLifetimeSeconds.ValueInt64()
	}

	rule, err := r.client.UpdateFederationRule(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating federation rule", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ruleToModel(ctx, rule, &plan))...)
}

func (r *federationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state federationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ArchiveFederationRule(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error archiving federation rule", err.Error())
	}
}

func (r *federationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func ruleToModel(ctx context.Context, rule *client.FederationRule, prior *federationRuleModel) federationRuleModel {
	m := federationRuleModel{
		ID:                     types.StringValue(rule.ID),
		Name:                   types.StringValue(rule.Name),
		IssuerID:               types.StringValue(rule.IssuerID),
		AppliesToAllWorkspaces: types.BoolValue(rule.AppliesToAllWorkspaces),
		OAuthScope:             types.StringValue(rule.OAuthScope),
		TokenLifetimeSeconds:   types.Int64Value(rule.TokenLifetimeSeconds),
		CreatedAt:              types.StringValue(rule.CreatedAt),
		WorkspaceID:            types.StringNull(),
	}
	if rule.WorkspaceID != "" {
		m.WorkspaceID = types.StringValue(rule.WorkspaceID)
	}
	if rule.Target != nil {
		m.ServiceAccountID = types.StringValue(rule.Target.ServiceAccountID)
	} else if prior != nil {
		m.ServiceAccountID = prior.ServiceAccountID
	}

	if rule.Match != nil {
		mm := &ruleMatchModel{
			SubjectPrefix: types.StringNull(),
			Audience:      types.StringNull(),
			Claims:        types.MapNull(types.StringType),
			Condition:     types.StringNull(),
		}
		if rule.Match.SubjectPrefix != "" {
			mm.SubjectPrefix = types.StringValue(rule.Match.SubjectPrefix)
		}
		if rule.Match.Audience != "" {
			mm.Audience = types.StringValue(rule.Match.Audience)
		}
		if rule.Match.Condition != "" {
			mm.Condition = types.StringValue(rule.Match.Condition)
		}
		if len(rule.Match.Claims) > 0 {
			claims, _ := types.MapValueFrom(ctx, types.StringType, rule.Match.Claims)
			mm.Claims = claims
		}
		m.Match = mm
	} else if prior != nil {
		m.Match = prior.Match
	}
	return m
}
