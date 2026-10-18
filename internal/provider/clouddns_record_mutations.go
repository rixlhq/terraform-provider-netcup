package provider

import (
	"context"
	"errors"
	"fmt"

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
	if committed, err := r.change(ctx, &data, nil); err != nil {
		if committed {
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}
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
	if committed, err := r.change(ctx, &data, &old); err != nil {
		if committed {
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}
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

func (r *cloudDNSRecordResource) change(ctx context.Context, data, old *cloudDNSRecordModel) (bool, error) {
	if err := r.checkChange(ctx, *data, old); err != nil {
		return false, err
	}
	var remove []clouddnsclient.RecordInput
	if old != nil {
		remove = []clouddnsclient.RecordInput{cloudDNSInput(*old)}
	}
	revision, err := r.client.ChangeRecords(ctx, data.DomainID.ValueInt64(), []clouddnsclient.RecordInput{cloudDNSInput(*data)}, remove)
	if err != nil {
		return false, err
	}
	data.ID = types.StringValue(fmt.Sprintf("%d/pending", data.DomainID.ValueInt64()))
	records, err := r.client.Records(ctx, data.DomainID.ValueInt64(), revision.Identifier)
	if err != nil {
		return true, err
	}
	// An update creates a new record identifier; match the desired content instead of the old ID.
	desired := *data
	desired.ID = types.StringNull()
	record := findCloudDNSRecord(desired, records)
	if record == nil {
		return true, errors.New("CloudDNS changeset succeeded but the requested record was not returned")
	}
	return true, updateCloudDNSModel(data, *record)
}

func (r *cloudDNSRecordResource) checkChange(ctx context.Context, data cloudDNSRecordModel, old *cloudDNSRecordModel) error {
	records, err := activeCloudDNSRecords(ctx, r.client, data.DomainID.ValueInt64())
	if err != nil {
		return err
	}
	desired := data
	desired.ID = types.StringNull()
	existing := findCloudDNSRecord(desired, records)
	if old == nil {
		if existing != nil {
			return errors.New("CloudDNS record already exists; import it before managing it")
		}
		return nil
	}
	previous := findCloudDNSRecord(*old, records)
	if previous == nil || previous.TTL == nil {
		return errors.New("CloudDNS record changed since planning; refresh the plan before updating")
	}
	if previous.Immutable {
		return errors.New("immutable CloudDNS records cannot be updated")
	}
	current := *old
	if err = updateCloudDNSModel(&current, *previous); err != nil {
		return err
	}
	if cloudDNSInput(current) != cloudDNSInput(*old) {
		return errors.New("CloudDNS record changed since planning; refresh the plan before updating")
	}
	if existing != nil && existing.Identifier != previous.Identifier {
		return errors.New("CloudDNS record already exists; refusing to replace another record")
	}
	return nil
}
