/*
Copyright (c) 2023 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	RemoveMode       types.String `tfsdk:"remove_mode"`
	DesiredRetention types.Int64  `tfsdk:"desired_retention"`
	RetentionUnit    types.String `tfsdk:"retention_unit"`
	RetentionInMin   types.String `tfsdk:"retention_in_min"`
}

// SdcList struct for sdc info response mapping to terrafrom
type SdcList struct {
	SdcID         types.String `tfsdk:"sdc_id"`
	LimitIops     types.Int64  `tfsdk:"limit_iops"`
	LimitBwInMbps types.Int64  `tfsdk:"limit_bw_in_mbps"`
	SdcName       types.String `tfsdk:"sdc_name"`
	AccessMode    types.String `tfsdk:"access_mode"`
}
