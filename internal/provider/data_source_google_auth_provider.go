package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &GoogleAuthProviderDataSource{}

func NewGoogleAuthProviderDataSource() datasource.DataSource {
	return &GoogleAuthProviderDataSource{}
}

type GoogleAuthProviderDataSource struct {
	client *Client
}

type GoogleAuthProviderDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	ClientID   types.String `tfsdk:"client_id"`
	Context    types.String `tfsdk:"context"`
	IsDefault  types.Bool   `tfsdk:"is_default"`
	IsDisabled types.Bool   `tfsdk:"is_disabled"`
}

func (d *GoogleAuthProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_auth_provider"
}

func (d *GoogleAuthProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone Google Auth Provider data source",
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
				MarkdownDescription: "Google OAuth2 Client ID",
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

func (d *GoogleAuthProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GoogleAuthProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GoogleAuthProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := d.client.doRequest("GET", "/google_auth_providers/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Google auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ClientID   string `json:"client_id"`
			Context    string `json:"context"`
			IsDefault  bool   `json:"is_default"`
			IsDisabled bool   `json:"is_disabled"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.ClientID = types.StringValue(result.Data.ClientID)
	data.Context = types.StringValue(result.Data.Context)
	data.IsDefault = types.BoolValue(result.Data.IsDefault)
	data.IsDisabled = types.BoolValue(result.Data.IsDisabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
