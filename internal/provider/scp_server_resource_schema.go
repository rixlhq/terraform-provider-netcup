package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var scpServerResourceSchema = schema.Schema{
	MarkdownDescription: "Manages mutable attributes of an existing netcup SCP virtual server. Servers cannot be created or deleted through the SCP API; this resource adopts an existing server by `server_id` and applies patches.",
	Attributes: map[string]schema.Attribute{
		attrServerID: schema.Int64Attribute{
			Required: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
			Description: "Id of the server to manage.",
		},
		"hostname": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Server hostname.",
		},
		"nickname": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "User-defined nickname for the server.",
		},
		"autostart": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the server should start automatically.",
		},
		"uefi": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether UEFI is enabled.",
		},
		"bootorder": schema.ListAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Boot order as a list of values such as HDD, CDROM, NETWORK.",
		},
		"os_optimization": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "OS optimization (LINUX, WINDOWS, BSD, LINUX_LEGACY, UNKNOWN).",
		},
		"keyboard_layout": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Keyboard layout used by the virtual console.",
		},
		"cpu_topology": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"socket_count": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Number of CPU sockets.",
				},
				"cores_per_socket_count": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Number of CPU cores per socket.",
				},
			},
			Description: "CPU topology configuration.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Server name as returned by the SCP API.",
		},
		"disabled": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the server is disabled.",
		},
		"rescue_system_active": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the rescue system is active.",
		},
		"snapshot_allowed": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether snapshots are allowed for this server.",
		},
	},
}
