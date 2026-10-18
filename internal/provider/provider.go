package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
)

var (
	_ provider.Provider                       = &NetcupProvider{}
	_ provider.ProviderWithFunctions          = &NetcupProvider{}
	_ provider.ProviderWithEphemeralResources = &NetcupProvider{}
)

// NetcupProvider implements the netcup Terraform provider.
type NetcupProvider struct {
	version string
}

// providerData holds the configured API clients and is passed to resources and data sources.
type providerData = providerdata.Data

// NetcupProviderModel describes the provider configuration.
type NetcupProviderModel struct {
	CloudDNSAPIKey  types.String `tfsdk:"clouddns_api_key"`
	CloudDNSBaseURL types.String `tfsdk:"clouddns_base_url"`
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
		MarkdownDescription: "Terraform provider for managing netcup infrastructure via the CloudDNS REST API and SCP REST API (Server Control Panel OpenAPI). Supports CloudDNS records and domains, virtual servers (VPS / Root Server), reverse DNS (rDNS IPv4/IPv6), network interfaces, firewall policies, snapshots, failover IPs, VLANs, custom images, ISOs, provider-defined functions, and ephemeral tokens under Terraform Plugin Framework protocol version 6.",
		Attributes: map[string]schema.Attribute{
			"clouddns_api_key":  schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "CloudDNS bearer API key from CCP Master Data > API > API Keys. Can be set via `NETCUP_CLOUDDNS_API_KEY`."},
			"clouddns_base_url": schema.StringAttribute{Optional: true, MarkdownDescription: "Override the CloudDNS API base URL (default https://api.netcup.com). Can be set via `NETCUP_CLOUDDNS_BASE_URL`."},
			"scp_access_token": schema.StringAttribute{
				MarkdownDescription: "Bearer access token for the netcup SCP REST API. Can be set via the `NETCUP_SCP_ACCESS_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"scp_refresh_token": schema.StringAttribute{
				MarkdownDescription: "Offline refresh token for the netcup SCP REST API. Can be set via the `NETCUP_SCP_REFRESH_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"scp_base_url": schema.StringAttribute{
				MarkdownDescription: "Override the netcup SCP REST API base URL. Can be set via the `NETCUP_SCP_BASE_URL` environment variable.",
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

	data = applyEnvOverrides(data)

	scpClient, hasSCP := newSCPClient(data)

	cloudClient, hasCloud, err := newCloudDNSClient(data)
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Client Configuration Error", err.Error())
		return
	}

	if !hasSCP && !hasCloud {
		resp.Diagnostics.AddError(
			"Missing Credentials",
			"Configure CloudDNS credentials (clouddns_api_key) or SCP credentials (scp_access_token or scp_refresh_token).",
		)
		return
	}

	resp.DataSourceData = &providerData{SCP: scpClient, CloudDNS: cloudClient}
	resp.ResourceData = &providerData{SCP: scpClient, CloudDNS: cloudClient}
}

func (p *NetcupProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCloudDNSRecordResource,
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
		NewScpServerInterfaceResource,
		NewScpServerInterfaceFirewallResource,
	}
}

func (p *NetcupProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	sources := make([]func() datasource.DataSource, 0, 3+len(scp.DataSources()))
	sources = append(sources,
		NewCloudDNSDomainsDataSource,
		NewCloudDNSRecordsDataSource,
		NewScpServerMetricsDataSource,
	)
	sources = append(sources, scp.DataSources()...)
	return sources
}

func (p *NetcupProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		NewIPVersionFunction,
	}
}

func (p *NetcupProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewScpAccessTokenEphemeralResource,
	}
}

// New returns a factory for the netcup provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NetcupProvider{version: version}
	}
}
