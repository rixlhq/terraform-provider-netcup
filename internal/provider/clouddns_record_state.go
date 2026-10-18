package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

func (r *cloudDNSRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cloudDNSRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	records, err := activeCloudDNSRecords(ctx, r.client, data.DomainID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Read Error", err.Error())
		return
	}
	record := findCloudDNSRecord(data, records)
	if record == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if err = updateCloudDNSModel(&data, *record); err != nil {
		resp.Diagnostics.AddError("CloudDNS Read Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudDNSRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) == 2 && parts[1] != "" {
		domainID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || domainID < 1 {
			resp.Diagnostics.AddError("Invalid Import ID", "domain_id must be a positive integer.")
			return
		}
		data := cloudDNSRecordModel{ID: types.StringValue(req.ID), DomainID: types.Int64Value(domainID)}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Support composite format: domain_id/name/type/rdata
	if len(parts) >= 4 {
		domainID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || domainID < 1 {
			resp.Diagnostics.AddError("Invalid Import ID", "domain_id must be a positive integer.")
			return
		}
		name := parts[1]
		recType := strings.ToUpper(parts[2])
		rdata := strings.Join(parts[3:], "/")
		data := cloudDNSRecordModel{
			ID:       types.StringValue(req.ID),
			DomainID: types.Int64Value(domainID),
			Name:     types.StringValue(name),
			Type:     types.StringValue(recType),
			RData:    types.StringValue(rdata),
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Expected either domain_id/record_identifier (e.g. '12345/rec-uuid') or domain_id/name/type/rdata (e.g. '12345/www/A/192.0.2.1').",
	)
}

func findCloudDNSRecord(data cloudDNSRecordModel, records []clouddnsclient.Record) *clouddnsclient.Record {
	id := strings.TrimPrefix(data.ID.ValueString(), strconv.FormatInt(data.DomainID.ValueInt64(), 10)+"/")
	for _, record := range records {
		if record.Identifier == id {
			return &record
		}
	}
	// Record identifiers may change across revisions; the record's DNS identity survives.
	for _, record := range records {
		if record.Name == data.Name.ValueString() && record.Type == data.Type.ValueString() && record.RData == data.RData.ValueString() {
			return &record
		}
	}
	return nil
}

func updateCloudDNSModel(data *cloudDNSRecordModel, record clouddnsclient.Record) error {
	if record.Identifier == "" || record.TTL == nil {
		return errors.New("CloudDNS record has no identifier or explicit TTL")
	}
	data.ID = types.StringValue(fmt.Sprintf("%d/%s", data.DomainID.ValueInt64(), record.Identifier))
	data.Name = types.StringValue(record.Name)
	data.Type = types.StringValue(record.Type)
	data.RData = types.StringValue(record.RData)
	data.TTL = types.Int64Value(*record.TTL)
	return nil
}

func cloudDNSInput(data cloudDNSRecordModel) clouddnsclient.RecordInput {
	return clouddnsclient.RecordInput{Name: data.Name.ValueString(), Type: data.Type.ValueString(), RData: data.RData.ValueString(), TTL: data.TTL.ValueInt64()}
}
