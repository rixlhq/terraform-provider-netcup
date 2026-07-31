package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var scpUserFirewallPolicyRulesAttributes = map[string]schema.Attribute{
	"action": schema.StringAttribute{
		Required:    true,
		Description: "Firewall action. ACCEPT or DROP.",
	},
	"description": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Rule description.",
	},
	"destination_ports": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Destination port or range.",
	},
	"destinations": schema.ListAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Computed:    true,
		Description: "Destination CIDRs/addresses.",
	},
	"direction": schema.StringAttribute{
		Required:    true,
		Description: "Rule direction. INGRESS or EGRESS.",
	},
	"number_of_effective_rules": schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Description: "Number of effective rules generated from this rule.",
	},
	"protocol": schema.StringAttribute{
		Required:    true,
		Description: "Protocol. TCP, UDP, ICMP or ICMPv6.",
	},
	"source_ports": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Source port or range.",
	},
	"sources": schema.ListAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Computed:    true,
		Description: "Source CIDRs/addresses.",
	},
}

var scpUserFirewallPolicySchema = schema.Schema{
	Description: "Manages a firewall policy in the netcup SCP account.",
	Attributes: map[string]schema.Attribute{
		"description": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Policy description.",
		},
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "SCP firewall policy id.",
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Policy name.",
		},
		"rules": schema.ListNestedAttribute{
			Optional: true,
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: scpUserFirewallPolicyRulesAttributes,
			},
			Description: "Firewall rules, evaluated in order.",
		},
		attrUserID: schema.Int64Attribute{
			Required:    true,
			Description: "Id of the SCP user account that owns the policy.",
		},
	},
}

var scpUserFirewallPolicySpec = scpCrudResourceSpec{
	typeName:     "scp_user_firewall_policy",
	createPath:   "/api/v1/users/{user_id}/firewall-policies",
	readPath:     "/api/v1/users/{user_id}/firewall-policies/{id}",
	updatePath:   "/api/v1/users/{user_id}/firewall-policies/{id}",
	deletePath:   "/api/v1/users/{user_id}/firewall-policies/{id}",
	createMethod: "POST",
	readMethod:   "GET",
	updateMethod: "PUT",
	deleteMethod: methodDelete,
	responseRoot: "firewallPolicy",
	pathParams:   []string{attrUserID, "id"},
	bodyExclude:  []string{attrUserID, "id"},
}

// NewScpUserFirewallPolicyResource returns a resource that manages netcup SCP
// firewall policies for a user account.
func NewScpUserFirewallPolicyResource() resource.Resource {
	return &scpCrudResource{
		schema: scpUserFirewallPolicySchema,
		spec:   scpUserFirewallPolicySpec,
	}
}
