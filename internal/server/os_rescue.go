package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// ResourceType is the type name of the Hetzner Robot OS Rescue resource.
const ResourceOSRescueType = "hetznerrobot_os_rescue"

type ServerInput struct {
	ID   string
	Name string
}

func ResourceOSRescue() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOSRescueCreate,
		ReadContext:   schema.NoopContext,
		UpdateContext: resourceOSRescueUpdate,
		DeleteContext: schema.NoopContext,
		Schema: map[string]*schema.Schema{
			"server_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The server will be renamed to this name.",
			},
			"server_number": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Server ID.",
			},
			"rescue_os": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "linux",
				Description: "Operating system for rescue mode (e.g. linux, freebsd).",
			},
			"ssh_keys": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of SSH keys to be added during the rescue mode.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Server public IP.",
			},
			"ssh_password": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Current Rescue System root password.",
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
	serverNumber := d.Get("server_number").(string)
	rescueOS := d.Get("rescue_os").(string)
	sshKeysRaw := d.Get("ssh_keys").([]any)

	sshKeys := make([]string, 0, len(sshKeysRaw))
	for _, key := range sshKeysRaw {
		sshKeys = append(sshKeys, key.(string))
	}

	serverID, err := strconv.Atoi(serverNumber)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID %s: %w", serverNumber, err))
	}

	rescueResp, err := hClient.EnableRescueMode(ctx, serverID, rescueOS, sshKeys)
	if err != nil {
		return diag.FromErr(
			fmt.Errorf("failed to enable rescue mode for server %d: %w", serverID, err),
		)
	}
	ip := rescueResp.Rescue.ServerIP
	pass := rescueResp.Rescue.Password

	if err := hClient.RebootServer(ctx, serverID, "hw"); err != nil {
		return diag.FromErr(
			fmt.Errorf("failed to reboot server %d with power reset: %w", serverID, err),
		)
	}

	if err := waitForSSH(ip, 3*time.Minute, 10*time.Second); err != nil {
		return diag.FromErr(fmt.Errorf("SSH not available on server %d: %w", serverID, err))
	}

	_, err = hClient.RenameServer(ctx, serverID, serverName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to rename server %d: %w", serverID, err))
	}

	if err := d.Set("ip", ip); err != nil {
		return diag.FromErr(fmt.Errorf("error setting ip attribute: %w", err))
	}

	if err := d.Set("ssh_password", pass); err != nil {
		return diag.FromErr(fmt.Errorf("error setting ssh_password attribute: %w", err))
	}

	d.SetId(serverNumber)

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
	serverNumber := d.Get("server_number").(string)

	serverID, err := strconv.Atoi(serverNumber)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID %s: %w", serverNumber, err))
	}

	serverInfo, err := hClient.FetchServerByID(ctx, serverNumber)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch server %s info: %w", serverNumber, err))
	}

	if serverName != serverInfo.ServerName {
		_, err := hClient.RenameServer(ctx, serverID, serverName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to rename server %d: %w", serverID, err))
		}
	}

	return nil
}

func waitForSSH(ip string, timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", ip), 5*time.Second)
		if err == nil {
			_ = conn.Close()
			fmt.Printf("[INFO] SSH is available on the server %s\n", ip)
			return nil
		}

		fmt.Printf(
			"[WARN] Waiting for SSH on %s... Retrying in %v seconds\n",
			ip,
			interval.Seconds(),
		)
		time.Sleep(interval)
	}

	return fmt.Errorf("SSH not available on %s after %v", ip, timeout)
}
