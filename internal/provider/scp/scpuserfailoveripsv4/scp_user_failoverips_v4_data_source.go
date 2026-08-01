package scpuserfailoveripsv4

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

var _ datasource.DataSource = &ScpUserFailoveripsV4DataSource{}

// NewScpUserFailoveripsV4DataSource returns a new data source.
func NewScpUserFailoveripsV4DataSource() datasource.DataSource {
	return &ScpUserFailoveripsV4DataSource{}
}

// ScpUserFailoveripsV4DataSource maps the /api/v1/users/{userId}/failoverips/v4 SCP endpoint to a Terraform data source.
type ScpUserFailoveripsV4DataSource struct {
	client *scpclient.Client
}

func (d *ScpUserFailoveripsV4DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_user_failoverips_v4"
}

func (d *ScpUserFailoveripsV4DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpUserFailoveripsV4DataSourceSchema(ctx)
}

func (d *ScpUserFailoveripsV4DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScpUserFailoveripsV4DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpUserFailoveripsV4Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}
	path := fmt.Sprintf("/api/v1/users/%s/failoverips/v4", strconv.FormatInt(data.UserId.ValueInt64(), 10))
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

	if arr, ok := jsonVal.([]any); ok {
		jsonVal = map[string]any{"scp_user_failoverips_v4": arr}
	}

	schema := ScpUserFailoveripsV4DataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}
