package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure profileDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &profileDataSource{}

// NewProfileDataSource returns a new profile data source.
func NewProfileDataSource() datasource.DataSource {
	return &profileDataSource{}
}

// profileDataSource queries an existing CloudLab profile.
type profileDataSource struct {
	client *Client
}

// profileDataSourceModel maps the data source schema data.
type profileDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Project           types.String `tfsdk:"project"`
	Creator           types.String `tfsdk:"creator"`
	Version           types.Int64  `tfsdk:"version"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	RepositoryURL     types.String `tfsdk:"repository_url"`
	RepositoryRefspec types.String `tfsdk:"repository_refspec"`
	RepositoryHash    types.String `tfsdk:"repository_hash"`
	RepositoryGithook types.String `tfsdk:"repository_githook"`
	Public            types.Bool   `tfsdk:"public"`
	ProjectWritable   types.Bool   `tfsdk:"project_writable"`
}

// Metadata returns the data source type name.
func (d *profileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

// Schema defines the schema for the data source.
func (d *profileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries an existing CloudLab profile by its UUID or project,name identifier. " +
			"Use this data source to reference profiles that were created outside of Terraform " +
			"or in a separate Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID or project,name) of the profile to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the profile.",
				Computed:    true,
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project that owns the profile.",
				Computed:    true,
			},
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the profile.",
				Computed:    true,
			},
			"version": schema.Int64Attribute{
				Description: "The current version number of the profile.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the profile was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the profile was last updated.",
				Computed:    true,
			},
			"repository_url": schema.StringAttribute{
				Description: "The URL of the repository (for repository-backed profiles).",
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
			"public": schema.BoolAttribute{
				Description: "Whether the profile can be instantiated by any CloudLab user.",
				Computed:    true,
			},
			"project_writable": schema.BoolAttribute{
				Description: "Whether other members of the project can modify this profile.",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the data source.
func (d *profileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the profile data.
func (d *profileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state profileDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading CloudLab profile", map[string]any{"id": state.ID.ValueString()})

	profile, err := d.client.GetProfile(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Profile", err.Error())
		return
	}

	state.ID = types.StringValue(profile.ID)
	state.Name = types.StringValue(profile.Name)
	state.Project = types.StringValue(profile.Project)
	state.Creator = types.StringValue(profile.Creator)
	state.Version = types.Int64Value(profile.Version)
	state.CreatedAt = types.StringValue(profile.CreatedAt)
	state.Public = types.BoolValue(profile.Public)
	state.ProjectWritable = types.BoolValue(profile.ProjectWritable)

	if profile.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(*profile.UpdatedAt)
	} else {
		state.UpdatedAt = types.StringNull()
	}

	if profile.RepositoryURL != nil {
		state.RepositoryURL = types.StringValue(*profile.RepositoryURL)
	} else {
		state.RepositoryURL = types.StringNull()
	}

	if profile.RepositoryRefspec != nil {
		state.RepositoryRefspec = types.StringValue(*profile.RepositoryRefspec)
	} else {
		state.RepositoryRefspec = types.StringNull()
	}

	if profile.RepositoryHash != nil {
		state.RepositoryHash = types.StringValue(*profile.RepositoryHash)
	} else {
		state.RepositoryHash = types.StringNull()
	}

	if profile.RepositoryGithook != nil {
		state.RepositoryGithook = types.StringValue(*profile.RepositoryGithook)
	} else {
		state.RepositoryGithook = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
