package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	// ResourceOSRescueType is the type name of the Hetzner Robot OS Rescue resource.
	ResourceOSRescueType = "hetznerrobot_os_rescue"
	waitMin              = 3
	retryAfterSec        = 10
)

// ResourceOSRescue defines the os_rescue terraform resource.
func ResourceOSRescue() *schema.Resource {
	return &schema.Resource{
		Description: `Reboot a server into Hetzner Robot rescue system:
1. activate the Hetzner Robot rescue system
2. issue a hw reset (equivalent to pressing the reset button)
3. wait for the rescue system's SSH port to come up
4. rename the server

Updates only handle server_name changes; all other fields are effectively immutable.
Read and Delete are no-ops, so destroying the resource does not deactivate rescue mode or reboot the server back to its installed OS.`,
		CreateContext: resourceOSRescueCreate,
		ReadContext:   schema.NoopContext,
		UpdateContext: resourceOSRescueUpdate,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"server_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name to assign to the server after the rescue system is reachable. The only field whose change is honored by Update.",
			},
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Server ID (Hetzner server number).",
			},
			"rescue_os": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "linux",
				Description: "Operating system for rescue mode (e.g. linux, freebsd).",
			},
			"ssh_keys": {
				Type:     schema.TypeList,
				Optional: true,
				Description: "List of public SSH keys to install in the rescue system's authorized_keys. " +
					"If non-empty, the rescue system disables password authentication and `ssh_password` will be empty. " +
					"If left empty, Hetzner generates a one-shot root password (returned in `ssh_password`).",
				Elem: &schema.Schema{Type: schema.TypeString},
			},
			"reboot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
				Description: "Whether to trigger a hardware reset to boot into the rescue system after activation. " +
					"When `false`, the rescue system is armed for the next boot but the server is left in its current running state; " +
					"the caller is expected to reboot it. Skipping the reboot also skips the wait for SSH on the rescue system. " +
					"Only takes effect on Create — flipping this on an existing resource forces recreate.",
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Public IPv4 of the server.",
			},
			"ssh_password": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "One-shot root password for the rescue system. Set only when ssh_keys is empty; otherwise this is empty and you authenticate with one of the listed keys.",
			},
		},
	}
}

func resourceOSRescueCreate(
	ctx context.Context,
	d *schema.ResourceData,
	meta any,
) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverName := d.Get("server_name").(string)
	serverID := d.Get("server_id").(string)
	rescueOS := d.Get("rescue_os").(string)
	sshKeys := parseSSHKeys(d.Get("ssh_keys").([]any))

	rescueResp, err := hClient.EnableRescueMode(ctx, serverID, rescueOS, sshKeys)
	if err != nil {
		return diag.FromErr(
			fmt.Errorf("failed to enable rescue mode for server %s: %w", serverID, err),
		)
	}

	ip := rescueResp.Rescue.ServerIP
	pass := rescueResp.Rescue.Password

	if d.Get("reboot").(bool) {
		err = hClient.RebootServer(ctx, serverID, "hw")
		if err != nil {
			return diag.FromErr(
				fmt.Errorf("failed to reboot server %s with power reset: %w", serverID, err),
			)
		}

		err = waitForSSH(ctx, ip, waitMin*time.Minute, retryAfterSec*time.Second)
		if err != nil {
			return diag.FromErr(fmt.Errorf("SSH not available on server %s: %w", serverID, err))
		}
	}

	_, err = hClient.RenameServer(ctx, serverID, serverName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to rename server %s: %w", serverID, err))
	}

	err = d.Set("ip", ip)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error setting ip attribute: %w", err))
	}

	err = d.Set("ssh_password", pass)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error setting ssh_password attribute: %w", err))
	}

	d.SetId(serverID)

	return nil
}

func resourceOSRescueUpdate(
	ctx context.Context,
	d *schema.ResourceData,
	meta any,
) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverName := d.Get("server_name").(string)
	serverID := d.Get("server_id").(string)

	serverInfo, err := hClient.FetchServerByID(ctx, serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch server %s info: %w", serverID, err))
	}

	if serverName != serverInfo.ServerName {
		_, err := hClient.RenameServer(ctx, serverID, serverName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to rename server %s: %w", serverID, err))
		}
	}

	return nil
}

func parseSSHKeys(raw []any) []string {
	keys := make([]string, 0, len(raw))
	for _, key := range raw {
		keys = append(keys, key.(string))
	}

	return keys
}

func waitForSSH(
	ctx context.Context,
	ip string,
	timeout time.Duration,
	interval time.Duration,
) error {
	const waitTime = 5

	//exhaustruct:ignore
	dialer := &net.Dialer{
		Timeout: waitTime * time.Second,
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(ctx, "tcp", ip+":22")
		if err == nil {
			_ = conn.Close()

			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("SSH not available on %s after %v", ip, timeout)
}
