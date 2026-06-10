package provider

import (
	"context"

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
	URL    types.String `tfsdk:"url"`
	APIKey types.String `tfsdk:"api_key"`
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
		MarkdownDescription: "Provider for managing [Fider](https://fider.io) self-hosted feedback platform configuration.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Fider tenant (e.g. `https://feedback.example.com`). Can also be set via the `FIDER_URL` environment variable.",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Admin API key for the Fider tenant. Can also be set via the `FIDER_API_KEY` environment variable.",
				Required:            true,
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

	client := fider.NewClient(data.URL.ValueString(), data.APIKey.ValueString())
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *FiderProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOAuthConfigResource,
		NewTenantSettingsResource,
	}
}

func (p *FiderProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *FiderProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}
