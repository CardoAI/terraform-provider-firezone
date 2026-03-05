package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ActorResource{}
var _ resource.ResourceWithImportState = &ActorResource{}

func NewActorResource() resource.Resource {
	return &ActorResource{}
}

type ActorResource struct {
	client *Client
}

type ActorResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Email                 types.String `tfsdk:"email"`
	Type                  types.String `tfsdk:"type"`
	AllowEmailOtpSignIn   types.Bool   `tfsdk:"allow_email_otp_sign_in"`
}

func (r *ActorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actor"
}

func (r *ActorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone Actor resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Actor name",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Actor email (optional for service accounts)",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Actor type (account_admin_user, account_user, service_account)",
				Required:            true,
			},
			"allow_email_otp_sign_in": schema.BoolAttribute{
				MarkdownDescription: "Allow email OTP sign in",
				Optional:            true,
			},
		},
	}
}

func (r *ActorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ActorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ActorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := map[string]interface{}{
		"actor": map[string]interface{}{
			"name":                    data.Name.ValueString(),
			"type":                    data.Type.ValueString(),
			"allow_email_otp_sign_in": data.AllowEmailOtpSignIn.ValueBool(),
		},
	}

	// Add email if provided
	if !data.Email.IsNull() && data.Email.ValueString() != "" {
		reqBody["actor"].(map[string]interface{})["email"] = data.Email.ValueString()
	}

	body, err := r.client.doRequest("POST", "/actors", reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create actor: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                  string `json:"id"`
			Name                string `json:"name"`
			Email               string `json:"email"`
			Type                string `json:"type"`
			AllowEmailOtpSignIn bool   `json:"allow_email_otp_sign_in"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Name)
	if result.Data.Email != "" {
		data.Email = types.StringValue(result.Data.Email)
	}
	data.Type = types.StringValue(result.Data.Type)
	data.AllowEmailOtpSignIn = types.BoolValue(result.Data.AllowEmailOtpSignIn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ActorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.doRequest("GET", "/actors/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read actor: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                  string `json:"id"`
			Name                string `json:"name"`
			Email               string `json:"email"`
			Type                string `json:"type"`
			AllowEmailOtpSignIn bool   `json:"allow_email_otp_sign_in"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	if result.Data.Email != "" {
		data.Email = types.StringValue(result.Data.Email)
	}
	data.Type = types.StringValue(result.Data.Type)
	data.AllowEmailOtpSignIn = types.BoolValue(result.Data.AllowEmailOtpSignIn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ActorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := map[string]interface{}{
		"actor": map[string]interface{}{
			"name":                    data.Name.ValueString(),
			"type":                    data.Type.ValueString(),
			"allow_email_otp_sign_in": data.AllowEmailOtpSignIn.ValueBool(),
		},
	}

	// Add email if provided
	if !data.Email.IsNull() && data.Email.ValueString() != "" {
		reqBody["actor"].(map[string]interface{})["email"] = data.Email.ValueString()
	}

	body, err := r.client.doRequest("PUT", "/actors/"+data.ID.ValueString(), reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update actor: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                  string `json:"id"`
			Name                string `json:"name"`
			Email               string `json:"email"`
			Type                string `json:"type"`
			AllowEmailOtpSignIn bool   `json:"allow_email_otp_sign_in"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	if result.Data.Email != "" {
		data.Email = types.StringValue(result.Data.Email)
	}
	data.Type = types.StringValue(result.Data.Type)
	data.AllowEmailOtpSignIn = types.BoolValue(result.Data.AllowEmailOtpSignIn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ActorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.doRequest("DELETE", "/actors/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete actor: %s", err))
		return
	}
}

func (r *ActorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
