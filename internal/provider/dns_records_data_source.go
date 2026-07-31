package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ datasource.DataSource = &DNSRecordsDataSource{}

var dnsRecordModelType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":          types.StringType,
	"hostname":    types.StringType,
	"type":        types.StringType,
	"destination": types.StringType,
	"priority":    types.Int64Type,
}}

// DNSRecordsDataSource reads all DNS records for a netcup zone.
type DNSRecordsDataSource struct {
	client *client.Client
	scp    *scpclient.Client
}

type dnsRecordsDataSourceModel struct {
	DomainName types.String `tfsdk:"domain_name"`
	TTL        types.Int64  `tfsdk:"ttl"`
	Records    types.List   `tfsdk:"records"`
}

type dnsRecordModel struct {
	ID          types.String `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	Type        types.String `tfsdk:"type"`
	Destination types.String `tfsdk:"destination"`
	Priority    types.Int64  `tfsdk:"priority"`
}

func NewDNSRecordsDataSource() datasource.DataSource {
	return &DNSRecordsDataSource{}
}

func (d *DNSRecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

func (d *DNSRecordsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads all DNS records for a netcup DNS zone.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "Domain name of the DNS zone.",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "TTL for all records in the zone.",
				Computed:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "List of DNS records in the zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Record ID.",
							Computed:            true,
						},
						"hostname": schema.StringAttribute{
							MarkdownDescription: "Record hostname.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Record type.",
							Computed:            true,
						},
						"destination": schema.StringAttribute{
							MarkdownDescription: "Record destination.",
							Computed:            true,
						},
						"priority": schema.Int64Attribute{
							MarkdownDescription: "Record priority.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *DNSRecordsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T", req.ProviderData),
		)
		return
	}

	d.client = data.CCP
	d.scp = data.SCP
}

func (d *DNSRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsRecordsDataSourceModel
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

	set, err := d.client.InfoDnsRecords(ctx, sessionID, data.DomainName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	data.TTL = types.Int64Value(zone.TTL)

	records := make([]dnsRecordModel, 0, len(set.DNSRecords))
	for _, rec := range set.DNSRecords {
		m := dnsRecordModel{
			ID:          types.StringValue(rec.ID),
			Hostname:    types.StringValue(rec.Hostname),
			Type:        types.StringValue(rec.Type),
			Destination: types.StringValue(rec.Destination),
		}
		if rec.Priority != nil {
			m.Priority = types.Int64Value(*rec.Priority)
		}
		records = append(records, m)
	}

	list, diags := types.ListValueFrom(ctx, dnsRecordModelType, records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Records = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
