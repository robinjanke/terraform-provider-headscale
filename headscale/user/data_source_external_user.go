package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/awlsring/terraform-provider-headscale/internal/service"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &externalUserDataSource{}
	_ datasource.DataSourceWithConfigure = &externalUserDataSource{}
)

func DataSourceExternal() datasource.DataSource {
	return &externalUserDataSource{}
}

type externalUserDataSource struct {
	client service.Headscale
}

type externalUserModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ProviderID        types.String `tfsdk:"provider_id"`
	DisplayName       types.String `tfsdk:"display_name"`
	Email             types.String `tfsdk:"email"`
	ProfilePictureURL types.String `tfsdk:"profile_picture_url"`
	Provider          types.String `tfsdk:"provider"`
	CreatedAt         types.String `tfsdk:"created_at"`
	CreateIfNotExists types.Bool   `tfsdk:"create_if_not_exists"`
}

func (d *externalUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_user"
}

func (d *externalUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(service.Headscale)
}

func (d *externalUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an OIDC (external) Headscale user. Optionally creates the user when missing.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Headscale username (preferred_username from OIDC).",
			},
			"provider_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OIDC provider identifier (iss+sub). Required when create_if_not_exists is true.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Display name used when creating the user.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Email used when creating the user.",
			},
			"profile_picture_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Profile picture URL used when creating the user.",
			},
			"create_if_not_exists": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, create an OIDC user when none matches. Defaults to false.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Headscale user id.",
			},
			"provider": schema.StringAttribute{
				Computed:    true,
				Description: "Auth provider of the user (expected: oidc).",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation timestamp of the user.",
			},
		},
	}
}

func (d *externalUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config externalUserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(config.Name.ValueString())
	providerID := strings.TrimSpace(config.ProviderID.ValueString())
	createIfNotExists := false
	if !config.CreateIfNotExists.IsNull() && !config.CreateIfNotExists.IsUnknown() {
		createIfNotExists = config.CreateIfNotExists.ValueBool()
	}

	if name == "" && providerID == "" {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Set name and/or provider_id to look up an external Headscale user.",
		)
		return
	}
	if createIfNotExists && providerID == "" {
		resp.Diagnostics.AddError(
			"provider_id required",
			"create_if_not_exists=true requires provider_id so the OIDC user can be created with the correct iss+sub identifier.",
		)
		return
	}
	if createIfNotExists && name == "" {
		resp.Diagnostics.AddError(
			"name required",
			"create_if_not_exists=true requires name for the OIDC user.",
		)
		return
	}

	user, cliConflict, err := d.client.FindExternalUser(ctx, name, providerID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to look up external user", err.Error())
		return
	}
	if cliConflict && user == nil {
		resp.Diagnostics.AddError(
			"CLI user name conflict",
			fmt.Sprintf("A non-OIDC Headscale user already exists with name %q; refusing to create or use it as an external user.", name),
		)
		return
	}

	if user == nil {
		if !createIfNotExists {
			resp.Diagnostics.AddError(
				"External user not found",
				"No OIDC user matched the given name/provider_id. Set create_if_not_exists=true to create one.",
			)
			return
		}

		created, createErr := d.client.CreateUser(ctx, service.CreateUserInput{
			Name:        name,
			ProviderID:  providerID,
			Email:       strings.TrimSpace(config.Email.ValueString()),
			DisplayName: strings.TrimSpace(config.DisplayName.ValueString()),
			PictureURL:  strings.TrimSpace(config.ProfilePictureURL.ValueString()),
		})
		if createErr != nil {
			resp.Diagnostics.AddError("Unable to create external user", createErr.Error())
			return
		}
		user = created
	}

	state := externalUserModel{
		ID:                types.StringValue(user.ID),
		Name:              types.StringValue(user.Name),
		ProviderID:        types.StringValue(user.ProviderID),
		Provider:          types.StringValue(user.Provider),
		CreatedAt:         types.StringValue(user.CreatedAt.DeepCopy().String()),
		CreateIfNotExists: types.BoolValue(createIfNotExists),
	}
	if user.DisplayName != "" {
		state.DisplayName = types.StringValue(user.DisplayName)
	} else if !config.DisplayName.IsNull() {
		state.DisplayName = config.DisplayName
	} else {
		state.DisplayName = types.StringNull()
	}
	if user.Email != "" {
		state.Email = types.StringValue(user.Email)
	} else if !config.Email.IsNull() {
		state.Email = config.Email
	} else {
		state.Email = types.StringNull()
	}
	if user.ProfilePicURL != "" {
		state.ProfilePictureURL = types.StringValue(user.ProfilePicURL)
	} else if !config.ProfilePictureURL.IsNull() {
		state.ProfilePictureURL = config.ProfilePictureURL
	} else {
		state.ProfilePictureURL = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
