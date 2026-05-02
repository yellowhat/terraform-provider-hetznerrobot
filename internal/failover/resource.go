// Package failover defines the failover terraform resource.
package failover

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// ResourceType is the type name of the Hetzner Robot failover resource.
const ResourceType = "hetznerrobot_failover"

// Resource defines the failover terraform resource.
func Resource() *schema.Resource {
	return &schema.Resource{
		Description: "Routes a Hetzner Robot failover IP to a target server. " +
			"Destroying the resource resets the routing back to the failover IP's primary server.",
		CreateContext: create,
		ReadContext:   read,
		UpdateContext: update,
		DeleteContext: delete_,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The failover IP to route. Must be an existing failover IP on the account.",
			},
			"active_server_ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The main IP of the server the failover IP should currently route to.",
			},
			"server_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Main IP of the failover IP's primary (owner) server.",
			},
			"server_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Server number of the failover IP's primary (owner) server.",
			},
			"netmask": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Netmask reported by the API for the failover IP.",
			},
		},
	}
}

func create(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	ip := d.Get("ip").(string)
	activeServerIP := d.Get("active_server_ip").(string)

	err := hClient.SetFailover(ctx, ip, activeServerIP)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to route failover %s: %w", ip, err))
	}

	d.SetId(ip)

	return read(ctx, d, meta)
}

func read(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	rec, err := hClient.FetchFailover(ctx, d.Id())
	if err != nil {
		if errors.Is(err, client.ErrFailoverNotFound) {
			d.SetId("")

			return nil
		}

		return diag.FromErr(fmt.Errorf("failed to read failover %s: %w", d.Id(), err))
	}

	for k, v := range map[string]any{
		"ip":               rec.IP,
		"active_server_ip": rec.ActiveServerIP,
		"server_ip":        rec.ServerIP,
		"server_number":    rec.ServerNumber,
		"netmask":          rec.Netmask,
	} {
		err = d.Set(k, v)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error setting %s attribute: %w", k, err))
		}
	}

	return nil
}

func update(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	if d.HasChange("active_server_ip") {
		err := hClient.SetFailover(ctx, d.Id(), d.Get("active_server_ip").(string))
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to reroute failover %s: %w", d.Id(), err))
		}
	}

	return read(ctx, d, meta)
}

//nolint:revive // delete is a Go builtin; trailing underscore avoids shadowing.
func delete_(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	err := hClient.DeleteFailover(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to reset failover %s: %w", d.Id(), err))
	}

	return nil
}
