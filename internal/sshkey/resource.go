// Package sshkey defines the ssh_key terraform resource.
package sshkey

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// ResourceType is the type name of the Hetzner Robot SSH key resource.
const ResourceType = "hetznerrobot_ssh_key"

// Resource defines the ssh_key terraform resource.
func Resource() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an SSH key in the Hetzner Robot account-level key registry. " +
			"Registered keys can be referenced by fingerprint when activating the rescue " +
			"system or ordering new servers.",
		CreateContext: create,
		ReadContext:   read,
		UpdateContext: update,
		DeleteContext: delete_,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name for the key.",
			},
			"data": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Public key in OpenSSH `authorized_keys` format (e.g. `ssh-ed25519 AAAA...`). The key body is immutable; changing it forces recreate.",
			},
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "MD5 fingerprint computed by Hetzner. Used as the resource ID.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Key algorithm reported by Hetzner (e.g. `ED25519`, `RSA`).",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Key size in bits.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp at which the key was registered.",
			},
		},
	}
}

func create(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	key, err := hClient.CreateSSHKey(ctx, d.Get("name").(string), d.Get("data").(string))
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create ssh key: %w", err))
	}

	d.SetId(key.Fingerprint)

	return read(ctx, d, meta)
}

func read(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	key, err := hClient.FetchSSHKey(ctx, d.Id())
	if err != nil {
		if errors.Is(err, client.ErrSSHKeyNotFound) {
			d.SetId("")

			return nil
		}

		return diag.FromErr(fmt.Errorf("failed to read ssh key %s: %w", d.Id(), err))
	}

	for k, v := range map[string]any{
		"name":        key.Name,
		"data":        key.Data,
		"fingerprint": key.Fingerprint,
		"type":        key.Type,
		"size":        key.Size,
		"created_at":  key.CreatedAt,
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

	if d.HasChange("name") {
		err := hClient.RenameSSHKey(ctx, d.Id(), d.Get("name").(string))
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to rename ssh key %s: %w", d.Id(), err))
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

	err := hClient.DeleteSSHKey(ctx, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete ssh key %s: %w", d.Id(), err))
	}

	return nil
}
