package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

type cloudDNSDataSource struct {
	client  *clouddnsclient.Client
	records bool
}

type cloudDNSDomainModel struct {
	ID           types.Int64  `tfsdk:"id"`
	FQDN         types.String `tfsdk:"fqdn"`
	IsDNSManaged types.Bool   `tfsdk:"is_dns_managed"`
}

type cloudDNSRecordDataModel struct {
	Identifier types.String `tfsdk:"identifier"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	RData      types.String `tfsdk:"rdata"`
	TTL        types.Int64  `tfsdk:"ttl"`
	Immutable  types.Bool   `tfsdk:"immutable"`
}

// NewCloudDNSDomainsDataSource lists all domains visible to a CloudDNS API key.
func NewCloudDNSDomainsDataSource() datasource.DataSource { return &cloudDNSDataSource{} }

// NewCloudDNSRecordsDataSource reads the active CloudDNS zone without creating a revision.
func NewCloudDNSRecordsDataSource() datasource.DataSource { return &cloudDNSDataSource{records: true} }

func (d *cloudDNSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clouddns_domains"
	if d.records {
		resp.TypeName = req.ProviderTypeName + "_clouddns_records"
	}
}

func (d *cloudDNSDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredCloudDNSClient(req.ProviderData, &resp.Diagnostics)
}

func (d *cloudDNSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists domains visible to the netcup CloudDNS API key, including whether netcup manages their DNS. Requires clouddns_api_key.",
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":             schema.Int64Attribute{Computed: true, Description: "Domain ID used by the CloudDNS API."},
				"fqdn":           schema.StringAttribute{Computed: true, Description: "Fully qualified domain name."},
				"is_dns_managed": schema.BoolAttribute{Computed: true, Description: "Whether netcup manages DNS for this domain."},
			}}},
		},
	}
	if !d.records {
		return
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Reads records from the active netcup CloudDNS revision. Does not create or commit revisions. Requires clouddns_api_key.", Attributes: map[string]schema.Attribute{
		"domain_id": schema.Int64Attribute{Required: true, Description: "Domain ID from netcup_clouddns_domains.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
		"records": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"identifier": schema.StringAttribute{Computed: true, Description: "Record identifier in this revision."},
			"name":       schema.StringAttribute{Computed: true, Description: "Record name relative to the zone."},
			"type":       schema.StringAttribute{Computed: true, Description: "DNS record type."},
			"rdata":      schema.StringAttribute{Computed: true, Description: "Record data in DNS presentation format."},
			"ttl":        schema.Int64Attribute{Computed: true, Description: "TTL in seconds; may be null for system records."},
			"immutable":  schema.BoolAttribute{Computed: true, Description: "Whether this is a system-managed, immutable record."},
		}}},
	}}
}

func (d *cloudDNSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.records {
		d.readRecords(ctx, req, resp)
		return
	}
	domains, err := d.client.Domains(ctx)
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Read Error", err.Error())
		return
	}
	data := struct {
		Domains []cloudDNSDomainModel `tfsdk:"domains"`
	}{Domains: make([]cloudDNSDomainModel, 0, len(domains))}
	for _, domain := range domains {
		data.Domains = append(data.Domains, cloudDNSDomainModel{ID: types.Int64Value(domain.ID), FQDN: types.StringValue(domain.FQDN), IsDNSManaged: types.BoolValue(domain.IsDNSManaged)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *cloudDNSDataSource) readRecords(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data struct {
		DomainID types.Int64               `tfsdk:"domain_id"`
		Records  []cloudDNSRecordDataModel `tfsdk:"records"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	records, err := activeCloudDNSRecords(ctx, d.client, data.DomainID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Read Error", err.Error())
		return
	}
	data.Records = make([]cloudDNSRecordDataModel, 0, len(records))
	for _, record := range records {
		ttl := types.Int64Null()
		if record.TTL != nil {
			ttl = types.Int64Value(*record.TTL)
		}
		data.Records = append(data.Records, cloudDNSRecordDataModel{
			Identifier: types.StringValue(record.Identifier), Name: types.StringValue(record.Name), Type: types.StringValue(record.Type), RData: types.StringValue(record.RData), TTL: ttl, Immutable: types.BoolValue(record.Immutable),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
