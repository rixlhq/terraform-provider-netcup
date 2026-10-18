package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

func (r *cloudDNSRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cloudDNSRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.change(ctx, &data, nil); err != nil {
		resp.Diagnostics.AddError("CloudDNS Create Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudDNSRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, old cloudDNSRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.change(ctx, &data, &old); err != nil {
		resp.Diagnostics.AddError("CloudDNS Update Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cloudDNSRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cloudDNSRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	records, err := activeCloudDNSRecords(ctx, r.client, data.DomainID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Delete Error", err.Error())
		return
	}
	record := findCloudDNSRecord(data, records)
	if record == nil {
		return
	}
	if record.Immutable {
		resp.Diagnostics.AddError("Immutable CloudDNS Record", "System-managed DNS records cannot be deleted.")
		return
	}
	if err = updateCloudDNSModel(&data, *record); err == nil {
		_, err = r.client.ChangeRecords(ctx, data.DomainID.ValueInt64(), nil, []clouddnsclient.RecordInput{cloudDNSInput(data)})
	}
	if err != nil {
		resp.Diagnostics.AddError("CloudDNS Delete Error", err.Error())
	}
}

func (r *cloudDNSRecordResource) change(ctx context.Context, data, old *cloudDNSRecordModel) error {
	var remove []clouddnsclient.RecordInput
	if old != nil {
		remove = []clouddnsclient.RecordInput{cloudDNSInput(*old)}
	}
	revision, err := r.client.ChangeRecords(ctx, data.DomainID.ValueInt64(), []clouddnsclient.RecordInput{cloudDNSInput(*data)}, remove)
	if err != nil {
		return err
	}
	records, err := r.client.Records(ctx, data.DomainID.ValueInt64(), revision.Identifier)
	if err != nil {
		return err
	}
	// An update creates a new record identifier; match the desired content instead of the old ID.
	desired := *data
	desired.ID = types.StringNull()
	record := findCloudDNSRecord(desired, records)
	if record == nil {
		return errors.New("CloudDNS changeset succeeded but the requested record was not returned")
	}
	return updateCloudDNSModel(data, *record)
}
