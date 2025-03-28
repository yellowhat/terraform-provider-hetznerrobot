package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/yellowhat/terraform-provider-hetznerrobot/client"
	"github.com/yellowhat/terraform-provider-hetznerrobot/data_sources"
	"github.com/yellowhat/terraform-provider-hetznerrobot/resources"
	"github.com/yellowhat/terraform-provider-hetznerrobot/shared"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Hetzner Robot API username.",
				DefaultFunc: schema.EnvDefaultFunc("HETZNERROBOT_USERNAME", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Hetzner Robot API password.",
				DefaultFunc: schema.EnvDefaultFunc("HETZNERROBOT_PASSWORD", nil),
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HETZNERROBOT_URL", "https://robot-ws.your-server.de"),
				Description: "Base URL for the Hetzner Robot API.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"hetznerrobot_firewall":  resources.ResourceFirewall(),
			"hetznerrobot_os_rescue": resources.ResourceOSRescue(),
			"hetznerrobot_vswitch":   resources.ResourceVSwitch(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"hetznerrobot_server":  data_sources.DataSourceServers(),
			"hetznerrobot_vswitch": data_sources.DataSourceVSwitches(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	var diags diag.Diagnostics
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	url := d.Get("url").(string)
	if username == "" || password == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Missing credentials",
			Detail:   "Both username and password must be provided.",
		})
		return nil, diags
	}
	config := &shared.ProviderConfig{
		Username: username,
		Password: password,
		BaseURL:  url,
	}
	client := client.NewHetznerRobotClient(config)
	return client, diags
}
