package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// NewScpUserResource returns an update-only resource for an SCP user account.
func NewScpUserResource() resource.Resource {
	return &scpCrudResource{
		schema: userResourceSchema(),
		spec: scpCrudResourceSpec{
			typeName:     "scp_user",
			createPath:   "/api/v1/users/{user_id}",
			readPath:     "/api/v1/users/{user_id}",
			updatePath:   "/api/v1/users/{user_id}",
			createMethod: "PUT",
			readMethod:   "GET",
			updateMethod: "PUT",
			noDelete:     true,
			pathParams:   []string{"user_id"},
			bodyExclude:  []string{"user_id", "id", "username", "firstname", "lastname", "email", "company"},
		},
	}
}

func userResourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Updates an existing netcup SCP user account. Users cannot be created or deleted through the API; this resource adopts the user by `user_id` and applies `UserSave` fields.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the user account.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "User ID echoed from the API.",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username of the account.",
				Computed:            true,
			},
			"firstname": schema.StringAttribute{
				MarkdownDescription: "First name.",
				Computed:            true,
			},
			"lastname": schema.StringAttribute{
				MarkdownDescription: "Last name.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address.",
				Computed:            true,
			},
			"company": schema.StringAttribute{
				MarkdownDescription: "Company name.",
				Optional:            true,
				Computed:            true,
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "Account language (`en` or `de`).",
				Required:            true,
			},
			"time_zone": schema.StringAttribute{
				MarkdownDescription: "Account time zone.",
				Required:            true,
			},
			"show_nickname": schema.BoolAttribute{
				MarkdownDescription: "Whether to show the server nickname.",
				Optional:            true,
				Computed:            true,
			},
			"passwordless_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether passwordless mode is enabled.",
				Optional:            true,
				Computed:            true,
			},
			"secure_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether secure mode is enabled.",
				Optional:            true,
				Computed:            true,
			},
			"api_ip_login_restrictions": schema.StringAttribute{
				MarkdownDescription: "Comma-separated list of IP addresses allowed for API login.",
				Optional:            true,
				Computed:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "New password for the account.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"old_password": schema.StringAttribute{
				MarkdownDescription: "Old password, required when changing the password.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"soap_webservice_password": schema.StringAttribute{
				MarkdownDescription: "Password for the SOAP web service.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}
