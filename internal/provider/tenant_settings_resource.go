package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m11s-io/terraform-provider-fider/internal/fider"
)

var _ resource.Resource = &TenantSettingsResource{}

type TenantSettingsResource struct {
	client *fider.Client
}

type TenantSettingsModel struct {
	Title          types.String `tfsdk:"title"`
	Invitation     types.String `tfsdk:"invitation"`
	WelcomeMessage types.String `tfsdk:"welcome_message"`
	WelcomeHeader  types.String `tfsdk:"welcome_header"`
	CNAME          types.String `tfsdk:"cname"`
	Locale         types.String `tfsdk:"locale"`
}

func NewTenantSettingsResource() resource.Resource {
	return &TenantSettingsResource{}
}

func (r *TenantSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_settings"
}

func (r *TenantSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages general settings for a Fider tenant: board name, custom domain (CNAME), invitation text, and locale.\n\n" +
			"> **Note:** Fider has no GET endpoint for general settings. Terraform state is the source of truth. " +
			"Out-of-band changes to these settings will not be detected until the next `terraform apply`.",
		Attributes: map[string]schema.Attribute{
			"title": schema.StringAttribute{
				MarkdownDescription: "Public name of the feedback board (max 60 characters). Shown in the page title and emails.",
				Required:            true,
			},
			"cname": schema.StringAttribute{
				MarkdownDescription: "Custom domain for this tenant (e.g. `feedback.mysite.com`). " +
					"The domain must resolve to your Fider instance. Leave empty to use the default subdomain.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"invitation": schema.StringAttribute{
				MarkdownDescription: "Short tagline shown on the feedback board (max 60 characters).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"welcome_message": schema.StringAttribute{
				MarkdownDescription: "Markdown welcome message shown on the home page.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"welcome_header": schema.StringAttribute{
				MarkdownDescription: "Short welcome header shown above the post list (max 100 characters).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"locale": schema.StringAttribute{
				MarkdownDescription: "Language locale for the board UI (e.g. `en`, `pt-BR`, `de`). Defaults to `en`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("en"),
			},
		},
	}
}

func (r *TenantSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TenantSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TenantSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateTenantSettings(ctx, modelToSettings(data)); err != nil {
		resp.Diagnostics.AddError("Error creating tenant settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read is a no-op: Fider has no GET endpoint for general settings.
// State written on Create/Update is kept as-is.
func (r *TenantSettingsResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (r *TenantSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TenantSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateTenantSettings(ctx, modelToSettings(data)); err != nil {
		resp.Diagnostics.AddError("Error updating tenant settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete clears the CNAME and resets optional fields; the tenant itself is not deleted.
func (r *TenantSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TenantSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdateTenantSettings(ctx, fider.TenantSettings{
		Title:  data.Title.ValueString(),
		Locale: data.Locale.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error clearing tenant settings", err.Error())
	}
}

func modelToSettings(m TenantSettingsModel) fider.TenantSettings {
	return fider.TenantSettings{
		Title:          m.Title.ValueString(),
		Invitation:     m.Invitation.ValueString(),
		WelcomeMessage: m.WelcomeMessage.ValueString(),
		WelcomeHeader:  m.WelcomeHeader.ValueString(),
		CNAME:          m.CNAME.ValueString(),
		Locale:         m.Locale.ValueString(),
	}
}
