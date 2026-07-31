package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// NewScpUserSshKeyResource returns a resource that creates and deletes SSH keys
// for an SCP user account.
func NewScpUserSshKeyResource() resource.Resource {
	return &scpCrudResource{
		schema: userSshKeyResourceSchema(),
		spec: scpCrudResourceSpec{
			typeName:               "scp_user_ssh_key",
			createPath:             "/api/v1/users/{user_id}/ssh-keys",
			readPath:               "/api/v1/users/{user_id}/ssh-keys",
			deletePath:             "/api/v1/users/{user_id}/ssh-keys/{id}",
			createMethod:           "POST",
			readMethod:             "GET",
			deleteMethod:           "DELETE",
			pathParams:             []string{"user_id", "id"},
			bodyExclude:            []string{"user_id", "id", "created_at"},
			readFromList:           true,
			listSearchConfigAttr:   "id",
			listSearchResponseAttr: "id",
		},
	}
}

func userSshKeyResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an SSH key for a netcup SCP user account. SSH keys cannot be updated in place; changing `name` or `key` replaces the resource.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the user account that owns the SSH key.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "ID of the SSH key assigned by the API.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the SSH key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The public SSH key.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp of the SSH key.",
				Computed:            true,
			},
		},
	}
}
