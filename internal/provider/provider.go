package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ provider.Provider = &NetcupProvider{}

// NetcupProvider implements the netcup Terraform provider.
type NetcupProvider struct {
	version string
}

// providerData holds the configured API clients and is passed to resources and data sources.
type providerData = providerdata.Data

// NetcupProviderModel describes the provider configuration.
type NetcupProviderModel struct {
	APIKey          types.String `tfsdk:"api_key"`
	APIPassword     types.String `tfsdk:"api_password"`
	CustomerNumber  types.String `tfsdk:"customer_number"`
	Endpoint        types.String `tfsdk:"endpoint"`
	SCPAccessToken  types.String `tfsdk:"scp_access_token"`
	SCPRefreshToken types.String `tfsdk:"scp_refresh_token"`
	SCPBaseURL      types.String `tfsdk:"scp_base_url"`
}

func (p *NetcupProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netcup"
	resp.Version = p.version
}

func (p *NetcupProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing netcup CCP DNS and SCP resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Netcup API key generated in the Customer Control Panel for the CCP/DNS API.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_password": schema.StringAttribute{
				MarkdownDescription: "Netcup API password generated in the Customer Control Panel for the CCP/DNS API.",
				Optional:            true,
				Sensitive:           true,
			},
			"customer_number": schema.StringAttribute{
				MarkdownDescription: "Netcup customer number for the CCP/DNS API.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Override the netcup CCP JSON API endpoint. Defaults to the production endpoint.",
				Optional:            true,
			},
			"scp_access_token": schema.StringAttribute{
				MarkdownDescription: "Bearer access token for the netcup SCP REST API.",
				Optional:            true,
				Sensitive:           true,
			},
			"scp_refresh_token": schema.StringAttribute{
				MarkdownDescription: "Offline refresh token for the netcup SCP REST API.",
				Optional:            true,
				Sensitive:           true,
			},
			"scp_base_url": schema.StringAttribute{
				MarkdownDescription: "Override the netcup SCP REST API base URL. Defaults to https://www.servercontrolpanel.de/scp-core.",
				Optional:            true,
			},
		},
	}
}

func (p *NetcupProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NetcupProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ccpClient, hasCCP, err := newCCPClient(data)
	if err != nil {
		resp.Diagnostics.AddError("CCP Client Configuration Error", err.Error())
		return
	}

	scpClient, hasSCP := newSCPClient(data)

	if !hasCCP && !hasSCP {
		resp.Diagnostics.AddError(
			"Missing Credentials",
			"Either CCP credentials (api_key, api_password, customer_number) or SCP credentials (scp_access_token) must be configured.",
		)
		return
	}

	resp.DataSourceData = &providerData{CCP: ccpClient, SCP: scpClient}
	resp.ResourceData = &providerData{CCP: ccpClient, SCP: scpClient}
}

func newCCPClient(data NetcupProviderModel) (*client.Client, bool, error) {
	hasCCP := !data.APIKey.IsNull() && !data.APIPassword.IsNull() && !data.CustomerNumber.IsNull()
	if !hasCCP {
		return nil, false, nil
	}

	endpoint := ""
	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		endpoint = data.Endpoint.ValueString()
	}

	c, err := client.New(
		data.CustomerNumber.ValueString(),
		data.APIKey.ValueString(),
		data.APIPassword.ValueString(),
		endpoint,
		nil,
	)
	if err != nil {
		return nil, true, err
	}
	return c, true, nil
}

func newSCPClient(data NetcupProviderModel) (*scpclient.Client, bool) {
	hasSCP := !data.SCPAccessToken.IsNull()
	if !hasSCP {
		return nil, false
	}

	baseURL := ""
	if !data.SCPBaseURL.IsNull() && !data.SCPBaseURL.IsUnknown() {
		baseURL = data.SCPBaseURL.ValueString()
	}
	return scpclient.New(
		data.SCPAccessToken.ValueString(),
		data.SCPRefreshToken.ValueString(),
		baseURL,
		nil,
	), true
}

func (p *NetcupProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDNSRecordResource,
		NewDNSZoneResource,
		NewScpServerResource,
		NewScpServerActionResource,
		NewScpServerSnapshotResource,
		NewScpRdnsResource,
		NewScpUserFirewallPolicyResource,
		NewScpFailoverIpV4Resource,
		NewScpFailoverIpV6Resource,
		NewScpUserVlanResource,
		NewScpUserResource,
		NewScpUserSshKeyResource,
		NewScpTaskActionResource,
	}
}

func (p *NetcupProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	sources := make([]func() datasource.DataSource, 0, 3+len(scp.DataSources()))
	sources = append(sources,
		NewDNSRecordsDataSource,
		NewDNSZoneDataSource,
		NewScpServerMetricsDataSource,
	)
	sources = append(sources, scp.DataSources()...)
	return sources
}

// New returns a factory for the netcup provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NetcupProvider{version: version}
	}
}
