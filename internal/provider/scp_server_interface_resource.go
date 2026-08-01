package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &ScpServerInterfaceResource{}

type ScpServerInterfaceResource struct {
	client *scpclient.Client
}

func NewScpServerInterfaceResource() resource.Resource {
	return &ScpServerInterfaceResource{}
}

func (r *ScpServerInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_interface"
}

func (r *ScpServerInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a network interface for a netcup SCP server. The interface is created with a VLAN and network driver; the MAC address is returned by the SCP task result and used as the Terraform id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Terraform identifier for this network interface (the MAC address).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the server to attach the interface to.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			attrVlanID: schema.Int64Attribute{
				MarkdownDescription: "ID of the VLAN for this interface.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_driver": schema.StringAttribute{
				MarkdownDescription: "Network driver for this interface (e.g. virtio, e1000, rtl8139).",
				Required:            true,
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "MAC address assigned to this interface by the SCP API.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"speed_in_mbits": schema.Int64Attribute{
				MarkdownDescription: "Interface speed in MBit/s reported by the SCP API.",
				Computed:            true,
			},
		},
	}
}

type scpServerInterfaceResourceModel struct {
	ServerID      types.Int64  `tfsdk:"server_id"`
	VlanId        types.Int64  `tfsdk:"vlan_id"`
	NetworkDriver types.String `tfsdk:"network_driver"`
	Mac           types.String `tfsdk:"mac"`
	SpeedInMbits  types.Int64  `tfsdk:"speed_in_mbits"`
	ID            types.String `tfsdk:"id"`
}

type serverInterfaceResult struct {
	Mac              string `json:"mac"`
	Driver           string `json:"driver"`
	Mtu              int64  `json:"mtu"`
	SpeedInMBits     int64  `json:"speedInMBits"`
	TrafficThrottled bool   `json:"trafficThrottled"`
	VlanInterface    bool   `json:"vlanInterface"`
	VlanId           int64  `json:"vlanId"`
}

func (r *ScpServerInterfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.SCP
}

func (r *ScpServerInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	var plan scpServerInterfaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createBody := map[string]any{
		"vlanId":        plan.VlanId.ValueInt64(),
		"networkDriver": plan.NetworkDriver.ValueString(),
	}
	body, err := json.Marshal(createBody)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces", plan.ServerID.ValueInt64())
	result, err := r.client.Post(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}
	if len(result) == 0 {
		resp.Diagnostics.AddError("SCP API Error", "create network interface did not return a task result with the generated MAC")
		return
	}

	var iface serverInterfaceResult
	if err := json.Unmarshal(result, &iface); err != nil {
		resp.Diagnostics.AddError("SCP API Decode Error", err.Error())
		return
	}
	if iface.Mac == "" {
		resp.Diagnostics.AddError("SCP API Error", "create network interface task result did not contain a MAC address")
		return
	}

	plan.Mac = types.StringValue(iface.Mac)
	plan.ID = types.StringValue(iface.Mac)
	if iface.SpeedInMBits != 0 {
		plan.SpeedInMbits = types.Int64Value(iface.SpeedInMBits)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScpServerInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	var state scpServerInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces/%s", state.ServerID.ValueInt64(), state.Mac.ValueString())
	body, err := r.client.Get(ctx, path, nil)
	if err != nil {
		if scpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	var iface serverInterfaceResult
	if err := json.Unmarshal(body, &iface); err != nil {
		resp.Diagnostics.AddError("SCP API Decode Error", err.Error())
		return
	}

	state.Mac = types.StringValue(iface.Mac)
	if iface.SpeedInMBits != 0 {
		state.SpeedInMbits = types.Int64Value(iface.SpeedInMBits)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScpServerInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	var plan, state scpServerInterfaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateBody := map[string]any{
		"driver": plan.NetworkDriver.ValueString(),
	}
	body, err := json.Marshal(updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces/%s", state.ServerID.ValueInt64(), state.Mac.ValueString())
	if _, err := r.client.Patch(ctx, path, body); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	readBody, err := r.client.Get(ctx, path, nil)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	var iface serverInterfaceResult
	if err := json.Unmarshal(readBody, &iface); err != nil {
		resp.Diagnostics.AddError("SCP API Decode Error", err.Error())
		return
	}

	state.NetworkDriver = plan.NetworkDriver
	state.Mac = types.StringValue(iface.Mac)
	if iface.SpeedInMBits != 0 {
		state.SpeedInMbits = types.Int64Value(iface.SpeedInMBits)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScpServerInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	var state scpServerInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces/%s", state.ServerID.ValueInt64(), state.Mac.ValueString())
	if _, err := r.client.Delete(ctx, path); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}
}
