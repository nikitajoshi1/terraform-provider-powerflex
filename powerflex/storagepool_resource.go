package powerflex

import (
	"context"
	"terraform-provider-powerflex/helper"

	"github.com/dell/goscaleio"
	scaleiotypes "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &storagepoolResource{}
	_ resource.ResourceWithConfigure   = &storagepoolResource{}
	_ resource.ResourceWithImportState = &storagepoolResource{}
)

// StoragepoolResource - function to return resource interface
func StoragepoolResource() resource.Resource {
	return &storagepoolResource{}
}

type storagepoolResource struct {
	client *goscaleio.Client
}

type storagepoolResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProtectionDomainID   types.String `tfsdk:"protection_domain_id"`
	ProtectionDomainName types.String `tfsdk:"protection_domain_name"`
	Name                 types.String `tfsdk:"name"`
	MediaType            types.String `tfsdk:"media_type"`
	UseRmcache           types.Bool   `tfsdk:"use_rmcache"`
	UseRfcache           types.Bool   `tfsdk:"use_rfcache"`
}

func (r *storagepoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storagepool"
}

func (r *storagepoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = StoragepoolReourceSchema
}

func (r *storagepoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*goscaleio.Client)
}

// getNewProtectionDomainEx function to get Protection Domain
func getNewProtectionDomainEx(c *goscaleio.Client, pdID string, pdName string, href string) (*goscaleio.ProtectionDomain, error) {
	system, _ := getFirstSystem(c)
	pdr := goscaleio.NewProtectionDomainEx(c, &scaleiotypes.ProtectionDomain{})
	if pdID != "" {
		protectionDomain, err := system.FindProtectionDomain(pdID, "", "")
		pdr.ProtectionDomain = protectionDomain
		if err != nil {
			return nil, err
		}
	} else {
		protectionDomain, err := system.FindProtectionDomain("", pdName, "")
		pdr.ProtectionDomain = protectionDomain
		if err != nil {
			return nil, err
		}
	}
	return pdr, nil
}

// Function used to Create Storagepool Resource
func (r *storagepoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Create storagepool")
	// Retrieve values from plan
	var plan storagepoolResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pd, err := getNewProtectionDomainEx(r.client, plan.ProtectionDomainID.ValueString(), plan.ProtectionDomainName.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Protection Domain",
			"Could not get Protection Domain, unexpected err: "+err.Error(),
		)
		return
	}

	payload := &scaleiotypes.StoragePoolParam{
		Name:      plan.Name.ValueString(),
		MediaType: plan.MediaType.ValueString(),
	}

	if plan.UseRmcache.String() == "true" {
		payload.UseRmcache = "true"
	} else {
		payload.UseRmcache = "false"
	}

	if plan.UseRfcache.String() == "true" {
		payload.UseRfcache = "true"
	} else {
		payload.UseRfcache = "false"
	}

	sp, err := pd.CreateStoragePool(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Storage Pool",
			"Could not create Storage Pool, unexpected error: "+err.Error(),
		)
		return
	}

	spResponse, err := pd.FindStoragePool(sp, "", "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Storagepool after creation",
			"Could not get Storagepool, unexpected error: "+err.Error(),
		)
		return
	}

	state := updateStoragepoolState(spResponse, plan)
	tflog.Debug(ctx, "Create Storagepool :-- "+helper.PrettyJSON(sp))
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Function used to Read Storagepool Resource
func (r *storagepoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Read Storagepool")
	// Get current state
	var state storagepoolResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pd, err := getNewProtectionDomainEx(r.client, state.ProtectionDomainID.ValueString(), state.ProtectionDomainName.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Protection Domain",
			"Could not get Protection Domain, unexpected err: "+err.Error(),
		)
		return
	}

	spr, err := pd.FindStoragePool(state.ID.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Powerflex Storagepool",
			err.Error(),
		)
		return
	}
	spResponse := updateStoragepoolState(spr, state)
	tflog.Debug(ctx, "Read Storagepool :-- "+helper.PrettyJSON(spr))
	diags = resp.State.Set(ctx, spResponse)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Function used to Update Storagepool Resource
func (r *storagepoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Update Storagepool")
	// Retrieve values from plan
	var plan storagepoolResourceModel
	var err1 error

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	//Get Current State
	var state storagepoolResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pd, err := getNewProtectionDomainEx(r.client, plan.ProtectionDomainID.ValueString(), plan.ProtectionDomainName.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Protection Domain",
			"Could not get Protection Domain, unexpected err: "+err.Error(),
		)
		return
	}

	spResponse, err := pd.FindStoragePool(state.ID.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while getting Storagepool", err.Error(),
		)
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		_, err := pd.ModifyStoragePoolName(state.ID.ValueString(), plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error while updating name of Storagepool", err.Error(),
			)
		}
	}

	if plan.MediaType.ValueString() != state.MediaType.ValueString() {
		_, err := pd.ModifyStoragePoolMedia(state.ID.ValueString(), plan.MediaType.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error while updating media type of Storagepool", err.Error(),
			)
		}
	}

	rm := goscaleio.NewStoragePoolEx(r.client, spResponse)

	if !state.UseRmcache.Equal(plan.UseRmcache) {
		err := rm.ModifyRMCache(plan.UseRmcache.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error while updating rm_cache of Storagepool", err.Error(),
			)
		}
	}

	if !state.UseRfcache.Equal(plan.UseRfcache) {
		if plan.UseRfcache.String() == "true" {
			_, err1 = pd.EnableRFCache(spResponse.ID)

		} else {
			_, err1 = pd.DisableRFCache(spResponse.ID)
		}
	}

	if err1 != nil {
		resp.Diagnostics.AddError(
			"Error while updating rf_cache of Storagepool", err.Error(),
		)
	}

	spResponse, err = pd.FindStoragePool(state.ID.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error while getting Storagepool", err.Error(),
		)
		return
	}

	state1 := updateStoragepoolState(spResponse, plan)
	tflog.Debug(ctx, "Update Storagepool :-- "+helper.PrettyJSON(spResponse))
	diags = resp.State.Set(ctx, state1)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Function used to Delete Storagepool Resource
func (r *storagepoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Delete Storagepool")
	// Retrieve values from state
	var state storagepoolResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pd, err := getNewProtectionDomainEx(r.client, state.ProtectionDomainID.ValueString(), state.ProtectionDomainName.ValueString(), "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Protection Domain",
			"Could not get Protection Domain, unexpected err: "+err.Error(),
		)
		return
	}

	err = pd.DeleteStoragePool(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Storagepool",
			"Couldn't Delete Storagepool "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.RemoveResource(ctx)
}

// Function used to ImportState for Storagepool Resource
func (r *storagepoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// func getStoragePool(client *goscaleio.Client, spID string) (*scaleiotypes.StoragePool, error) {
// 	spr, err := client.FindStoragePool(spID, "", "", "")
// 	if err != nil {
// 		return nil, err
// 	}
// 	return spr, nil
// }

// Function to update the State for Storagepool Resource
func updateStoragepoolState(storagepool *scaleiotypes.StoragePool, plan storagepoolResourceModel) (state storagepoolResourceModel) {
	state.ProtectionDomainID = plan.ProtectionDomainID
	state.ProtectionDomainName = plan.ProtectionDomainName
	state.ID = types.StringValue(storagepool.ID)
	state.Name = types.StringValue(storagepool.Name)
	state.MediaType = types.StringValue(storagepool.MediaType)
	state.UseRmcache = types.BoolValue(storagepool.UseRmcache)
	state.UseRfcache = types.BoolValue(storagepool.UseRfcache)

	return state
}
