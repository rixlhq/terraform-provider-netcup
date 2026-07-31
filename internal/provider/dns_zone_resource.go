package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &DNSZoneResource{}
var _ resource.ResourceWithImportState = &DNSZoneResource{}

// DNSZoneResource manages the TTL of an existing netcup DNS zone.
type DNSZoneResource struct {
	client *client.Client
	scp    *scpclient.Client
}

type dnsZoneResourceModel struct {
	DomainName   types.String `tfsdk:"domain_name"`
	TTL          types.Int64  `tfsdk:"ttl"`
	Serial       types.String `tfsdk:"serial"`
	Refresh      types.String `tfsdk:"refresh"`
	Retry        types.String `tfsdk:"retry"`
	Expire       types.String `tfsdk:"expire"`
	DnsSecStatus types.Bool   `tfsdk:"dnssec_status"`
}

func NewDNSZoneResource() resource.Resource {
	return &DNSZoneResource{}
}

func (r *DNSZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (r *DNSZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DNS zone settings for an existing netcup domain.",
		Attributes: map[string]schema.Attribute{
			"domain_name": schema.StringAttribute{
				MarkdownDescription: "Domain name of the DNS zone.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "Time to live (TTL) for all records in the zone, in seconds.",
				Required:            true,
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

func (r *DNSZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerData, got: %T", req.ProviderData),
		)
		return
	}

	r.client = data.CCP
	r.scp = data.SCP
}

func (r *DNSZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dnsZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.updateZone(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	r.updateModel(&data, zone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsZoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.readZone(ctx, data.DomainName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	r.updateModel(&data, zone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dnsZoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.updateZone(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	r.updateModel(&data, zone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Deleting the resource only removes it from state; the zone continues to exist at netcup.
}

func (r *DNSZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := dnsZoneResourceModel{DomainName: types.StringValue(req.ID)}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSZoneResource) updateZone(ctx context.Context, data *dnsZoneResourceModel) (*client.DNSZone, error) {
	sessionID, err := r.client.Login(ctx)
	if err != nil {
		return nil, err
	}
	defer r.client.Logout(ctx, sessionID)

	zone := &client.DNSZone{
		Name: data.DomainName.ValueString(),
		TTL:  data.TTL.ValueInt64(),
	}

	return r.client.UpdateDnsZone(ctx, sessionID, data.DomainName.ValueString(), zone)
}

func (r *DNSZoneResource) readZone(ctx context.Context, domainName string) (*client.DNSZone, error) {
	sessionID, err := r.client.Login(ctx)
	if err != nil {
		return nil, err
	}
	defer r.client.Logout(ctx, sessionID)

	return r.client.InfoDnsZone(ctx, sessionID, domainName)
}

func (r *DNSZoneResource) updateModel(data *dnsZoneResourceModel, zone *client.DNSZone) {
	data.TTL = types.Int64Value(zone.TTL)
	data.Serial = types.StringValue(zone.Serial)
	data.Refresh = types.StringValue(zone.Refresh)
	data.Retry = types.StringValue(zone.Retry)
	data.Expire = types.StringValue(zone.Expire)
	data.DnsSecStatus = types.BoolValue(zone.DnsSecStatus)
}
