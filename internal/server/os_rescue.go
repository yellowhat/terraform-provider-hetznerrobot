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
			"server_id": {
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
	serverID := d.Get("server_id").(string)
	rescueOS := d.Get("rescue_os").(string)
	sshKeysRaw := d.Get("ssh_keys").([]any)

	sshKeys := make([]string, 0, len(sshKeysRaw))
	for _, key := range sshKeysRaw {
		sshKeys = append(sshKeys, key.(string))
	}

	rescueResp, err := hClient.EnableRescueMode(ctx, serverID, rescueOS, sshKeys)
	if err != nil {
		return diag.FromErr(
			fmt.Errorf("failed to enable rescue mode for server %s: %w", serverID, err),
		)
	}

	ip := rescueResp.Rescue.ServerIP
	pass := rescueResp.Rescue.Password

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
