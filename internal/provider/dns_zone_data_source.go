package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
)

var _ datasource.DataSource = &DNSZoneDataSource{}

// DNSZoneDataSource reads the settings of a netcup DNS zone.
type DNSZoneDataSource struct {
	client *client.Client
}

type dnsZoneDataSourceModel struct {
	DomainName   types.String `tfsdk:"domain_name"`
	TTL          types.Int64  `tfsdk:"ttl"`
	Serial       types.String `tfsdk:"serial"`
	Refresh      types.String `tfsdk:"refresh"`
	Retry        types.String `tfsdk:"retry"`
	Expire       types.String `tfsdk:"expire"`
	DnsSecStatus types.Bool   `tfsdk:"dnssec_status"`
}

func NewDNSZoneDataSource() datasource.DataSource {
	return &DNSZoneDataSource{}
}

func (d *DNSZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (d *DNSZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads DNS zone settings for a netcup domain.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "Domain name of the DNS zone.",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live (TTL) for the zone, in seconds.",
				Computed:            true,
			},
			"serial": schema.StringAttribute{
				MarkdownDescription: "Serial number of the zone.",
				Computed:            true,
			},
			"refresh": schema.StringAttribute{
				MarkdownDescription: "Refresh interval of the zone.",
				Computed:            true,
			},
			"retry": schema.StringAttribute{
				MarkdownDescription: "Retry interval of the zone.",
				Computed:            true,
			},
			"expire": schema.StringAttribute{
				MarkdownDescription: "Expire time of the zone.",
				Computed:            true,
			},
			"dnssec_status": schema.BoolAttribute{
				MarkdownDescription: "DNSSEC status of the zone.",
				Computed:            true,
			},
		},
	}
}

func (d *DNSZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *DNSZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsZoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID, err := d.client.Login(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Login Error", err.Error())
		return
	}
	defer d.client.Logout(ctx, sessionID)

	zone, err := d.client.InfoDnsZone(ctx, sessionID, data.DomainName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	data.TTL = types.Int64Value(zone.TTL)
	data.Serial = types.StringValue(zone.Serial)
	data.Refresh = types.StringValue(zone.Refresh)
	data.Retry = types.StringValue(zone.Retry)
	data.Expire = types.StringValue(zone.Expire)
	data.DnsSecStatus = types.BoolValue(zone.DnsSecStatus)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
