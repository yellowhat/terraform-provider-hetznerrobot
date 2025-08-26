package client_test

import (
	"context"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

func TestGetFirewall(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		IP         string
		wantStatus string
	}

	testCases := []testCase{
		{
			name:       "successful firewall retrieval",
			IP:         "1.2.3.4",
			wantStatus: "active",
		},
	}

	for _, test := range testCases {
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

			if test.wantStatus != firewall.Status {
				t.Errorf("Status: want %v, got %v", test.wantStatus, firewall.Status)
			}

			if !firewall.WhitelistHetznerServices {
				t.Errorf("GetFirewall() WhitelistHetznerServices = %v, want true", firewall.WhitelistHetznerServices)
			}
		})
	}
}
