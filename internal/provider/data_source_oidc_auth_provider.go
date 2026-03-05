package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OIDCAuthProviderDataSource{}

func NewOIDCAuthProviderDataSource() datasource.DataSource {
	return &OIDCAuthProviderDataSource{}
}

type OIDCAuthProviderDataSource struct {
	client *Client
}

type OIDCAuthProviderDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	ClientID             types.String `tfsdk:"client_id"`
	DiscoveryDocumentURI types.String `tfsdk:"discovery_document_uri"`
	Scope                types.String `tfsdk:"scope"`
	Context              types.String `tfsdk:"context"`
	IsDefault            types.Bool   `tfsdk:"is_default"`
	IsDisabled           types.Bool   `tfsdk:"is_disabled"`
}

func (d *OIDCAuthProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_auth_provider"
}

func (d *OIDCAuthProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone OIDC Auth Provider data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Provider ID",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Provider name",
				Computed:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 Client ID",
				Computed:            true,
			},
			"discovery_document_uri": schema.StringAttribute{
				MarkdownDescription: "OIDC Discovery Document URI",
				Computed:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "OAuth2 scope",
				Computed:            true,
			},
			"context": schema.StringAttribute{
				MarkdownDescription: "Provider context",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is the default provider",
				Computed:            true,
			},
			"is_disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the provider is disabled",
				Computed:            true,
			},
		},
	}
}

func (d *OIDCAuthProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", "Expected *Client")
		return
	}

	d.client = client
}

func (d *OIDCAuthProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OIDCAuthProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := d.client.doRequest("GET", "/oidc_auth_providers/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OIDC auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                   string `json:"id"`
			Name                 string `json:"name"`
			ClientID             string `json:"client_id"`
			DiscoveryDocumentURI string `json:"discovery_document_uri"`
			Scope                string `json:"scope"`
			Context              string `json:"context"`
			IsDefault            bool   `json:"is_default"`
			IsDisabled           bool   `json:"is_disabled"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.ClientID = types.StringValue(result.Data.ClientID)
	data.DiscoveryDocumentURI = types.StringValue(result.Data.DiscoveryDocumentURI)
	data.Scope = types.StringValue(result.Data.Scope)
	data.Context = types.StringValue(result.Data.Context)
	data.IsDefault = types.BoolValue(result.Data.IsDefault)
	data.IsDisabled = types.BoolValue(result.Data.IsDisabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
