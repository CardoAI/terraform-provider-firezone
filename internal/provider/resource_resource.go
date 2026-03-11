package provider

import (
	"strings"
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

var _ resource.Resource = &ResourceResource{}
var _ resource.ResourceWithImportState = &ResourceResource{}

func NewResourceResource() resource.Resource {
	return &ResourceResource{}
}

type ResourceResource struct {
	client *Client
}

type ResourceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Address            types.String `tfsdk:"address"`
	AddressDescription types.String `tfsdk:"address_description"`
	Type               types.String `tfsdk:"type"`
	SiteID             types.String `tfsdk:"site_id"`
	IPStack            types.String `tfsdk:"ip_stack"`
}

func (r *ResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *ResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firezone Resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Resource name",
				Required:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Resource address (IP, CIDR, or DNS)",
				Required:            true,
			},
			"address_description": schema.StringAttribute{
				MarkdownDescription: "Resource address description",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Resource type (ip, cidr, dns)",
				Required:            true,
			},
			"site_id": schema.StringAttribute{
				MarkdownDescription: "Site ID to connect the resource to. Required for new resources, but not returned by API for existing resources.",
				Optional:            true,
				Computed:            true,
			},
			"ip_stack": schema.StringAttribute{
				MarkdownDescription: "IP stack type (ipv4_only, ipv6_only, dual). Only supported for DNS resources.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	reqBody := map[string]interface{}{
		"resource": map[string]interface{}{
			"name":                data.Name.ValueString(),
			"address":             data.Address.ValueString(),
			"address_description": data.AddressDescription.ValueString(),
			"type":                data.Type.ValueString(),
		},
	}

	// Add site_id if provided
	if !data.SiteID.IsNull() && data.SiteID.ValueString() != "" {
		reqBody["resource"].(map[string]interface{})["site_id"] = data.SiteID.ValueString()
	}

	// Add ip_stack if provided
	if !data.IPStack.IsNull() && data.IPStack.ValueString() != "" {
		reqBody["resource"].(map[string]interface{})["ip_stack"] = data.IPStack.ValueString()
	}

	body, err := r.client.doRequest("POST", "/resources", reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create resource: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                 string  `json:"id"`
			Name               string  `json:"name"`
			Address            string  `json:"address"`
			AddressDescription string  `json:"address_description"`
			Type               string  `json:"type"`
			IPStack            *string `json:"ip_stack"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.ID = types.StringValue(result.Data.ID)
	data.Name = types.StringValue(result.Data.Name)
	data.Address = types.StringValue(result.Data.Address)
	data.AddressDescription = types.StringValue(result.Data.AddressDescription)
	data.Type = types.StringValue(result.Data.Type)
	// site_id is preserved from plan (API doesn't return it)
	if result.Data.IPStack != nil {
		data.IPStack = types.StringValue(*result.Data.IPStack)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store the current ip_stack value from state
	currentIPStack := data.IPStack

	body, err := r.client.doRequest("GET", "/resources/"+data.ID.ValueString(), nil)
	if err != nil {
		// Check if it's a 404 error (resource deleted outside Terraform)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			// Resource no longer exists, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                 string  `json:"id"`
			Name               string  `json:"name"`
			Address            string  `json:"address"`
			AddressDescription string  `json:"address_description"`
			Type               string  `json:"type"`
			IPStack            *string `json:"ip_stack"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.Address = types.StringValue(result.Data.Address)
	data.AddressDescription = types.StringValue(result.Data.AddressDescription)
	data.Type = types.StringValue(result.Data.Type)
	// site_id is preserved from state (API doesn't return it)
	
	// Handle ip_stack: use API value if present, otherwise preserve configured value
	if result.Data.IPStack != nil {
		data.IPStack = types.StringValue(*result.Data.IPStack)
	} else {
		// API doesn't return ip_stack, preserve the configured value
		data.IPStack = currentIPStack
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store the planned ip_stack value
	plannedIPStack := data.IPStack

	reqBody := map[string]interface{}{
		"resource": map[string]interface{}{
			"name":                data.Name.ValueString(),
			"address":             data.Address.ValueString(),
			"address_description": data.AddressDescription.ValueString(),
			"type":                data.Type.ValueString(),
		},
	}

	// Add site_id if provided
	if !data.SiteID.IsNull() && data.SiteID.ValueString() != "" {
		reqBody["resource"].(map[string]interface{})["site_id"] = data.SiteID.ValueString()
	}

	// Add ip_stack if provided
	if !data.IPStack.IsNull() && data.IPStack.ValueString() != "" {
		reqBody["resource"].(map[string]interface{})["ip_stack"] = data.IPStack.ValueString()
	}

	body, err := r.client.doRequest("PUT", "/resources/"+data.ID.ValueString(), reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update resource: %s", err))
		return
	}

	var result struct {
		Data struct {
			ID                 string  `json:"id"`
			Name               string  `json:"name"`
			Address            string  `json:"address"`
			AddressDescription string  `json:"address_description"`
			Type               string  `json:"type"`
			IPStack            *string `json:"ip_stack"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	data.Name = types.StringValue(result.Data.Name)
	data.Address = types.StringValue(result.Data.Address)
	data.AddressDescription = types.StringValue(result.Data.AddressDescription)
	data.Type = types.StringValue(result.Data.Type)
	// site_id is preserved from plan (API doesn't return it)
	
	// Handle ip_stack: use API value if present, otherwise preserve planned value
	if result.Data.IPStack != nil {
		data.IPStack = types.StringValue(*result.Data.IPStack)
	} else {
		// API doesn't return ip_stack, preserve the planned value
		data.IPStack = plannedIPStack
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.doRequest("DELETE", "/resources/"+data.ID.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete resource: %s", err))
		return
	}
}

func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
