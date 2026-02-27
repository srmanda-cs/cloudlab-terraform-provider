package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure manifestDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &manifestDataSource{}

// NewManifestDataSource returns a new manifest data source.
func NewManifestDataSource() datasource.DataSource {
	return &manifestDataSource{}
}

// manifestDataSource queries the manifests (node details) of a running experiment.
type manifestDataSource struct {
	client *Client
}

// manifestNodeInterfaceModel maps a network interface on a node.
type manifestNodeInterfaceModel struct {
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
}

// manifestNodeModel maps a node in a manifest.
type manifestNodeModel struct {
	ClientID   types.String                 `tfsdk:"client_id"`
	Hostname   types.String                 `tfsdk:"hostname"`
	Interfaces []manifestNodeInterfaceModel `tfsdk:"interfaces"`
}

// manifestEntryModel maps a manifest entry for a single aggregate.
type manifestEntryModel struct {
	Aggregate types.String        `tfsdk:"aggregate"`
	Nodes     []manifestNodeModel `tfsdk:"nodes"`
}

// manifestDataSourceModel maps the data source schema data.
type manifestDataSourceModel struct {
	ExperimentID types.String         `tfsdk:"experiment_id"`
	Manifests    []manifestEntryModel `tfsdk:"manifests"`
}

// Metadata returns the data source type name.
func (d *manifestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest"
}

// Schema defines the schema for the data source.
func (d *manifestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves the manifests for a running CloudLab experiment. " +
			"The manifest contains the assigned node hostnames, IP addresses, and network interfaces " +
			"for all nodes in the experiment.",
		Attributes: map[string]schema.Attribute{
			"experiment_id": schema.StringAttribute{
				Description: "The UUID of the running experiment to retrieve manifests for.",
				Required:    true,
			},
			"manifests": schema.ListNestedAttribute{
				Description: "The list of manifests, one per CloudLab aggregate/site.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"aggregate": schema.StringAttribute{
							Description: "The CloudLab aggregate (site) this manifest applies to.",
							Computed:    true,
						},
						"nodes": schema.ListNestedAttribute{
							Description: "The list of nodes provisioned at this aggregate.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"client_id": schema.StringAttribute{
										Description: "The client-assigned node identifier from the profile.",
										Computed:    true,
									},
									"hostname": schema.StringAttribute{
										Description: "The fully qualified hostname of the node.",
										Computed:    true,
									},
									"interfaces": schema.ListNestedAttribute{
										Description: "The network interfaces on this node.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Description: "The interface name (e.g., eth0).",
													Computed:    true,
												},
												"address": schema.StringAttribute{
													Description: "The IP address assigned to this interface.",
													Computed:    true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure sets the provider-configured client on the data source.
func (d *manifestDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the manifest data for the experiment.
func (d *manifestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state manifestDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading CloudLab experiment manifests", map[string]any{
		"experiment_id": state.ExperimentID.ValueString(),
	})

	manifests, err := d.client.GetManifests(state.ExperimentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Experiment Manifests", err.Error())
		return
	}

	var manifestModels []manifestEntryModel
	for _, m := range manifests {
		entry := manifestEntryModel{
			Aggregate: types.StringValue(m.Aggregate),
		}

		for _, n := range m.Nodes {
			node := manifestNodeModel{
				ClientID: types.StringValue(n.ClientID),
				Hostname: types.StringValue(n.Hostname),
			}

			for _, iface := range n.Interfaces {
				node.Interfaces = append(node.Interfaces, manifestNodeInterfaceModel{
					Name:    types.StringValue(iface.Name),
					Address: types.StringValue(iface.Address),
				})
			}

			entry.Nodes = append(entry.Nodes, node)
		}

		manifestModels = append(manifestModels, entry)
	}

	state.Manifests = manifestModels

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
