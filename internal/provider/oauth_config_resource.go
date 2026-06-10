package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m11s-io/terraform-provider-fider/internal/fider"
)

var _ resource.Resource = &OAuthConfigResource{}

type OAuthConfigResource struct {
	client *fider.Client
}

type OAuthConfigModel struct {
	Provider          types.String `tfsdk:"provider"`
	DisplayName       types.String `tfsdk:"display_name"`
	Status            types.Int64  `tfsdk:"status"`
	ClientID          types.String `tfsdk:"client_id"`
	ClientSecret      types.String `tfsdk:"client_secret"`
	AuthorizeURL      types.String `tfsdk:"authorize_url"`
	TokenURL          types.String `tfsdk:"token_url"`
	ProfileURL        types.String `tfsdk:"profile_url"`
	Scope             types.String `tfsdk:"scope"`
	IsTrusted         types.Bool   `tfsdk:"is_trusted"`
	JSONUserIDPath    types.String `tfsdk:"json_user_id_path"`
	JSONUserNamePath  types.String `tfsdk:"json_user_name_path"`
	JSONUserEmailPath types.String `tfsdk:"json_user_email_path"`
	JSONUserRolesPath types.String `tfsdk:"json_user_roles_path"`
	AllowedRoles      types.String `tfsdk:"allowed_roles"`
}

func NewOAuthConfigResource() resource.Resource {
	return &OAuthConfigResource{}
}

func (r *OAuthConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_config"
}

func (r *OAuthConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom OAuth2 provider configuration for a Fider tenant. " +
			"This resource requires an admin API key for the specific tenant.\n\n" +
			"> **Note:** Fider has no delete API for custom OAuth configs. " +
			"On `terraform destroy`, the config is disabled (status=1) rather than removed.",
		Attributes: map[string]schema.Attribute{
			"provider": schema.StringAttribute{
				MarkdownDescription: "Unique provider slug. Used as the identifier and in the OAuth callback URL " +
					"(`/oauth/<provider>/callback`). Changing this forces a new resource.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name shown on the login button (max 50 characters).",
				Required:            true,
			},
			"status": schema.Int64Attribute{
				MarkdownDescription: "Provider status. `1` = disabled, `2` = enabled. Defaults to `2`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client ID.",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client secret.",
				Required:            true,
				Sensitive:           true,
			},
			"authorize_url": schema.StringAttribute{
				MarkdownDescription: "OAuth2 authorization endpoint URL.",
				Required:            true,
			},
			"token_url": schema.StringAttribute{
				MarkdownDescription: "OAuth2 token endpoint URL.",
				Required:            true,
			},
			"profile_url": schema.StringAttribute{
				MarkdownDescription: "URL to fetch the user profile (userinfo endpoint). Optional.",
				Optional:            true,
				Computed:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Space-separated OAuth2 scopes to request (e.g. `openid profile email`).",
				Required:            true,
			},
			"is_trusted": schema.BoolAttribute{
				MarkdownDescription: "When `true`, users from this provider are automatically trusted. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"json_user_id_path": schema.StringAttribute{
				MarkdownDescription: "JSONPath expression to extract the user ID from the profile response (e.g. `sub`).",
				Required:            true,
			},
			"json_user_name_path": schema.StringAttribute{
				MarkdownDescription: "JSONPath expression to extract the user display name (e.g. `name`).",
				Optional:            true,
				Computed:            true,
			},
			"json_user_email_path": schema.StringAttribute{
				MarkdownDescription: "JSONPath expression to extract the user email (e.g. `email`).",
				Optional:            true,
				Computed:            true,
			},
			"json_user_roles_path": schema.StringAttribute{
				MarkdownDescription: "JSONPath expression to extract roles for role-based access control.",
				Optional:            true,
				Computed:            true,
			},
			"allowed_roles": schema.StringAttribute{
				MarkdownDescription: "Comma-separated list of roles allowed to sign in via this provider.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *OAuthConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*fider.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *fider.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *OAuthConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OAuthConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := modelToConfig(data)
	if err := r.client.SaveOAuthConfig(ctx, cfg); err != nil {
		resp.Diagnostics.AddError("Error creating OAuth config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OAuthConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OAuthConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetOAuthConfig(ctx, data.Provider.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading OAuth config", err.Error())
		return
	}
	if cfg == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.DisplayName = types.StringValue(cfg.DisplayName)
	data.Status = types.Int64Value(int64(cfg.Status))
	data.ClientID = types.StringValue(cfg.ClientID)
	// Fider masks the secret in GET responses — keep the value already in state
	// to avoid perpetual drift detection.
	if !isMasked(cfg.ClientSecret) {
		data.ClientSecret = types.StringValue(cfg.ClientSecret)
	}
	data.AuthorizeURL = types.StringValue(cfg.AuthorizeURL)
	data.TokenURL = types.StringValue(cfg.TokenURL)
	data.ProfileURL = types.StringValue(cfg.ProfileURL)
	data.Scope = types.StringValue(cfg.Scope)
	data.IsTrusted = types.BoolValue(cfg.IsTrusted)
	data.JSONUserIDPath = types.StringValue(cfg.JSONUserIDPath)
	data.JSONUserNamePath = types.StringValue(cfg.JSONUserNamePath)
	data.JSONUserEmailPath = types.StringValue(cfg.JSONUserEmailPath)
	data.JSONUserRolesPath = types.StringValue(cfg.JSONUserRolesPath)
	data.AllowedRoles = types.StringValue(cfg.AllowedRoles)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OAuthConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OAuthConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := modelToConfig(data)
	if err := r.client.SaveOAuthConfig(ctx, cfg); err != nil {
		resp.Diagnostics.AddError("Error updating OAuth config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OAuthConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OAuthConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DisableOAuthConfig(ctx, data.Provider.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error disabling OAuth config", err.Error())
	}
}

func modelToConfig(m OAuthConfigModel) fider.OAuthConfig {
	return fider.OAuthConfig{
		Provider:          m.Provider.ValueString(),
		DisplayName:       m.DisplayName.ValueString(),
		Status:            int(m.Status.ValueInt64()),
		ClientID:          m.ClientID.ValueString(),
		ClientSecret:      m.ClientSecret.ValueString(),
		AuthorizeURL:      m.AuthorizeURL.ValueString(),
		TokenURL:          m.TokenURL.ValueString(),
		ProfileURL:        m.ProfileURL.ValueString(),
		Scope:             m.Scope.ValueString(),
		IsTrusted:         m.IsTrusted.ValueBool(),
		JSONUserIDPath:    m.JSONUserIDPath.ValueString(),
		JSONUserNamePath:  m.JSONUserNamePath.ValueString(),
		JSONUserEmailPath: m.JSONUserEmailPath.ValueString(),
		JSONUserRolesPath: m.JSONUserRolesPath.ValueString(),
		AllowedRoles:      m.AllowedRoles.ValueString(),
	}
}

// isMasked returns true when Fider has returned a redacted secret (e.g. "abc...xyz").
func isMasked(secret string) bool {
	return strings.Contains(secret, "...")
}
