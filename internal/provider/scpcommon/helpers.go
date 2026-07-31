package scpcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// DecodeJSONResponse decodes JSON using json.Number to preserve numeric precision.
func DecodeJSONResponse(body []byte) (interface{}, error) {
	var v interface{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// JSONToTfValue converts a JSON-decoded value into a tftypes.Value that matches t.
func JSONToTfValue(ctx context.Context, t tftypes.Type, v interface{}) (tftypes.Value, error) {
	if v == nil {
		return tftypes.NewValue(t, nil), nil
	}

	if t.Is(tftypes.String) {
		s, err := toString(v)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.String, s), nil
	}
	if t.Is(tftypes.Number) {
		n, err := toBigFloat(v)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.Number, n), nil
	}
	if t.Is(tftypes.Bool) {
		b, ok := v.(bool)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected bool, got %T", v)
		}
		return tftypes.NewValue(tftypes.Bool, b), nil
	}
	if t.Is(tftypes.DynamicPseudoType) {
		return dynamicJSONToTfValue(ctx, v)
	}

	switch ty := t.(type) {
	case tftypes.Object:
		m, ok := v.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected object for type %s, got %T", t, v)
		}
		vals := make(map[string]tftypes.Value, len(ty.AttributeTypes))
		for attr, attrType := range ty.AttributeTypes {
			attrVal, ok := m[attr]
			if !ok {
				vals[attr] = tftypes.NewValue(attrType, nil)
				continue
			}
			converted, err := JSONToTfValue(ctx, attrType, attrVal)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[attr] = converted
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.List:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected list for type %s, got %T", t, v)
		}
		vals := make([]tftypes.Value, 0, len(s))
		for _, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Set:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected set for type %s, got %T", t, v)
		}
		vals := make([]tftypes.Value, 0, len(s))
		for _, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Map:
		m, ok := v.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected map for type %s, got %T", t, v)
		}
		vals := make(map[string]tftypes.Value, len(m))
		for k, elem := range m {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[k] = converted
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Tuple:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected tuple for type %s, got %T", t, v)
		}
		if len(s) != len(ty.ElementTypes) {
			return tftypes.Value{}, fmt.Errorf("expected tuple of length %d, got %d", len(ty.ElementTypes), len(s))
		}
		vals := make([]tftypes.Value, 0, len(s))
		for i, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementTypes[i], elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported tftypes.Type %T", t)
	}
}

func toString(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return val.String(), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

func toBigFloat(v interface{}) (*big.Float, error) {
	switch val := v.(type) {
	case json.Number:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val.String(), 10)
		if err != nil {
			return nil, err
		}
		return f, nil
	case float64:
		return big.NewFloat(val).SetPrec(128), nil
	case int:
		return big.NewFloat(float64(val)).SetPrec(128), nil
	case int64:
		return big.NewFloat(float64(val)).SetPrec(128), nil
	case string:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val, 10)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to number", v)
	}
}

func dynamicJSONToTfValue(ctx context.Context, v interface{}) (tftypes.Value, error) {
	switch val := v.(type) {
	case nil:
		return tftypes.NewValue(tftypes.DynamicPseudoType, nil), nil
	case bool:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case json.Number:
		n, err := toBigFloat(val)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, n), nil
	case float64:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(val).SetPrec(128)), nil
	case int:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(float64(val)).SetPrec(128)), nil
	case string:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case []interface{}:
		elems := make([]tftypes.Value, 0, len(val))
		for _, e := range val {
			el, err := dynamicJSONToTfValue(ctx, e)
			if err != nil {
				return tftypes.Value{}, err
			}
			elems = append(elems, el)
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, elems), nil
	case map[string]interface{}:
		vals := make(map[string]tftypes.Value, len(val))
		for k, e := range val {
			el, err := dynamicJSONToTfValue(ctx, e)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[k] = el
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, vals), nil
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported dynamic value type %T", val)
	}
}
