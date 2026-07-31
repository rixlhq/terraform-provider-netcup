package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &ScpTaskActionResource{}

// NewScpTaskActionResource returns a one-off resource for task actions.
func NewScpTaskActionResource() resource.Resource {
	return &ScpTaskActionResource{}
}

// ScpTaskActionResource triggers one-off actions against SCP tasks.
type ScpTaskActionResource struct {
	client *scpclient.Client
}

// ScpTaskActionResourceModel describes the task action input/output.
type ScpTaskActionResourceModel struct {
	TaskUuid types.String `tfsdk:"task_uuid"`
	Action   types.String `tfsdk:"action"`
	Triggers types.Map    `tfsdk:"triggers"`
	Id       types.String `tfsdk:"id"`
}

func (r *ScpTaskActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_task_action"
}

func (r *ScpTaskActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers one-off actions against an existing SCP task (currently `cancel`).",
		Attributes: map[string]schema.Attribute{
			"task_uuid": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the task to run the action on.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"action": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Action to execute. Supported: `cancel`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Map of values that, when changed, force the action to run again.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier for this action invocation.",
			},
		},
	}
}

func (r *ScpTaskActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScpTaskActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScpTaskActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.execute(ctx, &data); err != nil {
		resp.Diagnostics.AddError("SCP Task Action Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpTaskActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScpTaskActionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpTaskActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScpTaskActionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.execute(ctx, &data); err != nil {
		resp.Diagnostics.AddError("SCP Task Action Failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScpTaskActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *ScpTaskActionResource) execute(ctx context.Context, data *ScpTaskActionResourceModel) error {
	if r.client == nil {
		return fmt.Errorf("configure the provider with scp_access_token to use this resource")
	}

	switch data.Action.ValueString() {
	case "cancel":
		path := fmt.Sprintf("/api/v1/tasks/%s:cancel", data.TaskUuid.ValueString())
		if _, err := r.client.Put(ctx, path, nil); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported action %q", data.Action.ValueString())
	}

	data.Id = types.StringValue(fmt.Sprintf("%s/%s/%d", data.TaskUuid.ValueString(), data.Action.ValueString(), time.Now().Unix()))
	return nil
}
