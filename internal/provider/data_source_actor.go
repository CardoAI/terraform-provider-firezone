package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ActorDataSource{}

func NewActorDataSource() datasource.DataSource {
	return &ActorDataSource{}
}

type ActorDataSource struct {
	client *Client
}

type ActorDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Email types.String `tfsdk:"email"`
	Type  types.String `tfsdk:"type"`
}

func (d *ActorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actor"
}

func (d *ActorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone Actor data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Actor ID",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Actor name",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Actor email",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Actor type",
				Computed:            true,
			},
		},
	}
}

func (d *ActorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ActorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := d.client.doRequest("GET", "/actors/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read actor: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Type  string `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.Email = types.StringValue(result.Data.Email)
	data.Type = types.StringValue(result.Data.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
