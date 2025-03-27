package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (c *HetznerRobotClient) GetFirewall(ctx context.Context, ip string) (*HetznerRobotFirewall, error) {
	path := fmt.Sprintf("/firewall/%s", ip)
	resp, err := c.DoRequest("GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	var fwResp HetznerRobotFirewallResponse
	if err := json.NewDecoder(resp.Body).Decode(&fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse firewall response: %w", err)
	}

	return &fwResp.Firewall, nil
}

func (c *HetznerRobotClient) SetFirewall(ctx context.Context, firewall HetznerRobotFirewall, maxRetries int, waitTime time.Duration) error {
	path := fmt.Sprintf("/firewall/%s", firewall.IP)
	whitelistHOS := "false"
	if firewall.WhitelistHetznerServices {
		whitelistHOS = "true"
	}

	data := url.Values{}
	data.Set("whitelist_hos", whitelistHOS)
	data.Set("status", firewall.Status)

	for idx, rule := range firewall.Rules.Input {
		data.Set(fmt.Sprintf("rules[input][%d][ip_version]", idx), "ipv4")
		if rule.Name != "" {
			data.Set(fmt.Sprintf("rules[input][%d][name]", idx), rule.Name)
		}
		if rule.DstIP != "" {
			data.Set(fmt.Sprintf("rules[input][%d][dst_ip]", idx), rule.DstIP)
		}
		if rule.DstPort != "" {
			data.Set(fmt.Sprintf("rules[input][%d][dst_port]", idx), rule.DstPort)
		}
		if rule.SrcIP != "" {
			data.Set(fmt.Sprintf("rules[input][%d][src_ip]", idx), rule.SrcIP)
		}
		if rule.SrcPort != "" {
			data.Set(fmt.Sprintf("rules[input][%d][src_port]", idx), rule.SrcPort)
		}
		if rule.Protocol != "" {
			data.Set(fmt.Sprintf("rules[input][%d][protocol]", idx), rule.Protocol)
		}
		if rule.TCPFlags != "" {
			data.Set(fmt.Sprintf("rules[input][%d][tcp_flags]", idx), rule.TCPFlags)
		}
		data.Set(fmt.Sprintf("rules[input][%d][action]", idx), rule.Action)
	}

	resp, err := c.DoRequest("POST", path, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("failed to set firewall: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return c.waitForFirewallActive(ctx, firewall.IP, maxRetries, waitTime)
}

func (c *HetznerRobotClient) waitForFirewallActive(ctx context.Context, ip string, maxRetries int, waitTime time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		firewall, err := c.GetFirewall(ctx, ip)
		if err != nil {
			return fmt.Errorf("error checking firewall status: %w", err)
		}

		if firewall.Status == "active" {
			fmt.Println("Firewall is now active.")
			return nil
		}

		fmt.Printf("Waiting for firewall to become active... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(waitTime)
	}

	return fmt.Errorf("timeout waiting for firewall to become active")
}
