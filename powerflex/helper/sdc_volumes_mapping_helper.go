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

package helper

import (
	"terraform-provider-powerflex/powerflex/models"

	goscaleio_types "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// GetVolSetValueFromItems return the type for volume list
func GetVolSetValueFromItems(volumes []models.SdcVolumeModel) (basetypes.SetValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	volInfoElemType := types.ObjectType{
		AttrTypes: GetVolType(),
	}

	if len(volumes) == 0 {
		return types.SetNull(volInfoElemType), diags
	}

	objectVolInfos := []attr.Value{}
	for _, vol := range volumes {
		obj := map[string]attr.Value{
			"volume_id":        vol.VolumeID,
			"volume_name":      vol.VolumeName,
			"limit_iops":       vol.IOPSLimit,
			"limit_bw_in_mbps": vol.BWLimit,
			"access_mode":      vol.AccessMode,
		}
		objVal, dgs := types.ObjectValue(GetVolType(), obj)
		diags = append(diags, dgs...)
		objectVolInfos = append(objectVolInfos, objVal)
	}
	mappedSdcInfoVal, dgs := types.SetValue(volInfoElemType, objectVolInfos)
	diags = append(diags, dgs...)
	return mappedSdcInfoVal, diags
}

// GetVolType returns the volume type required for mapping
func GetVolType() map[string]attr.Type {
	return map[string]attr.Type{
		"volume_id":        types.StringType,
		"volume_name":      types.StringType,
		"limit_iops":       types.Int64Type,
		"limit_bw_in_mbps": types.Int64Type,
		"access_mode":      types.StringType,
	}
}

// GetVolValue returns the volume object required for mapping
func GetVolValue(vol *goscaleio_types.Volume) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(GetVolType(), map[string]attr.Value{
		"volume_id":        types.StringValue(vol.ID),
		"volume_name":      types.StringValue(vol.Name),
		"limit_iops":       types.Int64Value(int64(vol.MappedSdcInfo[0].LimitIops)),
		"limit_bw_in_mbps": types.Int64Value(int64(vol.MappedSdcInfo[0].LimitBwInMbps)),
		"access_mode":      types.StringValue(vol.MappedSdcInfo[0].AccessMode),
	})
}

// UpdateSDCVolMapState updates the state
func UpdateSDCVolMapState(mappedVolumes []*goscaleio_types.Volume, plan models.SdcVolumeMappingResourceModel) (models.SdcVolumeMappingResourceModel, diag.Diagnostics) {
	state := plan
	SDCAttrTypes := GetVolType()

	SDCElemType := types.ObjectType{
		AttrTypes: SDCAttrTypes,
	}

	objectSDCs := []attr.Value{}
	var diags diag.Diagnostics
	for _, vol := range mappedVolumes {
		objVal, dgs := GetVolValue(vol)
		diags = append(diags, dgs...)
		objectSDCs = append(objectSDCs, objVal)
		state.Name = types.StringValue(vol.MappedSdcInfo[0].SdcName)
		state.ID = types.StringValue(vol.MappedSdcInfo[0].SdcID)
	}
	setVal, dgs := types.SetValue(SDCElemType, objectSDCs)
	diags = append(diags, dgs...)
	state.VolumeList = setVal

	return state, diags
}
