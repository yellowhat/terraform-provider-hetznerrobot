// Package main is the entrypoint for the Hetzner Robot terraform provider.
package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/yellowhat/terraform-provider-hetznerrobot/hetznerrobot"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: hetznerrobot.Provider,
	})
}
