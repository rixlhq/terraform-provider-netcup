package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
)

var _ provider.Provider = &NetcupProvider{}

// NetcupProvider implements the netcup Terraform provider.
type NetcupProvider struct {
	version string
}

// NetcupProviderModel describes the provider configuration.
type NetcupProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	APIPassword    types.String `tfsdk:"api_password"`
	CustomerNumber types.String `tfsdk:"customer_number"`
	Endpoint       types.String `tfsdk:"endpoint"`
}

func (p *NetcupProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netcup"
	resp.Version = p.version
}

func (p *NetcupProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Netcup API key generated in the Customer Control Panel.",
				Required:            true,
				Sensitive:           true,
			},
			"api_password": schema.StringAttribute{
				MarkdownDescription: "Netcup API password generated in the Customer Control Panel.",
				Required:            true,
				Sensitive:           true,
			},
			"customer_number": schema.StringAttribute{
				MarkdownDescription: "Netcup customer number.",
				Required:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Override the netcup JSON API endpoint. Defaults to the production endpoint.",
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
		resp.Diagnostics.AddError("Client Configuration Error", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *NetcupProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDNSRecordResource,
		NewDNSZoneResource,
	}
}

func (p *NetcupProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDNSRecordsDataSource,
		NewDNSZoneDataSource,
	}
}

// New returns a factory for the netcup provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NetcupProvider{version: version}
	}
}
