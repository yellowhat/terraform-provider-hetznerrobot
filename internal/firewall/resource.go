package firewall

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	// ResourceType is the type name of the Hetzner Robot Firewall resource.
	ResourceType = "hetznerrobot_firewall"
	statusTrue   = "active"
)

func Resource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCreate,
		ReadContext:   resourceRead,
		UpdateContext: resourceUpdate,
		DeleteContext: resourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceFirewallImportState,
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the server to which the firewall will be applied.",
			},
			"active": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the firewall is active.",
			},
			"whitelist_hos": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether to whitelist Hetzner services.",
			},
			"rule": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Name of the firewall rule.",
						},
						"src_ip": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Source IP address.",
						},
						"src_port": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Source port.",
						},
						"dst_ip": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Destination IP address.",
						},
						"dst_port": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Destination port.",
						},
						"protocol": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Protocol (e.g., tcp, udp).",
						},
						"tcp_flags": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "TCP flags.",
						},
						"action": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(
								validation.StringInSlice([]string{"accept", "discard"}, false),
							),
							Description: "Action to take (accept or discard).",
						},
					},
				},
			},
		},
	}
}

func resourceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(ctx, serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	status := "disabled"
	if d.Get("active").(bool) {
		status = statusTrue
	}

	rules := buildFirewallRules(d.Get("rule").([]any))

	err = hClient.SetFirewall(ctx, client.Firewall{
		IP:                       server.IP,
		WhitelistHetznerServices: d.Get("whitelist_hos").(bool),
		Status:                   status,
		Rules:                    client.FirewallRules{Input: rules},
	}, 20, 15*time.Second)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)

	return resourceRead(ctx, d, meta)
}

func resourceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)

	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(ctx, serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	firewall, err := hClient.GetFirewall(ctx, server.IP)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = d.Set("active", firewall.Status == statusTrue); err != nil {
		return diag.FromErr(fmt.Errorf("error setting active attribute: %w", err))
	}

	if err = d.Set("whitelist_hos", firewall.WhitelistHetznerServices); err != nil {
		return diag.FromErr(fmt.Errorf("error setting whitelist_hos attribute: %w", err))
	}

	if err = d.Set("rule", flattenFirewallRules(firewall.Rules.Input)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting rule attribute: %w", err))
	}

	return nil
}

func resourceUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return resourceCreate(ctx, d, meta)
}

func resourceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(ctx, serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	// Set a rule to allow all traffic
	err = hClient.SetFirewall(ctx, client.Firewall{
		IP:                       server.IP,
		WhitelistHetznerServices: false,
		Status:                   "active",
		Rules: client.FirewallRules{
			Input: []client.FirewallRule{
				{
					Name:     "Allow all",
					SrcIP:    "",
					SrcPort:  "",
					DstIP:    "",
					DstPort:  "",
					Protocol: "",
					TCPFlags: "",
					Action:   "accept",
				},
			},
		},
	}, 20, 15*time.Second)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error setting firewall to allow all: %w", err))
	}

	d.SetId("")

	return nil
}

func resourceFirewallImportState(
	ctx context.Context,
	d *schema.ResourceData,
	meta any,
) ([]*schema.ResourceData, error) {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return nil, fmt.Errorf("invalid client type")
	}

	serverID := d.Id()
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	server, err := hClient.FetchServerByID(ctx, serverIDInt)
	if err != nil {
		return nil, fmt.Errorf("error fetching server: %w", err)
	}

	firewall, err := hClient.GetFirewall(ctx, server.IP)
	if err != nil {
		return nil, fmt.Errorf("could not find firewall for server ID %s: %w", serverID, err)
	}

	if err = d.Set("active", firewall.Status == "active"); err != nil {
		return nil, fmt.Errorf("error setting active attribute: %w", err)
	}

	if err = d.Set("whitelist_hos", firewall.WhitelistHetznerServices); err != nil {
		return nil, fmt.Errorf("error setting whitelist_hos attribute: %w", err)
	}

	if err = d.Set("rule", flattenFirewallRules(firewall.Rules.Input)); err != nil {
		return nil, fmt.Errorf("error setting rule attribute: %w", err)
	}

	if err = d.Set("server_id", serverID); err != nil {
		return nil, fmt.Errorf("error setting server_id attribute: %w", err)
	}

	d.SetId(serverID)

	return []*schema.ResourceData{d}, nil
}

// Helper functions
func buildFirewallRules(ruleList []any) []client.FirewallRule {
	rules := make([]client.FirewallRule, 0, len(ruleList))
	for _, ruleMap := range ruleList {
		ruleProps := ruleMap.(map[string]any)
		rules = append(rules, client.FirewallRule{
			Name:     ruleProps["name"].(string),
			SrcIP:    ruleProps["src_ip"].(string),
			SrcPort:  ruleProps["src_port"].(string),
			DstIP:    ruleProps["dst_ip"].(string),
			DstPort:  ruleProps["dst_port"].(string),
			Protocol: ruleProps["protocol"].(string),
			TCPFlags: ruleProps["tcp_flags"].(string),
			Action:   ruleProps["action"].(string),
		})
	}
	return rules
}

func flattenFirewallRules(rules []client.FirewallRule) []map[string]any {
	result := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		result = append(result, map[string]any{
			"name":      rule.Name,
			"src_ip":    rule.SrcIP,
			"src_port":  rule.SrcPort,
			"dst_ip":    rule.DstIP,
			"dst_port":  rule.DstPort,
			"protocol":  rule.Protocol,
			"tcp_flags": rule.TCPFlags,
			"action":    rule.Action,
		})
	}
	return result
}
