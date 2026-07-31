package provider

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildServerPatches(plan, state ScpServerResourceModel) [][]byte {
	var patches [][]byte

	patches = appendStringPatch(patches, plan.Hostname, state.Hostname, "hostname")
	patches = appendStringPatch(patches, plan.Nickname, state.Nickname, "nickname")
	patches = appendBoolPatch(patches, plan.Autostart, state.Autostart, "autostart")
	patches = appendBoolPatch(patches, plan.Uefi, state.Uefi, "uefi")
	patches = appendStringListPatch(patches, plan.Bootorder, state.Bootorder, "bootorder")
	patches = appendStringPatch(patches, plan.OsOptimization, state.OsOptimization, "os_optimization")
	patches = appendStringPatch(patches, plan.KeyboardLayout, state.KeyboardLayout, "keyboardLayout")
	patches = appendCpuTopologyPatch(patches, plan.CpuTopology, state.CpuTopology)

	return patches
}

func appendStringPatch(patches [][]byte, plan, state types.String, key string) [][]byte {
	if !stringChanged(plan, state) {
		return patches
	}
	body, err := json.Marshal(map[string]any{key: plan.ValueString()})
	if err != nil {
		return patches
	}
	return append(patches, body)
}

func appendBoolPatch(patches [][]byte, plan, state types.Bool, key string) [][]byte {
	if !boolChanged(plan, state) {
		return patches
	}
	body, err := json.Marshal(map[string]any{key: plan.ValueBool()})
	if err != nil {
		return patches
	}
	return append(patches, body)
}

func appendStringListPatch(patches [][]byte, plan, state types.List, key string) [][]byte {
	if !stringListChanged(plan, state) {
		return patches
	}
	order, err := stringListValue(plan)
	if err != nil {
		return patches
	}
	body, err := json.Marshal(map[string]any{key: order})
	if err != nil {
		return patches
	}
	return append(patches, body)
}

func appendCpuTopologyPatch(patches [][]byte, plan, state types.Object) [][]byte {
	if !objectChanged(plan, state) {
		return patches
	}
	topo, err := cpuTopologyValue(plan)
	if err != nil {
		return patches
	}
	body, err := json.Marshal(map[string]any{"cpuTopology": topo})
	if err != nil {
		return patches
	}
	return append(patches, body)
}

func stringChanged(a, b types.String) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() && a.ValueString() == b.ValueString() {
		return false
	}
	return true
}

func boolChanged(a, b types.Bool) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() && a.ValueBool() == b.ValueBool() {
		return false
	}
	return true
}

func stringListChanged(a, b types.List) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	la, err := stringListValue(a)
	if err != nil {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() {
		lb, err := stringListValue(b)
		if err == nil && stringSlicesEqual(la, lb) {
			return false
		}
	}
	return true
}

func stringListValue(l types.List) ([]string, error) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	elems := l.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected string element, got %T", e)
		}
		out = append(out, s.ValueString())
	}
	return out, nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func objectChanged(a, b types.Object) bool {
	if a.IsUnknown() || a.IsNull() {
		return false
	}
	if !b.IsNull() && !b.IsUnknown() {
		av, err := objectToMap(a)
		if err != nil {
			return true
		}
		bv, err := objectToMap(b)
		if err == nil && mapsEqual(av, bv) {
			return false
		}
	}
	return true
}

func cpuTopologyValue(o types.Object) (map[string]any, error) {
	m, err := objectToMap(o)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if v, ok := m["socket_count"]; ok {
		out["socketCount"] = v
	}
	if v, ok := m["cores_per_socket_count"]; ok {
		out["coresPerSocketCount"] = v
	}
	return out, nil
}

func objectToMap(o types.Object) (map[string]any, error) {
	if o.IsNull() || o.IsUnknown() {
		return nil, nil
	}
	attrs := o.Attributes()
	out := make(map[string]any, len(attrs))
	for k, v := range attrs {
		switch val := v.(type) {
		case types.Int64:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueInt64()
			}
		case types.String:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueString()
			}
		case types.Bool:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueBool()
			}
		case types.Number:
			if !val.IsNull() && !val.IsUnknown() {
				out[k] = val.ValueBigFloat()
			}
		default:
			return nil, fmt.Errorf("unsupported nested attribute type %T for %s", v, k)
		}
	}
	return out, nil
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		switch ta := va.(type) {
		case int64:
			tb, ok := vb.(int64)
			if !ok || ta != tb {
				return false
			}
		case *big.Float:
			tb, ok := vb.(*big.Float)
			if !ok || ta.Cmp(tb) != 0 {
				return false
			}
		case string:
			tb, ok := vb.(string)
			if !ok || ta != tb {
				return false
			}
		case bool:
			tb, ok := vb.(bool)
			if !ok || ta != tb {
				return false
			}
		default:
			return false
		}
	}
	return true
}
