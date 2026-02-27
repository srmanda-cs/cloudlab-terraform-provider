package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure nodeDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &nodeDataSource{}

// NewNodeDataSource returns a new node data source.
func NewNodeDataSource() datasource.DataSource {
	return &nodeDataSource{}
}

// nodeDataSource queries a specific node in a CloudLab experiment.
type nodeDataSource struct {
	client *Client
}

// nodeDataSourceModel maps the data source schema data.
type nodeDataSourceModel struct {
	ExperimentID  types.String `tfsdk:"experiment_id"`
	ClientID      types.String `tfsdk:"client_id"`
	URN           types.String `tfsdk:"urn"`
	Hostname      types.String `tfsdk:"hostname"`
	IPv4          types.String `tfsdk:"ipv4"`
	Status        types.String `tfsdk:"status"`
	State         types.String `tfsdk:"state"`
	RawState      types.String `tfsdk:"rawstate"`
	StartupStatus types.String `tfsdk:"startup_status"`
}

// Metadata returns the data source type name.
func (d *nodeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

// Schema defines the schema for the data source.
func (d *nodeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries a specific node in a running CloudLab experiment. " +
			"Returns detailed node status including hostname, IP address, and operational state.",
		Attributes: map[string]schema.Attribute{
			"experiment_id": schema.StringAttribute{
				Description: "The UUID of the running experiment.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The logical name (client ID) of the node within the experiment.",
				Required:    true,
			},
			"urn": schema.StringAttribute{
				Description: "The URN of the node.",
				Computed:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "The fully qualified hostname of the node.",
				Computed:    true,
			},
			"ipv4": schema.StringAttribute{
				Description: "The IPv4 address of the node.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the node.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the node.",
				Computed:    true,
			},
			"rawstate": schema.StringAttribute{
				Description: "The current raw state of the node.",
				Computed:    true,
			},
			"startup_status": schema.StringAttribute{
				Description: "The current status of the startup execution service.",
				Computed:    true,
			},
		},
	}
}

// Configure sets the provider-configured client on the data source.
func (d *nodeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the node data.
func (d *nodeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state nodeDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading CloudLab experiment node", map[string]any{
		"experiment_id": state.ExperimentID.ValueString(),
		"client_id":     state.ClientID.ValueString(),
	})

	node, err := d.client.GetExperimentNode(ctx, state.ExperimentID.ValueString(), state.ClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Experiment Node", err.Error())
		return
	}

	state.URN = types.StringValue(node.URN)
	state.ClientID = types.StringValue(node.ClientID)
	state.Hostname = types.StringValue(node.Hostname)
	state.IPv4 = types.StringValue(node.IPv4)
	state.Status = types.StringValue(node.Status)
	state.State = types.StringValue(node.State)
	state.RawState = types.StringValue(node.RawState)
	state.StartupStatus = types.StringValue(node.StartupStatus)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
