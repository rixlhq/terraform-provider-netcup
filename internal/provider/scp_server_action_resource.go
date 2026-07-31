package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &ScpServerActionResource{}

// NewScpServerActionResource returns a resource that triggers one-off actions
// against an existing netcup SCP server (start, stop, reset, rescue, snapshots,
// ISO attach, image setup, firewall reapply, etc.).
func NewScpServerActionResource() resource.Resource {
	return &ScpServerActionResource{}
}

type ScpServerActionResource struct {
	client *scpclient.Client
}

type ScpServerActionResourceModel struct {
	ServerId  types.Int64  `tfsdk:"server_id"`
	Action    types.String `tfsdk:"action"`
	Arguments types.Map    `tfsdk:"arguments"`
	Body      types.String `tfsdk:"body"`
	Triggers  types.Map    `tfsdk:"triggers"`
	Id        types.String `tfsdk:"id"`
}

type actionSpec struct {
	method       string
	path         string
	bodyBuilder  func(args map[string]string, body string) ([]byte, error)
	queryBuilder func(args map[string]string) url.Values
}

func (r *ScpServerActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_action"
}

func (r *ScpServerActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Description: "Id of the server to run the action on.",
			},
			"action": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Action to execute. Supported: start, stop, reset, powercycle, suspend, rescue_activate, rescue_deactivate, iso_attach, iso_detach, snapshot_create, snapshot_revert, snapshot_export, snapshot_delete, snapshot_dryrun, disk_format, image_setup, user_image_setup, storage_optimize, firewall_reapply, firewall_restore.",
			},
			"arguments": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "String arguments used to build the request path or query, e.g. snapshot_name, disk_name, mac, state_option.",
			},
			"body": schema.StringAttribute{
				Optional:    true,
				Description: "JSON request body for actions that require one (snapshot_create, image_setup, iso_attach, etc.).",
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Description: "Map of values that, when changed, force the action to run again.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier for this action invocation.",
			},
		},
	}
}

func (r *ScpServerActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScpServerActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScpServerActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.execute(ctx, &data); err != nil {
		resp.Diagnostics.AddError("SCP Action Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScpServerActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScpServerActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.execute(ctx, &data); err != nil {
		resp.Diagnostics.AddError("SCP Action Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpServerActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Action resources do not have a corresponding delete API call.
}

func (r *ScpServerActionResource) execute(ctx context.Context, data *ScpServerActionResourceModel) error {
	if r.client == nil {
		return fmt.Errorf("configure the provider with scp_access_token to use this resource")
	}

	args := mapStringMap(data.Arguments)
	serverID := data.ServerId.ValueInt64()

	spec, ok := actionSpecs[data.Action.ValueString()]
	if !ok {
		return fmt.Errorf("unsupported action %q", data.Action.ValueString())
	}

	path := fmt.Sprintf(spec.path, serverID)
	if argName, required := requiredArgForAction(data.Action.ValueString()); required {
		if _, ok := args[argName]; !ok {
			return fmt.Errorf("action %q requires argument %q", data.Action.ValueString(), argName)
		}
		path = fmt.Sprintf(path, args[argName])
	}

	body, err := spec.bodyBuilder(args, data.Body.ValueString())
	if err != nil {
		return err
	}

	query := spec.queryBuilder(args)
	if query != nil {
		path = path + "?" + query.Encode()
	}

	var response []byte
	switch spec.method {
	case "POST":
		response, err = r.client.Post(ctx, path, body)
	case "PATCH":
		response, err = r.client.Patch(ctx, path, body)
	case "PUT":
		response, err = r.client.Put(ctx, path, body)
	case "DELETE":
		response, err = r.client.Delete(ctx, path)
	default:
		err = fmt.Errorf("unsupported HTTP method %q", spec.method)
	}
	if err != nil {
		return err
	}
	_ = response

	data.Id = types.StringValue(fmt.Sprintf("%d/%s/%d", serverID, data.Action.ValueString(), time.Now().Unix()))
	return nil
}

func requiredArgForAction(action string) (string, bool) {
	switch action {
	case "snapshot_revert", "snapshot_export", "snapshot_delete":
		return "snapshot_name", true
	case "disk_format":
		return "disk_name", true
	case "firewall_reapply", "firewall_restore":
		return "mac", true
	}
	return "", false
}

func mapStringMap(m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := make(map[string]string, len(m.Elements()))
	for k, v := range m.Elements() {
		if s, ok := v.(types.String); ok {
			out[k] = s.ValueString()
		}
	}
	return out
}

func jsonBodyOrEmpty(body string) ([]byte, error) {
	if body == "" {
		return nil, nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

var actionSpecs = map[string]actionSpec{
	"start": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery(""),
	},
	"stop": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("OFF"),
		queryBuilder: powerQuery("POWEROFF"),
	},
	"reset": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery("RESET"),
	},
	"powercycle": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("ON"),
		queryBuilder: powerQuery("POWERCYCLE"),
	},
	"suspend": {
		method:       "PATCH",
		path:         "/api/v1/servers/%d",
		bodyBuilder:  powerBody("SUSPENDED"),
		queryBuilder: powerQuery(""),
	},
	"rescue_activate": {
		method:      "POST",
		path:        "/api/v1/servers/%d/rescuesystem",
		bodyBuilder: emptyBody,
	},
	"rescue_deactivate": {
		method:      "DELETE",
		path:        "/api/v1/servers/%d/rescuesystem",
		bodyBuilder: emptyBody,
	},
	"iso_attach": {
		method:      "POST",
		path:        "/api/v1/servers/%d/iso",
		bodyBuilder: jsonBody,
	},
	"iso_detach": {
		method:      "DELETE",
		path:        "/api/v1/servers/%d/iso",
		bodyBuilder: emptyBody,
	},
	"snapshot_create": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots",
		bodyBuilder: jsonBody,
	},
	"snapshot_revert": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots/%s/revert",
		bodyBuilder: emptyBody,
	},
	"snapshot_export": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots/%s/export",
		bodyBuilder: emptyBody,
	},
	"snapshot_delete": {
		method:      "DELETE",
		path:        "/api/v1/servers/%d/snapshots/%s",
		bodyBuilder: emptyBody,
	},
	"snapshot_dryrun": {
		method:      "POST",
		path:        "/api/v1/servers/%d/snapshots:dryrun",
		bodyBuilder: jsonBody,
	},
	"disk_format": {
		method:      "POST",
		path:        "/api/v1/servers/%d/disks/%s:format",
		bodyBuilder: emptyBody,
	},
	"image_setup": {
		method:      "POST",
		path:        "/api/v1/servers/%d/image",
		bodyBuilder: jsonBody,
	},
	"user_image_setup": {
		method:      "POST",
		path:        "/api/v1/servers/%d/user-image",
		bodyBuilder: jsonBody,
	},
	"storage_optimize": {
		method:      "POST",
		path:        "/api/v1/servers/%d/storageoptimization",
		bodyBuilder: emptyOrJSONBody,
	},
	"firewall_reapply": {
		method:      "POST",
		path:        "/api/v1/servers/%d/interfaces/%s/firewall:reapply",
		bodyBuilder: emptyBody,
	},
	"firewall_restore": {
		method:      "POST",
		path:        "/api/v1/servers/%d/interfaces/%s/firewall:restore-copied-policies",
		bodyBuilder: emptyBody,
	},
}

func powerBody(state string) func(map[string]string, string) ([]byte, error) {
	return func(args map[string]string, body string) ([]byte, error) {
		m := map[string]interface{}{"state": state}
		return json.Marshal(m)
	}
}

func powerQuery(defaultOption string) func(map[string]string) url.Values {
	return func(args map[string]string) url.Values {
		option := defaultOption
		if v, ok := args["state_option"]; ok && v != "" {
			option = v
		}
		if option == "" {
			return nil
		}
		return url.Values{"stateOption": []string{option}}
	}
}

func emptyBody(args map[string]string, body string) ([]byte, error) {
	return nil, nil
}

func emptyOrJSONBody(args map[string]string, body string) ([]byte, error) {
	if body == "" {
		return json.Marshal(map[string]interface{}{})
	}
	return jsonBody(args, body)
}

func jsonBody(args map[string]string, body string) ([]byte, error) {
	if body == "" {
		return nil, fmt.Errorf("action requires a JSON request body")
	}
	return jsonBodyOrEmpty(body)
}
