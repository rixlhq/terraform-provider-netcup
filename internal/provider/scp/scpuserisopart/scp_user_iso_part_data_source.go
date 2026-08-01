package scpuserisopart

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &ScpUserIsoPartDataSource{}

// NewScpUserIsoPartDataSource returns a new data source.
func NewScpUserIsoPartDataSource() datasource.DataSource {
	return &ScpUserIsoPartDataSource{}
}

// ScpUserIsoPartDataSource maps the /api/v1/users/{userId}/isos/{key}/{uploadId}/parts/{partNumber} SCP endpoint to a Terraform data source.
type ScpUserIsoPartDataSource struct {
	client *scpclient.Client
}

func (d *ScpUserIsoPartDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_user_iso_part"
}

func (d *ScpUserIsoPartDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ScpUserIsoPartDataSourceSchema(ctx)
}

func (d *ScpUserIsoPartDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScpUserIsoPartDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScpUserIsoPartModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "The netcup provider must be configured with scp_access_token to use this data source.")
		return
	}
	path := fmt.Sprintf("/api/v1/users/%s/isos/%s/%s/parts/%s", strconv.FormatInt(data.UserId.ValueInt64(), 10), data.Key.ValueString(), data.UploadId.ValueString(), strconv.FormatInt(data.PartNumber.ValueInt64(), 10))

	body, err := d.client.Get(ctx, path, nil)
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
		jsonVal = map[string]any{"scp_user_iso_part": arr}
	}

	schema := ScpUserIsoPartDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)
	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}
