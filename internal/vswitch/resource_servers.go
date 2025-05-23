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
			"vlan_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The VLAN ID for the existing vSwitch.",
			},
			"servers": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of server IDs to connect to the vSwitch.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceServersCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	vlan := d.Get("vlan_id").(int)
	vlanID := strconv.Itoa(vlan)

	servers := d.Get("servers")
	serverIDs := parseServerIDs(servers.([]any))
	serverObjs := parseServerIDsToVSwitchServers(serverIDs)

	if err := hClient.AddVSwitchServers(ctx, vlanID, serverObjs); err != nil {
		return diag.FromErr(fmt.Errorf("error adding servers to vSwitch: %w", err))
	}

	err := hClient.WaitForVSwitchReady(ctx, vlanID, waitMaxRetries, waitDuration)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for vSwitch readiness after create: %w", err))
	}

	d.SetId(vlanID)

	return resourceServersRead(ctx, d, meta)
}

func resourceServersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	vlanID := d.Id()

	vsw, err := hClient.FetchVSwitchByID(ctx, vlanID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading vSwitch: %w", err))
	}

	if err = d.Set("vlan_id", vsw.VLAN); err != nil {
		return diag.FromErr(fmt.Errorf("error setting vlan attribute: %w", err))
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

	vlanID := d.Id()

	var waitForReady bool

	if d.HasChange("servers") {
		if err := manageServers(ctx, d, hClient, vlanID); err != nil {
			return err
		}

		waitForReady = true
	}

	if waitForReady {
		if err := hClient.WaitForVSwitchReady(ctx, vlanID, waitMaxRetries, waitDuration); err != nil {
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
