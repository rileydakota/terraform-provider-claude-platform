package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rileydakota/terraform-provider-claude-platform/internal/client"
)

var _ datasource.DataSource = (*userDataSource)(nil)

func NewUserDataSource() datasource.DataSource { return &userDataSource{} }

type userDataSource struct {
	client *client.Client
}

type userDataModel struct {
	ID      types.String `tfsdk:"id"`
	Email   types.String `tfsdk:"email"`
	Name    types.String `tfsdk:"name"`
	Role    types.String `tfsdk:"role"`
	AddedAt types.String `tfsdk:"added_at"`
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an organization member by email. Useful for claudeplatform_workspace_member.",
		Attributes: map[string]schema.Attribute{
			"email":    schema.StringAttribute{Required: true, Description: "Member email address (exact match)."},
			"id":       schema.StringAttribute{Computed: true, Description: "User ID (user_...)."},
			"name":     schema.StringAttribute{Computed: true},
			"role":     schema.StringAttribute{Computed: true, Description: "Organization role."},
			"added_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Provider configuration error", err.Error())
		return
	}
	if c != nil {
		d.client = c
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.client.ListUsers(ctx, config.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing users", err.Error())
		return
	}
	if len(users) == 0 {
		resp.Diagnostics.AddError("User not found",
			fmt.Sprintf("No organization member with email %q.", config.Email.ValueString()))
		return
	}
	u := users[0]

	resp.Diagnostics.Append(resp.State.Set(ctx, userDataModel{
		ID:      types.StringValue(u.ID),
		Email:   types.StringValue(u.Email),
		Name:    types.StringValue(u.Name),
		Role:    types.StringValue(u.Role),
		AddedAt: types.StringValue(u.AddedAt),
	})...)
}
