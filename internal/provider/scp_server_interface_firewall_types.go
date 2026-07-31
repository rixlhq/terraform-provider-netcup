package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var errClientNotConfigured = errors.New("scp client not configured")

type scpServerInterfaceFirewallResourceModel struct {
	ServerID            types.Int64  `tfsdk:"server_id"`
	MAC                 types.String `tfsdk:"mac"`
	Active              types.Bool   `tfsdk:"active"`
	Consistent          types.Bool   `tfsdk:"consistent"`
	CopiedPolicyIDs     types.List   `tfsdk:"copied_policy_ids"`
	UserPolicyIDs       types.List   `tfsdk:"user_policy_ids"`
	IngressImplicitRule types.String `tfsdk:"ingress_implicit_rule"`
	EgressImplicitRule  types.String `tfsdk:"egress_implicit_rule"`
	ID                  types.String `tfsdk:"id"`
}

type firewallPolicyIdentifier struct {
	ID int64 `json:"id"`
}

type firewallSaveRequest struct {
	Active         bool                       `json:"active"`
	CopiedPolicies []firewallPolicyIdentifier `json:"copiedPolicies"`
	UserPolicies   []firewallPolicyIdentifier `json:"userPolicies"`
}

type firewallReadResponse struct {
	Active              bool             `json:"active"`
	Consistent          bool             `json:"consistent"`
	CopiedPolicies      []firewallPolicy `json:"copiedPolicies"`
	UserPolicies        []firewallPolicy `json:"userPolicies"`
	IngressImplicitRule string           `json:"ingressImplicitRule"`
	EgressImplicitRule  string           `json:"egressImplicitRule"`
}

type firewallPolicy struct {
	ID int64 `json:"id"`
}

var scpServerInterfaceFirewallSchemaAttributes = map[string]schema.Attribute{
	"id": schema.StringAttribute{
		Description: "Terraform identifier in the form 'server_id/mac'.",
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"server_id": schema.Int64Attribute{
		Description: "ID of the server.",
		Required:    true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	},
	"mac": schema.StringAttribute{
		Description: "MAC address of the network interface.",
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	},
	"active": schema.BoolAttribute{
		Description: "Whether the firewall is active.",
		Optional:    true,
		Computed:    true,
	},
	"copied_policy_ids": schema.ListAttribute{
		Description: "IDs of copied firewall policies to apply.",
		Optional:    true,
		Computed:    true,
		ElementType: types.Int64Type,
	},
	"user_policy_ids": schema.ListAttribute{
		Description: "IDs of user-defined firewall policies to apply.",
		Optional:    true,
		Computed:    true,
		ElementType: types.Int64Type,
	},
	"consistent": schema.BoolAttribute{
		Description: "Reports whether the firewall configuration is consistent.",
		Computed:    true,
	},
	"ingress_implicit_rule": schema.StringAttribute{
		Description: "Implicit ingress rule, either ACCEPT_ALL or DROP_ALL.",
		Computed:    true,
	},
	"egress_implicit_rule": schema.StringAttribute{
		Description: "Implicit egress rule, either ACCEPT_ALL or DROP_ALL.",
		Computed:    true,
	},
}

func idsToIdentifiers(ids []int64) []firewallPolicyIdentifier {
	out := make([]firewallPolicyIdentifier, len(ids))
	for i, id := range ids {
		out[i] = firewallPolicyIdentifier{ID: id}
	}
	return out
}

func policyIDs(policies []firewallPolicy) []int64 {
	out := make([]int64, len(policies))
	for i, p := range policies {
		out[i] = p.ID
	}
	return out
}

func intListToInt64s(ctx context.Context, list types.List) ([]int64, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	elements := make([]types.Int64, 0, len(list.Elements()))
	diags := list.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	out := make([]int64, len(elements))
	for i, e := range elements {
		out[i] = e.ValueInt64()
	}
	return out, nil
}

// splitImportID splits s by "/" and returns the parts if there are exactly n of them.
func splitImportID(s string, n int) []string {
	parts := strings.SplitN(s, "/", n)
	if len(parts) != n || strings.Contains(parts[n-1], "/") {
		return nil
	}
	return parts
}
