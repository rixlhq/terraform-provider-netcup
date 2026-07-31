package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

var _ resource.Resource = &DNSRecordResource{}
var _ resource.ResourceWithImportState = &DNSRecordResource{}

// DNSRecordResource manages a single DNS record in a netcup zone.
type DNSRecordResource struct {
	client *client.Client
	scp    *scpclient.Client
}

type dnsRecordResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Zone        types.String `tfsdk:"zone"`
	Hostname    types.String `tfsdk:"hostname"`
	Type        types.String `tfsdk:"type"`
	Destination types.String `tfsdk:"destination"`
	Priority    types.Int64  `tfsdk:"priority"`
}

func NewDNSRecordResource() resource.Resource {
	return &DNSRecordResource{}
}

func (r *DNSRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *DNSRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single DNS record for a domain hosted at netcup.",
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				MarkdownDescription: "Domain name of the DNS zone (e.g. example.com).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname of the record. Use '@' for the zone apex.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "DNS record type such as A, AAAA, CNAME, MX, TXT, etc.",
				Required:            true,
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "Target/destination of the record.",
				Required:            true,
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Record priority, required for MX records.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique ID assigned to the record by the netcup API.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DNSRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID, err := r.client.Login(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Login Error", err.Error())
		return
	}
	defer r.client.Logout(ctx, sessionID)

	record := r.toClientRecord(data)
	updatedSet, err := r.client.UpdateDnsRecords(ctx, sessionID, data.Zone.ValueString(), &client.DNSRecordSet{DNSRecords: []client.DNSRecord{record}})
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	match := findMatchingRecord(record, updatedSet.DNSRecords)
	if match == nil {
		resp.Diagnostics.AddError("API Error", "record was not returned after create")
		return
	}

	r.updateModel(&data, *match)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID, err := r.client.Login(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Login Error", err.Error())
		return
	}
	defer r.client.Logout(ctx, sessionID)

	set, err := r.client.InfoDnsRecords(ctx, sessionID, data.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	for _, rec := range set.DNSRecords {
		if rec.ID == data.ID.ValueString() {
			r.updateModel(&data, rec)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *DNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID, err := r.client.Login(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Login Error", err.Error())
		return
	}
	defer r.client.Logout(ctx, sessionID)

	record := r.toClientRecord(data)
	record.ID = data.ID.ValueString()

	updatedSet, err := r.client.UpdateDnsRecords(ctx, sessionID, data.Zone.ValueString(), &client.DNSRecordSet{DNSRecords: []client.DNSRecord{record}})
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	for _, rec := range updatedSet.DNSRecords {
		if rec.ID == record.ID {
			r.updateModel(&data, rec)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError("API Error", "record was not returned after update")
}

func (r *DNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID, err := r.client.Login(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Login Error", err.Error())
		return
	}
	defer r.client.Logout(ctx, sessionID)

	record := r.toClientRecord(data)
	record.ID = data.ID.ValueString()
	record.DeleteRecord = true

	_, err = r.client.UpdateDnsRecords(ctx, sessionID, data.Zone.ValueString(), &client.DNSRecordSet{DNSRecords: []client.DNSRecord{record}})
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
	}
}

func (r *DNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "expected 'zone/id'")
		return
	}

	data := dnsRecordResourceModel{
		Zone: types.StringValue(parts[0]),
		ID:   types.StringValue(parts[1]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DNSRecordResource) toClientRecord(data dnsRecordResourceModel) client.DNSRecord {
	rec := client.DNSRecord{
		Hostname:    data.Hostname.ValueString(),
		Type:        data.Type.ValueString(),
		Destination: data.Destination.ValueString(),
	}
	if !data.Priority.IsNull() && !data.Priority.IsUnknown() {
		p := data.Priority.ValueInt64()
		rec.Priority = &p
	}
	return rec
}

func (r *DNSRecordResource) updateModel(data *dnsRecordResourceModel, rec client.DNSRecord) {
	data.ID = types.StringValue(rec.ID)
	data.Hostname = types.StringValue(rec.Hostname)
	data.Type = types.StringValue(rec.Type)
	data.Destination = types.StringValue(rec.Destination)
	if rec.Priority != nil {
		data.Priority = types.Int64Value(*rec.Priority)
	} else {
		data.Priority = types.Int64Null()
	}
}

func findMatchingRecord(rec client.DNSRecord, records []client.DNSRecord) *client.DNSRecord {
	for _, r := range records {
		if r.Hostname == rec.Hostname && r.Type == rec.Type && r.Destination == rec.Destination {
			if rec.Priority != nil {
				if r.Priority == nil || *r.Priority != *rec.Priority {
					continue
				}
			}
			return &r
		}
	}
	return nil
}
