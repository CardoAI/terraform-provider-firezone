package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &OIDCAuthProviderResource{}
var _ resource.ResourceWithImportState = &OIDCAuthProviderResource{}

func NewOIDCAuthProviderResource() resource.Resource {
	return &OIDCAuthProviderResource{}
}

type OIDCAuthProviderResource struct {
	client *Client
}

type OIDCAuthProviderResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	ClientID                    types.String `tfsdk:"client_id"`
	ClientSecret                types.String `tfsdk:"client_secret"`
	DiscoveryDocumentURI        types.String `tfsdk:"discovery_document_uri"`
	Scope                       types.String `tfsdk:"scope"`
	Context                     types.String `tfsdk:"context"`
	IsDefault                   types.Bool   `tfsdk:"is_default"`
	IsDisabled                  types.Bool   `tfsdk:"is_disabled"`
	ClientSessionLifetimeSecs   types.Int64  `tfsdk:"client_session_lifetime_secs"`
	PortalSessionLifetimeSecs   types.Int64  `tfsdk:"portal_session_lifetime_secs"`
}

func (r *OIDCAuthProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_auth_provider"
}

func (r *OIDCAuthProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone OIDC Auth Provider resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Provider ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Provider name",
				Required:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 Client ID",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "OAuth2 Client Secret",
				Required:            true,
				Sensitive:           true,
			},
			"discovery_document_uri": schema.StringAttribute{
				MarkdownDescription: "OIDC Discovery Document URI",
				Required:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "OAuth2 scope (optional, e.g., 'openid email profile')",
				Optional:            true,
			},
			"context": schema.StringAttribute{
				MarkdownDescription: "Where the provider can be used: clients_and_portal, clients_only, portal_only",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("clients_and_portal"),
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is the default provider",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the provider is disabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"client_session_lifetime_secs": schema.Int64Attribute{
				MarkdownDescription: "Client session lifetime in seconds (optional)",
				Optional:            true,
			},
			"portal_session_lifetime_secs": schema.Int64Attribute{
				MarkdownDescription: "Portal session lifetime in seconds (optional)",
				Optional:            true,
			},
		},
	}
}

func (r *OIDCAuthProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}

	r.client = client
}

func (r *OIDCAuthProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"OIDC Provider Creation Not Supported",
		"Firezone does not support creating OIDC authentication providers via API. "+
			"Please create the OIDC provider manually in the Firezone admin portal, then import it into Terraform using:\n\n"+
			"  terraform import firezone_oidc_auth_provider.<name> <provider_id>\n\n"+
			"You can find the provider ID in the Firezone UI or by listing providers via the API.",
	)
}

func (r *OIDCAuthProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OIDCAuthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.doRequest("GET", "/oidc_auth_providers/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OIDC auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                        string  `json:"id"`
			Name                      string  `json:"name"`
			ClientID                  string  `json:"client_id"`
			DiscoveryDocumentURI      string  `json:"discovery_document_uri"`
			Scope                     *string `json:"scope"`
			Context                   string  `json:"context"`
			IsDefault                 bool    `json:"is_default"`
			IsDisabled                bool    `json:"is_disabled"`
			ClientSessionLifetimeSecs *int64  `json:"client_session_lifetime_secs"`
			PortalSessionLifetimeSecs *int64  `json:"portal_session_lifetime_secs"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.ClientID = types.StringValue(result.Data.ClientID)
	data.DiscoveryDocumentURI = types.StringValue(result.Data.DiscoveryDocumentURI)
	if result.Data.Scope != nil && *result.Data.Scope != "" {
		data.Scope = types.StringValue(*result.Data.Scope)
	} else {
		data.Scope = types.StringNull()
	}
	data.Context = types.StringValue(result.Data.Context)
	data.IsDefault = types.BoolValue(result.Data.IsDefault)
	data.IsDisabled = types.BoolValue(result.Data.IsDisabled)
	if result.Data.ClientSessionLifetimeSecs != nil {
		data.ClientSessionLifetimeSecs = types.Int64Value(*result.Data.ClientSessionLifetimeSecs)
	} else {
		data.ClientSessionLifetimeSecs = types.Int64Null()
	}
	if result.Data.PortalSessionLifetimeSecs != nil {
		data.PortalSessionLifetimeSecs = types.Int64Value(*result.Data.PortalSessionLifetimeSecs)
	} else {
		data.PortalSessionLifetimeSecs = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCAuthProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OIDCAuthProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := map[string]interface{}{
		"provider": map[string]interface{}{
			"name":                   data.Name.ValueString(),
			"client_id":              data.ClientID.ValueString(),
			"client_secret":          data.ClientSecret.ValueString(),
			"discovery_document_uri": data.DiscoveryDocumentURI.ValueString(),
			"context":                data.Context.ValueString(),
			"is_default":             data.IsDefault.ValueBool(),
			"is_disabled":            data.IsDisabled.ValueBool(),
		},
	}

	// Only include optional fields if they are explicitly set
	provider := reqBody["provider"].(map[string]interface{})
	if !data.Scope.IsNull() && !data.Scope.IsUnknown() {
		provider["scope"] = data.Scope.ValueString()
	}
	if !data.ClientSessionLifetimeSecs.IsNull() && !data.ClientSessionLifetimeSecs.IsUnknown() {
		provider["client_session_lifetime_secs"] = data.ClientSessionLifetimeSecs.ValueInt64()
	}
	if !data.PortalSessionLifetimeSecs.IsNull() && !data.PortalSessionLifetimeSecs.IsUnknown() {
		provider["portal_session_lifetime_secs"] = data.PortalSessionLifetimeSecs.ValueInt64()
	}

	body, err := r.client.doRequest("PUT", "/oidc_auth_providers/"+data.ID.ValueString(), reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update OIDC auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                        string `json:"id"`
			Name                      string `json:"name"`
			ClientID                  string `json:"client_id"`
			DiscoveryDocumentURI      string `json:"discovery_document_uri"`
			Scope                     string `json:"scope"`
			Context                   string `json:"context"`
			IsDefault                 bool   `json:"is_default"`
			IsDisabled                bool   `json:"is_disabled"`
			ClientSessionLifetimeSecs int64  `json:"client_session_lifetime_secs"`
			PortalSessionLifetimeSecs int64  `json:"portal_session_lifetime_secs"`
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
	data.ClientSessionLifetimeSecs = types.Int64Value(result.Data.ClientSessionLifetimeSecs)
	data.PortalSessionLifetimeSecs = types.Int64Value(result.Data.PortalSessionLifetimeSecs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCAuthProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OIDCAuthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.doRequest("DELETE", "/oidc_auth_providers/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete OIDC auth provider: %s", err))
		return
	}
}

func (r *OIDCAuthProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
