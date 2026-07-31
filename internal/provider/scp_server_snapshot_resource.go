package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var scpServerSnapshotDownloadInfosAttributes = map[string]schema.Attribute{
	"filename": schema.StringAttribute{
		Computed:    true,
		Description: "Filename of the exported snapshot.",
	},
	"presigned_url": schema.StringAttribute{
		Computed:    true,
		Description: "Presigned URL for downloading the exported snapshot.",
	},
	"presigned_url_validity_duration_in_hours": schema.NumberAttribute{
		Computed:    true,
		Description: "Validity of the presigned URL in hours.",
	},
	"headers": schema.MapAttribute{
		ElementType: types.ListType{ElemType: types.StringType},
		Computed:    true,
		Description: "HTTP headers required to download the snapshot.",
	},
}

var scpServerSnapshotSchema = schema.Schema{
	Description: "Manages snapshots of a netcup SCP virtual server.",
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "Terraform identifier, set to the snapshot name.",
		},
		attrServerID: schema.Int64Attribute{
			Required:    true,
			Description: "ID of the server to snapshot.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Name of the snapshot.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Description of the snapshot.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"disk_name": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Name of the disk to snapshot. If omitted the whole server is snapshotted.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"online_snapshot": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the snapshot should be created while the server is running.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
			},
		},
		"uuid": schema.StringAttribute{
			Computed:    true,
			Description: "UUID of the snapshot assigned by SCP.",
		},
		"disks": schema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "List of disk names included in the snapshot.",
		},
		"creation_time": schema.StringAttribute{
			Computed:    true,
			Description: "Timestamp when the snapshot was created.",
		},
		"state": schema.StringAttribute{
			Computed:    true,
			Description: "Current state of the snapshot.",
		},
		"online": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the snapshot was created online.",
		},
		"exported": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the snapshot has been exported.",
		},
		"exported_size_in_ki_b": schema.NumberAttribute{
			Computed:    true,
			Description: "Size of the exported snapshot in KiB.",
		},
		"download_infos": schema.SingleNestedAttribute{
			Computed:    true,
			Attributes:  scpServerSnapshotDownloadInfosAttributes,
			Description: "Download information available after the snapshot is exported.",
		},
	},
}

var scpServerSnapshotSpec = scpCrudResourceSpec{
	typeName:        "scp_server_snapshot",
	createPath:      "/api/v1/servers/{server_id}/snapshots",
	readPath:        "/api/v1/servers/{server_id}/snapshots/{name}",
	deletePath:      "/api/v1/servers/{server_id}/snapshots/{name}",
	createMethod:    "POST",
	readMethod:      "GET",
	deleteMethod:    methodDelete,
	pathParams:      []string{attrServerID, "name"},
	bodyExclude:     []string{attrServerID, "id", "uuid", "disks", "creation_time", "state", "online", "exported", "exported_size_in_ki_b", "download_infos"},
	idFromAttr:      "name",
	createReadsBack: true,
}

// NewScpServerSnapshotResource returns a resource that manages snapshots of an
// SCP virtual server. Snapshots cannot be updated in place, so any change that
// would alter an existing snapshot triggers a replacement.
func NewScpServerSnapshotResource() resource.Resource {
	return &scpCrudResource{
		schema: scpServerSnapshotSchema,
		spec:   scpServerSnapshotSpec,
	}
}
