package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

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

// ---------------------------------------------------------------------------
// RSpec XML parsing types
// ---------------------------------------------------------------------------

// rspecXML is the top-level RSpec XML structure.
type rspecXML struct {
	XMLName xml.Name    `xml:"rspec"`
	Nodes   []rspecNode `xml:"node"`
}

// rspecNode represents a node element in an RSpec.
type rspecNode struct {
	ClientID   string       `xml:"client_id,attr"`
	Host       rspecHost    `xml:"host"`
	Interfaces []rspecIface `xml:"interface"`
}

// rspecHost represents the <host> element with hostname and IPv4.
type rspecHost struct {
	Name string `xml:"name,attr"`
	IPv4 string `xml:"ipv4,attr"`
}

// rspecIface represents a network interface in an RSpec node.
type rspecIface struct {
	ClientID string    `xml:"client_id,attr"`
	IPs      []rspecIP `xml:"ip"`
}

// rspecIP represents an IP address on an interface.
type rspecIP struct {
	Address string `xml:"address,attr"`
	Type    string `xml:"type,attr"`
}

// parseRSpecNodes parses node information from a raw RSpec XML string.
func parseRSpecNodes(rspecXMLStr string) []manifestNodeModel {
	var rspec rspecXML
	if err := xml.NewDecoder(strings.NewReader(rspecXMLStr)).Decode(&rspec); err != nil {
		return nil
	}

	var nodes []manifestNodeModel
	for _, n := range rspec.Nodes {
		node := manifestNodeModel{
			ClientID: types.StringValue(n.ClientID),
			Hostname: types.StringValue(n.Host.Name),
		}

		// Add the host IPv4 as the primary interface if present
		if n.Host.IPv4 != "" {
			node.Interfaces = append(node.Interfaces, manifestNodeInterfaceModel{
				Name:    types.StringValue("eth0"),
				Address: types.StringValue(n.Host.IPv4),
			})
		}

		// Add any additional interface IPs
		for _, iface := range n.Interfaces {
			for _, ip := range iface.IPs {
				if ip.Type == "ipv4" || ip.Type == "" {
					name := iface.ClientID
					if name == "" {
						name = "iface"
					}
					node.Interfaces = append(node.Interfaces, manifestNodeInterfaceModel{
						Name:    types.StringValue(name),
						Address: types.StringValue(ip.Address),
					})
				}
			}
		}

		nodes = append(nodes, node)
	}
	return nodes
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
							Description: "The CloudLab aggregate (site) URN this manifest applies to.",
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
													Description: "The interface name.",
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

	rawManifests, err := d.client.GetRawManifests(ctx, state.ExperimentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Experiment Manifests", err.Error())
		return
	}

	var manifestModels []manifestEntryModel
	for urn, rspecXMLStr := range rawManifests {
		entry := manifestEntryModel{
			Aggregate: types.StringValue(urn),
			Nodes:     parseRSpecNodes(rspecXMLStr),
		}
		manifestModels = append(manifestModels, entry)
	}

	state.Manifests = manifestModels

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
