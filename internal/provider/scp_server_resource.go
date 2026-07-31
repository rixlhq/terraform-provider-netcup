package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &ScpServerResource{}

// NewScpServerResource returns a resource that manages mutable attributes of an
// existing netcup SCP server. The server itself cannot be created or deleted
// through the SCP API; this resource adopts an existing server by id and
// applies patches for hostname, nickname, autostart, uefi, bootorder, etc.
func NewScpServerResource() resource.Resource {
	return &ScpServerResource{}
}

type ScpServerResource struct {
	client *scpclient.Client
}

type ScpServerResourceModel struct {
	ServerId           types.Int64  `tfsdk:"server_id"`
	Hostname           types.String `tfsdk:"hostname"`
	Nickname           types.String `tfsdk:"nickname"`
	Autostart          types.Bool   `tfsdk:"autostart"`
	Uefi               types.Bool   `tfsdk:"uefi"`
	Bootorder          types.List   `tfsdk:"bootorder"`
	OsOptimization     types.String `tfsdk:"os_optimization"`
	KeyboardLayout     types.String `tfsdk:"keyboard_layout"`
	CpuTopology        types.Object `tfsdk:"cpu_topology"`
	Name               types.String `tfsdk:"name"`
	Disabled           types.Bool   `tfsdk:"disabled"`
	RescueSystemActive types.Bool   `tfsdk:"rescue_system_active"`
	SnapshotAllowed    types.Bool   `tfsdk:"snapshot_allowed"`
}

func (r *ScpServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server"
}

func (r *ScpServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = scpServerResourceSchema
}

func (r *ScpServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.SCP
}

func (r *ScpServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScpServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token to use this resource.")
		return
	}

	if err := r.applyPatches(ctx, plan, ScpServerResourceModel{}, plan.ServerId.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	tfVal, err := r.readServer(ctx, plan.ServerId.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", err.Error())
		return
	}
	resp.State.Raw = tfVal
}

func (r *ScpServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScpServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfVal, err := r.readServer(ctx, state.ServerId.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", err.Error())
		return
	}
	resp.State.Raw = tfVal
}

func (r *ScpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ScpServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyPatches(ctx, plan, state, plan.ServerId.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	tfVal, err := r.readServer(ctx, plan.ServerId.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", err.Error())
		return
	}
	resp.State.Raw = tfVal
}

func (r *ScpServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Servers cannot be deleted through the SCP API. Removing the resource from
	// state is sufficient.
}

func (r *ScpServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("server_id must be an integer: %s", err))
		return
	}

	tfVal, err := r.readServer(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", err.Error())
		return
	}
	resp.State.Raw = tfVal
}

func (r *ScpServerResource) readServer(ctx context.Context, serverID int64) (tftypes.Value, error) {
	path := fmt.Sprintf("/api/v1/servers/%d", serverID)
	body, err := r.client.Get(ctx, path, nil)
	if err != nil {
		return tftypes.Value{}, err
	}

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		return tftypes.Value{}, err
	}

	flat, err := flattenServerJSON(serverID, jsonVal)
	if err != nil {
		return tftypes.Value{}, err
	}

	schema := serverResourceSchema(ctx, r)
	tfType := schema.Type().TerraformType(ctx)
	return scpcommon.JSONToTfValue(ctx, tfType, flat)
}

func serverResourceSchema(ctx context.Context, r *ScpServerResource) schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func flattenServerJSON(serverID int64, v any) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected server object, got %T", v)
	}

	flat := make(map[string]any, len(m))
	for k, val := range m {
		flat[scpcommon.TerraformIdentifier(k)] = val
	}
	flat[attrServerID] = json.Number(strconv.FormatInt(serverID, 10))

	if live, ok := flat["server_live_info"].(map[string]any); ok {
		for k, val := range live {
			key := scpcommon.TerraformIdentifier(k)
			if _, exists := flat[key]; !exists {
				flat[key] = val
			}
		}
	}

	return flat, nil
}

func (r *ScpServerResource) applyPatches(ctx context.Context, plan, state ScpServerResourceModel, serverID int64) error {
	patches := buildServerPatches(plan, state)

	path := fmt.Sprintf("/api/v1/servers/%d", serverID)
	for _, body := range patches {
		if _, err := r.client.Patch(ctx, path, body); err != nil {
			return err
		}
	}
	return nil
}
