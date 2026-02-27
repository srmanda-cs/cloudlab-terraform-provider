package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	snapshotPollInterval = 15 * time.Second
	snapshotPollTimeout  = 60 * time.Minute

	snapshotStatusReady  = "ready"
	snapshotStatusFailed = "failed"
)

// Ensure snapshotResource satisfies the resource.Resource interface.
var _ resource.Resource = &snapshotResource{}

// NewSnapshotResource returns a new snapshot resource.
func NewSnapshotResource() resource.Resource {
	return &snapshotResource{}
}

// snapshotResource manages a CloudLab node image snapshot.
type snapshotResource struct {
	client *Client
}

// snapshotResourceModel maps the resource schema data.
type snapshotResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ExperimentID    types.String `tfsdk:"experiment_id"`
	ClientID        types.String `tfsdk:"client_id"`
	ImageName       types.String `tfsdk:"image_name"`
	WholeDisk       types.Bool   `tfsdk:"whole_disk"`
	WaitForComplete types.Bool   `tfsdk:"wait_for_complete"`
	// Computed read-only fields
	Status          types.String `tfsdk:"status"`
	StatusTimestamp types.String `tfsdk:"status_timestamp"`
	ImageSize       types.Int64  `tfsdk:"image_size"`
	ImageURN        types.String `tfsdk:"image_urn"`
	ErrorMessage    types.String `tfsdk:"error_message"`
}

// Metadata returns the resource type name.
func (r *snapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

// Schema defines the schema for the resource.
func (r *snapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudLab node image snapshot. Creates an image snapshot of a running node " +
			"in an experiment. The image can then be used as a base image in future experiments. " +
			"Deleting this resource does not delete the created image — it only removes it from Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the snapshot request.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"experiment_id": schema.StringAttribute{
				Description: "The UUID of the running experiment containing the node to snapshot.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "The logical name (client ID) of the node to snapshot.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_name": schema.StringAttribute{
				Description: "The name of the image to create or update.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"whole_disk": schema.BoolAttribute{
				Description: "If true, take a whole disk image. Defaults to false (partition image).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"wait_for_complete": schema.BoolAttribute{
				Description: "If true (default), Terraform will wait until the snapshot completes " +
					"before finishing. Set to false to return immediately after the snapshot is initiated.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			// Computed read-only attributes
			"status": schema.StringAttribute{
				Description: "The current status of the snapshot operation.",
				Computed:    true,
			},
			"status_timestamp": schema.StringAttribute{
				Description: "The timestamp of the last status update.",
				Computed:    true,
			},
			"image_size": schema.Int64Attribute{
				Description: "The current size of the image in KB.",
				Computed:    true,
			},
			"image_urn": schema.StringAttribute{
				Description: "The URN of the created image.",
				Computed:    true,
			},
			"error_message": schema.StringAttribute{
				Description: "Error message if the snapshot failed.",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the resource.
func (r *snapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create initiates a node snapshot.
func (r *snapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan snapshotResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotReq := &SnapshotRequest{
		ImageName: plan.ImageName.ValueString(),
		WholeDisk: plan.WholeDisk.ValueBool(),
	}

	tflog.Info(ctx, "Creating CloudLab node snapshot", map[string]any{
		"experiment_id": plan.ExperimentID.ValueString(),
		"client_id":     plan.ClientID.ValueString(),
		"image_name":    snapshotReq.ImageName,
	})

	status, err := r.client.StartSnapshot(
		ctx,
		plan.ExperimentID.ValueString(),
		plan.ClientID.ValueString(),
		snapshotReq,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Starting Snapshot", err.Error())
		return
	}

	plan = mapSnapshotStatusToModel(status, plan)

	// Optionally wait for completion
	waitForComplete := plan.WaitForComplete.ValueBool()
	if waitForComplete {
		tflog.Info(ctx, "Waiting for snapshot to complete", map[string]any{"snapshot_id": status.ID})
		status, err = r.waitForSnapshot(ctx, plan.ExperimentID.ValueString(), status.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error Waiting for Snapshot", err.Error())
			return
		}
		plan = mapSnapshotStatusToModel(status, plan)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the snapshot status.
func (r *snapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state snapshotResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		return
	}

	status, err := r.client.GetSnapshotStatus(ctx, state.ExperimentID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Snapshot Status", err.Error())
		return
	}

	state = mapSnapshotStatusToModel(status, state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update is not supported — all snapshot attributes require replacement.
func (r *snapshotResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Snapshot parameters cannot be changed in-place. All changes require creating a new snapshot.",
	)
}

// Delete removes the snapshot from Terraform state.
// Note: The CloudLab API does not provide a delete endpoint for snapshots/images through this path.
// The created image persists in CloudLab even after the Terraform resource is destroyed.
func (r *snapshotResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Snapshot images are not deleted via the API snapshot endpoint.
	// They persist in CloudLab. This resource is removed from state only.
}

// waitForSnapshot polls until the snapshot reaches a terminal status.
func (r *snapshotResource) waitForSnapshot(ctx context.Context, experimentID, snapshotID string) (*SnapshotStatus, error) {
	deadline := time.Now().Add(snapshotPollTimeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for snapshot %s to complete", snapshotID)
		}

		status, err := r.client.GetSnapshotStatus(ctx, experimentID, snapshotID)
		if err != nil {
			return nil, fmt.Errorf("error polling snapshot status: %w", err)
		}

		switch status.Status {
		case snapshotStatusReady:
			return status, nil
		case snapshotStatusFailed:
			msg := fmt.Sprintf("snapshot %s failed", snapshotID)
			if status.ErrorMessage != nil {
				msg = fmt.Sprintf("snapshot %s failed: %s", snapshotID, *status.ErrorMessage)
			}
			return nil, fmt.Errorf("%s", msg)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(snapshotPollInterval):
		}
	}
}

// mapSnapshotStatusToModel maps an API SnapshotStatus response to the Terraform model.
func mapSnapshotStatusToModel(status *SnapshotStatus, model snapshotResourceModel) snapshotResourceModel {
	model.ID = types.StringValue(status.ID)
	model.Status = types.StringValue(status.Status)
	model.ImageURN = types.StringValue(status.ImageURN)

	if status.StatusTimestamp != nil {
		model.StatusTimestamp = types.StringValue(*status.StatusTimestamp)
	} else {
		model.StatusTimestamp = types.StringNull()
	}

	if status.ImageSize != nil {
		model.ImageSize = types.Int64Value(*status.ImageSize)
	} else {
		model.ImageSize = types.Int64Null()
	}

	if status.ErrorMessage != nil {
		model.ErrorMessage = types.StringValue(*status.ErrorMessage)
	} else {
		model.ErrorMessage = types.StringNull()
	}

	return model
}
