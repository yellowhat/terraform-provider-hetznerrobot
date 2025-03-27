package client

import (
	"net/http"

	"hetznerrobot-provider/shared"
)

type NotFoundError struct {
	Message string
}

type VSwitchCloudNetwork struct {
	ID      int    `json:"id"`
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

type Server struct {
	IP         string `json:"server_ip"`
	IPv6Net    string `json:"server_ipv6_net"`
	Number     int    `json:"server_number"`
	ServerName string `json:"server_name"`
	Product    string `json:"product"`
	Datacenter string `json:"dc"`
	Traffic    string `json:"traffic"`
	Status     string `json:"status"`
	Cancelled  bool   `json:"cancelled"`
	PaidUntil  string `json:"paid_until"`
}

type HetznerRobotClient struct {
	Config *shared.ProviderConfig
	Client *http.Client
}

type VSwitch struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	VLAN      int               `json:"vlan"`
	Cancelled bool              `json:"cancelled"`
	Servers   []VSwitchServer   `json:"server"`
	Subnets   []VSwitchSubnet   `json:"subnets"`
	CloudNets []VSwitchCloudNet `json:"cloud_networks"`
}

type VSwitchServer struct {
	ServerNumber  int    `json:"server_number,omitempty"`
	ServerIP      string `json:"server_ip,omitempty"`
	ServerIPv6Net string `json:"server_ipv6_net,omitempty"`
	Status        string `json:"status,omitempty"`
}

type VSwitchSubnet struct {
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

type VSwitchCloudNet struct {
	ID      int    `json:"id"`
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

type HetznerRobotFirewall struct {
	IP                       string                    `json:"ip"`
	WhitelistHetznerServices bool                      `json:"whitelist_hos"`
	Status                   string                    `json:"status"`
	Rules                    HetznerRobotFirewallRules `json:"rules"`
}

type HetznerRobotFirewallRules struct {
	Input []HetznerRobotFirewallRule `json:"input"`
}

type HetznerRobotFirewallRule struct {
	Name     string `json:"name,omitempty"`
	SrcIP    string `json:"src_ip,omitempty"`
	SrcPort  string `json:"src_port,omitempty"`
	DstIP    string `json:"dst_ip,omitempty"`
	DstPort  string `json:"dst_port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	TCPFlags string `json:"tcp_flags,omitempty"`
	Action   string `json:"action"`
}

type HetznerRobotFirewallResponse struct {
	Firewall HetznerRobotFirewall `json:"firewall"`
}

type HetznerResetResponse struct {
	Reset struct {
		Type string `json:"type"`
	} `json:"reset"`
}

type HetznerRescueResponse struct {
	Rescue struct {
		ServerIP string `json:"server_ip"`
		Password string `json:"password"`
	} `json:"rescue"`
}

type HetznerRenameResponse struct {
	Server struct {
		ServerName string `json:"server_name"`
	} `json:"server"`
}
