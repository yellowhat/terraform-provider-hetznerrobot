package client_test

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const testFirewallJSON = `{
  "firewall": {
    "ip": "1.2.3.4",
    "whitelist_hos": true,
    "filter_ipv6": false,
    "status": "active",
    "rules": {
      "input": [
        {"ip_version": "ipv4", "name": "allow-ssh", "src_ip": "0.0.0.0/0", "dst_port": "22", "protocol": "tcp", "action": "accept"},
        {"ip_version": null, "name": "allow-http", "src_ip": "0.0.0.0/0", "dst_port": "80", "protocol": "tcp", "action": "accept"}
      ]
    }
  }
}`

//nolint:gochecknoglobals
var testFirewall = client.Firewall{
	IP:                       "1.2.3.4",
	WhitelistHetznerServices: true,
	FilterIPv6:               false,
	Status:                   "active",
	Rules: client.FirewallRules{
		//exhaustruct:ignore
		Input: []client.FirewallRule{
			{
				IPVersion: "ipv4",
				Name:      "allow-ssh",
				SrcIP:     "0.0.0.0/0",
				DstPort:   "22",
				Protocol:  "tcp",
				Action:    "accept",
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

func TestGetFirewall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		want    *client.Firewall
		wantErr string
	}{
		{
			name:    "success",
			status:  http.StatusOK,
			body:    testFirewallJSON,
			want:    &testFirewall,
			wantErr: "",
		},
		{
			name:    "not found",
			status:  http.StatusNotFound,
			body:    apiError(http.StatusNotFound, "SERVER_NOT_FOUND"),
			want:    nil,
			wantErr: "unexpected response status: 404",
		},
		{
			name:    "invalid json",
			status:  http.StatusOK,
			body:    `{`,
			want:    nil,
			wantErr: "failed to parse firewall response",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /firewall/1.2.3.4": respond(test.status, test.body),
			})

			got, err := api.GetFirewall(context.Background(), "1.2.3.4")
			assertErr(t, err, test.wantErr)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("firewall\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestSetFirewall(t *testing.T) {
	t.Parallel()

	// ip_version defaults to ipv4, empty fields are omitted.
	wantForm := url.Values{
		"whitelist_hos":               []string{"true"},
		"filter_ipv6":                 []string{"false"},
		"status":                      []string{"active"},
		"rules[input][0][ip_version]": []string{"ipv4"},
		"rules[input][0][name]":       []string{"allow-ssh"},
		"rules[input][0][src_ip]":     []string{"0.0.0.0/0"},
		"rules[input][0][dst_port]":   []string{"22"},
		"rules[input][0][protocol]":   []string{"tcp"},
		"rules[input][0][action]":     []string{"accept"},
		"rules[input][1][ip_version]": []string{"ipv4"},
		"rules[input][1][name]":       []string{"allow-http"},
		"rules[input][1][src_ip]":     []string{"0.0.0.0/0"},
		"rules[input][1][dst_port]":   []string{"80"},
		"rules[input][1][protocol]":   []string{"tcp"},
		"rules[input][1][action]":     []string{"accept"},
	}

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /firewall/1.2.3.4": expectForm(t, wantForm, http.StatusAccepted, testFirewallJSON),
		// SetFirewall polls until the firewall is active.
		"GET /firewall/1.2.3.4": respond(http.StatusOK, testFirewallJSON),
	})

	err := api.SetFirewall(context.Background(), testFirewall)
	if err != nil {
		t.Errorf("SetFirewall() error: %v", err)
	}
}

// Mixed rules: ip_version must not leak between indexes.
func TestSetFirewallIPv6(t *testing.T) {
	t.Parallel()

	firewall := client.Firewall{
		IP:                       "1.2.3.4",
		WhitelistHetznerServices: false,
		FilterIPv6:               true,
		Status:                   "active",
		Rules: client.FirewallRules{
			//exhaustruct:ignore
			Input: []client.FirewallRule{
				{
					Name:     "allow-ssh",
					SrcIP:    "0.0.0.0/0",
					DstPort:  "22",
					Protocol: "tcp",
					Action:   "accept",
				},
				{
					IPVersion: "ipv6",
					Name:      "allow-ssh-v6",
					SrcIP:     "::/0",
					DstPort:   "22",
					Protocol:  "tcp",
					Action:    "accept",
				},
			},
		},
	}

	wantForm := url.Values{
		"whitelist_hos":               []string{"false"},
		"filter_ipv6":                 []string{"true"},
		"status":                      []string{"active"},
		"rules[input][0][ip_version]": []string{"ipv4"},
		"rules[input][0][name]":       []string{"allow-ssh"},
		"rules[input][0][src_ip]":     []string{"0.0.0.0/0"},
		"rules[input][0][dst_port]":   []string{"22"},
		"rules[input][0][protocol]":   []string{"tcp"},
		"rules[input][0][action]":     []string{"accept"},
		"rules[input][1][ip_version]": []string{"ipv6"},
		"rules[input][1][name]":       []string{"allow-ssh-v6"},
		"rules[input][1][src_ip]":     []string{"::/0"},
		"rules[input][1][dst_port]":   []string{"22"},
		"rules[input][1][protocol]":   []string{"tcp"},
		"rules[input][1][action]":     []string{"accept"},
	}

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /firewall/1.2.3.4": expectForm(t, wantForm, http.StatusOK, testFirewallJSON),
		"GET /firewall/1.2.3.4":  respond(http.StatusOK, testFirewallJSON),
	})

	err := api.SetFirewall(context.Background(), firewall)
	if err != nil {
		t.Errorf("SetFirewall() error: %v", err)
	}
}

func TestSetFirewallError(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /firewall/1.2.3.4": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "FIREWALL_IN_PROCESS"),
		),
	})

	err := api.SetFirewall(context.Background(), testFirewall)
	assertErr(t, err, "unexpected response status: 409")
}
