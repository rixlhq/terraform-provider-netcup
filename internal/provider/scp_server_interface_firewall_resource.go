package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var (
	_ resource.Resource                = &ScpServerInterfaceFirewallResource{}
	_ resource.ResourceWithImportState = &ScpServerInterfaceFirewallResource{}
)

// ScpServerInterfaceFirewallResource manages firewall rules for a server network
// interface identified by server ID and MAC address.
type ScpServerInterfaceFirewallResource struct {
	client *scpclient.Client
}

func NewScpServerInterfaceFirewallResource() resource.Resource {
	return &ScpServerInterfaceFirewallResource{}
}

func (r *ScpServerInterfaceFirewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_interface_firewall"
}

func (r *ScpServerInterfaceFirewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the firewall configuration for a server network interface.",
		Attributes:  scpServerInterfaceFirewallSchemaAttributes,
	}
}

func (r *ScpServerInterfaceFirewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScpServerInterfaceFirewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data scpServerInterfaceFirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.saveFirewall(ctx, data); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	readData, err := r.readFirewall(ctx, data.ServerID.ValueInt64(), data.MAC.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	data = *readData
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerInterfaceFirewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data scpServerInterfaceFirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, err := r.readFirewall(ctx, data.ServerID.ValueInt64(), data.MAC.ValueString())
	if err != nil {
		if scpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	data = *readData
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerInterfaceFirewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data scpServerInterfaceFirewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.saveFirewall(ctx, data); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	readData, err := r.readFirewall(ctx, data.ServerID.ValueInt64(), data.MAC.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	data = *readData
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerInterfaceFirewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data scpServerInterfaceFirewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	empty := scpServerInterfaceFirewallResourceModel{
		ServerID: data.ServerID,
		MAC:      data.MAC,
		Active:   types.BoolValue(false),
	}

	if err := r.saveFirewall(ctx, empty); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
	}
}

func (r *ScpServerInterfaceFirewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitImportID(req.ID, 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "expected 'server_id/mac'")
		return
	}

	serverID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "server_id must be an integer")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), parts[1])...)
}

func (r *ScpServerInterfaceFirewallResource) saveFirewall(ctx context.Context, data scpServerInterfaceFirewallResourceModel) error {
	if r.client == nil {
		return errClientNotConfigured
	}

	active := true
	if !data.Active.IsNull() && !data.Active.IsUnknown() {
		active = data.Active.ValueBool()
	}

	copied, diags := intListToInt64s(ctx, data.CopiedPolicyIDs)
	if diags.HasError() {
		return fmt.Errorf("copied_policy_ids: %s", diags.Errors()[0].Summary())
	}
	user, diags := intListToInt64s(ctx, data.UserPolicyIDs)
	if diags.HasError() {
		return fmt.Errorf("user_policy_ids: %s", diags.Errors()[0].Summary())
	}

	req := firewallSaveRequest{
		Active:         active,
		CopiedPolicies: idsToIdentifiers(copied),
		UserPolicies:   idsToIdentifiers(user),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces/%s/firewall", data.ServerID.ValueInt64(), data.MAC.ValueString())
	_, err = r.client.Put(ctx, path, body)
	return err
}

func (r *ScpServerInterfaceFirewallResource) readFirewall(ctx context.Context, serverID int64, mac string) (*scpServerInterfaceFirewallResourceModel, error) {
	if r.client == nil {
		return nil, errClientNotConfigured
	}

	path := fmt.Sprintf("/api/v1/servers/%d/interfaces/%s/firewall", serverID, mac)
	body, err := r.client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var fw firewallReadResponse
	if err := json.Unmarshal(body, &fw); err != nil {
		return nil, err
	}

	copied, diags := types.ListValueFrom(ctx, types.Int64Type, policyIDs(fw.CopiedPolicies))
	if diags.HasError() {
		return nil, fmt.Errorf("copied_policy_ids: %s", diags.Errors()[0].Summary())
	}
	user, diags := types.ListValueFrom(ctx, types.Int64Type, policyIDs(fw.UserPolicies))
	if diags.HasError() {
		return nil, fmt.Errorf("user_policy_ids: %s", diags.Errors()[0].Summary())
	}

	return &scpServerInterfaceFirewallResourceModel{
		ServerID:            types.Int64Value(serverID),
		MAC:                 types.StringValue(mac),
		ID:                  types.StringValue(fmt.Sprintf("%d/%s", serverID, mac)),
		Active:              types.BoolValue(fw.Active),
		Consistent:          types.BoolValue(fw.Consistent),
		CopiedPolicyIDs:     copied,
		UserPolicyIDs:       user,
		IngressImplicitRule: types.StringValue(fw.IngressImplicitRule),
		EgressImplicitRule:  types.StringValue(fw.EgressImplicitRule),
	}, nil
}
