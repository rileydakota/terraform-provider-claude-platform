package provider

import (
	"context"
	"encoding/json"

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
	_ resource.Resource                = (*federationIssuerResource)(nil)
	_ resource.ResourceWithImportState = (*federationIssuerResource)(nil)
)

func NewFederationIssuerResource() resource.Resource { return &federationIssuerResource{} }

type federationIssuerResource struct {
	client *client.Client
}

type federationIssuerModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	IssuerURL types.String `tfsdk:"issuer_url"`
	JWKS      *jwksModel   `tfsdk:"jwks"`
	CACertPEM types.String `tfsdk:"ca_cert_pem"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type jwksModel struct {
	Type          types.String `tfsdk:"type"`
	DiscoveryBase types.String `tfsdk:"discovery_base"`
	URL           types.String `tfsdk:"url"`
	KeysJSON      types.String `tfsdk:"keys_json"`
}

func (r *federationIssuerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_issuer"
}

func (r *federationIssuerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Workload Identity Federation issuer (fdis_...): registers an OIDC identity " +
			"provider with the organization. Requires an org:admin OAuth token. Destroy archives the issuer; " +
			"archiving fails while a live federation rule references it. Note: an OAuth caller cannot update " +
			"an issuer that backs a rule scoped to anything other than workspace:developer/workspace:inference — " +
			"keep bootstrap (org:admin) rules on a dedicated, Console-managed issuer.",
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
			"issuer_url": schema.StringAttribute{
				Required: true,
				Description: "The OIDC issuer URL, compared byte-for-byte against the JWT iss claim. " +
					"Must be https on port 443 when Anthropic dials it (discovery mode without discovery_base).",
			},
			"ca_cert_pem": schema.StringAttribute{
				Optional:    true,
				Description: "PEM CA certificate for issuers serving TLS from a private CA (discovery/explicit_url modes).",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"jwks": schema.SingleNestedBlock{
				Description: "How Anthropic obtains the issuer's signing keys.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "discovery | explicit_url | inline.",
						Validators: []validator.String{
							stringvalidator.OneOf("discovery", "explicit_url", "inline"),
						},
					},
					"discovery_base": schema.StringAttribute{
						Optional:    true,
						Description: "discovery mode only: base URL for /.well-known/openid-configuration when it differs from issuer_url.",
					},
					"url": schema.StringAttribute{
						Optional:    true,
						Description: "explicit_url mode: the JWKS endpoint URL.",
					},
					"keys_json": schema.StringAttribute{
						Optional: true,
						Description: "inline mode: JSON array of JWK objects (the \"keys\" array of a JWKS document). " +
							"No automatic key refresh in inline mode — key rotation requires updating this attribute.",
					},
				},
			},
		},
	}
}

func (r *federationIssuerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		r.client = c
	}
}

func (m *federationIssuerModel) toRequest() (client.FederationIssuerRequest, error) {
	req := client.FederationIssuerRequest{
		Name:      m.Name.ValueString(),
		IssuerURL: m.IssuerURL.ValueString(),
		CACertPEM: m.CACertPEM.ValueString(),
	}
	if m.JWKS != nil {
		jwks := &client.JWKS{
			Type:          m.JWKS.Type.ValueString(),
			DiscoveryBase: m.JWKS.DiscoveryBase.ValueString(),
			URL:           m.JWKS.URL.ValueString(),
		}
		if keysJSON := m.JWKS.KeysJSON.ValueString(); keysJSON != "" {
			if err := json.Unmarshal([]byte(keysJSON), &jwks.Keys); err != nil {
				return req, err
			}
		}
		req.JWKS = jwks
	}
	return req, nil
}

func (r *federationIssuerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan federationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, err := plan.toRequest()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("jwks").AtName("keys_json"),
			"Invalid JWKS keys JSON", err.Error())
		return
	}

	iss, err := r.client.CreateFederationIssuer(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating federation issuer", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, issuerToModel(iss, &plan))...)
}

func (r *federationIssuerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state federationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	iss, err := r.client.GetFederationIssuer(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading federation issuer", err.Error())
		return
	}
	if iss.ArchivedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, issuerToModel(iss, &state))...)
}

func (r *federationIssuerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan federationIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, err := plan.toRequest()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("jwks").AtName("keys_json"),
			"Invalid JWKS keys JSON", err.Error())
		return
	}

	iss, err := r.client.UpdateFederationIssuer(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating federation issuer", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, issuerToModel(iss, &plan))...)
}

func (r *federationIssuerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state federationIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ArchiveFederationIssuer(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error archiving federation issuer",
			err.Error()+"\n\nNote: archiving fails with 400 while a live federation rule references this issuer.")
	}
}

func (r *federationIssuerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// issuerToModel maps the API object onto state. prior carries the
// previously-configured model so semantically-equal JSON (keys_json) and
// unset-vs-empty strings don't produce spurious drift.
func issuerToModel(iss *client.FederationIssuer, prior *federationIssuerModel) federationIssuerModel {
	m := federationIssuerModel{
		ID:        types.StringValue(iss.ID),
		Name:      types.StringValue(iss.Name),
		IssuerURL: types.StringValue(iss.IssuerURL),
		CreatedAt: types.StringValue(iss.CreatedAt),
		CACertPEM: types.StringNull(),
	}
	if iss.CACertPEM != "" {
		m.CACertPEM = types.StringValue(iss.CACertPEM)
	} else if prior != nil && !prior.CACertPEM.IsNull() && prior.CACertPEM.ValueString() == "" {
		m.CACertPEM = prior.CACertPEM
	}

	if iss.JWKS != nil {
		jm := &jwksModel{
			Type:          types.StringValue(iss.JWKS.Type),
			DiscoveryBase: types.StringNull(),
			URL:           types.StringNull(),
			KeysJSON:      types.StringNull(),
		}
		if iss.JWKS.DiscoveryBase != "" {
			jm.DiscoveryBase = types.StringValue(iss.JWKS.DiscoveryBase)
		}
		if iss.JWKS.URL != "" {
			jm.URL = types.StringValue(iss.JWKS.URL)
		}
		if len(iss.JWKS.Keys) > 0 {
			raw, _ := json.Marshal(iss.JWKS.Keys)
			apiKeys := string(raw)
			// Keep the user's formatting when semantically identical.
			if prior != nil && prior.JWKS != nil && !prior.JWKS.KeysJSON.IsNull() &&
				jsonSemanticallyEqual(prior.JWKS.KeysJSON.ValueString(), apiKeys) {
				jm.KeysJSON = prior.JWKS.KeysJSON
			} else {
				jm.KeysJSON = types.StringValue(apiKeys)
			}
		}
		m.JWKS = jm
	} else if prior != nil {
		m.JWKS = prior.JWKS
	}
	return m
}
