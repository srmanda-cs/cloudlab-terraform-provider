package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure vlanConnectionResource satisfies the resource.Resource interface.
var _ resource.Resource = &vlanConnectionResource{}

// NewVlanConnectionResource returns a new VLAN connection resource.
func NewVlanConnectionResource() resource.Resource {
	return &vlanConnectionResource{}
}

// vlanConnectionResource manages a shared VLAN connection between two CloudLab experiments.
type vlanConnectionResource struct {
	client *Client
}

// vlanConnectionResourceModel maps the resource schema data.
type vlanConnectionResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ExperimentID types.String `tfsdk:"experiment_id"`
	SourceLan    types.String `tfsdk:"source_lan"`
	TargetID     types.String `tfsdk:"target_id"`
	TargetLan    types.String `tfsdk:"target_lan"`
}

// Metadata returns the resource type name.
func (r *vlanConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vlan_connection"
}

// Schema defines the schema for the resource.
func (r *vlanConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a shared VLAN connection between two CloudLab experiments. " +
			"Creates a layer-2 connection between a LAN in one experiment and a LAN in another experiment. " +
			"Both experiments must be running and have shared VLANs configured in their profiles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A synthetic identifier for this VLAN connection, formatted as experiment_id/source_lan.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"experiment_id": schema.StringAttribute{
				Description: "The UUID of the source experiment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_lan": schema.StringAttribute{
				Description: "The client ID of the LAN in the source experiment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_id": schema.StringAttribute{
				Description: "The UUID or project,name of the target experiment to connect to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_lan": schema.StringAttribute{
				Description: "The client ID of the LAN in the target experiment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure sets the provider-configured client on the resource.
func (r *vlanConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates the VLAN connection.
func (r *vlanConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vlanConnectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	experimentID := plan.ExperimentID.ValueString()
	sourceLan := plan.SourceLan.ValueString()
	targetID := plan.TargetID.ValueString()
	targetLan := plan.TargetLan.ValueString()

	tflog.Info(ctx, "Connecting CloudLab VLAN", map[string]any{
		"experiment_id": experimentID,
		"source_lan":    sourceLan,
		"target_id":     targetID,
		"target_lan":    targetLan,
	})

	if err := r.client.ConnectExperimentVlan(experimentID, sourceLan, targetID, targetLan); err != nil {
		resp.Diagnostics.AddError("Error Connecting VLAN", err.Error())
		return
	}

	// Generate a synthetic ID for tracking this connection in state.
	plan.ID = types.StringValue(experimentID + "/" + sourceLan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the state. The CloudLab API does not provide a direct way to query
// VLAN connection status, so we keep state as-is (connection state is opaque).
func (r *vlanConnectionResource) Read(_ context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vlanConnectionResourceModel
	diags := req.State.Get(context.Background(), &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// No API for querying connection state — preserve existing state.
	diags = resp.State.Set(context.Background(), state)
	resp.Diagnostics.Append(diags...)
}

// Update is not supported for VLAN connections (all attributes require replace).
func (r *vlanConnectionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"VLAN connection attributes cannot be updated in-place. Please delete and recreate the connection.",
	)
}

// Delete disconnects the VLAN.
func (r *vlanConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vlanConnectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Disconnecting CloudLab VLAN", map[string]any{
		"experiment_id": state.ExperimentID.ValueString(),
		"source_lan":    state.SourceLan.ValueString(),
	})

	if err := r.client.DisconnectExperimentVlan(state.ExperimentID.ValueString(), state.SourceLan.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error Disconnecting VLAN", err.Error())
		return
	}
}
