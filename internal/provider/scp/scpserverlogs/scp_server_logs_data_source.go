package scpserverlogs

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &ScpServerLogsDataSource{}

// NewScpServerLogsDataSource returns a new data source.
func NewScpServerLogsDataSource() datasource.DataSource {
	return &ScpServerLogsDataSource{}
}

// ScpServerLogsDataSource maps the /api/v1/servers/{serverId}/logs SCP endpoint to a Terraform data source.
type ScpServerLogsDataSource struct {
	client *scpclient.Client
}

func (d *ScpServerLogsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_logs"
}

func (d *ScpServerLogsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpServerLogsDataSourceSchema(ctx)
}

func (d *ScpServerLogsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = pd.SCP
}

func (d *ScpServerLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpServerLogsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}
	path := fmt.Sprintf("/api/v1/servers/%s/logs", strconv.FormatInt(data.ServerId.ValueInt64(), 10))
	query := url.Values{}
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		query.Set("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}
	if !data.Offset.IsNull() && !data.Offset.IsUnknown() {
		query.Set("offset", strconv.FormatInt(data.Offset.ValueInt64(), 10))
	}

	body, err := d.client.Get(ctx, path, query)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", err.Error())
		return
	}

	if arr, ok := jsonVal.([]interface{}); ok {
		jsonVal = map[string]interface{}{"scp_server_logs": arr}
	}

	schema := ScpServerLogsDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}
