package powerflex

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var sdcDatasourceSchemaDescriptions = struct {
	SdcDatasourceSchema string

	InputID    string
	InputSdcID string
	// InputSystemid string
	InputName string

	Sdcs string // outpur slice

	LastUpdated        string
	SdcID              string
	SystemID           string
	Name               string
	SdcIP              string
	SdcApproved        string
	OnVMWare           string
	SdcGUID            string
	MdmConnectionState string
	Links              string
	LinksRel           string
	LinksHref          string
	Statistics         string
}{
	SdcDatasourceSchema: "",

	InputID:    "Input ID required only for testing.",
	InputSdcID: "Input SDC id to search for.",
	// InputSystemid: "",
	InputName: "SDC input sdc name to search for.",

	Sdcs: "result SDCs.", // outpur slice

	LastUpdated:        "SDC result last updated timestamp.",
	SdcID:              "SDC ID.",
	SystemID:           "SDC System ID.",
	Name:               "SDC name.",
	SdcIP:              "SDC IP.",
	SdcApproved:        "SDC is approved.",
	OnVMWare:           "SDC is onvmware.",
	SdcGUID:            "SDC GUID.",
	MdmConnectionState: "SDC MDM connection status.",
	Links:              "SDC Links.",
	LinksRel:           "SDC Links-Rel.",
	LinksHref:          "SDC Links-HREF.",
	Statistics:         "SDC Statistics.",
}

// SDCDataSourceScheme is variable for schematic for SDC Data Source
var SDCDataSourceScheme schema.Schema = schema.Schema{
	Description: sdcDatasourceSchemaDescriptions.SdcDatasourceSchema,
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: sdcDatasourceSchemaDescriptions.InputID,
			Optional:    true,
		},
		"sdc_id": schema.StringAttribute{
			Description: sdcDatasourceSchemaDescriptions.InputSdcID,
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot("name")),
			},
		},
		"name": schema.StringAttribute{
			Description: sdcDatasourceSchemaDescriptions.InputName,
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot("sdc_id")),
			},
		},
		"sdcs": schema.ListNestedAttribute{
			Description: sdcDatasourceSchemaDescriptions.Sdcs,
			Computed:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.SdcID,
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.Name,
						Computed:    true,
					},
					"sdc_guid": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.SdcGUID,
						Computed:    true,
					},
					"on_vmware": schema.BoolAttribute{
						Description: sdcDatasourceSchemaDescriptions.OnVMWare,
						Computed:    true,
					},
					"sdc_approved": schema.BoolAttribute{
						Description: sdcDatasourceSchemaDescriptions.SdcApproved,
						Computed:    true,
					},
					"system_id": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.SystemID,
						Computed:    true,
					},
					"sdc_ip": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.SdcIP,
						Computed:    true,
					},
					"mdm_connection_state": schema.StringAttribute{
						Description: sdcDatasourceSchemaDescriptions.MdmConnectionState,
						Computed:    true,
					},
					// "statistics": schema.ObjectAttribute{
					// 	Description: sdcDatasourceSchemaDescriptions.Statistics,
					// 	Computed:    true,
					// 	AttributeTypes: map[string]attr.Type{
					// 		"numofmappedvolumes": types.Int64Type,
					// 		"volumeids":          types.ListType{ElemType: types.StringType},
					// 		"userdatareadbwc": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 		"userdatawritebwc": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 		"userdatatrimbwc": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 		"userdatasdcreadlatency": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 		"userdatasdcwritelatency": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 		"userdatasdctrimlatency": types.ObjectType{
					// 			AttrTypes: map[string]attr.Type{
					// 				"totalweightinkb": types.Int64Type,
					// 				"numoccured":      types.Int64Type,
					// 				"numseconds":      types.Int64Type,
					// 			},
					// 		},
					// 	},
					// },
					"links": schema.ListNestedAttribute{
						Description: sdcDatasourceSchemaDescriptions.Links,
						Computed:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"rel": schema.StringAttribute{
									Description: sdcDatasourceSchemaDescriptions.LinksRel,
									Computed:    true,
								},
								"href": schema.StringAttribute{
									Description: sdcDatasourceSchemaDescriptions.LinksHref,
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	},
}
