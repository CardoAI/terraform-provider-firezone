package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &GoogleAuthProviderResource{}
var _ resource.ResourceWithImportState = &GoogleAuthProviderResource{}

func NewGoogleAuthProviderResource() resource.Resource {
	return &GoogleAuthProviderResource{}
}

type GoogleAuthProviderResource struct {
	client *Client
}

type GoogleAuthProviderResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	ClientID                    types.String `tfsdk:"client_id"`
	ClientSecret                types.String `tfsdk:"client_secret"`
	Context                     types.String `tfsdk:"context"`
	IsDefault                   types.Bool   `tfsdk:"is_default"`
	IsDisabled                  types.Bool   `tfsdk:"is_disabled"`
	ClientSessionLifetimeSecs   types.Int64  `tfsdk:"client_session_lifetime_secs"`
	PortalSessionLifetimeSecs   types.Int64  `tfsdk:"portal_session_lifetime_secs"`
}

func (r *GoogleAuthProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_google_auth_provider"
}

func (r *GoogleAuthProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone Google Auth Provider resource",
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
				MarkdownDescription: "Google OAuth2 Client ID",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Google OAuth2 Client Secret",
				Required:            true,
				Sensitive:           true,
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
				MarkdownDescription: "Client session lifetime in seconds",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(86400),
			},
			"portal_session_lifetime_secs": schema.Int64Attribute{
				MarkdownDescription: "Portal session lifetime in seconds",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
			},
		},
	}
}

func (r *GoogleAuthProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GoogleAuthProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Google Auth Provider Creation Not Supported",
		"Firezone does not support creating Google authentication providers via API. "+
			"Please create the Google provider manually in the Firezone admin portal, then import it into Terraform using:\n\n"+
			"  terraform import firezone_google_auth_provider.<name> <provider_id>\n\n"+
			"You can find the provider ID in the Firezone UI or by listing providers via the API.",
	)
}

func (r *GoogleAuthProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GoogleAuthProviderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.doRequest("GET", "/google_auth_providers/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Google auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                        string `json:"id"`
			Name                      string `json:"name"`
			ClientID                  string `json:"client_id"`
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
	data.Context = types.StringValue(result.Data.Context)
	data.IsDefault = types.BoolValue(result.Data.IsDefault)
	data.IsDisabled = types.BoolValue(result.Data.IsDisabled)
	data.ClientSessionLifetimeSecs = types.Int64Value(result.Data.ClientSessionLifetimeSecs)
	data.PortalSessionLifetimeSecs = types.Int64Value(result.Data.PortalSessionLifetimeSecs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GoogleAuthProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The Firezone API has no PATCH/PUT endpoint for Google auth providers.
	// Re-read current state from the API and preserve plan values for write-only fields.
	var plan GoogleAuthProviderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.doRequest("GET", "/google_auth_providers/"+plan.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Google auth provider: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                        string `json:"id"`
			Name                      string `json:"name"`
			ClientID                  string `json:"client_id"`
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

	plan.Name = types.StringValue(result.Data.Name)
	plan.ClientID = types.StringValue(result.Data.ClientID)
	plan.Context = types.StringValue(result.Data.Context)
	plan.IsDefault = types.BoolValue(result.Data.IsDefault)
	plan.IsDisabled = types.BoolValue(result.Data.IsDisabled)
	plan.ClientSessionLifetimeSecs = types.Int64Value(result.Data.ClientSessionLifetimeSecs)
	plan.PortalSessionLifetimeSecs = types.Int64Value(result.Data.PortalSessionLifetimeSecs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GoogleAuthProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The Firezone API has no DELETE endpoint for Google auth providers.
	// Remove from Terraform state only; the provider must be deleted manually via the admin portal.
}

func (r *GoogleAuthProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
