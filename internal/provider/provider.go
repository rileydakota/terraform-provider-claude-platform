package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var _ provider.Provider = (*claudePlatformProvider)(nil)

type claudePlatformProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &claudePlatformProvider{version: version}
	}
}

type providerModel struct {
	AdminAPIKey types.String     `tfsdk:"admin_api_key"`
	OAuthToken  types.String     `tfsdk:"oauth_token"`
	BaseURL     types.String     `tfsdk:"base_url"`
	Federation  *federationModel `tfsdk:"federation"`
}

type federationModel struct {
	FederationRuleID  types.String `tfsdk:"federation_rule_id"`
	OrganizationID    types.String `tfsdk:"organization_id"`
	ServiceAccountID  types.String `tfsdk:"service_account_id"`
	WorkspaceID       types.String `tfsdk:"workspace_id"`
	IdentityToken     types.String `tfsdk:"identity_token"`
	IdentityTokenFile types.String `tfsdk:"identity_token_file"`
}

func (p *claudePlatformProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "claudeplatform"
	resp.Version = p.version
}

func (p *claudePlatformProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Anthropic Claude platform organization resources via the Admin API: " +
			"workspaces, members, invites, API keys, service accounts, and Workload Identity Federation.",
		Attributes: map[string]schema.Attribute{
			"admin_api_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Admin API key (sk-ant-admin...). Falls back to ANTHROPIC_ADMIN_KEY. " +
					"Note: the WIF endpoints (service accounts, federation issuers/rules) reject admin " +
					"API keys and require an org:admin OAuth token.",
			},
			"oauth_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "OAuth bearer token with the org:admin scope. Falls back to " +
					"ANTHROPIC_OAUTH_TOKEN. Grants the full Admin API surface including WIF endpoints.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "API base URL. Falls back to ANTHROPIC_BASE_URL, then https://api.anthropic.com.",
			},
		},
		Blocks: map[string]schema.Block{
			"federation": schema.SingleNestedBlock{
				Description: "Mint the org:admin bearer token at plan time via Workload Identity " +
					"Federation (POST /v1/oauth/token) instead of supplying a static credential. " +
					"Requires a Console-created bootstrap rule with oauth_scope org:admin.",
				Attributes: map[string]schema.Attribute{
					"federation_rule_id": schema.StringAttribute{
						Optional:    true,
						Description: "Federation rule ID (fdrl_...).",
					},
					"organization_id": schema.StringAttribute{
						Optional:    true,
						Description: "Anthropic organization UUID.",
					},
					"service_account_id": schema.StringAttribute{
						Optional:    true,
						Description: "Target service account ID (svac_...).",
					},
					"workspace_id": schema.StringAttribute{
						Optional:    true,
						Description: "Workspace to scope the token to (wrkspc_... or \"default\"). Required only when the rule spans multiple workspaces.",
					},
					"identity_token": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "The OIDC JWT from your identity provider, as a literal string.",
					},
					"identity_token_file": schema.StringAttribute{
						Optional:    true,
						Description: "Filesystem path to the OIDC JWT (e.g. a projected service account token).",
					},
				},
			},
		},
	}
}

func (p *claudePlatformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := firstNonEmpty(config.BaseURL.ValueString(), os.Getenv("ANTHROPIC_BASE_URL"), client.DefaultBaseURL)
	adminKey := firstNonEmpty(config.AdminAPIKey.ValueString(), os.Getenv("ANTHROPIC_ADMIN_KEY"))
	oauthToken := firstNonEmpty(config.OAuthToken.ValueString(), os.Getenv("ANTHROPIC_OAUTH_TOKEN"))

	// Federation block: exchange an OIDC JWT for a short-lived bearer token.
	if config.Federation != nil {
		fed := config.Federation
		assertion := fed.IdentityToken.ValueString()
		if assertion == "" && fed.IdentityTokenFile.ValueString() != "" {
			raw, err := os.ReadFile(fed.IdentityTokenFile.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("federation").AtName("identity_token_file"),
					"Unable to read identity token file", err.Error())
				return
			}
			assertion = strings.TrimSpace(string(raw))
		}
		if assertion == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("federation"),
				"Missing identity token",
				"One of federation.identity_token or federation.identity_token_file must be set.")
			return
		}
		tok, err := client.ExchangeToken(ctx, nil, baseURL, client.TokenExchangeInput{
			Assertion:        assertion,
			FederationRuleID: fed.FederationRuleID.ValueString(),
			OrganizationID:   fed.OrganizationID.ValueString(),
			ServiceAccountID: fed.ServiceAccountID.ValueString(),
			WorkspaceID:      fed.WorkspaceID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Workload Identity Federation token exchange failed", err.Error())
			return
		}
		oauthToken = tok.AccessToken
	}

	if adminKey == "" && oauthToken == "" {
		resp.Diagnostics.AddError(
			"Missing credentials",
			"Configure one of: oauth_token (or ANTHROPIC_OAUTH_TOKEN), admin_api_key (or "+
				"ANTHROPIC_ADMIN_KEY), or a federation block. An org:admin OAuth token is required "+
				"for service accounts and federation issuers/rules.")
		return
	}

	c, err := client.New(client.Options{
		BaseURL:     baseURL,
		AdminAPIKey: adminKey,
		OAuthToken:  oauthToken,
		UserAgent:   fmt.Sprintf("terraform-provider-claude-platform/%s", p.version),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create API client", err.Error())
		return
	}

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *claudePlatformProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewWorkspaceResource,
		NewWorkspaceMemberResource,
		NewOrganizationInviteResource,
		NewServiceAccountResource,
		NewFederationIssuerResource,
		NewFederationRuleResource,
	}
}

func (p *claudePlatformProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewWorkspaceDataSource,
		NewWorkspacesDataSource,
		NewUserDataSource,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// clientFromProviderData extracts the configured client in resource/data
// source Configure methods.
func clientFromProviderData(data any) (*client.Client, error) {
	if data == nil {
		return nil, nil // Configure not called yet; framework calls again later
	}
	c, ok := data.(*client.Client)
	if !ok {
		return nil, fmt.Errorf("unexpected provider data type %T", data)
	}
	return c, nil
}
