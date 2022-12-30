package powerflex

import "github.com/hashicorp/terraform-plugin-framework/types"

// VolumeResourceModel maps the resource schema data.
type VolumeResourceModel struct {
	ProtectionDomainName               types.String `tfsdk:"protection_domain_name"`
	ProtectionDomainID                 types.String `tfsdk:"protection_domain_id"`
	StoragePoolName                    types.String `tfsdk:"storage_pool_name"`
	StoragePoolID                      types.String `tfsdk:"storage_pool_id"`
	VolumeType                         types.String `tfsdk:"volume_type"`
	UseRmCache                         types.Bool   `tfsdk:"use_rm_cache"`
	Size                               types.Int64  `tfsdk:"size"`
	CapacityUnit                       types.String `tfsdk:"capacity_unit"`
	VolumeSizeInKb                     types.String `tfsdk:"volume_size_in_kb"`
	Name                               types.String `tfsdk:"name"`
	MapSdcsID                          types.List   `tfsdk:"map_sdcs_id"`
	MappingToAllSdcsEnabled            types.Bool   `tfsdk:"mapping_to_all_sdcs_enabled"`
	IsObfuscated                       types.Bool   `tfsdk:"is_obfuscated"`
	ConsistencyGroupID                 types.String `tfsdk:"consistency_group_id"`
	VTreeID                            types.String `tfsdk:"vtree_id"`
	AncestorVolumeID                   types.String `tfsdk:"ancestor_volume_id"`
	MappedScsiInitiatorInfo            types.String `tfsdk:"mapped_scsi_initiator_info"`
	SizeInKb                           types.Int64  `tfsdk:"size_in_kb"`
	CreationTime                       types.Int64  `tfsdk:"creation_time"`
	ID                                 types.String `tfsdk:"id"`
	DataLayout                         types.String `tfsdk:"data_layout"`
	NotGenuineSnapshot                 types.Bool   `tfsdk:"not_genuine_snapshot"`
	AccessModeLimit                    types.String `tfsdk:"access_mode_limit"`
	SecureSnapshotExpTime              types.Int64  `tfsdk:"secure_snapshot_exp_time"`
	ManagedBy                          types.String `tfsdk:"managed_by"`
	LockedAutoSnapshot                 types.Bool   `tfsdk:"locked_auto_snapshot"`
	LockedAutoSnapshotMarkedForRemoval types.Bool   `tfsdk:"locked_auto_snapshot_marked_for_removal"`
	CompressionMethod                  types.String `tfsdk:"compression_method"`
	TimeStampIsAccurate                types.Bool   `tfsdk:"time_stamp_is_accurate"`
	OriginalExpiryTime                 types.Int64  `tfsdk:"original_expiry_time"`
	VolumeReplicationState             types.String `tfsdk:"volume_replication_state"`
	ReplicationJournalVolume           types.Bool   `tfsdk:"replication_journal_volume"`
	ReplicationTimeStamp               types.Int64  `tfsdk:"replication_time_stamp"`
	Links                              types.List   `tfsdk:"links"`
	MappedSdcInfo                      types.List   `tfsdk:"mapped_sdc_info"`
}
