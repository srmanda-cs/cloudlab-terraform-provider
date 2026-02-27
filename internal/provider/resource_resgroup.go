package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure resgroupResource satisfies the resource.Resource interface.
var _ resource.Resource = &resgroupResource{}

// NewResgroupResource returns a new reservation group resource.
func NewResgroupResource() resource.Resource {
	return &resgroupResource{}
}

// resgroupResource manages a CloudLab reservation group.
type resgroupResource struct {
	client *Client
}

// resgroupNodeTypeModel maps a node type reservation in a resgroup.
type resgroupNodeTypeModel struct {
	URN      types.String `tfsdk:"urn"`
	NodeType types.String `tfsdk:"node_type"`
	Count    types.Int64  `tfsdk:"count"`
}

// resgroupRangeModel maps a frequency range reservation in a resgroup.
type resgroupRangeModel struct {
	MinFreq types.Float64 `tfsdk:"min_freq"`
	MaxFreq types.Float64 `tfsdk:"max_freq"`
}

// resgroupRouteModel maps a named route reservation in a resgroup.
type resgroupRouteModel struct {
	Name types.String `tfsdk:"name"`
}

// resgroupResourceModel maps the resource schema data.
type resgroupResourceModel struct {
	ID          types.String            `tfsdk:"id"`
	Project     types.String            `tfsdk:"project"`
	Group       types.String            `tfsdk:"group"`
	Reason      types.String            `tfsdk:"reason"`
	StartAt     types.String            `tfsdk:"start_at"`
	ExpiresAt   types.String            `tfsdk:"expires_at"`
	Duration    types.Int64             `tfsdk:"duration"`
	PowderZones types.String            `tfsdk:"powder_zones"`
	NodeTypes   []resgroupNodeTypeModel `tfsdk:"node_types"`
	Ranges      []resgroupRangeModel    `tfsdk:"ranges"`
	Routes      []resgroupRouteModel    `tfsdk:"routes"`
	// Computed read-only fields
	Creator   types.String `tfsdk:"creator"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// Metadata returns the resource type name.
func (r *resgroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resgroup"
}

// Schema defines the schema for the resource.
func (r *resgroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a CloudLab reservation group. Reservation groups allow you to pre-reserve " +
			"specific hardware resources on CloudLab for a defined time window, ensuring availability " +
			"when you need to run experiments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier (UUID) of the reservation group assigned by CloudLab.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Description: "The CloudLab project for this reservation group.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group": schema.StringAttribute{
				Description: "The project subgroup for this reservation group.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"reason": schema.StringAttribute{
				Description: "A description of why you need to reserve these resources.",
				Required:    true,
			},
			"start_at": schema.StringAttribute{
				Description: "The time the reservation should start (RFC3339 format). " +
					"If omitted, the reservation starts immediately.",
				Optional: true,
				Computed: true,
			},
			"expires_at": schema.StringAttribute{
				Description: "The time the reservation expires (RFC3339 format). " +
					"Mutually exclusive with duration.",
				Optional: true,
				Computed: true,
			},
			"duration": schema.Int64Attribute{
				Description: "Duration of the reservation in hours, as an alternative to expires_at.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"powder_zones": schema.StringAttribute{
				Description: "Powder zone for radio reservations. One of: Outdoor, Indoor OTA Lab, Flux.",
				Optional:    true,
			},
			"node_types": schema.ListNestedAttribute{
				Description: "The list of node types and counts to reserve.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"urn": schema.StringAttribute{
							Description: "The aggregate URN for this reservation (e.g., urn:publicid:IDN+utah.cloudlab.us+authority+cm).",
							Required:    true,
						},
						"node_type": schema.StringAttribute{
							Description: "The hardware node type to reserve (e.g., xl170, m400).",
							Required:    true,
						},
						"count": schema.Int64Attribute{
							Description: "The number of nodes of this type to reserve.",
							Required:    true,
						},
					},
				},
			},
			"ranges": schema.ListNestedAttribute{
				Description: "Frequency range reservations (Powder/POWDER wireless testbed).",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"min_freq": schema.Float64Attribute{
							Description: "The start of the frequency range (inclusive) in MHz.",
							Required:    true,
						},
						"max_freq": schema.Float64Attribute{
							Description: "The end of the frequency range (inclusive) in MHz.",
							Required:    true,
						},
					},
				},
			},
			"routes": schema.ListNestedAttribute{
				Description: "Named route reservations.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The route name to reserve.",
							Required:    true,
						},
					},
				},
			},
			// Computed read-only attributes
			"creator": schema.StringAttribute{
				Description: "The CloudLab username who created the reservation group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the reservation group was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure sets the provider-configured client on the resource.
func (r *resgroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildResgroupCreateRequest builds a ResgroupCreateRequest from a model.
func buildResgroupCreateRequest(plan resgroupResourceModel) *ResgroupCreateRequest {
	createReq := &ResgroupCreateRequest{
		Project: plan.Project.ValueString(),
		Reason:  plan.Reason.ValueString(),
	}

	if !plan.Group.IsNull() && !plan.Group.IsUnknown() {
		createReq.Group = plan.Group.ValueString()
	}
	if !plan.StartAt.IsNull() && !plan.StartAt.IsUnknown() {
		v := plan.StartAt.ValueString()
		createReq.StartAt = &v
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		createReq.ExpiresAt = &v
	}
	if !plan.PowderZones.IsNull() && !plan.PowderZones.IsUnknown() {
		v := plan.PowderZones.ValueString()
		createReq.PowderZones = &v
	}

	if len(plan.NodeTypes) > 0 {
		nodeTypes := &ResgroupNodeTypes{}
		for _, n := range plan.NodeTypes {
			nodeTypes.NodeTypes = append(nodeTypes.NodeTypes, ResgroupNodeType{
				URN:      n.URN.ValueString(),
				NodeType: n.NodeType.ValueString(),
				Count:    n.Count.ValueInt64(),
			})
		}
		createReq.NodeTypes = nodeTypes
	}

	if len(plan.Ranges) > 0 {
		ranges := &ResgroupRanges{}
		for _, rng := range plan.Ranges {
			ranges.Ranges = append(ranges.Ranges, ResgroupRange{
				MinFreq: rng.MinFreq.ValueFloat64(),
				MaxFreq: rng.MaxFreq.ValueFloat64(),
			})
		}
		createReq.Ranges = ranges
	}

	if len(plan.Routes) > 0 {
		routes := &ResgroupRoutes{}
		for _, rt := range plan.Routes {
			routes.Routes = append(routes.Routes, ResgroupRoute{
				Name: rt.Name.ValueString(),
			})
		}
		createReq.Routes = routes
	}

	return createReq
}

// Create creates the reservation group and sets the initial Terraform state.
func (r *resgroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resgroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := buildResgroupCreateRequest(plan)

	var durationHours *int64
	if !plan.Duration.IsNull() && !plan.Duration.IsUnknown() {
		v := plan.Duration.ValueInt64()
		durationHours = &v
	}

	tflog.Info(ctx, "Creating CloudLab reservation group", map[string]any{
		"project": createReq.Project,
	})

	rg, err := r.client.CreateResgroup(ctx, createReq, durationHours)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Reservation Group", err.Error())
		return
	}

	plan = mapResgroupResponseToModel(rg, plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *resgroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resgroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rg, err := r.client.GetResgroup(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			tflog.Warn(ctx, "Reservation group not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Reservation Group", err.Error())
		return
	}

	state = mapResgroupResponseToModel(rg, state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update modifies mutable resgroup attributes via PUT.
func (r *resgroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resgroupResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resgroupID := state.ID.ValueString()
	modReq := buildResgroupCreateRequest(plan)

	var durationHours *int64
	if !plan.Duration.IsNull() && !plan.Duration.IsUnknown() {
		v := plan.Duration.ValueInt64()
		durationHours = &v
	}

	tflog.Info(ctx, "Modifying CloudLab reservation group", map[string]any{"id": resgroupID})
	rg, err := r.client.ModifyResgroup(ctx, resgroupID, modReq, durationHours)
	if err != nil {
		resp.Diagnostics.AddError("Error Modifying Reservation Group", err.Error())
		return
	}

	plan = mapResgroupResponseToModel(rg, plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the reservation group.
func (r *resgroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resgroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting CloudLab reservation group", map[string]any{"id": state.ID.ValueString()})

	if err := r.client.DeleteResgroup(ctx, state.ID.ValueString()); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Reservation Group", err.Error())
		return
	}
}

// mapResgroupResponseToModel maps an API response to the Terraform model.
func mapResgroupResponseToModel(rg *ResgroupResponse, model resgroupResourceModel) resgroupResourceModel {
	model.ID = types.StringValue(rg.ID)
	model.Project = types.StringValue(rg.Project)
	model.Reason = types.StringValue(rg.Reason)
	model.Creator = types.StringValue(rg.Creator)

	if rg.Group != "" {
		model.Group = types.StringValue(rg.Group)
	} else if model.Group.IsUnknown() {
		model.Group = types.StringNull()
	}

	if rg.CreatedAt != nil {
		model.CreatedAt = types.StringValue(*rg.CreatedAt)
	} else {
		model.CreatedAt = types.StringNull()
	}

	if rg.StartAt != nil {
		model.StartAt = types.StringValue(*rg.StartAt)
	} else {
		model.StartAt = types.StringNull()
	}

	if rg.ExpiresAt != nil {
		model.ExpiresAt = types.StringValue(*rg.ExpiresAt)
	} else {
		model.ExpiresAt = types.StringNull()
	}

	if rg.PowderZones != nil {
		model.PowderZones = types.StringValue(*rg.PowderZones)
	} else if model.PowderZones.IsUnknown() {
		model.PowderZones = types.StringNull()
	}

	// Map node types from response
	if rg.NodeTypes != nil && len(rg.NodeTypes.NodeTypes) > 0 {
		var nodeTypeModels []resgroupNodeTypeModel
		for _, nt := range rg.NodeTypes.NodeTypes {
			nodeTypeModels = append(nodeTypeModels, resgroupNodeTypeModel{
				URN:      types.StringValue(nt.URN),
				NodeType: types.StringValue(nt.NodeType),
				Count:    types.Int64Value(nt.Count),
			})
		}
		model.NodeTypes = nodeTypeModels
	}

	// Map ranges from response
	if rg.Ranges != nil && len(rg.Ranges.Ranges) > 0 {
		var rangeModels []resgroupRangeModel
		for _, rng := range rg.Ranges.Ranges {
			rangeModels = append(rangeModels, resgroupRangeModel{
				MinFreq: types.Float64Value(rng.MinFreq),
				MaxFreq: types.Float64Value(rng.MaxFreq),
			})
		}
		model.Ranges = rangeModels
	}

	// Map routes from response
	if rg.Routes != nil && len(rg.Routes.Routes) > 0 {
		var routeModels []resgroupRouteModel
		for _, rt := range rg.Routes.Routes {
			routeModels = append(routeModels, resgroupRouteModel{
				Name: types.StringValue(rt.Name),
			})
		}
		model.Routes = routeModels
	}

	return model
}
