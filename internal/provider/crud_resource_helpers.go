package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

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

func (r *scpCrudResource) buildBody(ctx context.Context, v tftypes.Value) ([]byte, error) {
	// Only bodyExclude is removed from the request body. Path parameters that
	// also belong in the body (e.g. "ip" for rDNS) must not be excluded by
	// default; they can be listed explicitly here when needed.
	exclude := make(map[string]bool, len(r.spec.bodyExclude))
	for _, attr := range r.spec.bodyExclude {
		exclude[scpcommon.SnakeToCamel(attr)] = true
	}

	converted, err := scpcommon.TfValueToJSON(ctx, v)
	if err != nil {
		return nil, err
	}
	bodyMap, ok := converted.(map[string]any)
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

func jsonValueAsString(v any) string {
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

func (r *scpCrudResource) asObject(v tftypes.Value) (map[string]tftypes.Value, error) {
	if v.IsNull() || !v.IsKnown() {
		return nil, errors.New("value is null or unknown")
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
