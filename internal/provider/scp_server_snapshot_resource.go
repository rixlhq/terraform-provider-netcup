package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
	Description: "Manages snapshots of a netcup SCP virtual server. Offline snapshots (online_snapshot = false) require disk_name and restart the server during creation. Online snapshots (online_snapshot = true) capture the running server without a disk selector.",
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
			Description: "Name of the disk to snapshot (e.g. \"vda\"). Required for offline snapshots (online_snapshot = false). Must not be set for online snapshots.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"online_snapshot": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the snapshot should be created while the server is running (default false). When false (offline snapshot), the server will restart and disk_name must be specified.",
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

func validateScpServerSnapshot(_ context.Context, v tftypes.Value) error {
	var obj map[string]tftypes.Value
	if err := v.As(&obj); err != nil {
		return nil
	}

	onlineVal, hasOnline := obj["online_snapshot"]
	diskVal, hasDiskVal := obj["disk_name"]

	if (hasOnline && !onlineVal.IsKnown()) || (hasDiskVal && !diskVal.IsKnown()) {
		return nil
	}

	isOnline := false
	if hasOnline && !onlineVal.IsNull() {
		var b bool
		if err := onlineVal.As(&b); err == nil {
			isOnline = b
		}
	}

	hasDisk := false
	if hasDiskVal && !diskVal.IsNull() {
		var s string
		if err := diskVal.As(&s); err == nil && strings.TrimSpace(s) != "" {
			hasDisk = true
		}
	}

	if !isOnline && !hasDisk {
		return errors.New("disk_name is required when creating an offline snapshot (online_snapshot = false); Netcup requires selecting a specific disk to snapshot offline and will restart the server")
	}

	return nil
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
	importIDAttrs:   []string{attrServerID, "name"},
	createReadsBack: true,
	validate:        validateScpServerSnapshot,
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
