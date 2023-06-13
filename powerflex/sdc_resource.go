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

package powerflex

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"terraform-provider-powerflex/helper"
	"time"

	"github.com/dell/goscaleio"
	goscaleio_types "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &sdcResource{}
	_ resource.ResourceWithConfigure   = &sdcResource{}
	_ resource.ResourceWithImportState = &sdcResource{}
)

// SDCResource - function to return resource interface
func SDCResource() resource.Resource {
	return &sdcResource{}
}

// sdcResource - struct to define sdc resource
type sdcResource struct {
	client        *goscaleio.Client
	gatewayClient *goscaleio.GatewayClient
}

// Metadata - function to return metadata for SDC resource.
func (r *sdcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sdc"
}

// Schema - function to return Schema for SDC resource.
func (r *sdcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = SDCReourceSchema
}

// Configure - function to return Configuration for SDC resource.
func (r *sdcResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*goscaleio.Client)

	// Create a new PowerFlex gateway client using the configuration values
	gatewayClient, err := goscaleio.NewGateway(r.client.GetConfigConnect().Endpoint, r.client.GetConfigConnect().Username, r.client.GetConfigConnect().Password, r.client.GetConfigConnect().Insecure, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create gateway API Client",
			"An unexpected error occurred when creating the gateway API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"gateway Client Error: "+err.Error(),
		)
		return
	}

	r.gatewayClient = gatewayClient
}

// Create - function to Create for SDC resource.
func (r *sdcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "[POWERFLEX] Create")

	var plan sdcResourceModel

	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdcDetailList := []SDCDetailDataModel{}
	diags = plan.SDCDetails.ElementsAs(ctx, &sdcDetailList, true)
	resp.Diagnostics.Append(diags...)

	system, err := getFirstSystem(r.client)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error in getting system instance on the PowerFlex cluster",
			err.Error(),
		)
		return
	}

	var chnagedSDCs []SDCDetailDataModel

	if plan.Name.ValueString() != "" && plan.ID.ValueString() != "" {

		nameChng, err := system.ChangeSdcName(plan.ID.ValueString(), plan.Name.ValueString())

		if err != nil {
			resp.Diagnostics.AddError(
				"[Create] Unable to Change name Powerflex sdc",
				err.Error(),
			)
			return
		}

		tflog.Debug(ctx, "[POWERFLEX] nameChng Result :-- "+helper.PrettyJSON(nameChng))

		finalSDC, err := system.GetSdcByID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Changed SDC",
				err.Error(),
			)
			return
		}

		changedSDCDetail := getSDCState(*finalSDC.Sdc, SDCDetailDataModel{})

		changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

		chnagedSDCs = append(chnagedSDCs, changedSDCDetail)

		data, dgs := updateState(chnagedSDCs, plan)
		resp.Diagnostics.Append(dgs...)

		diags = resp.State.Set(ctx, data)
		resp.Diagnostics.Append(diags...)

	} else if len(sdcDetailList) > 0 {

		resp.Diagnostics.Append(r.SDCExpansionOperations(ctx, plan, system, sdcDetailList, &chnagedSDCs)...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(r.UpdateSDCNamdPerfProfileOperations(ctx, sdcDetailList, system, &chnagedSDCs)...)

		data, dgs := updateState(chnagedSDCs, plan)
		resp.Diagnostics.Append(dgs...)

		diags = resp.State.Set(ctx, data)
		resp.Diagnostics.Append(diags...)

		tflog.Info(ctx, "SDC Details updated to state file successfully")

		return
	}
}

// Read - function to Read for SDC resource.
func (r *sdcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "[POWERFLEX] Read")
	var state sdcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdcDetailList := []SDCDetailDataModel{}
	diags = state.SDCDetails.ElementsAs(ctx, &sdcDetailList, true)
	resp.Diagnostics.Append(diags...)

	system, err := getFirstSystem(r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error in getting system instance on the PowerFlex cluster",
			err.Error(),
		)
		return
	}

	var chnagedSDCs []SDCDetailDataModel

	//For handling the import case
	if state.ID.ValueString() != "" && state.ID.ValueString() != "placeholder" && (state.Name.ValueString() == "" || state.Name.IsNull()) {

		for _, id := range strings.Split(state.ID.ValueString(), ",") {
			sdcData, err := system.GetSdcByID(id)

			if err != nil {
				resp.Diagnostics.AddError(
					"[Import] Unable to Find SDC by ID:"+id,
					err.Error(),
				)
				return
			}

			if sdcData != nil {
				changedSDCDetail := getSDCState(*sdcData.Sdc, SDCDetailDataModel{})

				chnagedSDCs = append(chnagedSDCs, changedSDCDetail)
			}
		}
	} else if state.Name.ValueString() != "" && !state.Name.IsNull() && state.ID.ValueString() != "" && state.ID.ValueString() != "placeholder" {

		//For handling the single SDC reanme operation
		singleSdc, err := system.FindSdc("ID", state.ID.ValueString())

		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Powerflex systems-sdcs Read",
				err.Error(),
			)
			return
		}

		changedSDCDetail := getSDCState(*singleSdc.Sdc, SDCDetailDataModel{})

		chnagedSDCs = append(chnagedSDCs, changedSDCDetail)
	} else if len(sdcDetailList) > 0 {

		//For handling the multiple sdc_details update
		for _, sdc := range sdcDetailList {

			var sdcData *goscaleio.Sdc

			if sdc.SDCID.ValueString() != "" {
				sdcData, err = system.GetSdcByID(sdc.SDCID.ValueString())

				if err != nil {
					resp.Diagnostics.AddError(
						"[Read] Unable to Find SDC by ID:"+sdc.SDCID.ValueString(),
						err.Error(),
					)
				}
			} else if sdc.IP.ValueString() != "" {
				sdcData, err = system.FindSdc("SdcIP", sdc.IP.ValueString())

				if err != nil {
					resp.Diagnostics.AddError(
						"[Read] Unable to Find SDC by IP:"+sdc.IP.ValueString(),
						err.Error(),
					)
				}
			} else if sdc.SDCName.ValueString() != "" {
				sdcData, err = system.FindSdc("Name", sdc.SDCName.ValueString())

				if err != nil {
					resp.Diagnostics.AddError(
						"[Read] Unable to Find SDC by Name:"+sdc.SDCName.ValueString(),
						err.Error(),
					)
				}
			}

			if sdcData != nil {
				changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

				chnagedSDCs = append(chnagedSDCs, changedSDCDetail)
			}
		}
	} else {
		resp.Diagnostics.AddError("[Read] Please provide valid SDC ID", "Please provide valid SDC ID")

		return
	}

	data, dgs := updateState(chnagedSDCs, state)
	resp.Diagnostics.Append(dgs...)

	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update - function to Update for SDC resource.
func (r *sdcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "[POWERFLEX] Update")
	// Retrieve values from plan
	var plan sdcResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state sdcResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	system, err := getFirstSystem(r.client)

	planSdcDetailList := []SDCDetailDataModel{}
	diags = plan.SDCDetails.ElementsAs(ctx, &planSdcDetailList, true)
	resp.Diagnostics.Append(diags...)

	stateSdcDetailList := []SDCDetailDataModel{}
	diags = state.SDCDetails.ElementsAs(ctx, &stateSdcDetailList, true)
	resp.Diagnostics.Append(diags...)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error in getting system instance on the PowerFlex cluster",
			err.Error(),
		)
		return
	}

	var chnagedSDCs []SDCDetailDataModel

	deletedSDC := findDeletedSDC(stateSdcDetailList, planSdcDetailList)

	if !(plan.Name.ValueString() != "" && plan.ID.ValueString() != "") {
		if len(deletedSDC) > 0 {

			for _, sdc := range deletedSDC {
				err := system.DeleteSdc(sdc.SDCID.ValueString())

				if err != nil {
					resp.Diagnostics.AddError(
						"[Update] Unable to Delete SDC by ID:"+sdc.SDCID.ValueString(),
						err.Error(),
					)
					return
				}
			}
		}
	}

	if plan.Name.ValueString() != "" && plan.ID.ValueString() != "" {

		nameChng, err := system.ChangeSdcName(plan.ID.ValueString(), plan.Name.ValueString())

		if err != nil {
			resp.Diagnostics.AddError(
				"[Update] Unable to Change name Powerflex sdc",
				err.Error(),
			)
			return
		}

		tflog.Debug(ctx, "[POWERFLEX] nameChng Result :-- "+helper.PrettyJSON(nameChng))

		finalSDC, err := system.GetSdcByID(plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"[Update] Unable to Read Changed SDC",
				err.Error(),
			)
			return
		}

		changedSDCDetail := getSDCState(*finalSDC.Sdc, SDCDetailDataModel{})

		if changedSDCDetail.LastUpdated.ValueString() == "" {
			changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
		}

		chnagedSDCs = append(chnagedSDCs, changedSDCDetail)

		data, dgs := updateState(chnagedSDCs, plan)
		resp.Diagnostics.Append(dgs...)

		diags = resp.State.Set(ctx, data)
		resp.Diagnostics.Append(diags...)

		return

	} else if len(planSdcDetailList) > 0 {

		resp.Diagnostics.Append(r.SDCExpansionOperations(ctx, plan, system, planSdcDetailList, &chnagedSDCs)...)
		if resp.Diagnostics.HasError() {

			//Handling the existing state file data
			for _, sdc := range planSdcDetailList {
				sdcData, _ := system.FindSdc("SdcIP", sdc.IP.ValueString())

				if sdcData != nil {
					changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

					if changedSDCDetail.LastUpdated.ValueString() == "" {
						changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
					}

					chnagedSDCs = append(chnagedSDCs, changedSDCDetail)
				}
			}

			data, dgs := updateState(chnagedSDCs, plan)
			resp.Diagnostics.Append(dgs...)

			diags = resp.State.Set(ctx, data)
			resp.Diagnostics.Append(diags...)

			return
		}

		resp.Diagnostics.Append(r.UpdateSDCNamdPerfProfileOperations(ctx, planSdcDetailList, system, &chnagedSDCs)...)

		data, dgs := updateState(chnagedSDCs, plan)
		resp.Diagnostics.Append(dgs...)

		diags = resp.State.Set(ctx, data)
		resp.Diagnostics.Append(diags...)

		tflog.Info(ctx, "SDC Details updated to state file successfully")

		return
	}
}

// Delete - function to Delete for SDC resource.
func (r *sdcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "[POWERFLEX] Delete")
	var state sdcResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sdcDetailList := []SDCDetailDataModel{}
	diags = state.SDCDetails.ElementsAs(ctx, &sdcDetailList, true)
	resp.Diagnostics.Append(diags...)

	system, err := getFirstSystem(r.client)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error in getting system instance on the PowerFlex cluster",
			err.Error(),
		)
		return
	}

	for _, sdc := range sdcDetailList {
		err := system.DeleteSdc(sdc.SDCID.ValueString())

		if err != nil {
			resp.Diagnostics.AddError(
				"[Delete] Unable to Delete SDC by ID:"+sdc.SDCID.ValueString(),
				err.Error(),
			)
			return
		}
	}

	resp.State.RemoveResource(ctx)

}

// ImportState - function to ImportState for SDC resource.
func (r *sdcResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "[POWERFLEX] ImportState :-- "+helper.PrettyJSON(req))
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// findDeletedSDC function to find deleted SDC Details in Plan
func findDeletedSDC(state, plan []SDCDetailDataModel) []SDCDetailDataModel {
	difference := []SDCDetailDataModel{}

	for _, obj1 := range state {
		found := false
		for _, obj2 := range plan {
			if obj2.IP.ValueString() != "" && obj1.IP == obj2.IP {
				found = true
				break
			} else if obj2.SDCID.ValueString() != "" && obj1.SDCID == obj2.SDCID {
				found = true
				break
			} else if obj2.SDCName.ValueString() != "" && obj1.SDCName == obj2.SDCName {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, obj1)
		}
	}

	return difference
}

// getSDCDetailType returns the SDC Detail type
func getSDCDetailType() map[string]attr.Type {
	return map[string]attr.Type{
		"sdc_id":               types.StringType,
		"ip":                   types.StringType,
		"username":             types.StringType,
		"password":             types.StringType,
		"operating_system":     types.StringType,
		"is_mdm_or_tb":         types.StringType,
		"is_sdc":               types.StringType,
		"performance_profile":  types.StringType,
		"name":                 types.StringType,
		"system_id":            types.StringType,
		"sdc_approved":         types.BoolType,
		"on_vmware":            types.BoolType,
		"sdc_guid":             types.StringType,
		"mdm_connection_state": types.StringType,
		"last_updated":         types.StringType,
	}
}

// getSDCDetailValue returns the SDC Detail model object value
func getSDCDetailValue(sdc SDCDetailDataModel) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValue(getSDCDetailType(), map[string]attr.Value{
		"sdc_id":               types.StringValue(sdc.SDCID.ValueString()),
		"ip":                   types.StringValue(sdc.IP.ValueString()),
		"username":             types.StringValue(sdc.UserName.ValueString()),
		"password":             types.StringValue(sdc.Password.ValueString()),
		"operating_system":     types.StringValue(sdc.OperatingSystem.ValueString()),
		"is_mdm_or_tb":         types.StringValue(sdc.IsMdmOrTb.ValueString()),
		"is_sdc":               types.StringValue(sdc.IsSdc.ValueString()),
		"performance_profile":  types.StringValue(sdc.PerformanceProfile.ValueString()),
		"name":                 types.StringValue(sdc.SDCName.ValueString()),
		"system_id":            types.StringValue(sdc.SystemID.ValueString()),
		"sdc_approved":         types.BoolValue(sdc.SdcApproved.ValueBool()),
		"on_vmware":            types.BoolValue(sdc.OnVMWare.ValueBool()),
		"sdc_guid":             types.StringValue(sdc.SdcGUID.ValueString()),
		"mdm_connection_state": types.StringValue(sdc.MdmConnectionState.ValueString()),
		"last_updated":         types.StringValue(sdc.LastUpdated.ValueString()),
	})
}

// updateState - function to update state file for SDC resource.
func updateState(sdcs []SDCDetailDataModel, plan sdcResourceModel) (sdcResourceModel, diag.Diagnostics) {
	state := plan
	var diags diag.Diagnostics

	SDCAttrTypes := getSDCDetailType()

	SDCElemType := types.ObjectType{
		AttrTypes: SDCAttrTypes,
	}

	objectSDCs := []attr.Value{}
	for _, sdc := range sdcs {
		objVal, dgs := getSDCDetailValue(sdc)
		diags = append(diags, dgs...)
		objectSDCs = append(objectSDCs, objVal)
	}
	setSdcs, dgs := types.ListValue(SDCElemType, objectSDCs)
	diags = append(diags, dgs...)

	state.SDCDetails = setSdcs

	if plan.ID.ValueString() != "" && len(strings.Split(plan.ID.ValueString(), ",")) == 1 {
		state.ID = plan.ID
	} else {
		state.ID = types.StringValue("placeholder")
	}

	return state, diags
}

// GetMDMIP function is used for fetch MDM IP from cluster details
func GetMDMIP(ctx context.Context, sdcDetails []SDCDetailDataModel) (string, error) {
	var mdmIP string

	for _, item := range sdcDetails {
		if strings.EqualFold(item.IsMdmOrTb.ValueString(), "Primary") {
			mdmIP = item.IP.ValueString()
			return mdmIP, nil
		}
	}
	return mdmIP, nil
}

// CheckForExpansion function is used for check for expansion
func CheckForExpansion(model []SDCDetailDataModel) bool {
	performaneChangeSdc := false

	for _, item := range model {
		if strings.EqualFold(item.IsSdc.ValueString(), "Yes") {
			performaneChangeSdc = true
			break
		}
	}
	return performaneChangeSdc
}

// ResetInstallerQueue function for the Abort, Clear and Move To Idle Execution
func ResetInstallerQueue(gatewayClient *goscaleio.GatewayClient) error {

	_, err := gatewayClient.AbortOperation()

	if err != nil {
		return fmt.Errorf("Error while Aborting Operation is %s", err.Error())
	}
	_, err = gatewayClient.ClearQueueCommand()

	if err != nil {
		return fmt.Errorf("Error while Clearing Queue is %s", err.Error())
	}

	_, err = gatewayClient.MoveToIdlePhase()

	if err != nil {
		return fmt.Errorf("Error while Move to Ideal Phase is %s", err.Error())
	}

	return nil
}

// SDCExpansionOperations function for the SDC Expansion Operation Like ParseCSV, Validate MDM and Installation
func (r *sdcResource) SDCExpansionOperations(ctx context.Context, plan sdcResourceModel, system *goscaleio.System, sdcDetails []SDCDetailDataModel, chnagedSDCs *[]SDCDetailDataModel) (dia diag.Diagnostics) {

	if CheckForExpansion(sdcDetails) {
		parsecsvRespose, parseCSVError := ParseCSVOperation(ctx, sdcDetails, r.gatewayClient)

		if parseCSVError != nil {
			dia.AddError(
				"Error while Parsing CSV",
				"unexpected error: "+parseCSVError.Error(),
			)
			return
		}

		// to make gateway available for installation
		queueOperationError := ResetInstallerQueue(r.gatewayClient)
		if queueOperationError != nil {
			dia.AddError(
				"Error Clearing Queue",
				"unexpected error: "+queueOperationError.Error(),
			)
			return
		}

		tflog.Info(ctx, "Gateway Installer changed to idle phase before initiating process")

		mdmIP, mdmIPError := GetMDMIP(ctx, sdcDetails)
		if mdmIPError != nil {
			dia.AddError(
				"Error while Getting MDM IP",
				"unexpected error: "+mdmIPError.Error(),
			)
			return
		}

		tflog.Info(ctx, "CSV File parsed ssucessfully")

		// Vaidate the MDM credentials
		validateMDMResponse, validateMDMError := ValidateMDMOperation(ctx, plan, r.gatewayClient, mdmIP)
		if validateMDMError != nil {
			dia.AddError(
				"Error While Validating MDM Details",
				"unexpected error: "+validateMDMResponse.Message,
			)
			return
		}

		if validateMDMResponse.StatusCode == 200 {

			tflog.Info(ctx, "MDM Details validated successfully")

			if !CheckForNewSDCIPs(strings.Split(parsecsvRespose.Message, ","), strings.Split(validateMDMResponse.Data, ",")) {
				installationError := InstallationOperations(ctx, plan, r.gatewayClient, parsecsvRespose)

				if installationError != nil {
					dia.AddError(
						"Error in Installation Process",
						"unexpected error: "+installationError.Error(),
					)
					return
				}
			}
			for _, sdc := range sdcDetails {

				if strings.EqualFold(sdc.IsSdc.ValueString(), "Yes") && sdc.SDCName.ValueString() == "" && sdc.PerformanceProfile.ValueString() == "" {
					sdcData, err := system.FindSdc("SdcIP", sdc.IP.ValueString())

					if err != nil {
						dia.AddError(
							"[Create] Unable to Find SDC by IP:"+sdc.IP.ValueString(),
							err.Error(),
						)
					}

					if sdcData != nil {
						changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

						if changedSDCDetail.LastUpdated.ValueString() == "" {
							changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
						}

						*chnagedSDCs = append(*chnagedSDCs, changedSDCDetail)
					}
				}
			}

		} else if validateMDMResponse.StatusCode != 200 {
			dia.AddError(
				"Error While Validating MDM Credentials",
				"unexpected error: "+validateMDMResponse.Message+" & Status Code: "+strconv.Itoa(validateMDMResponse.StatusCode),
			)
			return
		}
	}

	return
}

// UpdateSDCNamdPerfProfileOperations function for Update Name and Performance Profile of SDC
func (r *sdcResource) UpdateSDCNamdPerfProfileOperations(ctx context.Context, sdcDetailList []SDCDetailDataModel, system *goscaleio.System, chnagedSDCs *[]SDCDetailDataModel) (dia diag.Diagnostics) {

	for _, sdc := range sdcDetailList {

		if sdc.SDCName.ValueString() != "" || sdc.PerformanceProfile.ValueString() != "" {
			if sdc.SDCID.ValueString() == "" && sdc.IP.ValueString() != "" {
				sdcID, err := system.GetSdcIDByIP(sdc.IP.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Find SDC by IP:"+sdc.IP.ValueString(),
						err.Error(),
					)
				}

				sdc.SDCID = types.StringValue(sdcID)
			}

			if sdc.SDCID.ValueString() == "" && sdc.SDCName.ValueString() != "" {
				sdcData, err := system.FindSdc("Name", sdc.SDCName.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Find SDC by Name:"+sdc.SDCName.ValueString(),
						err.Error(),
					)
				}

				sdc.SDCID = types.StringValue(sdcData.Sdc.ID)
			}

			if sdc.SDCName.ValueString() != "" && sdc.SDCID.ValueString() != "" {

				nameExist, _ := checkForSDCName(system, sdc)

				if !nameExist {
					nameChng, err := system.ChangeSdcName(sdc.SDCID.ValueString(), sdc.SDCName.ValueString())

					if err != nil {
						dia.AddError(
							"[Create] Unable to Change Name Powerflex SDC by ID:"+sdc.SDCID.ValueString()+" Name:"+sdc.SDCName.ValueString(),
							err.Error(),
						)
					}

					tflog.Debug(ctx, fmt.Sprintf("[POWERFLEX] Name Change Result: %s  SDC ID: %s", helper.PrettyJSON(nameChng), sdc.SDCID))
				}
			}

			if sdc.PerformanceProfile.ValueString() != "" && sdc.SDCID.ValueString() != "" {
				perProfile, err := system.ChangeSdcPerfProfile(sdc.SDCID.ValueString(), sdc.PerformanceProfile.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Change Performance Profile Powerflex SDC by ID:"+sdc.SDCID.ValueString(),
						err.Error(),
					)
				}

				tflog.Debug(ctx, fmt.Sprintf("[POWERFLEX] Performance Profile Change Result: %s  SDC ID: %s", helper.PrettyJSON(perProfile), sdc.SDCID))
			}

			finalSDC, err := system.GetSdcByID(sdc.SDCID.ValueString())
			if err != nil {
				dia.AddError(
					"Unable to Read Changed SDC",
					err.Error(),
				)
				return
			}

			if finalSDC != nil {
				changedSDCDetail := getSDCState(*finalSDC.Sdc, sdc)

				if changedSDCDetail.LastUpdated.ValueString() == "" {
					changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
				}

				*chnagedSDCs = append(*chnagedSDCs, changedSDCDetail)
			}
		} else if strings.EqualFold(sdc.IsSdc.ValueString(), "No") && sdc.SDCName.ValueString() == "" && sdc.PerformanceProfile.ValueString() == "" {

			if sdc.SDCID.ValueString() != "" {
				sdcData, err := system.GetSdcByID(sdc.SDCID.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Find SDC by ID:"+sdc.SDCID.ValueString(),
						err.Error(),
					)
				}

				if sdcData != nil {
					changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

					if changedSDCDetail.LastUpdated.ValueString() == "" {
						changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
					}

					*chnagedSDCs = append(*chnagedSDCs, changedSDCDetail)
				}
			} else if sdc.IP.ValueString() != "" {
				sdcData, err := system.FindSdc("SdcIP", sdc.IP.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Find SDC by IP:"+sdc.IP.ValueString(),
						err.Error(),
					)
				}

				if sdcData != nil {
					changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

					if changedSDCDetail.LastUpdated.ValueString() == "" {
						changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
					}

					*chnagedSDCs = append(*chnagedSDCs, changedSDCDetail)
				}
			} else if sdc.SDCName.ValueString() != "" {
				sdcData, err := system.FindSdc("Name", sdc.SDCName.ValueString())

				if err != nil {
					dia.AddError(
						"[Create] Unable to Find SDC by Name:"+sdc.SDCName.ValueString(),
						err.Error(),
					)
				}

				if sdcData != nil {
					changedSDCDetail := getSDCState(*sdcData.Sdc, sdc)

					if changedSDCDetail.LastUpdated.ValueString() == "" {
						changedSDCDetail.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
					}

					*chnagedSDCs = append(*chnagedSDCs, changedSDCDetail)
				}
			}

		}
	}

	return
}

// ParseCSVOperation function for Handling Parsing CSV Operation
func ParseCSVOperation(ctx context.Context, sdcDetails []SDCDetailDataModel, gatewayClient *goscaleio.GatewayClient) (*goscaleio_types.GatewayResponse, error) {

	var parseCSVResponse goscaleio_types.GatewayResponse

	//Create a csv file from the input given by the user
	mydir, err := os.Getwd()
	if err != nil {
		return &parseCSVResponse, fmt.Errorf("Error While Reading Current Directory is %s", err.Error())
	}
	// Create a csv writer
	file, err := os.Create(mydir + "/Minimal.csv")
	if err != nil {
		return &parseCSVResponse, fmt.Errorf("Error While Creating Temp CSV is %s", err.Error())
	}
	defer file.Close()
	writer := csv.NewWriter(file)

	// Write the header row
	header := []string{"IPs", "Username", "Password", "Operating System", "Is MDM/TB", "Is SDC", "perfProfileForSDC"}
	err = writer.Write(header)
	if err != nil {
		return &parseCSVResponse, fmt.Errorf("Error While Writing Temp CSV is %s", err.Error())
	}

	var sdcIPs []string

	for _, item := range sdcDetails {

		if item.Password.ValueString() != "" {
			// Add mapped SDC
			csvStruct := CsvRow{
				IP:              item.IP.ValueString(),
				UserName:        item.UserName.ValueString(),
				Password:        item.Password.ValueString(),
				IsMdmOrTb:       item.IsMdmOrTb.ValueString(),
				OperatingSystem: item.OperatingSystem.ValueString(),
				IsSdc:           item.IsSdc.ValueString(),
			}

			if strings.EqualFold(csvStruct.IsSdc, "Yes") {
				sdcIPs = append(sdcIPs, csvStruct.IP)
			}

			if strings.EqualFold(item.PerformanceProfile.ValueString(), "HighPerformance") {
				csvStruct.PerformanceProfile = "High"
			}

			//Write the data row
			data := []string{csvStruct.IP, csvStruct.UserName, csvStruct.Password, csvStruct.OperatingSystem, csvStruct.IsMdmOrTb, csvStruct.IsSdc, csvStruct.PerformanceProfile} //, csvStruct.SDCName
			err = writer.Write(data)
			if err != nil {
				return &parseCSVResponse, fmt.Errorf("Error While Creating Temp CSV File is %s", err.Error())
			}
		}

	}
	writer.Flush()

	parsecsvRespose, parseCSVError := gatewayClient.ParseCSV(mydir + "/Minimal.csv")

	deletCSVError := os.Remove(mydir + "/Minimal.csv")
	if deletCSVError != nil {
		return &parseCSVResponse, fmt.Errorf("Error While Deleting Temp CSV File is %s", deletCSVError.Error())
	}

	if parseCSVError != nil {
		return &parseCSVResponse, fmt.Errorf("%s", parseCSVError.Error())
	}

	parsecsvRespose.Message = strings.Join(sdcIPs, ",")

	if parsecsvRespose.StatusCode != 200 {
		return &parseCSVResponse, fmt.Errorf("Meesage : %s, Error Cosde : %s", parsecsvRespose.Message, strconv.Itoa(parsecsvRespose.StatusCode))
	}

	return parsecsvRespose, nil
}

// ValidateMDMOperation function for Validate the MDM credentials
func ValidateMDMOperation(ctx context.Context, model sdcResourceModel, gatewayClient *goscaleio.GatewayClient, mdmIP string) (*goscaleio_types.GatewayResponse, error) {
	mapData := map[string]interface{}{
		"mdmUser":     "admin",
		"mdmPassword": model.MdmPassword.ValueString(),
	}
	mapData["mdmIps"] = []string{mdmIP}

	secureData := map[string]interface{}{
		"allowNonSecureCommunicationWithMdm": true,
		"allowNonSecureCommunicationWithLia": true,
		"disableNonMgmtComponentsAuth":       false,
	}
	mapData["securityConfiguration"] = secureData
	jsonres, _ := json.Marshal(mapData)

	validateMDMResponse, validateMDMError := gatewayClient.ValidateMDMDetails(jsonres)
	if validateMDMError != nil {
		return validateMDMResponse, fmt.Errorf("%s", validateMDMError.Error())
	}

	return validateMDMResponse, nil
}

// InstallationOperations function for begin instllation process
func InstallationOperations(ctx context.Context, model sdcResourceModel, gatewayClient *goscaleio.GatewayClient, parsecsvRespose *goscaleio_types.GatewayResponse) error {

	beginInstallationResponse, installationError := gatewayClient.BeginInstallation(parsecsvRespose.Data, "admin", model.MdmPassword.ValueString(), model.LiaPassword.ValueString(), true)

	if installationError != nil {
		return fmt.Errorf("Error while begin installation is %s", installationError.Error())
	}

	if beginInstallationResponse.StatusCode == 200 {
		currentPhase := "query"
		couterForStopExecution := 0

		tflog.Info(ctx, "Gateway Installation Begin, Current Phase - Query")

		for couterForStopExecution <= 5 {

			time.Sleep(1 * time.Minute)

			checkForPhaseCompleted, _ := gatewayClient.CheckForCompletionQueueCommands(currentPhase)

			if checkForPhaseCompleted.Data == "Completed" {
				couterForStopExecution = 0

				if currentPhase != "configure" {
					moveToNextPhaseResponse, err := gatewayClient.MoveToNextPhase()

					if err != nil {
						return fmt.Errorf("Error while moving to next phase is %s", err.Error())
					}

					if moveToNextPhaseResponse.StatusCode == 200 {
						if currentPhase == "query" {
							currentPhase = "upload"
							tflog.Info(ctx, "Gateway Installation phase changed to Upload")
						} else if currentPhase == "upload" {
							currentPhase = "install"
							tflog.Info(ctx, "Gateway Installation phase changed to Install")
						} else if currentPhase == "install" {
							currentPhase = "configure"
							tflog.Info(ctx, "Gateway Installation phase changed to Configure")
						}
					} else {
						return fmt.Errorf("Messsage: %s, Error Code: %s", moveToNextPhaseResponse.Message, strconv.Itoa(moveToNextPhaseResponse.StatusCode))
					}
				} else {
					// to make gateway available for installation
					queueOperationError := ResetInstallerQueue(gatewayClient)
					if queueOperationError != nil {
						return fmt.Errorf("Error Clearing Queue During Installation is %s", queueOperationError.Error())
					}

					couterForStopExecution = 10

					return nil
				}

			} else if checkForPhaseCompleted.Data == "Running" {
				couterForStopExecution++

				tflog.Info(ctx, "Gateway Installation operations are still running")

				if couterForStopExecution == 5 {
					// to make gateway available for installation
					queueOperationError := ResetInstallerQueue(gatewayClient)
					if queueOperationError != nil {
						return fmt.Errorf("Error Clearing Queue During Installation is %s", queueOperationError.Error())
					}

					return fmt.Errorf("Time Out,Some Operations of Installer running from since long")
				}

			} else {
				return fmt.Errorf("Error During Installation is %s", checkForPhaseCompleted.Message)
			}
		}
	} else {
		return fmt.Errorf("Message: %s, Error Code: %s", beginInstallationResponse.Message, strconv.Itoa(beginInstallationResponse.StatusCode))
	}

	return nil
}

// CheckForNewSDCIPs function to check SDC Alredy Installed or not
func CheckForNewSDCIPs(newSDCIPS []string, installedSDCIPs []string) bool {
	checkset := make(map[string]bool)
	for _, element := range newSDCIPS {
		checkset[element] = true
	}
	for _, value := range installedSDCIPs {
		if checkset[value] {
			delete(checkset, value)
		}
	}
	return len(checkset) == 0 //this implies that set is subset of superset
}

// getSDCState - function to return sdc result from goscaleio.
func getSDCState(sdc goscaleio_types.Sdc, model SDCDetailDataModel) (response SDCDetailDataModel) {

	if sdc.ID != "" {
		model.SDCID = types.StringValue(sdc.ID)
	}

	model.SDCName = types.StringValue(sdc.Name)

	if sdc.SdcGUID != "" {
		model.SdcGUID = types.StringValue(sdc.SdcGUID)
	}

	model.SdcApproved = types.BoolValue(sdc.SdcApproved)

	model.OnVMWare = types.BoolValue(sdc.OnVMWare)

	if sdc.SystemID != "" {
		model.SystemID = types.StringValue(sdc.SystemID)
	}

	model.PerformanceProfile = types.StringValue(sdc.PerfProfile)

	if sdc.SdcIP != "" {
		model.IP = types.StringValue(sdc.SdcIP)
	}

	if sdc.MdmConnectionState != "" {
		model.MdmConnectionState = types.StringValue(sdc.MdmConnectionState)
	}

	return model
}

// checkForSDCName - check for the SDC Name already exist or not
func checkForSDCName(system *goscaleio.System, sdcDetail SDCDetailDataModel) (bool, error) {

	sdcData, err := system.FindSdc("Name", sdcDetail.SDCName.ValueString())

	if err == nil && sdcData.Sdc.ID == sdcDetail.SDCID.ValueString() {
		return true, fmt.Errorf("SDC Name:%s already exist", sdcDetail.SDCName.ValueString())
	}

	return false, nil
}
