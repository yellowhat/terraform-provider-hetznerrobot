package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Firewall struct {
	IP                       string        `json:"ip"`
	WhitelistHetznerServices bool          `json:"whitelist_hos"`
	Status                   string        `json:"status"`
	Rules                    FirewallRules `json:"rules"`
}

type FirewallRules struct {
	Input []FirewallRule `json:"input"`
}

type FirewallRule struct {
	Name     string `json:"name,omitempty"`
	SrcIP    string `json:"src_ip,omitempty"`
	SrcPort  string `json:"src_port,omitempty"`
	DstIP    string `json:"dst_ip,omitempty"`
	DstPort  string `json:"dst_port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	TCPFlags string `json:"tcp_flags,omitempty"`
	Action   string `json:"action"`
}

type FirewallResponse struct {
	Firewall Firewall `json:"firewall"`
}

func (c *HetznerRobotClient) GetFirewall(ctx context.Context, ip string) (*Firewall, error) {
	path := fmt.Sprintf("/firewall/%s", ip)
	resp, err := c.DoRequest("GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, data)
	}

	var fwResp FirewallResponse
	if err := json.NewDecoder(resp.Body).Decode(&fwResp); err != nil {
		return nil, fmt.Errorf("failed to parse firewall response: %w", err)
	}

	return &fwResp.Firewall, nil
}

func (c *HetznerRobotClient) SetFirewall(
	ctx context.Context,
	firewall Firewall,
	maxRetries int,
	waitTime time.Duration,
) error {
	path := fmt.Sprintf("/firewall/%s", firewall.IP)

	data := url.Values{}
	data.Set("whitelist_hos", fmt.Sprintf("%t", firewall.WhitelistHetznerServices))
	data.Set("status", firewall.Status)

	for index, rule := range firewall.Rules.Input {
		data.Set(fmt.Sprintf("rules[input][%d][ip_version]", index), "ipv4")
		if rule.Name != "" {
			data.Set(fmt.Sprintf("rules[input][%d][name]", index), rule.Name)
		}
		if rule.SrcIP != "" {
			data.Set(fmt.Sprintf("rules[input][%d][src_ip]", index), rule.SrcIP)
		}
		if rule.SrcPort != "" {
			data.Set(fmt.Sprintf("rules[input][%d][src_port]", index), rule.SrcPort)
		}
		if rule.DstIP != "" {
			data.Set(fmt.Sprintf("rules[input][%d][dst_ip]", index), rule.DstIP)
		}
		if rule.DstPort != "" {
			data.Set(fmt.Sprintf("rules[input][%d][dst_port]", index), rule.DstPort)
		}
		if rule.Protocol != "" {
			data.Set(fmt.Sprintf("rules[input][%d][protocol]", index), rule.Protocol)
		}
		if rule.TCPFlags != "" {
			data.Set(fmt.Sprintf("rules[input][%d][tcp_flags]", index), rule.TCPFlags)
		}
		data.Set(fmt.Sprintf("rules[input][%d][action]", index), rule.Action)
	}

	resp, err := c.DoRequest(
		"POST",
		path,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("failed to set firewall: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, data)
	}

	return c.waitForFirewallActive(ctx, firewall.IP, maxRetries, waitTime)
}

func (c *HetznerRobotClient) waitForFirewallActive(
	ctx context.Context,
	ip string,
	maxRetries int,
	waitTime time.Duration,
) error {
	for i := range maxRetries {
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
