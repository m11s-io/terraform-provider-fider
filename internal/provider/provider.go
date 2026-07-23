package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m11s-io/terraform-provider-fider/internal/fider"
)

var _ provider.Provider = &FiderProvider{}
var _ provider.ProviderWithFunctions = &FiderProvider{}

type FiderProvider struct {
	version string
}

type FiderProviderModel struct {
	URL            types.String `tfsdk:"url"`
	APIKey         types.String `tfsdk:"api_key"`
	BootstrapURL   types.String `tfsdk:"bootstrap_url"`
	BootstrapToken types.String `tfsdk:"bootstrap_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FiderProvider{version: version}
	}
}

func (p *FiderProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "fider"
	resp.Version = p.version
}

func (p *FiderProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing [Fider](https://fider.io) self-hosted feedback platform configuration.\n\n" +
			"The provider supports two modes, which can be combined:\n\n" +
			"- **Bootstrap mode** (`bootstrap_url` + `bootstrap_token`): provisions and configures " +
			"tenants instance-wide through the internal bootstrap API. Used by `fider_tenant` and by " +
			"`fider_oauth_config` when its `tenant` attribute is set. Enables a single-apply workflow.\n" +
			"- **Tenant mode** (`url` + `api_key`): manages resources for a single existing tenant using " +
			"that tenant's admin API key.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of a single Fider tenant (e.g. `https://feedback.example.com`) for tenant-mode resources. Can also be set via the `FIDER_URL` environment variable.",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Admin API key for the Fider tenant (tenant mode). Can also be set via the `FIDER_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"bootstrap_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Fider instance for bootstrap-mode operations (e.g. `https://stage.internal.m11s.io`). Can also be set via the `FIDER_BOOTSTRAP_URL` environment variable.",
				Optional:            true,
			},
			"bootstrap_token": schema.StringAttribute{
				MarkdownDescription: "Instance-wide bootstrap token (`BOOTSTRAP_TOKEN` on the Fider server) used to authenticate bootstrap-mode operations. Can also be set via the `FIDER_BOOTSTRAP_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *FiderProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data FiderProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := fider.ClientConfig{
		URL:            firstNonEmpty(data.URL.ValueString(), os.Getenv("FIDER_URL")),
		APIKey:         firstNonEmpty(data.APIKey.ValueString(), os.Getenv("FIDER_API_KEY")),
		BootstrapURL:   firstNonEmpty(data.BootstrapURL.ValueString(), os.Getenv("FIDER_BOOTSTRAP_URL")),
		BootstrapToken: firstNonEmpty(data.BootstrapToken.ValueString(), os.Getenv("FIDER_BOOTSTRAP_TOKEN")),
	}

	client := fider.NewClient(cfg)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *FiderProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOAuthConfigResource,
		NewTenantSettingsResource,
		NewTenantResource,
	}
}

func (p *FiderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *FiderProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
