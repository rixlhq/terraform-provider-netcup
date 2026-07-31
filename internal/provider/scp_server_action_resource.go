package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func (r *ScpServerActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_action"
}

func (r *ScpServerActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = scpServerActionSchema
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
		return errors.New("configure the provider with scp_access_token to use this resource")
	}

	args := mapStringMap(data.Arguments)
	serverID := data.ServerId.ValueInt64()

	spec, ok := actionSpecs[data.Action.ValueString()]
	if !ok {
		return fmt.Errorf("unsupported action %q", data.Action.ValueString())
	}

	body, err := spec.bodyBuilder(args, data.Body.ValueString())
	if err != nil {
		return err
	}

	path, err := actionPath(spec, serverID, data.Action.ValueString(), args)
	if err != nil {
		return err
	}

	if err := r.sendActionRequest(ctx, spec.method, path, body); err != nil {
		return err
	}

	data.Id = types.StringValue(fmt.Sprintf("%d/%s/%d", serverID, data.Action.ValueString(), time.Now().Unix()))
	return nil
}

func (r *ScpServerActionResource) sendActionRequest(ctx context.Context, method, path string, body []byte) error {
	switch method {
	case "POST":
		_, err := r.client.Post(ctx, path, body)
		return err
	case "PATCH":
		_, err := r.client.Patch(ctx, path, body)
		return err
	case "PUT":
		_, err := r.client.Put(ctx, path, body)
		return err
	case methodDelete:
		_, err := r.client.Delete(ctx, path)
		return err
	default:
		return fmt.Errorf("unsupported HTTP method %q", method)
	}
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
