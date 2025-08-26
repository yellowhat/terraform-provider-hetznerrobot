package client_test

import (
	"context"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

func TestGetFirewall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		IP         string
		wantHOS    bool
		wantStatus string
		wantRules  []client.FirewallRule
	}{
		{
			name:       "successful firewall retrieval",
			IP:         "1.2.3.4",
			wantHOS:    true,
			wantStatus: "active",
			//exhaustruct:ignore
			wantRules: []client.FirewallRule{
				{
					Name:     "allow-ssh",
					SrcIP:    "0.0.0.0/0",
					DstPort:  "22",
					Protocol: "tcp",
					Action:   "accept",
				},
				{
					Name:     "allow-http",
					SrcIP:    "0.0.0.0/0",
					DstPort:  "80",
					Protocol: "tcp",
					Action:   "accept",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := mockServer()
			defer server.Close()

			client := client.New(&client.ProviderConfig{
				Username: testUsername,
				Password: testPassword,
				BaseURL:  server.URL,
			})

			firewall, err := client.GetFirewall(context.Background(), test.IP)
			if err != nil {
				t.Errorf("GetFirewall() error: %v", err)
			}

			if test.IP != firewall.IP {
				t.Errorf("IP: want %v, got %v", test.IP, firewall.IP)
			}

			if test.wantHOS != firewall.WhitelistHetznerServices {
				t.Errorf(
					"WhitelistHetznerServices: want %t, got %t",
					test.wantHOS,
					firewall.WhitelistHetznerServices,
				)
			}

			if test.wantStatus != firewall.Status {
				t.Errorf("Status: want %v, got %v", test.wantStatus, firewall.Status)
			}

			if len(test.wantRules) != len(firewall.Rules.Input) {
				t.Errorf(
					"Rules length: want %d, got %d",
					len(test.wantRules),
					len(firewall.Rules.Input),
				)
			}

			for i, wantRule := range test.wantRules {
				gotRule := firewall.Rules.Input[i]
				if wantRule.Name != gotRule.Name {
					t.Errorf("Rule[%d] Name: want %v, got %v", i, wantRule.Name, gotRule.Name)
				}

				if wantRule.SrcIP != gotRule.SrcIP {
					t.Errorf("Rule[%d] SrcIP: want %v, got %v", i, wantRule.SrcIP, gotRule.SrcIP)
				}

				if wantRule.DstPort != gotRule.DstPort {
					t.Errorf(
						"Rule[%d] DstPort: want %v, got %v",
						i,
						wantRule.DstPort,
						gotRule.DstPort,
					)
				}

				if wantRule.Protocol != gotRule.Protocol {
					t.Errorf(
						"Rule[%d] Protocol: want %v, got %v",
						i,
						wantRule.Protocol,
						gotRule.Protocol,
					)
				}

				if wantRule.Action != gotRule.Action {
					t.Errorf("Rule[%d] Action: want %v, got %v", i, wantRule.Action, gotRule.Action)
				}
			}
		})
	}
}
