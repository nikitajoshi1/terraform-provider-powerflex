package powerflex

import (
	"context"
	"fmt"

	"github.com/dell/goscaleio"
	scaleiotypes "github.com/dell/goscaleio/types/v1"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	frameworkTypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// getFirstSystem - finds available first system and returns it.
func getFirstSystem(rc *goscaleio.Client) (*goscaleio.System, error) {
	allSystems, err := rc.GetSystems()
	if err != nil {
		return nil, fmt.Errorf("Error in goscaleio GetSystems")
	}
	if numSys := len((allSystems)); numSys == 0 {
		return nil, fmt.Errorf("no systems found")
	} else if numSys > 1 {
		return nil, fmt.Errorf("more than one system found")
	}
	system, err := rc.FindSystem(allSystems[0].ID, "", "")
	if err != nil {
		return nil, fmt.Errorf("Error in goscaleio FindSystem")
	}
	return system, nil
}

// getNewProtectionDomainEx function to get Protection Domain
func getNewProtectionDomainEx(c *goscaleio.Client, pdID string, pdName string, href string) (*goscaleio.ProtectionDomain, error) {
	system, err := getFirstSystem(c)
	if err != nil {
		return nil, err
	}
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

// getSdcType function returns SDC type
func getSdcType(c *goscaleio.Client, sdcID string) (*goscaleio.Sdc, error) {
	system, err := getFirstSystem(c)
	if err != nil {
		return nil, err
	}
	return system.GetSdcByID(sdcID)
}

// getVolumeType function returns volume type
func getVolumeType(c *goscaleio.Client, volID string) (*goscaleio.Volume, error) {
	volumes, err := c.GetVolume("", volID, "", "", false)
	if err != nil {
		return nil, err
	}

	volume := volumes[0]
	volType := goscaleio.NewVolume(c)
	volType.Volume = volume
	return volType, nil
}

// boolDefaultModifier is a plan modifier that sets a default value for a
// types.BoolType attribute when it is not configured. The attribute must be
// marked as Optional and Computed. When setting the state during the resource
// Create, Read, or Update methods, this default value must also be included or
// the Terraform CLI will generate an error.
type boolDefaultModifier struct {
	Default bool
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m boolDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %t", m.Default)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m boolDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%t`", m.Default)
}

// PlanModifyBool runs the logic of the plan modifier.
// Access to the configuration, plan, and state is available in `req`, while
// `resp` contains fields for updating the planned value, triggering resource
// replacement, and returning diagnostics.
func (m boolDefaultModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// If the value is unknown or known, do not set default value.
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		resp.PlanValue = frameworkTypes.BoolValue(m.Default)
	}
}

func boolDefault(defaultValue bool) planmodifier.Bool {
	return boolDefaultModifier{
		Default: defaultValue,
	}
}
