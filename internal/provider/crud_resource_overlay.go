package provider

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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
		return "", errors.New("value is null or unknown")
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
		return strconv.FormatBool(b), nil
	}
	return "", fmt.Errorf("cannot use %s as path parameter", t)
}
