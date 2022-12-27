package powerflex

import (
	"context"
	"os"

	"github.com/dell/goscaleio"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &powerflexProvider{}
)

// New returns the powerflex provider
func New() provider.Provider {
	return &powerflexProvider{}
}

type powerflexProvider struct{}

type powerflexProviderModel struct {
	EndPoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *powerflexProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "powerflex"
}

func (p *powerflexProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description:         "the endpoint to which it needs to be connected.",
				MarkdownDescription: "the endpoint to which it needs to be connected.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				Description:         "The username required for authentication.",
				MarkdownDescription: "The username required for authentication.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				Description:         "The password required for the authentication.",
				MarkdownDescription: "The password required for the authentication.",
				Required:            true,
				Sensitive:           true,
			},
			"insecure": schema.BoolAttribute{
				Description:         "Specifies if the user wants to do SSL verification.",
				MarkdownDescription: "Specifies if the user wants to do SSL verification.",
				Optional:            true,
			},
		},
	}
}

func (p *powerflexProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring powerflex client")

	var config powerflexProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.EndPoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown powerflex API EndPoint",
			"The provider cannot create the powerflex API client as there is an unknown configuration value for the powerflex API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the POWERFLEX_ENDPOINT environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown powerflex API Username",
			"The provider cannot create the powerflex API client as there is an unknown configuration value for the powerflex API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the POWERFLEX_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown powerflex API Password",
			"The provider cannot create the powerflex API client as there is an unknown configuration value for the powerflex API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the POWERFLEX_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("POWERFLEX_ENDPOINT")
	username := os.Getenv("POWERFLEX_USERNAME")
	password := os.Getenv("POWERFLEX_PASSWORD")
	insecure := os.Getenv("POWERFLEX_INSECURE") == "true"

	if !config.EndPoint.IsNull() {
		endpoint = config.EndPoint.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing powerflex API Endpoint",
			"The provider cannot create the powerflex API client as there is a missing or empty value for the powerflex API endpoint. "+
				"Set the endpoint value in the configuration or use the POWERFLEX_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing powerflex API Username",
			"The provider cannot create the powerflex API client as there is a missing or empty value for the powerflex API username. "+
				"Set the username value in the configuration or use the POWERFLEX_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing powerflex API Password",
			"The provider cannot create the powerflex API client as there is a missing or empty value for the powerflex API password. "+
				"Set the password value in the configuration or use the POWERFLEX_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "powerflex_endpoint", endpoint)
	ctx = tflog.SetField(ctx, "powerflex_username", username)
	ctx = tflog.SetField(ctx, "powerflex_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "powerflex_password")
	ctx = tflog.SetField(ctx, "insecure", insecure)
	tflog.Debug(ctx, "Creating powerflex client")

	// Create a new powerflex client using the configuration values
	Client, err := goscaleio.NewClientWithArgs(endpoint, "", true, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create powerflex API Client",
			"An unexpected error occurred when creating the powerflex API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"powerflex Client Error: "+err.Error(),
		)
		return
	}

	var goscaleioConf goscaleio.ConfigConnect = goscaleio.ConfigConnect{}
	goscaleioConf.Endpoint = endpoint
	goscaleioConf.Username = username
	goscaleioConf.Version = ""
	goscaleioConf.Password = password

	_, err = Client.Authenticate(&goscaleioConf)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Authenticate Goscaleio API Client",
			"An unexpected error occurred when authenticating the Goscaleio API Client. "+
				"Unable to Authenticate Goscaleio API Client.\n\n"+
				"powerflex Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = Client
	resp.ResourceData = Client

	tflog.Info(ctx, "Configured powerflex client", map[string]any{"success": true})
}

// DataSources - returns array of all datasources.
func (p *powerflexProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		VolumeDataSource,
		// StoragePoolDataSource,
		SDCDataSource,
		StoragePoolDataSource,
	}
}

// Resources - returns array of all resources.
func (p *powerflexProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		SDCResource,
	}
	return []func() resource.Resource{}
}
