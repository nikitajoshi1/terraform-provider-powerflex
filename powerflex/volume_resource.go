package powerflex

import (
	"context"
	"strconv"

	"github.com/dell/goscaleio"
	pftypes "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &volumeResource{}
	_ resource.ResourceWithConfigure   = &volumeResource{}
	_ resource.ResourceWithImportState = &volumeResource{}
)

// NewVolumeResource is a helper function to simplify the provider implementation.
func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource is the resource implementation.
type volumeResource struct {
	client *goscaleio.Client
}

// Metadata returns the resource type name.
func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

// Schema defines the schema for the resource.
func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = VolumeResourceSchema
}

// Configure adds the provider configured client to the data source.
func (r *volumeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*goscaleio.Client)
}

// ModifyPlan modify resource plan attribute value
func (r *volumeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan VolumeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if !plan.Size.IsNull() && !plan.Size.IsUnknown() && !plan.CapacityUnit.IsUnknown() {
		// check if size is in granularity of 8 or not
		if plan.Size.ValueInt64()%8 != 0 && plan.CapacityUnit.ValueString() == "GB" {
			resp.Diagnostics.AddError(
				"Error: Size Must be in granularity of 8GB",
				"Could not assign volume with size. sizeInGb ("+strconv.FormatInt(plan.Size.ValueInt64(), 10)+") must be a positive number in granularity of 8 GB.",
			)
		}
		VSIKB := convertToKB(plan.CapacityUnit.ValueString(), plan.Size.ValueInt64())
		plan.SizeInKb = types.Int64Value(VSIKB)
		diags = resp.Plan.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan VolumeResourceModel
	var pdr *goscaleio.ProtectionDomain
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pdr, diags = r.getProtectionDomainID(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.getStoragePoolID(pdr, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumeCreate := &pftypes.VolumeParam{
		ProtectionDomainID: plan.ProtectionDomainID.ValueString(),
		StoragePoolID:      plan.StoragePoolID.ValueString(),
		UseRmCache:         strconv.FormatBool(plan.UseRmCache.ValueBool()),
		CompressionMethod:  plan.CompressionMethod.ValueString(),
		VolumeType:         plan.VolumeType.ValueString(),
		VolumeSizeInKb:     strconv.FormatInt(plan.SizeInKb.ValueInt64(), 10),
		Name:               plan.Name.ValueString(),
	}
	spr, err0 := getStoragePoolInstance(r.client, volumeCreate.StoragePoolID, volumeCreate.ProtectionDomainID)
	if err0 != nil {
		resp.Diagnostics.AddError(
			"Error getting storage pool with id: "+volumeCreate.StoragePoolID+" or protection pool with id: "+volumeCreate.ProtectionDomainID,
			"unexpected error: "+err0.Error(),
		)
		return
	}
	// platform fails silently for compression method "None".
	if (spr.StoragePool.DataLayout != "FineGranularity") && (plan.CompressionMethod.ValueString() != "") {
		resp.Diagnostics.AddError(
			"error setting the compression method",
			"compression may only be set on volumes with Fine Granularity layout on storage pool. This storage pool has "+spr.StoragePool.DataLayout+" layout.",
		)
		return
	}
	volCreateResponse, err1 := spr.CreateVolume(volumeCreate)
	if err1 != nil {
		resp.Diagnostics.AddError(
			"Error creating volume",
			"unexpected error: "+err1.Error(),
		)
		return
	}
	volsResponse, err2 := spr.GetVolume("", volCreateResponse.ID, "", "", false)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume after creation",
			"unexpected error: "+err2.Error(),
		)
		return
	}
	vol := volsResponse[0]
	vr := goscaleio.NewVolume(r.client)
	vr.Volume = vol
	if !plan.AccessMode.IsNull() {
		err3 := vr.SetVolumeAccessModeLimit(plan.AccessMode.ValueString())
		if err3 != nil {
			resp.Diagnostics.AddError(
				"Error setting access mode on volume",
				"unexpected error: "+err3.Error(),
			)
		}
	}

	volsResponse, err7 := spr.GetVolume("", volCreateResponse.ID, "", "", false)
	if err7 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume after mapping the sdcs",
			"Could not get volume after mapping the sdcs, unexpected error: "+err2.Error(),
		)
		return
	}
	vol = volsResponse[0]
	dgs := refreshVolumeState(vol, &plan)
	resp.Diagnostics.Append(dgs...)
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state VolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	volsResponse, err2 := r.client.GetVolume("", state.ID.ValueString(), "", "", false)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume",
			"Could not get volume, unexpected error: "+err2.Error(),
		)
		return
	}
	vol := volsResponse[0]
	dgs := refreshVolumeState(vol, &state)
	resp.Diagnostics.Append(dgs...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get plan values
	var plan VolumeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state VolumeResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volsplan, err2 := r.client.GetVolume("", state.ID.ValueString(), "", "", false)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume",
			"unexpected error: "+err2.Error(),
		)
		return
	}
	volresource := goscaleio.NewVolume(r.client)
	volresource.Volume = volsplan[0]

	// updating the name of volume if there is change in plan
	if !plan.Name.IsUnknown() && plan.Name.ValueString() != state.Name.ValueString() {
		err3 := volresource.SetVolumeName(plan.Name.ValueString())
		if err3 != nil {
			resp.Diagnostics.AddError(
				"Error renaming the volume",
				"unexpected error: "+err3.Error(),
			)
		}
	}

	// updating the size of the volume if there is change in plan
	if plan.SizeInKb.ValueInt64() != state.SizeInKb.ValueInt64() {
		sizeInGb := plan.SizeInKb.ValueInt64() / 1048576
		sizeInGB := strconv.FormatInt(int64(sizeInGb), 10)
		err4 := volresource.SetVolumeSize(sizeInGB)
		if err4 != nil {
			resp.Diagnostics.AddError(
				"Error setting the volume size",
				"unexpected error: "+err4.Error(),
			)
		} else {
			state.Size = plan.Size
			state.CapacityUnit = plan.CapacityUnit
		}
	}

	// prompt error on change in volume type, as we can't update the volume type after the creation
	if !plan.VolumeType.IsUnknown() && !plan.VolumeType.Equal(state.VolumeType) {
		resp.Diagnostics.AddError(
			"volume type cannot be update after volume creation.",
			"unexpected error: volume type change is not supported",
		)
	}

	// updating the use rm cache if there is change in plan
	if !plan.UseRmCache.IsUnknown() && plan.UseRmCache.ValueBool() != state.UseRmCache.ValueBool() {
		err5 := volresource.SetVolumeUseRmCache(plan.UseRmCache.ValueBool())
		if err5 != nil {
			resp.Diagnostics.AddError(
				"Error setting the use rm cache",
				"unexpected error: "+err5.Error(),
			)
		}
	}

	// updating the compression if there is change in plan
	if !plan.CompressionMethod.IsUnknown() && !plan.CompressionMethod.Equal(state.CompressionMethod) {
		err6 := volresource.SetCompressionMethod(plan.CompressionMethod.ValueString())
		if err6 != nil {
			resp.Diagnostics.AddError(
				"Error setting the compression method",
				"unexpected error: "+err6.Error(),
			)
		}
	}

	// changing the access mode
	if !plan.AccessMode.IsUnknown() && plan.AccessMode.ValueString() != state.AccessMode.ValueString() {
		err := volresource.SetVolumeAccessModeLimit(plan.AccessMode.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error setting the access mode",
				"unexpected error: "+err.Error(),
			)
		}
	}

	vols, err2 := r.client.GetVolume("", state.ID.ValueString(), "", "", false)
	if err2 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume",
			"Could not get volume, unexpected error: "+err2.Error(),
		)
		return
	}
	vol := vols[0]
	dgs := refreshVolumeState(vol, &state)
	resp.Diagnostics.Append(dgs...)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state VolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	volsplan, err1 := r.client.GetVolume("", state.ID.ValueString(), "", "", false)
	if err1 != nil {
		resp.Diagnostics.AddError(
			"Error getting volume",
			"Could not get volume, unexpected error: "+err1.Error(),
		)
		return
	}
	volresource := goscaleio.NewVolume(r.client)
	volresource.Volume = volsplan[0]

	// finally removing the volume after unmap operation
	err := volresource.RemoveVolume(state.RemoveMode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Removing Volume",
			"Couldn't remove volume "+err.Error(),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getProtectionDomainID updates the protection domain ID in the plan
func (r *volumeResource) getProtectionDomainID(plan *VolumeResourceModel) (*goscaleio.ProtectionDomain, diag.Diagnostics) {
	sr, err := getFirstSystem(r.client)
	var diags diag.Diagnostics
	if err != nil {
		diags.AddError(
			"Error in getting system instance on the PowerFlex cluster",
			err.Error(),
		)
		return nil, diags
	}

	pdr := goscaleio.NewProtectionDomain(r.client)

	if !plan.ProtectionDomainName.IsUnknown() {
		protectionDomain, err := sr.FindProtectionDomain("", plan.ProtectionDomainName.ValueString(), "")
		if err != nil {
			diags.AddError(
				"Error getting protection domain",
				"Could not get protection domain with name: "+plan.ProtectionDomainName.String()+", \n unexpected error: "+err.Error(),
			)
			return nil, diags
		}
		pdr.ProtectionDomain = protectionDomain
		plan.ProtectionDomainID = types.StringValue(protectionDomain.ID)
	} else if !plan.ProtectionDomainID.IsUnknown() {
		protectionDomain, err := sr.FindProtectionDomain(plan.ProtectionDomainID.ValueString(), "", "")
		if err != nil {
			diags.AddError(
				"Error getting protection domain with id",
				"Could not get protection domain with id: "+plan.ProtectionDomainName.ValueString()+", \n unexpected error: "+err.Error(),
			)
			return nil, diags
		}
		pdr.ProtectionDomain = protectionDomain
		plan.ProtectionDomainName = types.StringValue(protectionDomain.Name)
	}
	return pdr, diags
}

// getStoragePoolID updates the storage pool ID in the plan
func (r *volumeResource) getStoragePoolID(pdr *goscaleio.ProtectionDomain, plan *VolumeResourceModel) (diags diag.Diagnostics) {
	if !plan.StoragePoolName.IsUnknown() {
		storagePool, err := pdr.FindStoragePool("", plan.StoragePoolName.ValueString(), "")
		if err != nil {
			diags.AddError(
				"Error getting storage pool",
				"Could not get storage pool with name: "+plan.StoragePoolName.ValueString()+", \n unexpected error: "+err.Error(),
			)
			return
		}
		plan.StoragePoolID = types.StringValue(storagePool.ID)
	} else if !plan.StoragePoolID.IsUnknown() {
		storagePool, err := pdr.FindStoragePool(plan.StoragePoolID.ValueString(), "", "")
		if err != nil {
			diags.AddError(
				"Error getting storage pool with id",
				"Could not get storage pool with with id: "+plan.StoragePoolID.ValueString()+", \n unexpected error: "+err.Error(),
			)
			return
		}
		plan.StoragePoolName = types.StringValue(storagePool.Name)
	}
	return
}
