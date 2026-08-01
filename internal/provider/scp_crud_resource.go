package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	typeName        string
	createPath      string
	readPath        string
	updatePath      string
	deletePath      string
	createMethod    string
	readMethod      string
	updateMethod    string
	deleteMethod    string
	responseRoot    string
	pathParams      []string
	bodyExclude     []string
	idFromAttr      string
	createReadsBack bool
	updateReadsBack bool
	// noDelete removes the resource from state without calling an API endpoint.
	// Use for update-only resources that cannot be deleted (e.g. VLANs).
	noDelete bool
	// readFromList indicates the read endpoint returns an array and the resource
	// state must be located inside that array. listSearchConfigAttr is the
	// Terraform attribute name to match; listSearchResponseAttr is the matching
	// field in each list item.
	readFromList           bool
	listSearchConfigAttr   string
	listSearchResponseAttr string
	// importIDAttrs lists the top-level Terraform attribute names, in order,
	// that make up the import ID. The ID is expected as a slash-separated string
	// (e.g. "12345/67890"). If empty, the resource does not support import.
	importIDAttrs []string
}

var _ resource.ResourceWithImportState = &scpCrudResource{}

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

func (r *scpCrudResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}

func (r *scpCrudResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	list, ok := decoded.([]any)
	if !ok {
		return tftypes.Value{}, errors.New("expected list response for readFromList")
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
		m, ok := item.(map[string]any)
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

func (r *scpCrudResource) responseToState(ctx context.Context, tfType tftypes.Type, body []byte) (tftypes.Value, error) {
	decoded, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		return tftypes.Value{}, err
	}

	if r.spec.responseRoot != "" {
		m, ok := decoded.(map[string]any)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected object response for root %q", r.spec.responseRoot)
		}
		for part := range strings.SplitSeq(r.spec.responseRoot, ".") {
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
			m, ok = next.(map[string]any)
			if !ok {
				return scpcommon.JSONToTfValue(ctx, tfType, decoded)
			}
		}
		decoded = m
	}

	return scpcommon.JSONToTfValue(ctx, tfType, decoded)
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
	case methodDelete:
		return r.client.Delete(ctx, path)
	default:
		return nil, fmt.Errorf("unsupported HTTP method %q", method)
	}
}
