package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// NewScpUserVlanResource returns an update-only resource for a user VLAN's name.
func NewScpUserVlanResource() resource.Resource {
	return &scpCrudResource{
		schema: userVlanResourceSchema(),
		spec: scpCrudResourceSpec{
			typeName:        "scp_user_vlan",
			createPath:      "/api/v1/users/{user_id}/vlans/{vlan_id}",
			readPath:        "/api/v1/users/{user_id}/vlans/{vlan_id}",
			updatePath:      "/api/v1/users/{user_id}/vlans/{vlan_id}",
			createMethod:    "PUT",
			updateMethod:    "PUT",
			readMethod:      "GET",
			createReadsBack: true,
			updateReadsBack: true,
			noDelete:        true,
			pathParams:      []string{attrUserID, "vlan_id"},
			bodyExclude:     []string{attrUserID, "vlan_id", "id"},
			idFromAttr:      "vlan_id",
		},
	}
}

func userVlanResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Updates the name of an existing netcup SCP user VLAN. VLANs cannot be created or deleted through the API.",
		Attributes: map[string]schema.Attribute{
			attrUserID: schema.Int64Attribute{
				MarkdownDescription: "ID of the user that owns the VLAN.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"vlan_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the VLAN to update.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "New name for the VLAN.",
				Required:            true,
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "Terraform identifier for this VLAN.",
				Computed:            true,
			},
		},
	}
}
