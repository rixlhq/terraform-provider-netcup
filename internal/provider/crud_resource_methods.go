package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func (r *scpCrudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	plan := req.Plan.Raw
	tfType := r.schema.Type().TerraformType(ctx)

	path, err := r.buildPath(plan, r.spec.createPath)
	if err != nil {
		resp.Diagnostics.AddError("Path Error", err.Error())
		return
	}

	body, err := r.buildBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Body Error", err.Error())
		return
	}

	method := r.spec.createMethod
	if method == "" {
		method = "POST"
	}
	respBody, err := r.doRequest(ctx, method, path, body)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	var apiStateVal tftypes.Value
	if len(respBody) > 0 && !r.spec.createReadsBack {
		apiStateVal, err = r.responseToState(ctx, tfType, respBody)
		if err == nil {
			stateVal := overlayKnown(plan, apiStateVal)
			resp.State.Raw = r.applyIdFromAttr(plan, stateVal)
			return
		}
	}

	apiStateVal, err = r.readState(ctx, plan, tfType)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Read Error", err.Error())
		return
	}
	stateVal := overlayKnown(plan, apiStateVal)
	resp.State.Raw = r.applyIdFromAttr(plan, stateVal)
}

func (r *scpCrudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	state := req.State.Raw
	tfType := r.schema.Type().TerraformType(ctx)

	apiStateVal, err := r.readState(ctx, state, tfType)
	if err != nil {
		if scpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	// Preserve path parameters and other known state values the API response
	// does not include.
	stateVal := overlayKnown(state, apiStateVal)
	resp.State.Raw = r.applyIdFromAttr(state, stateVal)
}

func (r *scpCrudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	plan := req.Plan.Raw
	state := req.State.Raw
	tfType := r.schema.Type().TerraformType(ctx)

	// Combine state and plan so that known configuration values (including
	// the computed id and required path parameters) survive API responses
	// that do not echo every attribute.
	combined := overlayKnown(state, plan)

	// Path parameters are known from the combined state/plan, while the
	// request body is built from the planned configuration.
	path, err := r.buildPath(combined, r.spec.updatePath)
	if err != nil {
		resp.Diagnostics.AddError("Path Error", err.Error())
		return
	}

	body, err := r.buildBody(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Body Error", err.Error())
		return
	}

	method := r.spec.updateMethod
	if method == "" {
		method = "PUT"
	}
	respBody, err := r.doRequest(ctx, method, path, body)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	var apiStateVal tftypes.Value
	if len(respBody) > 0 && !r.spec.updateReadsBack {
		apiStateVal, err = r.responseToState(ctx, tfType, respBody)
		if err == nil {
			resp.State.Raw = overlayKnown(combined, apiStateVal)
			return
		}
	}

	apiStateVal, err = r.readState(ctx, combined, tfType)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Read Error", err.Error())
		return
	}
	resp.State.Raw = overlayKnown(combined, apiStateVal)
}

func (r *scpCrudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.spec.noDelete {
		resp.State.RemoveResource(ctx)
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	path, err := r.buildPath(req.State.Raw, r.spec.deletePath)
	if err != nil {
		resp.Diagnostics.AddError("Path Error", err.Error())
		return
	}

	method := r.spec.deleteMethod
	if method == "" {
		method = methodDelete
	}
	if _, err := r.doRequest(ctx, method, path, nil); err != nil {
		if !scpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("SCP API Error", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
}
