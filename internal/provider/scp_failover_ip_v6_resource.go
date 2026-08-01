package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// NewScpFailoverIpV6Resource returns a resource that routes an existing IPv6
// failover IP to a server.
func NewScpFailoverIpV6Resource() resource.Resource {
	return &scpCrudResource{
		schema: failoverIpV6ResourceSchema(),
		spec: scpCrudResourceSpec{
			typeName:               "scp_failover_ip_v6",
			createPath:             "/api/v1/users/{user_id}/failoverips/v6/{failover_ip_id}",
			readPath:               "/api/v1/users/{user_id}/failoverips/v6",
			updatePath:             "/api/v1/users/{user_id}/failoverips/v6/{failover_ip_id}",
			createMethod:           "PATCH",
			updateMethod:           "PATCH",
			pathParams:             []string{attrUserID, attrFailoverIPID},
			bodyExclude:            []string{attrUserID, attrFailoverIPID, "network_prefix", "network_prefix_length", "id", "editable"},
			noDelete:               true,
			readFromList:           true,
			listSearchConfigAttr:   attrFailoverIPID,
			listSearchResponseAttr: "id",
			idFromAttr:             attrFailoverIPID,
			importIDAttrs:          []string{attrUserID, attrFailoverIPID},
		},
	}
}

func failoverIpV6ResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Routes an existing netcup SCP IPv6 failover IP to a server. Failover IPs are pre-allocated; this resource only updates the `server_id` routing.",
		Attributes: map[string]schema.Attribute{
			attrUserID: schema.Int64Attribute{
				MarkdownDescription: "ID of the user that owns the failover IP.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			attrFailoverIPID: schema.Int64Attribute{
				MarkdownDescription: "ID of the failover IP to route.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			attrServerID: schema.Int64Attribute{
				MarkdownDescription: "ID of the server to route the failover IP to.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "Terraform identifier for this failover IP.",
				Computed:            true,
			},
			"network_prefix": schema.StringAttribute{
				MarkdownDescription: "The IPv6 network prefix.",
				Computed:            true,
			},
			"network_prefix_length": schema.Int64Attribute{
				MarkdownDescription: "Length of the IPv6 network prefix.",
				Computed:            true,
			},
			"editable": schema.BoolAttribute{
				MarkdownDescription: "Whether the failover IP is editable.",
				Computed:            true,
			},
		},
	}
}
