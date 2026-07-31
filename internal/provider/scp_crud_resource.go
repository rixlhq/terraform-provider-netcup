package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

// scpCrudResourceSpec describes a single SCP REST resource. Path templates use
// placeholders like {user_id} that are matched against the top-level Terraform
// attribute names (snake_case). responseRoot is a dot-separated path into the
// JSON response that contains the actual resource object (e.g. "firewallPolicy").
type scpCrudResourceSpec struct {
	typeName     string
	createPath   string
	readPath     string
	updatePath   string
	deletePath   string
	createMethod string
	readMethod   string
	updateMethod string
	deleteMethod string
	responseRoot string
	pathParams   []string
	bodyExclude  []string
}

// scpCrudResource is a generic Terraform resource backed by one SCP REST
// entity. It operates directly on tftypes.Value so it can reuse a generated
// Terraform schema without requiring a typed model for every resource.
type scpCrudResource struct {
	client *scpclient.Client
	schema schema.Schema
	spec   scpCrudResourceSpec
}

func (r *scpCrudResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.typeName
}

func (r *scpCrudResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *scpCrudResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.SCP
}

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

	body, err := r.buildBody(plan)
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

	stateVal, err := r.responseToState(ctx, tfType, respBody)
	if err != nil {
		resp.Diagnostics.AddError("State Error", err.Error())
		return
	}

	resp.State.Raw = stateVal
}

func (r *scpCrudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	state := req.State.Raw
	tfType := r.schema.Type().TerraformType(ctx)

	path, err := r.buildPath(state, r.spec.readPath)
	if err != nil {
		resp.Diagnostics.AddError("Path Error", err.Error())
		return
	}

	method := r.spec.readMethod
	if method == "" {
		method = "GET"
	}
	respBody, err := r.doRequest(ctx, method, path, nil)
	if err != nil {
		if scpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("SCP API Error", err.Error())
		return
	}

	stateVal, err := r.responseToState(ctx, tfType, respBody)
	if err != nil {
		resp.Diagnostics.AddError("State Error", err.Error())
		return
	}

	resp.State.Raw = stateVal
}

func (r *scpCrudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing SCP Client", "Configure the provider with scp_access_token or scp_refresh_token to use this resource.")
		return
	}

	plan := req.Plan.Raw
	state := req.State.Raw
	tfType := r.schema.Type().TerraformType(ctx)

	// Path parameters like the computed id are known from state, while the
	// request body is built from the planned configuration.
	path, err := r.buildPath(state, r.spec.updatePath)
	if err != nil {
		resp.Diagnostics.AddError("Path Error", err.Error())
		return
	}

	body, err := r.buildBody(plan)
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

	// Some updates return the resource object, others return a task wrapper.
	if len(respBody) > 0 {
		stateVal, err := r.responseToState(ctx, tfType, respBody)
		if err == nil {
			resp.State.Raw = stateVal
			return
		}
		// If the response does not match the resource schema, fall back to
		// reading the resource back from the API.
	}

	readPath, err := r.buildPath(state, r.spec.readPath)
	if err != nil {
		resp.Diagnostics.AddError("Read Path Error", err.Error())
		return
	}
	readBody, err := r.doRequest(ctx, "GET", readPath, nil)
	if err != nil {
		resp.Diagnostics.AddError("SCP API Read Error", err.Error())
		return
	}
	stateVal, err := r.responseToState(ctx, tfType, readBody)
	if err != nil {
		resp.Diagnostics.AddError("State Error", err.Error())
		return
	}
	resp.State.Raw = stateVal
}

func (r *scpCrudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
		method = "DELETE"
	}
	if _, err := r.doRequest(ctx, method, path, nil); err != nil {
		if !scpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("SCP API Error", err.Error())
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *scpCrudResource) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	switch method {
	case "GET":
		return r.client.Get(ctx, path, nil)
	case "POST":
		return r.client.Post(ctx, path, body)
	case "PUT":
		return r.client.Put(ctx, path, body)
	case "PATCH":
		return r.client.Patch(ctx, path, body)
	case "DELETE":
		return r.client.Delete(ctx, path)
	default:
		return nil, fmt.Errorf("unsupported HTTP method %q", method)
	}
}

func (r *scpCrudResource) buildPath(v tftypes.Value, template string) (string, error) {
	obj, err := r.asObject(v)
	if err != nil {
		return "", err
	}

	path := template
	for _, attr := range r.spec.pathParams {
		placeholder := "{" + attr + "}"
		if !strings.Contains(path, placeholder) {
			// Not all methods use every path parameter (e.g. Create does not
			// need a computed id). Skip placeholders that are absent from the
			// template for this method.
			continue
		}
		val, ok := obj[attr]
		if !ok {
			return "", fmt.Errorf("missing path parameter %q", attr)
		}
		str, err := r.valueAsString(val)
		if err != nil {
			return "", fmt.Errorf("path parameter %q: %w", attr, err)
		}
		path = strings.Replace(path, placeholder, str, 1)
	}

	return path, nil
}

func (r *scpCrudResource) buildBody(v tftypes.Value) ([]byte, error) {
	exclude := make(map[string]bool, len(r.spec.pathParams)+len(r.spec.bodyExclude))
	for _, attr := range r.spec.pathParams {
		// TfValueToJSON converts snake_case attribute names to camelCase JSON keys.
		exclude[scpcommon.SnakeToCamel(attr)] = true
	}
	for _, attr := range r.spec.bodyExclude {
		exclude[scpcommon.SnakeToCamel(attr)] = true
	}

	converted, err := scpcommon.TfValueToJSON(context.Background(), v)
	if err != nil {
		return nil, err
	}
	bodyMap, ok := converted.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected object body, got %T", converted)
	}
	for attr := range bodyMap {
		if exclude[attr] {
			delete(bodyMap, attr)
		}
	}

	if len(bodyMap) == 0 {
		return nil, nil
	}
	return json.Marshal(bodyMap)
}

func (r *scpCrudResource) responseToState(ctx context.Context, tfType tftypes.Type, body []byte) (tftypes.Value, error) {
	decoded, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		return tftypes.Value{}, err
	}

	if r.spec.responseRoot != "" {
		m, ok := decoded.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected object response for root %q", r.spec.responseRoot)
		}
		for _, part := range strings.Split(r.spec.responseRoot, ".") {
			if part == "" {
				continue
			}
			next, ok := m[part]
			if !ok {
				// Some methods return the resource directly while others wrap it.
				// Fall back to the top-level response when the configured root is
				// not present.
				return scpcommon.JSONToTfValue(ctx, tfType, decoded)
			}
			m, ok = next.(map[string]interface{})
			if !ok {
				return scpcommon.JSONToTfValue(ctx, tfType, decoded)
			}
		}
		decoded = m
	}

	return scpcommon.JSONToTfValue(ctx, tfType, decoded)
}

func (r *scpCrudResource) asObject(v tftypes.Value) (map[string]tftypes.Value, error) {
	if v.IsNull() || !v.IsKnown() {
		return nil, fmt.Errorf("value is null or unknown")
	}
	objType, ok := v.Type().(tftypes.Object)
	if !ok {
		return nil, fmt.Errorf("expected object value, got %s", v.Type())
	}
	m := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	if err := v.As(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *scpCrudResource) valueAsString(v tftypes.Value) (string, error) {
	if v.IsNull() || !v.IsKnown() {
		return "", fmt.Errorf("value is null or unknown")
	}
	t := v.Type()
	if t.Is(tftypes.String) {
		var s string
		if err := v.As(&s); err != nil {
			return "", err
		}
		return s, nil
	}
	if t.Is(tftypes.Number) {
		var n big.Float
		if err := v.As(&n); err != nil {
			return "", err
		}
		return n.Text('f', 0), nil
	}
	if t.Is(tftypes.Bool) {
		var b bool
		if err := v.As(&b); err != nil {
			return "", err
		}
		return fmt.Sprintf("%t", b), nil
	}
	return "", fmt.Errorf("cannot use %s as path parameter", t)
}
