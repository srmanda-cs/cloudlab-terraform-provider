package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure profileResource satisfies the resource.Resource interface.
var _ resource.Resource = &profileResource{}

// NewProfileResource returns a new profile resource.
func NewProfileResource() resource.Resource {
	return &profileResource{}
}

// profileResource manages a CloudLab experiment profile.
type profileResource struct {
	client *Client
}

// profileResourceModel maps the resource schema data.
type profileResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Project         types.String `tfsdk:"project"`
	Script          types.String `tfsdk:"script"`
	RepositoryURL   types.String `tfsdk:"repository_url"`
	Public          types.Bool   `tfsdk:"public"`
	ProjectWritable types.Bool   `tfsdk:"project_writable"`
	// Computed read-only fields
	Creator           types.String `tfsdk:"creator"`
	Version           types.Int64  `tfsdk:"version"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	RepositoryRefspec types.String `tfsdk:"repository_refspec"`
	RepositoryHash    types.String `tfsdk:"repository_hash"`
	RepositoryGithook types.String `tfsdk:"repository_githook"`
}

// Metadata returns the resource type name.
func (r *profileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

// Schema defines the schema for the resource.
func (r *profileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudLab experiment profile. A profile defines the topology template " +
			"(nodes, networks, hardware types) used to instantiate experiments. " +
			"Provide either a geni-lib Python script or a repository URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the profile assigned by CloudLab.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the profile. Must be unique within the project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project that owns this profile.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"script": schema.StringAttribute{
				Description: "A geni-lib Python script that defines the experiment topology. " +
					"Mutually exclusive with repository_url. Can be updated in-place.",
				Optional: true,
			},
			"repository_url": schema.StringAttribute{
				Description: "URL of a git repository containing the profile. " +
					"Mutually exclusive with script.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public": schema.BoolAttribute{
				Description: "If true, the profile can be instantiated by any CloudLab user. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"project_writable": schema.BoolAttribute{
				Description: "If true, other members of the project can modify this profile. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			// Computed read-only attributes
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the profile.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Description: "The current version number of the profile.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the profile was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the profile was last updated.",
				Computed:    true,
			},
			"repository_refspec": schema.StringAttribute{
				Description: "The refspec of the profile (for repository-backed profiles).",
				Computed:    true,
			},
			"repository_hash": schema.StringAttribute{
				Description: "The commit hash of the profile (for repository-backed profiles).",
				Computed:    true,
			},
			"repository_githook": schema.StringAttribute{
				Description: "The Portal URL of the repository githook (for repository-backed profiles).",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the resource.
func (r *profileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the profile and sets the initial Terraform state.
func (r *profileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan profileResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &ProfileCreateRequest{
		Name:            plan.Name.ValueString(),
		Project:         plan.Project.ValueString(),
		Public:          plan.Public.ValueBool(),
		ProjectWritable: plan.ProjectWritable.ValueBool(),
	}

	if !plan.Script.IsNull() && !plan.Script.IsUnknown() {
		createReq.Script = plan.Script.ValueString()
	}
	if !plan.RepositoryURL.IsNull() && !plan.RepositoryURL.IsUnknown() {
		createReq.RepositoryURL = plan.RepositoryURL.ValueString()
	}

	tflog.Info(ctx, "Creating CloudLab profile", map[string]any{
		"name":    createReq.Name,
		"project": createReq.Project,
	})

	profile, err := r.client.CreateProfile(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Profile", err.Error())
		return
	}

	plan = mapProfileResponseToModel(profile, plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *profileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state profileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile, err := r.client.GetProfile(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Profile not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Profile", err.Error())
		return
	}

	state = mapProfileResponseToModel(profile, state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update modifies mutable profile attributes: script, public, project_writable.
// For repository-backed profiles, it also triggers a repo update if repository_url hasn't changed.
func (r *profileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state profileResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileID := state.ID.ValueString()
	var profile *ProfileResponse
	var err error

	// Check if any PATCH-able fields changed (script, public, project_writable)
	scriptChanged := !plan.Script.Equal(state.Script)
	publicChanged := !plan.Public.Equal(state.Public)
	projectWritableChanged := !plan.ProjectWritable.Equal(state.ProjectWritable)

	if scriptChanged || publicChanged || projectWritableChanged {
		modReq := &ProfileModifyRequest{}
		if scriptChanged && !plan.Script.IsNull() && !plan.Script.IsUnknown() {
			v := plan.Script.ValueString()
			modReq.Script = &v
		}
		if publicChanged {
			v := plan.Public.ValueBool()
			modReq.Public = &v
		}
		if projectWritableChanged {
			v := plan.ProjectWritable.ValueBool()
			modReq.ProjectWritable = &v
		}

		tflog.Info(ctx, "Modifying CloudLab profile", map[string]any{"id": profileID})
		profile, err = r.client.ModifyProfile(ctx, profileID, modReq)
		if err != nil {
			resp.Diagnostics.AddError("Error Modifying Profile", err.Error())
			return
		}
	}

	// If no changes needed a PATCH call, refresh state
	if profile == nil {
		profile, err = r.client.GetProfile(ctx, profileID)
		if err != nil {
			resp.Diagnostics.AddError("Error Reading Profile", err.Error())
			return
		}
	}

	plan = mapProfileResponseToModel(profile, plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the profile.
func (r *profileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state profileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting CloudLab profile", map[string]any{"id": state.ID.ValueString()})

	if err := r.client.DeleteProfile(ctx, state.ID.ValueString()); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Profile", err.Error())
		return
	}
}

// mapProfileResponseToModel maps an API response to the Terraform model.
func mapProfileResponseToModel(profile *ProfileResponse, model profileResourceModel) profileResourceModel {
	model.ID = types.StringValue(profile.ID)
	model.Name = types.StringValue(profile.Name)
	model.Project = types.StringValue(profile.Project)
	model.Creator = types.StringValue(profile.Creator)
	model.Version = types.Int64Value(profile.Version)
	model.Public = types.BoolValue(profile.Public)
	model.ProjectWritable = types.BoolValue(profile.ProjectWritable)
	model.CreatedAt = types.StringValue(profile.CreatedAt)

	if profile.UpdatedAt != nil {
		model.UpdatedAt = types.StringValue(*profile.UpdatedAt)
	} else {
		model.UpdatedAt = types.StringNull()
	}

	if profile.RepositoryURL != nil {
		model.RepositoryURL = types.StringValue(*profile.RepositoryURL)
	} else if model.RepositoryURL.IsUnknown() {
		model.RepositoryURL = types.StringNull()
	}

	if profile.RepositoryRefspec != nil {
		model.RepositoryRefspec = types.StringValue(*profile.RepositoryRefspec)
	} else {
		model.RepositoryRefspec = types.StringNull()
	}

	if profile.RepositoryHash != nil {
		model.RepositoryHash = types.StringValue(*profile.RepositoryHash)
	} else {
		model.RepositoryHash = types.StringNull()
	}

	if profile.RepositoryGithook != nil {
		model.RepositoryGithook = types.StringValue(*profile.RepositoryGithook)
	} else {
		model.RepositoryGithook = types.StringNull()
	}

	// Populate script from current_version if available and not already set by user
	if profile.CurrentVersion != nil && profile.CurrentVersion.Script != nil {
		if model.Script.IsNull() || model.Script.IsUnknown() {
			model.Script = types.StringValue(*profile.CurrentVersion.Script)
		}
	}

	return model
}
