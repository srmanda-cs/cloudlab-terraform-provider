package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure resgroupDataSource satisfies the datasource.DataSource interface.
var _ datasource.DataSource = &resgroupDataSource{}

// NewResgroupDataSource returns a new resgroup data source.
func NewResgroupDataSource() datasource.DataSource {
	return &resgroupDataSource{}
}

// resgroupDataSource queries an existing CloudLab reservation group.
type resgroupDataSource struct {
	client *Client
}

// resgroupNodeTypeDSModel maps a node type reservation for a data source.
type resgroupNodeTypeDSModel struct {
	URN      types.String `tfsdk:"urn"`
	NodeType types.String `tfsdk:"node_type"`
	Count    types.Int64  `tfsdk:"count"`
}

// resgroupRangeDSModel maps a frequency range reservation for a data source.
type resgroupRangeDSModel struct {
	MinFreq types.Float64 `tfsdk:"min_freq"`
	MaxFreq types.Float64 `tfsdk:"max_freq"`
}

// resgroupRouteDSModel maps a route reservation for a data source.
type resgroupRouteDSModel struct {
	Name types.String `tfsdk:"name"`
}

// resgroupDataSourceModel maps the data source schema data.
type resgroupDataSourceModel struct {
	ID          types.String              `tfsdk:"id"`
	Project     types.String              `tfsdk:"project"`
	Group       types.String              `tfsdk:"group"`
	Reason      types.String              `tfsdk:"reason"`
	Creator     types.String              `tfsdk:"creator"`
	CreatedAt   types.String              `tfsdk:"created_at"`
	StartAt     types.String              `tfsdk:"start_at"`
	ExpiresAt   types.String              `tfsdk:"expires_at"`
	PowderZones types.String              `tfsdk:"powder_zones"`
	NodeTypes   []resgroupNodeTypeDSModel `tfsdk:"node_types"`
	Ranges      []resgroupRangeDSModel    `tfsdk:"ranges"`
	Routes      []resgroupRouteDSModel    `tfsdk:"routes"`
}

// Metadata returns the data source type name.
func (d *resgroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resgroup"
}

// Schema defines the schema for the data source.
func (d *resgroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries an existing CloudLab reservation group by its UUID. " +
			"Use this data source to reference reservation groups that were created outside of Terraform " +
			"or in a separate Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the reservation group to look up.",
				Required:    true,
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project this reservation group belongs to.",
				Computed:    true,
			},
			"group": schema.StringAttribute{
				Description: "The project subgroup this reservation group belongs to.",
				Computed:    true,
			},
			"reason": schema.StringAttribute{
				Description: "The reason the reservation was created.",
				Computed:    true,
			},
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the reservation group.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the reservation group was created.",
				Computed:    true,
			},
			"start_at": schema.StringAttribute{
				Description: "The time the reservation starts.",
				Computed:    true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The time the reservation expires.",
				Computed:    true,
			},
			"powder_zones": schema.StringAttribute{
				Description: "The Powder zone for radio reservations.",
				Computed:    true,
			},
			"node_types": schema.ListNestedAttribute{
				Description: "The list of node type reservations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"urn": schema.StringAttribute{
							Description: "The aggregate URN of the reservation.",
							Computed:    true,
						},
						"node_type": schema.StringAttribute{
							Description: "The hardware node type reserved.",
							Computed:    true,
						},
						"count": schema.Int64Attribute{
							Description: "The number of nodes reserved.",
							Computed:    true,
						},
					},
				},
			},
			"ranges": schema.ListNestedAttribute{
				Description: "The list of frequency range reservations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"min_freq": schema.Float64Attribute{
							Description: "The start of the frequency range (inclusive) in MHz.",
							Computed:    true,
						},
						"max_freq": schema.Float64Attribute{
							Description: "The end of the frequency range (inclusive) in MHz.",
							Computed:    true,
						},
					},
				},
			},
			"routes": schema.ListNestedAttribute{
				Description: "The list of named route reservations.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The route name reserved.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure sets the provider-configured client on the data source.
func (d *resgroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the resgroup data.
func (d *resgroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state resgroupDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading CloudLab reservation group", map[string]any{"id": state.ID.ValueString()})

	rg, err := d.client.GetResgroup(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Reservation Group", err.Error())
		return
	}

	state.ID = types.StringValue(rg.ID)
	state.Project = types.StringValue(rg.Project)
	state.Reason = types.StringValue(rg.Reason)
	state.Creator = types.StringValue(rg.Creator)

	if rg.Group != "" {
		state.Group = types.StringValue(rg.Group)
	} else {
		state.Group = types.StringNull()
	}

	if rg.CreatedAt != nil {
		state.CreatedAt = types.StringValue(*rg.CreatedAt)
	} else {
		state.CreatedAt = types.StringNull()
	}

	if rg.StartAt != nil {
		state.StartAt = types.StringValue(*rg.StartAt)
	} else {
		state.StartAt = types.StringNull()
	}

	if rg.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*rg.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}

	if rg.PowderZones != nil {
		state.PowderZones = types.StringValue(*rg.PowderZones)
	} else {
		state.PowderZones = types.StringNull()
	}

	if rg.NodeTypes != nil && len(rg.NodeTypes.NodeTypes) > 0 {
		var nodeTypeModels []resgroupNodeTypeDSModel
		for _, nt := range rg.NodeTypes.NodeTypes {
			nodeTypeModels = append(nodeTypeModels, resgroupNodeTypeDSModel{
				URN:      types.StringValue(nt.URN),
				NodeType: types.StringValue(nt.NodeType),
				Count:    types.Int64Value(nt.Count),
			})
		}
		state.NodeTypes = nodeTypeModels
	}

	if rg.Ranges != nil && len(rg.Ranges.Ranges) > 0 {
		var rangeModels []resgroupRangeDSModel
		for _, rng := range rg.Ranges.Ranges {
			rangeModels = append(rangeModels, resgroupRangeDSModel{
				MinFreq: types.Float64Value(rng.MinFreq),
				MaxFreq: types.Float64Value(rng.MaxFreq),
			})
		}
		state.Ranges = rangeModels
	}

	if rg.Routes != nil && len(rg.Routes.Routes) > 0 {
		var routeModels []resgroupRouteDSModel
		for _, rt := range rg.Routes.Routes {
			routeModels = append(routeModels, resgroupRouteDSModel{
				Name: types.StringValue(rt.Name),
			})
		}
		state.Routes = routeModels
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
