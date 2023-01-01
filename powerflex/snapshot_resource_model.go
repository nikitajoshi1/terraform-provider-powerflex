package powerflex

import "github.com/hashicorp/terraform-plugin-framework/types"

// SnapshotResourceModel maps the resource schema data.
type SnapshotResourceModel struct {
	Name               types.String `tfsdk:"name"`
	VolumeID           types.String `tfsdk:"volume_id"`
	AccessMode         types.String `tfsdk:"access_mode"`
	ID                 types.String `tfsdk:"id"`
	Size               types.Int64  `tfsdk:"size"`
	CapacityUnit       types.String `tfsdk:"capacity_unit"`
	VolumeSizeInKb     types.String `tfsdk:"volume_size_in_kb"`
	SizeInKb           types.Int64  `tfsdk:"size_in_kb"`
	LockedAutoSnapshot types.Bool   `tfsdk:"locked_auto_snapshot"`
	MapSdcIds          types.List   `tfsdk:"map_sdcs_id"`
}
