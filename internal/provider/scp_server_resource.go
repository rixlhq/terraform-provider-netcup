package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
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
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages mutable attributes of an existing netcup SCP virtual server. Servers cannot be created or deleted through the SCP API; this resource adopts an existing server by `server_id` and applies patches.",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Description: "Id of the server to manage.",
			},
			"hostname": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Server hostname.",
			},
			"nickname": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "User-defined nickname for the server.",
			},
			"autostart": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the server should start automatically.",
			},
			"uefi": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether UEFI is enabled.",
			},
			"bootorder": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Boot order as a list of values such as HDD, CDROM, NETWORK.",
			},
			"os_optimization": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OS optimization (LINUX, WINDOWS, BSD, LINUX_LEGACY, UNKNOWN).",
			},
			"keyboard_layout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Keyboard layout used by the virtual console.",
			},
			"cpu_topology": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"socket_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Number of CPU sockets.",
					},
					"cores_per_socket_count": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Number of CPU cores per socket.",
					},
				},
				Description: "CPU topology configuration.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Server name as returned by the SCP API.",
			},
			"disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the server is disabled.",
			},
			"rescue_system_active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the rescue system is active.",
			},
			"snapshot_allowed": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether snapshots are allowed for this server.",
			},
		},
	}
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

	schema := r.schema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	return scpcommon.JSONToTfValue(ctx, tfType, flat)
}

func (r *ScpServerResource) schema(ctx context.Context) schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func flattenServerJSON(serverID int64, v interface{}) (map[string]interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected server object, got %T", v)
	}

	flat := make(map[string]interface{}, len(m))
	for k, val := range m {
		flat[scpcommon.TerraformIdentifier(k)] = val
	}
	flat["server_id"] = json.Number(strconv.FormatInt(serverID, 10))

	if live, ok := flat["server_live_info"].(map[string]interface{}); ok {
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
	patches, err := r.buildPatches(plan, state)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/servers/%d", serverID)
	for _, body := range patches {
		if _, err := r.client.Patch(ctx, path, body); err != nil {
			return err
		}
	}
	return nil
}

func (r *ScpServerResource) buildPatches(plan, state ScpServerResourceModel) ([][]byte, error) {
	var patches [][]byte

	if stringChanged(plan.Hostname, state.Hostname) {
		body, err := json.Marshal(map[string]interface{}{"hostname": plan.Hostname.ValueString()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if stringChanged(plan.Nickname, state.Nickname) {
		body, err := json.Marshal(map[string]interface{}{"nickname": plan.Nickname.ValueString()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if boolChanged(plan.Autostart, state.Autostart) {
		body, err := json.Marshal(map[string]interface{}{"autostart": plan.Autostart.ValueBool()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if boolChanged(plan.Uefi, state.Uefi) {
		body, err := json.Marshal(map[string]interface{}{"uefi": plan.Uefi.ValueBool()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if stringListChanged(plan.Bootorder, state.Bootorder) {
		order, err := stringListValue(plan.Bootorder)
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(map[string]interface{}{"bootorder": order})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if stringChanged(plan.OsOptimization, state.OsOptimization) {
		body, err := json.Marshal(map[string]interface{}{"os_optimization": plan.OsOptimization.ValueString()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if stringChanged(plan.KeyboardLayout, state.KeyboardLayout) {
		body, err := json.Marshal(map[string]interface{}{"keyboardLayout": plan.KeyboardLayout.ValueString()})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	if objectChanged(plan.CpuTopology, state.CpuTopology) {
		topo, err := cpuTopologyValue(plan.CpuTopology)
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(map[string]interface{}{"cpuTopology": topo})
		if err != nil {
			return nil, err
		}
		patches = append(patches, body)
	}

	return patches, nil
}

func stringChanged(a, b types.String) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() && a.ValueString() == b.ValueString() {
		return false
	}
	return true
}

func boolChanged(a, b types.Bool) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() && a.ValueBool() == b.ValueBool() {
		return false
	}
	return true
}

func stringListChanged(a, b types.List) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	la, err := stringListValue(a)
	if err != nil {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() {
		lb, err := stringListValue(b)
		if err == nil && stringSlicesEqual(la, lb) {
			return false
		}
	}
	return true
}

func stringListValue(l types.List) ([]string, error) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	elems := l.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected string element, got %T", e)
		}
		out = append(out, s.ValueString())
	}
	return out, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func objectChanged(a, b types.Object) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() {
		// Compare the JSON representation of the values.
		av, err := objectToMap(a)
		if err != nil {
			return true
		}
		bv, err := objectToMap(b)
		if err == nil && mapsEqual(av, bv) {
			return false
		}
	}
	return true
}

func cpuTopologyValue(o types.Object) (map[string]interface{}, error) {
	m, err := objectToMap(o)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{})
	if v, ok := m["socket_count"]; ok {
		out["socketCount"] = v
	}
	if v, ok := m["cores_per_socket_count"]; ok {
		out["coresPerSocketCount"] = v
	}
	return out, nil
}

func objectToMap(o types.Object) (map[string]interface{}, error) {
	if o.IsNull() || o.IsUnknown() {
		return nil, nil
	}
	attrs := o.Attributes()
	out := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case types.Int64:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueInt64()
			}
		case types.String:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueString()
			}
		case types.Bool:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueBool()
			}
		case types.Number:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueBigFloat()
			}
		default:
			return nil, fmt.Errorf("unsupported nested attribute type %T for %s", v, k)
		}
	}
	return out, nil
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		switch ta := va.(type) {
		case int64:
			tb, ok := vb.(int64)
			if !ok || ta != tb {
				return false
			}
		case *big.Float:
			tb, ok := vb.(*big.Float)
			if !ok || ta.Cmp(tb) != 0 {
				return false
			}
		case string:
			tb, ok := vb.(string)
			if !ok || ta != tb {
				return false
			}
		case bool:
			tb, ok := vb.(bool)
			if !ok || ta != tb {
				return false
			}
		default:
			return false
		}
	}
	return true
}
