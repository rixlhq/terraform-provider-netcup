package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &ScpServerMetricsDataSource{}

// NewScpServerMetricsDataSource returns a new data source for server metrics.
func NewScpServerMetricsDataSource() datasource.DataSource {
	return &ScpServerMetricsDataSource{}
}

// ScpServerMetricsDataSource reads CPU, disk, network and network packet metrics.
type ScpServerMetricsDataSource struct {
	client *scpclient.Client
}

// ScpServerMetricsDataSourceModel describes the metrics data source input/output.
type ScpServerMetricsDataSourceModel struct {
	ServerId types.Int64  `tfsdk:"server_id"`
	Metric   types.String `tfsdk:"metric"`
	Hours    types.Int64  `tfsdk:"hours"`
	Json     types.String `tfsdk:"json"`
}

func (d *ScpServerMetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_server_metrics"
}

func (d *ScpServerMetricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Reads server metrics from the SCP REST API. The raw JSON response is returned so it can be parsed with terraform's jsondecode function.",
		Attributes: map[string]dschema.Attribute{
			attrServerID: dschema.Int64Attribute{
				MarkdownDescription: "ID of the server to query metrics for.",
				Required:            true,
			},
			"metric": dschema.StringAttribute{
				MarkdownDescription: "Metric to query. Valid values are `cpu`, `disk`, `network` and `network_packet`.",
				Required:            true,
			},
			"hours": dschema.Int64Attribute{
				MarkdownDescription: "Number of hours to look back (maximum 1440).",
				Optional:            true,
			},
			"json": dschema.StringAttribute{
				MarkdownDescription: "Raw JSON response from the SCP metrics endpoint.",
				Computed:            true,
			},
		},
	}
}

func (d *ScpServerMetricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScpServerMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpServerMetricsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}

	metricPaths := map[string]string{
		"cpu":            "cpu",
		"disk":           "disk",
		"network":        "network",
		"network_packet": "network/packet",
	}
	suffix, ok := metricPaths[data.Metric.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Invalid metric", "metric must be one of cpu, disk, network or network_packet, got: "+data.Metric.ValueString())
		return
	}

	path := fmt.Sprintf("/api/v1/servers/%s/metrics/%s", strconv.FormatInt(data.ServerId.ValueInt64(), 10), suffix)
	query := url.Values{}
	if !data.Hours.IsNull() && !data.Hours.IsUnknown() {
		query.Set("hours", strconv.FormatInt(data.Hours.ValueInt64(), 10))
	}

	body, err := d.client.Get(ctx, path, query)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	data.Json = types.StringValue(string(body))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
