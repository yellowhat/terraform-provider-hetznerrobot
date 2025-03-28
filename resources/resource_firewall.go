package resources

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/yellowhat/terraform-provider-hetznerrobot/client"
)

func ResourceFirewall() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallCreate,
		ReadContext:   resourceFirewallRead,
		UpdateContext: resourceFirewallUpdate,
		DeleteContext: resourceFirewallDelete,
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
						"name":      {Type: schema.TypeString, Optional: true, Description: "Name of the firewall rule."},
						"dst_ip":    {Type: schema.TypeString, Optional: true, Description: "Destination IP address."},
						"dst_port":  {Type: schema.TypeString, Optional: true, Description: "Destination port."},
						"src_ip":    {Type: schema.TypeString, Optional: true, Description: "Source IP address."},
						"src_port":  {Type: schema.TypeString, Optional: true, Description: "Source port."},
						"protocol":  {Type: schema.TypeString, Optional: true, Description: "Protocol (e.g., tcp, udp)."},
						"tcp_flags": {Type: schema.TypeString, Optional: true, Description: "TCP flags."},
						"action": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"accept", "discard"}, false)),
							Description:      "Action to take (accept or discard).",
						},
					},
				},
			},
		},
	}
}

func resourceFirewallCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	status := "disabled"
	if d.Get("active").(bool) {
		status = "active"
	}

	rules := buildFirewallRules(d.Get("rule").([]any))

	err = hClient.SetFirewall(ctx, client.HetznerRobotFirewall{
		IP:                       server.IP,
		WhitelistHetznerServices: d.Get("whitelist_hos").(bool),
		Status:                   status,
		Rules:                    client.HetznerRobotFirewallRules{Input: rules},
	}, 20, 15*time.Second)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(serverID)
	return resourceFirewallRead(ctx, d, meta)
}

func resourceFirewallRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)

	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	firewall, err := hClient.GetFirewall(ctx, server.IP)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("active", firewall.Status == "active")
	d.Set("whitelist_hos", firewall.WhitelistHetznerServices)
	d.Set("rule", flattenFirewallRules(firewall.Rules.Input))

	return nil
}

func resourceFirewallUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return resourceFirewallCreate(ctx, d, meta)
}

func resourceFirewallDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return diag.Errorf("invalid client type")
	}

	serverID := d.Get("server_id").(string)
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("invalid server ID: %w", err))
	}

	server, err := hClient.FetchServerByID(serverIDInt)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error fetching server: %w", err))
	}

	// Set a rule to allow all traffic
	err = hClient.SetFirewall(ctx, client.HetznerRobotFirewall{
		IP:                       server.IP,
		WhitelistHetznerServices: false,
		Status:                   "active",
		Rules: client.HetznerRobotFirewallRules{
			Input: []client.HetznerRobotFirewallRule{
				{
					Name:     "Allow all",
					Protocol: "",
					DstIP:    "",
					SrcIP:    "",
					DstPort:  "",
					SrcPort:  "",
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

func resourceFirewallImportState(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	hClient, ok := meta.(*client.HetznerRobotClient)
	if !ok {
		return nil, fmt.Errorf("invalid client type")
	}

	serverID := d.Id()
	serverIDInt, err := strconv.Atoi(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %w", err)
	}

	server, err := hClient.FetchServerByID(serverIDInt)
	if err != nil {
		return nil, fmt.Errorf("error fetching server: %w", err)
	}

	firewall, err := hClient.GetFirewall(ctx, server.IP)
	if err != nil {
		return nil, fmt.Errorf("could not find firewall for server ID %s: %w", serverID, err)
	}

	d.Set("active", firewall.Status == "active")
	d.Set("whitelist_hos", firewall.WhitelistHetznerServices)
	d.Set("rule", flattenFirewallRules(firewall.Rules.Input))
	d.Set("server_id", serverID)
	d.SetId(serverID)

	return []*schema.ResourceData{d}, nil
}

// Helper functions
func buildFirewallRules(ruleList []any) []client.HetznerRobotFirewallRule {
	var rules []client.HetznerRobotFirewallRule
	for _, ruleMap := range ruleList {
		ruleProps := ruleMap.(map[string]any)
		rules = append(rules, client.HetznerRobotFirewallRule{
			Name:     ruleProps["name"].(string),
			DstIP:    ruleProps["dst_ip"].(string),
			DstPort:  ruleProps["dst_port"].(string),
			SrcIP:    ruleProps["src_ip"].(string),
			SrcPort:  ruleProps["src_port"].(string),
			Protocol: ruleProps["protocol"].(string),
			TCPFlags: ruleProps["tcp_flags"].(string),
			Action:   ruleProps["action"].(string),
		})
	}
	return rules
}

func flattenFirewallRules(rules []client.HetznerRobotFirewallRule) []map[string]any {
	var result []map[string]any
	for _, rule := range rules {
		result = append(result, map[string]any{
			"name":      rule.Name,
			"dst_ip":    rule.DstIP,
			"dst_port":  rule.DstPort,
			"src_ip":    rule.SrcIP,
			"src_port":  rule.SrcPort,
			"protocol":  rule.Protocol,
			"tcp_flags": rule.TCPFlags,
			"action":    rule.Action,
		})
	}
	return result
}
