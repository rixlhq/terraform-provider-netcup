package scpuserfailoveripsv6

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

var _ datasource.DataSource = &ScpUserFailoveripsV6DataSource{}

// NewScpUserFailoveripsV6DataSource returns a new data source.
func NewScpUserFailoveripsV6DataSource() datasource.DataSource {
	return &ScpUserFailoveripsV6DataSource{}
}

// ScpUserFailoveripsV6DataSource maps the /api/v1/users/{userId}/failoverips/v6 SCP endpoint to a Terraform data source.
type ScpUserFailoveripsV6DataSource struct {
	client *scpclient.Client
}

func (d *ScpUserFailoveripsV6DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_user_failoverips_v6"
}

func (d *ScpUserFailoveripsV6DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpUserFailoveripsV6DataSourceSchema(ctx)
}

func (d *ScpUserFailoveripsV6DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScpUserFailoveripsV6DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpUserFailoveripsV6Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}

	path := "/api/v1/users/{userId}/failoverips/v6"
	query := url.Values{}
	if !data.Ip.IsNull() && !data.Ip.IsUnknown() {
		query.Set("ip", data.Ip.ValueString())
	}
	if !data.ServerId.IsNull() && !data.ServerId.IsUnknown() {
		query.Set("serverId", strconv.FormatInt(data.ServerId.ValueInt64(), 10))
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
		jsonVal = map[string]interface{}{"scp_user_failoverips_v6": arr}
	}

	schema := ScpUserFailoveripsV6DataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}
