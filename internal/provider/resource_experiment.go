package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure experimentResource satisfies the resource.Resource interface.
var _ resource.Resource = &experimentResource{}

// NewExperimentResource returns a new experiment resource.
func NewExperimentResource() resource.Resource {
	return &experimentResource{}
}

// experimentResource manages a CloudLab experiment.
type experimentResource struct {
	client *Client
}

// experimentResourceModel maps the resource schema data.
type experimentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Project        types.String `tfsdk:"project"`
	ProfileName    types.String `tfsdk:"profile_name"`
	ProfileProject types.String `tfsdk:"profile_project"`
	Duration       types.Int64  `tfsdk:"duration"`
	StartAt        types.String `tfsdk:"start_at"`
	StopAt         types.String `tfsdk:"stop_at"`
	Creator        types.String `tfsdk:"creator"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	WaitForReady   types.Bool   `tfsdk:"wait_for_ready"`
}

// Metadata returns the resource type name.
func (r *experimentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experiment"
}

// Schema defines the schema for the resource.
func (r *experimentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudLab experiment. Creating an experiment provisions physical or virtual " +
			"machines on the CloudLab testbed using a specified profile (topology template). " +
			"Deleting the experiment terminates and releases all associated resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the experiment assigned by CloudLab.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A human-readable name for the experiment. Must be unique within the project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project to instantiate the experiment in.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_name": schema.StringAttribute{
				Description: "The name of the profile (topology template) used to create the experiment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_project": schema.StringAttribute{
				Description: "The project that owns the profile.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"duration": schema.Int64Attribute{
				Description: "Initial experiment duration in hours.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"start_at": schema.StringAttribute{
				Description: "Schedule the experiment to start at a future time (RFC3339 format).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"stop_at": schema.StringAttribute{
				Description: "Schedule the experiment to stop at a future time (RFC3339 format).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"wait_for_ready": schema.BoolAttribute{
				Description: "If true (default), Terraform will wait until the experiment reaches 'ready' status " +
					"before completing. Set to false to return immediately after creation is submitted.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the experiment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The current status of the experiment (e.g., created, ready, failed).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the experiment was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "The timestamp when the experiment is scheduled to expire.",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the resource.
func (r *experimentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the experiment and sets the initial Terraform state.
func (r *experimentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan experimentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &ExperimentCreateRequest{
		Name:           plan.Name.ValueString(),
		Project:        plan.Project.ValueString(),
		ProfileName:    plan.ProfileName.ValueString(),
		ProfileProject: plan.ProfileProject.ValueString(),
	}

	if !plan.Duration.IsNull() && !plan.Duration.IsUnknown() {
		v := plan.Duration.ValueInt64()
		createReq.Duration = &v
	}
	if !plan.StartAt.IsNull() && !plan.StartAt.IsUnknown() {
		v := plan.StartAt.ValueString()
		createReq.StartAt = &v
	}
	if !plan.StopAt.IsNull() && !plan.StopAt.IsUnknown() {
		v := plan.StopAt.ValueString()
		createReq.StopAt = &v
	}

	tflog.Info(ctx, "Creating CloudLab experiment", map[string]any{
		"name":    createReq.Name,
		"project": createReq.Project,
	})

	exp, err := r.client.CreateExperiment(createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Experiment", err.Error())
		return
	}

	waitForReady := true
	if !plan.WaitForReady.IsNull() && !plan.WaitForReady.IsUnknown() {
		waitForReady = plan.WaitForReady.ValueBool()
	}

	if waitForReady {
		tflog.Info(ctx, "Waiting for experiment to become ready", map[string]any{"id": exp.ID})
		exp, err = r.client.WaitForExperiment(ctx, exp.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error Waiting for Experiment", err.Error())
			return
		}
	}

	plan = mapExperimentResponseToModel(exp, plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *experimentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state experimentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exp, err := r.client.GetExperiment(state.ID.ValueString())
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Experiment not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Experiment", err.Error())
		return
	}

	state = mapExperimentResponseToModel(exp, state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource. Experiments are immutable after creation;
// all changes require a replace (handled by RequiresReplace plan modifiers).
func (r *experimentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"CloudLab experiments cannot be updated in-place. All configuration changes require a new experiment.",
	)
}

// Delete terminates the experiment.
func (r *experimentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state experimentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting CloudLab experiment", map[string]any{"id": state.ID.ValueString()})

	if err := r.client.DeleteExperiment(state.ID.ValueString()); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Already gone, nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error Deleting Experiment", err.Error())
		return
	}
}

// mapExperimentResponseToModel maps an API response to the Terraform model.
func mapExperimentResponseToModel(exp *ExperimentResponse, model experimentResourceModel) experimentResourceModel {
	model.ID = types.StringValue(exp.ID)
	model.Name = types.StringValue(exp.Name)
	model.Project = types.StringValue(exp.Project)
	model.ProfileName = types.StringValue(exp.ProfileName)
	model.ProfileProject = types.StringValue(exp.ProfileProject)
	model.Creator = types.StringValue(exp.Creator)
	model.Status = types.StringValue(exp.Status)
	model.CreatedAt = types.StringValue(exp.CreatedAt)

	if exp.ExpiresAt != nil {
		model.ExpiresAt = types.StringValue(*exp.ExpiresAt)
	} else {
		model.ExpiresAt = types.StringNull()
	}

	if model.WaitForReady.IsNull() || model.WaitForReady.IsUnknown() {
		model.WaitForReady = types.BoolValue(true)
	}

	return model
}
