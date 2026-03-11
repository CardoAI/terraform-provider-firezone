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
	"github.com/hashicorp/terraform-plugin-framework/attr"
)

var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

type GroupMembershipResource struct {
	client *Client
}

type GroupMembershipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	GroupID   types.String `tfsdk:"group_id"`
	ActorIDs  types.Set    `tfsdk:"actor_ids"`
}

func (r *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages group membership (actors in a group). This resource manages the complete list of members for a group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Membership resource ID (same as group_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "Group ID",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"actor_ids": schema.SetAttribute{
				MarkdownDescription: "Set of actor IDs that are members of this group",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert actor IDs from set to strings
	var actorIDs []string
	resp.Diagnostics.Append(data.ActorIDs.ElementsAs(ctx, &actorIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use PATCH to add members
	reqBody := map[string]interface{}{
		"memberships": map[string]interface{}{
			"add": actorIDs,
		},
	}

	_, err := r.client.doRequest("PATCH", "/groups/"+data.GroupID.ValueString()+"/memberships", reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add group members: %s", err))
		return
	}

	data.ID = data.GroupID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.client.doRequest("GET", "/groups/"+data.GroupID.ValueString()+"/memberships", nil)
	if err != nil {
		// Check if it's a 404 error (resource deleted outside Terraform)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			// Resource no longer exists, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group memberships: %s", err))
		return
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse response: %s", err))
		return
	}

	// Convert to types.Set
	actorIDs := make([]attr.Value, len(result.Data))
	for i, member := range result.Data {
		actorIDs[i] = types.StringValue(member.ID)
	}

	actorIDsSet, diags := types.SetValue(types.StringType, actorIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ActorIDs = actorIDsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state GroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert sets to string slices
	var planActorIDsList, stateActorIDsList []string
	resp.Diagnostics.Append(plan.ActorIDs.ElementsAs(ctx, &planActorIDsList, false)...)
	resp.Diagnostics.Append(state.ActorIDs.ElementsAs(ctx, &stateActorIDsList, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which actors to add and remove
	planActorIDs := make(map[string]bool)
	for _, id := range planActorIDsList {
		planActorIDs[id] = true
	}

	stateActorIDs := make(map[string]bool)
	for _, id := range stateActorIDsList {
		stateActorIDs[id] = true
	}

	var toAdd, toRemove []string
	for id := range planActorIDs {
		if !stateActorIDs[id] {
			toAdd = append(toAdd, id)
		}
	}

	for id := range stateActorIDs {
		if !planActorIDs[id] {
			toRemove = append(toRemove, id)
		}
	}

	// Use PATCH to add/remove members
	if len(toAdd) > 0 || len(toRemove) > 0 {
		reqBody := map[string]interface{}{
			"memberships": map[string]interface{}{},
		}

		if len(toAdd) > 0 {
			reqBody["memberships"].(map[string]interface{})["add"] = toAdd
		}
		if len(toRemove) > 0 {
			reqBody["memberships"].(map[string]interface{})["remove"] = toRemove
		}

		_, err := r.client.doRequest("PATCH", "/groups/"+plan.GroupID.ValueString()+"/memberships", reqBody)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update group memberships: %s", err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert set to string slice
	var actorIDs []string
	resp.Diagnostics.Append(data.ActorIDs.ElementsAs(ctx, &actorIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(actorIDs) > 0 {
		reqBody := map[string]interface{}{
			"memberships": map[string]interface{}{
				"remove": actorIDs,
			},
		}

		_, err := r.client.doRequest("PATCH", "/groups/"+data.GroupID.ValueString()+"/memberships", reqBody)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove group members: %s", err))
			return
		}
	}
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: group_id
	groupID := req.ID
	
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), groupID)...)
}
