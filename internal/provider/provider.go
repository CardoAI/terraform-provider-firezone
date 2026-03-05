package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &FirezoneProvider{}

type FirezoneProvider struct {
	version string
}

type FirezoneProviderModel struct {
	ApiUrl types.String `tfsdk:"api_url"`
	ApiKey types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FirezoneProvider{
			version: version,
		}
	}
}

func (p *FirezoneProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "firezone"
	resp.Version = p.version
}

func (p *FirezoneProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Firezone API URL",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Firezone API Key",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *FirezoneProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data FirezoneProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiUrl := os.Getenv("FIREZONE_API_URL")
	apiKey := os.Getenv("FIREZONE_API_KEY")

	if !data.ApiUrl.IsNull() {
		apiUrl = data.ApiUrl.ValueString()
	}

	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	}

	if apiUrl == "" {
		resp.Diagnostics.AddError(
			"Missing API URL",
			"The provider cannot create the Firezone API client as there is a missing or empty value for the API URL. "+
				"Set the api_url value in the configuration or use the FIREZONE_API_URL environment variable.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider cannot create the Firezone API client as there is a missing or empty value for the API Key. "+
				"Set the api_key value in the configuration or use the FIREZONE_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := NewClient(apiUrl, apiKey)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *FirezoneProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewActorResource,
		NewGroupResource,
		NewResourceResource,
		NewPolicyResource,
		NewSiteResource,
		NewGoogleAuthProviderResource,
		NewOIDCAuthProviderResource,
		NewGroupMembershipResource,
	}
}

func (p *FirezoneProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewActorDataSource,
		NewGroupDataSource,
		NewGoogleAuthProviderDataSource,
		NewOIDCAuthProviderDataSource,
	}
}
