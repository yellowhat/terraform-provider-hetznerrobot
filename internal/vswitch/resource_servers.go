package vswitch

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	// ServersResourceType is the type name of the Hetzner Robot vSwitch Servers resource.
	ServersResourceType = "hetznerrobot_vswitch_servers"
)

// ServersResource defines the vswitch servers terraform resource.
func ServersResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServersCreate,
		ReadContext:   resourceServersRead,
		UpdateContext: resourceServersUpdate,
		DeleteContext: resourceServersDelete,

		Schema: map[string]*schema.Schema{
			"vswitch_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Existing vSwitch ID.",
			},
			"servers": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of server IDs to attach to the vSwitch.",
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

func resourceServersCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	vswID := d.Get("vswitch_id").(string)

	servers := d.Get("servers")
	serverIDs := parseServerIDs(servers.([]any))
	serverObjs := parseServerIDsToVSwitchServers(serverIDs)

	if err := hClient.AddVSwitchServers(ctx, vswID, serverObjs); err != nil {
		return diag.FromErr(fmt.Errorf("error adding servers to vSwitch: %w", err))
	}

	err := hClient.WaitForVSwitchReady(ctx, vswID, waitMaxRetries, waitDuration)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for vSwitch readiness after create: %w", err))
	}

	d.SetId(vswID)

	return resourceServersRead(ctx, d, meta)
}

func resourceServersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	id := d.Id()

	vsw, err := hClient.FetchVSwitchByID(ctx, id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading vSwitch: %w", err))
	}

	vswID := strconv.Itoa(vsw.ID)

	if err = d.Set("vswitch_id", vswID); err != nil {
		return diag.FromErr(fmt.Errorf("error setting vswitch_id attribute: %w", err))
	}

	servers := flattenServers(vsw.Servers)
	sort.Ints(servers)

	if err = d.Set("servers", servers); err != nil {
		return diag.FromErr(fmt.Errorf("error setting servers attribute: %w", err))
	}

	return nil
}

func resourceServersUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	vswID := d.Id()

	var waitForReady bool

	if d.HasChange("servers") {
		if err := manageServers(ctx, d, hClient, vswID); err != nil {
			return err
		}

		waitForReady = true
	}

	if waitForReady {
		if err := hClient.WaitForVSwitchReady(ctx, vswID, waitMaxRetries, waitDuration); err != nil {
			return diag.FromErr(
				fmt.Errorf("error waiting for vSwitch readiness after update: %w", err),
			)
		}
	}

	return resourceServersRead(ctx, d, meta)
}

func resourceServersDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	id := d.Id()
	servers := d.Get("servers")
	serverIDs := parseServerIDs(servers.([]any))
	serverObjs := parseServerIDsToVSwitchServers(serverIDs)

	if err := hClient.RemoveVSwitchServers(ctx, id, serverObjs); err != nil {
		return diag.FromErr(fmt.Errorf("error removing servers from vSwitch: %w", err))
	}

	d.SetId("")

	return nil
}
