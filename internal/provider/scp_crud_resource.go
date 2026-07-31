package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
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
// idFromAttr copies a known non-computed attribute (e.g. "ip" or "name") into
// the computed "id" attribute for resources whose API response does not echo an id.
// createReadsBack and updateReadsBack force a GET after POST/PUT because some
// SCP endpoints return a task or an empty body rather than the resource object.
type scpCrudResourceSpec struct {
	typeName      string
	createPath    string
	readPath      string
	updatePath    string
	deletePath    string
	createMethod  string
	readMethod    string
	updateMethod  string
	deleteMethod  string
	responseRoot  string
	pathParams    []string
	bodyExclude   []string
	idFromAttr    string
	createReadsBack bool
	updateReadsBack bool
	// noDelete removes the resource from state without calling an API endpoint.
	// Use for update-only resources that cannot be deleted (e.g. VLANs).
	noDelete bool
	// readFromList indicates the read endpoint returns an array and the resource
	// state must be located inside that array. listSearchConfigAttr is the
	// Terraform attribute name to match; listSearchResponseAttr is the matching
	// field in each list item.
	readFromList         bool
	listSearchConfigAttr string
	listSearchResponseAttr string
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

	apiStateVal, err := r.responseToState(ctx, tfType, respBody)
	if err != nil {
		resp.Diagnostics.AddError("State Error", err.Error())
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

func (r *scpCrudResource) readState(ctx context.Context, base tftypes.Value, tfType tftypes.Type) (tftypes.Value, error) {
	path, err := r.buildPath(base, r.spec.readPath)
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("read path: %w", err)
	}
	readBody, err := r.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("scp api read: %w", err)
	}

	if r.spec.readFromList {
		return r.responseFromList(ctx, base, tfType, readBody)
	}
	return r.responseToState(ctx, tfType, readBody)
}

func (r *scpCrudResource) responseFromList(ctx context.Context, base tftypes.Value, tfType tftypes.Type, body []byte) (tftypes.Value, error) {
	decoded, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		return tftypes.Value{}, err
	}
	list, ok := decoded.([]interface{})
	if !ok {
		return tftypes.Value{}, fmt.Errorf("expected list response for readFromList")
	}

	searchVal, err := r.valueAsStringByAttr(base, r.spec.listSearchConfigAttr)
	if err != nil {
		return tftypes.Value{}, fmt.Errorf("search value: %w", err)
	}

	responseAttr := r.spec.listSearchResponseAttr
	if responseAttr == "" {
		responseAttr = r.spec.listSearchConfigAttr
	}

	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		candidate, ok := m[responseAttr]
		if !ok {
			continue
		}
		if jsonValueAsString(candidate) != searchVal {
			continue
		}

		itemJSON, err := json.Marshal(m)
		if err != nil {
			return tftypes.Value{}, err
		}
		return r.responseToState(ctx, tfType, itemJSON)
	}

	return tftypes.Value{}, fmt.Errorf("list item with %s=%s not found", responseAttr, searchVal)
}

func (r *scpCrudResource) valueAsStringByAttr(v tftypes.Value, attr string) (string, error) {
	obj, err := r.asObject(v)
	if err != nil {
		return "", err
	}
	val, ok := obj[attr]
	if !ok {
		return "", fmt.Errorf("missing attribute %q", attr)
	}
	return r.valueAsString(val)
}

func jsonValueAsString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		n := big.NewFloat(t)
		return n.Text('f', 0)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprintf("%v", v)
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
	// Only bodyExclude is removed from the request body. Path parameters that
	// also belong in the body (e.g. "ip" for rDNS) must not be excluded by
	// default; they can be listed explicitly here when needed.
	exclude := make(map[string]bool, len(r.spec.bodyExclude))
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

// overlayKnown returns a new tftypes.Value of the same type as response,
// replacing any null or unknown response attributes with the corresponding
// known value from base. This is needed because SCP API responses often omit
// path parameters (such as user_id) and other configured fields.
func overlayKnown(base, response tftypes.Value) tftypes.Value {
	if !response.IsKnown() || response.IsNull() {
		if base.IsKnown() && !base.IsNull() {
			return base
		}
		return response
	}

	t := response.Type()
	switch ty := t.(type) {
	case tftypes.Object:
		respObj := make(map[string]tftypes.Value)
		_ = response.As(&respObj)

		baseObj := make(map[string]tftypes.Value)
		if base.IsKnown() && !base.IsNull() {
			_ = base.As(&baseObj)
		}

		vals := make(map[string]tftypes.Value, len(ty.AttributeTypes))
		for attr, attrType := range ty.AttributeTypes {
			respAttr, ok := respObj[attr]
			if !ok || !respAttr.IsKnown() || respAttr.IsNull() {
				if baseAttr, ok := baseObj[attr]; ok && baseAttr.IsKnown() && !baseAttr.IsNull() {
					vals[attr] = baseAttr
					continue
				}
				vals[attr] = tftypes.NewValue(attrType, nil)
				continue
			}

			baseAttr := tftypes.NewValue(attrType, nil)
			if b, ok := baseObj[attr]; ok {
				baseAttr = b
			}
			vals[attr] = overlayKnown(baseAttr, respAttr)
		}
		return tftypes.NewValue(t, vals)
	case tftypes.List, tftypes.Set, tftypes.Tuple, tftypes.Map:
		return response
	default:
		return response
	}
}

// applyIdFromAttr copies the value of a source attribute (e.g. "ip") into the
// computed "id" attribute when the API response does not include an id. The value
// is first looked up in the API response, then in the base request value so
// resources whose natural key is only returned by the API (e.g. interface mac)
// still get a stable Terraform id.
func (r *scpCrudResource) applyIdFromAttr(base, v tftypes.Value) tftypes.Value {
	if r.spec.idFromAttr == "" {
		return v
	}
	objType, ok := v.Type().(tftypes.Object)
	if !ok {
		return v
	}
	if _, hasID := objType.AttributeTypes["id"]; !hasID {
		return v
	}

	obj := make(map[string]tftypes.Value)
	_ = v.As(&obj)

	idVal, ok := obj["id"]
	if ok && idVal.IsKnown() && !idVal.IsNull() {
		return v
	}

	srcVal := tftypes.NewValue(objType.AttributeTypes[r.spec.idFromAttr], nil)
	if v.IsKnown() && !v.IsNull() {
		if respObj, ok := obj[r.spec.idFromAttr]; ok && respObj.IsKnown() && !respObj.IsNull() {
			srcVal = respObj
		}
	}

	if !srcVal.IsKnown() || srcVal.IsNull() {
		baseObj := make(map[string]tftypes.Value)
		if base.IsKnown() && !base.IsNull() {
			_ = base.As(&baseObj)
		}
		if b, ok := baseObj[r.spec.idFromAttr]; ok && b.IsKnown() && !b.IsNull() {
			srcVal = b
		}
	}

	if !srcVal.IsKnown() || srcVal.IsNull() {
		return v
	}

	idType := objType.AttributeTypes["id"]
	if !srcVal.Type().Is(idType) {
		return v
	}
	obj["id"] = srcVal
	return tftypes.NewValue(v.Type(), obj)
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
