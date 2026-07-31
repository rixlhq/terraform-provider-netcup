package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// NewScpRdnsResource returns a resource that manages reverse DNS entries for
// IPv4 and IPv6 addresses through the netcup SCP REST API.
func NewScpRdnsResource() resource.Resource {
	return &scpCrudResource{
		schema: schema.Schema{
			Description: "Manages reverse DNS entries for an IPv4 or IPv6 address.",
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					Computed:    true,
					Description: "Terraform identifier, set to the IP address.",
				},
				"ip_version": schema.StringAttribute{
					Required:    true,
					Description: "IP version. Must be ipv4 or ipv6.",
				},
				"ip": schema.StringAttribute{
					Required:    true,
					Description: "The IPv4 or IPv6 address whose rDNS entry is managed.",
				},
				"rdns": schema.StringAttribute{
					Required:    true,
					Description: "Reverse DNS hostname.",
				},
			},
		},
		spec: scpCrudResourceSpec{
			typeName:     "scp_rdns",
			createPath:   "/api/v1/rdns/{ip_version}",
			readPath:     "/api/v1/rdns/{ip_version}/{ip}",
			updatePath:   "/api/v1/rdns/{ip_version}",
			deletePath:   "/api/v1/rdns/{ip_version}/{ip}",
			createMethod: "POST",
			readMethod:   "GET",
			updateMethod: "POST",
			deleteMethod: "DELETE",
			pathParams:   []string{"ip_version", "ip"},
			bodyExclude:  []string{"ip_version", "id"},
			idFromAttr:   "ip",
		},
	}
}
