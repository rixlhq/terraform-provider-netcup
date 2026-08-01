package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var scpServerActionSchema = schema.Schema{
	MarkdownDescription: "Triggers one-off actions against an existing netcup SCP virtual server, such as start, stop, reset, snapshots, ISO attach, image setup, and disk driver updates.",
	Attributes: map[string]schema.Attribute{
		attrServerID: schema.Int64Attribute{
			Required: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
			Description: "Id of the server to run the action on.",
		},
		"action": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			Description: "Action to execute. Supported: start, stop, reset, powercycle, suspend, rescue_activate, rescue_deactivate, iso_attach, iso_detach, snapshot_create, snapshot_revert, snapshot_export, snapshot_delete, snapshot_dryrun, disk_format, disk_driver_update, image_setup, user_image_setup, storage_optimize, firewall_reapply, firewall_restore.",
		},
		"arguments": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "String arguments used to build the request path or query, e.g. snapshot_name, disk_name, mac, state_option, driver.",
		},
		"body": schema.StringAttribute{
			Optional:    true,
			Description: "JSON request body for actions that require one (snapshot_create, image_setup, iso_attach, etc.).",
		},
		"triggers": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.Map{
				mapplanmodifier.RequiresReplace(),
			},
			Description: "Map of values that, when changed, force the action to run again.",
		},
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "Identifier for this action invocation.",
		},
	},
}
