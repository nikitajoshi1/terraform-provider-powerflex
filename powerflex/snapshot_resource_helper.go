package powerflex

import (
	"strconv"

	pftypes "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// READWRITE represents access mode limit of snapshot
	READWRITE = "ReadWrite"
	// READONLY represents access mode limit of snapshot
	READONLY = "ReadOnly"
	// SecondsThreshold represents platform epoch drift
	SecondsThreshold = 300
	// DayInMins represents day in min
	DayInMins = 24 * HourInMins
	// HourInMins represents hour in min
	HourInMins = 60
	// MinuteInSeconds represents min in sec.
	MinuteInSeconds = 60
)

// SnapshotResourceModel maps the resource schema data.
type SnapshotResourceModel struct {
	Name             types.String `tfsdk:"name"`
	VolumeID         types.String `tfsdk:"volume_id"`
	VolumeName       types.String `tfsdk:"volume_name"`
	AccessMode       types.String `tfsdk:"access_mode"`
	ID               types.String `tfsdk:"id"`
	Size             types.Int64  `tfsdk:"size"`
	CapacityUnit     types.String `tfsdk:"capacity_unit"`
	SizeInKb         types.Int64  `tfsdk:"size_in_kb"`
	LockAutoSnapshot types.Bool   `tfsdk:"lock_auto_snapshot"`
	SdcList          types.Set    `tfsdk:"sdc_list"`
	RemoveMode       types.String `tfsdk:"remove_mode"`
	DesiredRetention types.Int64  `tfsdk:"desired_retention"`
	RetentionUnit    types.String `tfsdk:"retention_unit"`
	RetentionInMin   types.String `tfsdk:"retention_in_min"`
}

// SdcList struct for sdc info response mapping to terrafrom
type SdcList struct {
	SdcID         string `tfsdk:"sdc_id"`
	LimitIops     int    `tfsdk:"limit_iops"`
	LimitBwInMbps int    `tfsdk:"limit_bw_in_mbps"`
	SdcName       string `tfsdk:"sdc_name"`
	AccessMode    string `tfsdk:"access_mode"`
}

// SdcInfoAttrTypes for defining sdc list struct into terraform type
var SdcInfoAttrTypes = map[string]attr.Type{
	"sdc_id":           types.StringType,
	"limit_iops":       types.Int64Type,
	"limit_bw_in_mbps": types.Int64Type,
	"sdc_name":         types.StringType,
	"access_mode":      types.StringType,
}

func refreshState(snap *pftypes.Volume, prestate *SnapshotResourceModel) (diags diag.Diagnostics) {
	var drift int64
	prestate.ID = types.StringValue(snap.ID)
	prestate.Name = types.StringValue(snap.Name)
	prestate.AccessMode = types.StringValue(snap.AccessModeLimit)
	prestate.SizeInKb = types.Int64Value(int64(snap.SizeInKb))
	switch prestate.CapacityUnit.ValueString() {
	case "TB":
		prestate.Size = types.Int64Value(int64(snap.SizeInKb / TiKB))
	case "GB":
		prestate.Size = types.Int64Value(int64(snap.SizeInKb / GiKB))
	}
	prestate.LockAutoSnapshot = types.BoolValue(snap.LockedAutoSnapshot)
	diff1 := int64(snap.SecureSnapshotExpTime) - int64(snap.CreationTime)
	if prestate.RetentionUnit.ValueString() == "days" {
		drift = diff1 - prestate.DesiredRetention.ValueInt64()*DayInMins*MinuteInSeconds
	} else {
		drift = diff1 - prestate.DesiredRetention.ValueInt64()*HourInMins*MinuteInSeconds
	}
	if diff1 > 0 && drift > SecondsThreshold && drift < -SecondsThreshold {
		prestate.RetentionInMin = types.StringValue(strconv.FormatInt(diff1/60, 10))
	}
	sdcInfoElemType := types.ObjectType{
		AttrTypes: SdcInfoAttrTypes,
	}
	objectSdcInfos := []attr.Value{}
	for _, msi := range snap.MappedSdcInfo {
		// refreshing state for drift outside terraform
		obj := map[string]attr.Value{
			"sdc_id":           types.StringValue(msi.SdcID),
			"limit_iops":       types.Int64Value(int64(msi.LimitIops)),
			"limit_bw_in_mbps": types.Int64Value(int64(msi.LimitBwInMbps)),
			"sdc_name":         types.StringValue(msi.SdcName),
			"access_mode":      types.StringValue(msi.AccessMode),
		}
		objVal, diag1 := types.ObjectValue(SdcInfoAttrTypes, obj)
		diags = append(diags, diag1...)
		objectSdcInfos = append(objectSdcInfos, objVal)
	}
	mappedSdcInfoVal, diag2 := types.SetValue(sdcInfoElemType, objectSdcInfos)
	diags = append(diags, diag2...)
	prestate.SdcList = mappedSdcInfoVal
	return
}

func convertToMin(desireRetention int64, retentionUnit string) string {
	retentionMin := ""
	if retentionUnit == "days" {
		retentionMin = strconv.FormatInt(desireRetention*DayInMins, 10)
	} else {
		retentionMin = strconv.FormatInt(desireRetention*HourInMins, 10)
	}
	return retentionMin
}

// coverterKB fucntion to convert size into kb
func converterKB(capacityUnit string, size int64) int64 {
	var valInKiB int64
	switch capacityUnit {
	case "TB":
		valInKiB = size * TiKB
	case "GB":
		valInKiB = size * GiKB
	default:
		return 0
	}
	return int64(valInKiB)
}
