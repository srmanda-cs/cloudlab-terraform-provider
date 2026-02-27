package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure experimentResource satisfies the resource.Resource and
// resource.ResourceWithImportState interfaces.
var _ resource.Resource = &experimentResource{}
var _ resource.ResourceWithImportState = &experimentResource{}

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
	Group          types.String `tfsdk:"group"`
	ProfileName    types.String `tfsdk:"profile_name"`
	ProfileProject types.String `tfsdk:"profile_project"`
	Duration       types.Int64  `tfsdk:"duration"`
	StartAt        types.String `tfsdk:"start_at"`
	StopAt         types.String `tfsdk:"stop_at"`
	ParamsetName   types.String `tfsdk:"paramset_name"`
	ParamsetOwner  types.String `tfsdk:"paramset_owner"`
	Bindings       types.Map    `tfsdk:"bindings"`
	Refspec        types.String `tfsdk:"refspec"`
	SSHPubKey      types.String `tfsdk:"sshpubkey"`
	WaitForReady   types.Bool   `tfsdk:"wait_for_ready"`
	// Extension fields (mutable via Update)
	ExpiresAt    types.String `tfsdk:"expires_at"`
	ExtendBy     types.Int64  `tfsdk:"extend_by"`
	ExtendReason types.String `tfsdk:"extend_reason"`
	// Computed read-only fields
	Creator           types.String `tfsdk:"creator"`
	Updater           types.String `tfsdk:"updater"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	StartedAt         types.String `tfsdk:"started_at"`
	URL               types.String `tfsdk:"url"`
	WBStoreID         types.String `tfsdk:"wbstore_id"`
	RepositoryURL     types.String `tfsdk:"repository_url"`
	RepositoryRefspec types.String `tfsdk:"repository_refspec"`
	RepositoryHash    types.String `tfsdk:"repository_hash"`
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
			"group": schema.StringAttribute{
				Description: "The project subgroup to instantiate the experiment in.",
				Optional:    true,
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
				Validators: []validator.String{
					validateRFC3339(),
				},
			},
			"stop_at": schema.StringAttribute{
				Description: "Schedule the experiment to stop at a future time (RFC3339 format).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validateRFC3339(),
				},
			},
			"paramset_name": schema.StringAttribute{
				Description: "Optional name of a parameter set to apply to the profile.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"paramset_owner": schema.StringAttribute{
				Description: "The owner of the parameter set.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bindings": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Optional map of parameter bindings to apply to the profile. " +
					"Values must be strings (use `tostring()` for numeric parameters if needed).",
				Optional: true,
			},
			"refspec": schema.StringAttribute{
				Description: "For repository-based profiles, optionally specify a refspec[:hash] to use instead of HEAD.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sshpubkey": schema.StringAttribute{
				Description: "Additional SSH public key to install in the experiment.",
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
			// Mutable extension attributes
			"expires_at": schema.StringAttribute{
				Description: "The timestamp when the experiment is scheduled to expire (RFC3339 format). " +
					"Setting or updating this performs a PUT /experiments/{id} to set an absolute expiry. " +
					"Mutually exclusive in intent with extend_by — use one or the other per apply.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validateRFC3339(),
				},
			},
			"extend_by": schema.Int64Attribute{
				Description: "Number of hours to add to the experiment's current expiration. " +
					"Changing this value performs a PUT /experiments/{id} with extend_by set to the new value. " +
					"To extend by additional hours in a subsequent apply, increment this value " +
					"(e.g. 24 → 48 means \"I have extended by 48 h total in this config\"). " +
					"Mutually exclusive in intent with expires_at — use one or the other per apply.",
				Optional: true,
			},
			"extend_reason": schema.StringAttribute{
				Description: "Reason provided when extending the experiment lifetime (via expires_at or extend_by).",
				Optional:    true,
			},
			// Computed read-only attributes
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the experiment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updater": schema.StringAttribute{
				Description: "The CloudLab username who last updated the experiment.",
				Computed:    true,
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
			"started_at": schema.StringAttribute{
				Description: "The timestamp when the experiment was actually started.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the Portal status page for the experiment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wbstore_id": schema.StringAttribute{
				Description: "The ID of the experiment's WB store.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_url": schema.StringAttribute{
				Description: "The URL of the repository (for repository-backed profiles).",
				Computed:    true,
			},
			"repository_refspec": schema.StringAttribute{
				Description: "The refspec of the experiment (for repository-backed profiles).",
				Computed:    true,
			},
			"repository_hash": schema.StringAttribute{
				Description: "The commit hash of the experiment (for repository-backed profiles).",
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

	if !plan.Group.IsNull() && !plan.Group.IsUnknown() {
		createReq.Group = plan.Group.ValueString()
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
	if !plan.ParamsetName.IsNull() && !plan.ParamsetName.IsUnknown() {
		v := plan.ParamsetName.ValueString()
		createReq.ParamsetName = &v
	}
	if !plan.ParamsetOwner.IsNull() && !plan.ParamsetOwner.IsUnknown() {
		v := plan.ParamsetOwner.ValueString()
		createReq.ParamsetOwner = &v
	}
	if !plan.Bindings.IsNull() && !plan.Bindings.IsUnknown() {
		createReq.Bindings = bindingsFromMap(plan.Bindings)
	}
	if !plan.Refspec.IsNull() && !plan.Refspec.IsUnknown() {
		v := plan.Refspec.ValueString()
		createReq.Refspec = &v
	}
	if !plan.SSHPubKey.IsNull() && !plan.SSHPubKey.IsUnknown() {
		v := plan.SSHPubKey.ValueString()
		createReq.SSHPubKey = &v
	}

	tflog.Info(ctx, "Creating CloudLab experiment", map[string]any{
		"name":    createReq.Name,
		"project": createReq.Project,
	})

	exp, err := r.client.CreateExperiment(ctx, createReq)
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

	exp, err := r.client.GetExperiment(ctx, state.ID.ValueString())
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

// Update handles mutable changes: extending experiment lifetime and modifying bindings.
func (r *experimentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state experimentResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	experimentID := state.ID.ValueString()
	var exp *ExperimentResponse
	var err error

	// Handle extension (expires_at or extend_by changed)
	expiresAtChanged := !plan.ExpiresAt.Equal(state.ExpiresAt)
	extendByChanged := !plan.ExtendBy.Equal(state.ExtendBy)

	if expiresAtChanged || extendByChanged {
		extReq := &ExperimentExtendRequest{}
		if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
			v := plan.ExpiresAt.ValueString()
			extReq.ExpiresAt = &v
		}
		if !plan.ExtendBy.IsNull() && !plan.ExtendBy.IsUnknown() {
			v := plan.ExtendBy.ValueInt64()
			extReq.ExtendBy = &v
		}
		if !plan.ExtendReason.IsNull() && !plan.ExtendReason.IsUnknown() {
			v := plan.ExtendReason.ValueString()
			extReq.Reason = &v
		}
		tflog.Info(ctx, "Extending CloudLab experiment", map[string]any{"id": experimentID})
		exp, err = r.client.ExtendExperiment(ctx, experimentID, extReq)
		if err != nil {
			resp.Diagnostics.AddError("Error Extending Experiment", err.Error())
			return
		}
	}

	// Handle bindings modification
	if !plan.Bindings.Equal(state.Bindings) {
		var bindings map[string]any
		if !plan.Bindings.IsNull() && !plan.Bindings.IsUnknown() {
			bindings = bindingsFromMap(plan.Bindings)
		}
		modReq := &ExperimentModifyRequest{Bindings: bindings}
		tflog.Info(ctx, "Modifying CloudLab experiment bindings", map[string]any{"id": experimentID})
		exp, err = r.client.ModifyExperiment(ctx, experimentID, modReq)
		if err != nil {
			resp.Diagnostics.AddError("Error Modifying Experiment", err.Error())
			return
		}
	}

	// If no API call was made, refresh state
	if exp == nil {
		exp, err = r.client.GetExperiment(ctx, experimentID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Experiment", err.Error())
			return
		}
	}

	plan = mapExperimentResponseToModel(exp, plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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

	if err := r.client.DeleteExperiment(ctx, state.ID.ValueString()); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Already gone, nothing to do.
			return
		}
		resp.Diagnostics.AddError("Error Deleting Experiment", err.Error())
		return
	}
}

// ImportState implements resource.ResourceWithImportState.
// The import ID is the experiment UUID assigned by CloudLab.
func (r *experimentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// bindingsFromMap converts a types.Map (string element type) to a map[string]any
// suitable for sending to the CloudLab API as profile parameter bindings.
func bindingsFromMap(m types.Map) map[string]any {
	result := make(map[string]any, len(m.Elements()))
	for k, v := range m.Elements() {
		if sv, ok := v.(types.String); ok {
			result[k] = sv.ValueString()
		}
	}
	return result
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
	model.URL = types.StringValue(exp.URL)
	model.WBStoreID = types.StringValue(exp.WBStoreID)

	if exp.Group != "" {
		model.Group = types.StringValue(exp.Group)
	} else if model.Group.IsUnknown() {
		model.Group = types.StringNull()
	}

	if exp.Updater != nil {
		model.Updater = types.StringValue(*exp.Updater)
	} else {
		model.Updater = types.StringNull()
	}

	if exp.StartedAt != nil {
		model.StartedAt = types.StringValue(*exp.StartedAt)
	} else {
		model.StartedAt = types.StringNull()
	}

	if exp.ExpiresAt != nil {
		model.ExpiresAt = types.StringValue(*exp.ExpiresAt)
	} else {
		model.ExpiresAt = types.StringNull()
	}

	if exp.RepositoryURL != nil {
		model.RepositoryURL = types.StringValue(*exp.RepositoryURL)
	} else {
		model.RepositoryURL = types.StringNull()
	}

	if exp.RepositoryRefspec != nil {
		model.RepositoryRefspec = types.StringValue(*exp.RepositoryRefspec)
	} else {
		model.RepositoryRefspec = types.StringNull()
	}

	if exp.RepositoryHash != nil {
		model.RepositoryHash = types.StringValue(*exp.RepositoryHash)
	} else {
		model.RepositoryHash = types.StringNull()
	}

	if model.WaitForReady.IsNull() || model.WaitForReady.IsUnknown() {
		model.WaitForReady = types.BoolValue(true)
	}

	return model
}
