package scpmaintenance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &ScpMaintenanceDataSource{}

// NewScpMaintenanceDataSource returns a new data source.
func NewScpMaintenanceDataSource() datasource.DataSource {
	return &ScpMaintenanceDataSource{}
}

// ScpMaintenanceDataSource maps the /api/v1/maintenance SCP endpoint to a Terraform data source.
type ScpMaintenanceDataSource struct {
	client *scpclient.Client
}

func (d *ScpMaintenanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_maintenance"
}

func (d *ScpMaintenanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpMaintenanceDataSourceSchema(ctx)
	resp.Schema.DeprecationMessage = "Netcup is removing the underlying SCP endpoint GET /api/v1/maintenance by 31.12.2026. This data source will be removed in a future major version."
}

func (d *ScpMaintenanceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providerdata.Data, got: %T", req.ProviderData),
		)
		return
	}

	if pd.SCP == nil {
		return
	}

	d.client = pd.SCP
}

func (d *ScpMaintenanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ScpMaintenanceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}

	body, err := d.client.Get(ctx, "/api/v1/maintenance", nil)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	var apiResp struct {
		FinishAt string `json:"finishAt"`
		StartAt  string `json:"startAt"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", err.Error())
		return
	}

	state.FinishAt = types.StringValue(apiResp.FinishAt)
	state.StartAt = types.StringValue(apiResp.StartAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
