package scpservers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &ScpServersDataSource{}

// NewScpServersDataSource returns a new data source.
func NewScpServersDataSource() datasource.DataSource {
	return &ScpServersDataSource{}
}

// ScpServersDataSource maps the /api/v1/servers SCP endpoint to a Terraform data source.
type ScpServersDataSource struct {
	client *scpclient.Client
}

func (d *ScpServersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_servers"
}

func (d *ScpServersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpServersDataSourceSchema(ctx)
}

func (d *ScpServersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func scpServersQuery(data ScpServersModel) url.Values {
	query := url.Values{}
	if !data.FirewallPolicyId.IsNull() && !data.FirewallPolicyId.IsUnknown() {
		query.Set("firewallPolicyId", strconv.FormatInt(data.FirewallPolicyId.ValueInt64(), 10))
	}
	if !data.Ip.IsNull() && !data.Ip.IsUnknown() {
		query.Set("ip", data.Ip.ValueString())
	}
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		query.Set("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		query.Set("name", data.Name.ValueString())
	}
	if !data.Offset.IsNull() && !data.Offset.IsUnknown() {
		query.Set("offset", strconv.FormatInt(data.Offset.ValueInt64(), 10))
	}
	if !data.Q.IsNull() && !data.Q.IsUnknown() {
		query.Set("q", data.Q.ValueString())
	}
	return query
}

func (d *ScpServersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpServersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}
	body, err := d.client.Get(ctx, "/api/v1/servers", scpServersQuery(data))
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", err.Error())
		return
	}

	if arr, ok := jsonVal.([]any); ok {
		jsonVal = map[string]any{"scp_servers": arr}
	}

	schema := ScpServersDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}
