package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure experimentDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &experimentDataSource{}

// NewExperimentDataSource returns a new experiment data source.
func NewExperimentDataSource() datasource.DataSource {
	return &experimentDataSource{}
}

// experimentDataSource queries an existing CloudLab experiment.
type experimentDataSource struct {
	client *Client
}

// experimentDataSourceModel maps the data source schema data.
type experimentDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Project        types.String `tfsdk:"project"`
	ProfileName    types.String `tfsdk:"profile_name"`
	ProfileProject types.String `tfsdk:"profile_project"`
	Creator        types.String `tfsdk:"creator"`
	Status         types.String `tfsdk:"status"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
}

// Metadata returns the data source type name.
func (d *experimentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experiment"
}

// Schema defines the schema for the data source.
func (d *experimentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries an existing CloudLab experiment by its UUID. " +
			"Use this data source to reference experiments that were created outside of Terraform " +
			"or in a separate Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the experiment to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The human-readable name of the experiment.",
				Computed:    true,
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project the experiment belongs to.",
				Computed:    true,
			},
			"profile_name": schema.StringAttribute{
				Description: "The name of the profile used to create the experiment.",
				Computed:    true,
			},
			"profile_project": schema.StringAttribute{
				Description: "The project that owns the profile.",
				Computed:    true,
			},
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the experiment.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the experiment.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the experiment was created.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The timestamp when the experiment is scheduled to expire.",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the data source.
func (d *experimentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *provider.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read fetches the experiment data.
func (d *experimentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state experimentDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading CloudLab experiment", map[string]any{"id": state.ID.ValueString()})

	exp, err := d.client.GetExperiment(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Experiment", err.Error())
		return
	}

	state.ID = types.StringValue(exp.ID)
	state.Name = types.StringValue(exp.Name)
	state.Project = types.StringValue(exp.Project)
	state.ProfileName = types.StringValue(exp.ProfileName)
	state.ProfileProject = types.StringValue(exp.ProfileProject)
	state.Creator = types.StringValue(exp.Creator)
	state.Status = types.StringValue(exp.Status)
	state.CreatedAt = types.StringValue(exp.CreatedAt)

	if exp.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*exp.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
