package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// apiCall invokes one client method, keeping only its error.
type apiCall struct {
	name       string
	decodes    bool  // parses a JSON body on 200
	notFound   error // sentinel returned on 404, nil for a generic error
	notFoundOK bool  // 404 is treated as success
	run        func(ctx context.Context, api *client.HetznerRobotClient) error
}

// allCalls lists every exported endpoint method (DoRequest is tested separately).
//
//nolint:gochecknoglobals
var allCalls = []apiCall{
	{
		name:       "FetchFailover",
		decodes:    true,
		notFound:   client.ErrFailoverNotFound,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchFailover(ctx, "1.2.3.4"))
		},
	},
	{
		name:       "SetFailover",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.SetFailover(ctx, "1.2.3.4", "5.6.7.8")
		},
	},
	{
		name:       "DeleteFailover",
		decodes:    false,
		notFound:   nil,
		notFoundOK: true,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteFailover(ctx, "1.2.3.4")
		},
	},
	{
		name:       "GetFirewall",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.GetFirewall(ctx, "1.2.3.4"))
		},
	},
	{
		name:       "SetFirewall",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.SetFirewall(ctx, testFirewall)
		},
	},
	{
		name:       "FetchServerByID",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchServerByID(ctx, "1"))
		},
	},
	{
		name:       "FetchServersByIDs",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchServersByIDs(ctx, []string{"1"}))
		},
	},
	{
		name:       "FetchAllServers",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchAllServers(ctx))
		},
	},
	{
		name:       "RenameServer",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.RenameServer(ctx, "1", "new"))
		},
	},
	{
		name:       "EnableRescueMode",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.EnableRescueMode(ctx, "1", "linux", []string{"aa:bb"}))
		},
	},
	{
		name:       "RebootServer",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RebootServer(ctx, "1", "sw")
		},
	},
	{
		name:       "FetchSSHKey",
		decodes:    true,
		notFound:   client.ErrSSHKeyNotFound,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchSSHKey(ctx, "aa:bb"))
		},
	},
	{
		name:       "CreateSSHKey",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.CreateSSHKey(ctx, "key", "ssh-ed25519 AAAA"))
		},
	},
	{
		name:       "RenameSSHKey",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RenameSSHKey(ctx, "aa:bb", "new")
		},
	},
	{
		name:       "DeleteSSHKey",
		decodes:    false,
		notFound:   nil,
		notFoundOK: true,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteSSHKey(ctx, "aa:bb")
		},
	},
	{
		name:       "FetchVSwitchByID",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchVSwitchByID(ctx, "1"))
		},
	},
	{
		name:       "FetchVSwitchesByIDs",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchVSwitchesByIDs(ctx, []string{"1"}))
		},
	},
	{
		name:       "FetchAllVSwitches",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchAllVSwitches(ctx))
		},
	},
	{
		name:       "CreateVSwitch",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.CreateVSwitch(ctx, "vsw", 4000))
		},
	},
	{
		name:       "UpdateVSwitch",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.UpdateVSwitch(ctx, "1", "vsw", 4000)
		},
	},
	{
		name:       "DeleteVSwitch",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteVSwitch(ctx, "1", "now")
		},
	},
	{
		name:       "AddVSwitchServers",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.AddVSwitchServers(ctx, "1", testVSwitch(1).Servers)
		},
	},
	{
		name:       "RemoveVSwitchServers",
		decodes:    false,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RemoveVSwitchServers(ctx, "1", testVSwitch(1).Servers)
		},
	},
	{
		name:       "WaitForVSwitchReady",
		decodes:    true,
		notFound:   nil,
		notFoundOK: false,
		run: func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.WaitForVSwitchReady(ctx, "1")
		},
	},
}

// errOnly drops a method's value, keeping its error.
func errOnly[T any](_ T, err error) error {
	return err
}

// Every method must fail, and keep the cause, when the request cannot be sent.
func TestTransportError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for _, call := range allCalls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			err := call.run(ctx, newTestClient(t, nil))
			if err == nil {
				t.Fatal("want error, got nil")
			}

			assertErrIs(t, err, context.Canceled)
		})
	}
}

// Every method must fail when an error body is unreadable, reporting the read
// failure, or only the status for calls that discard the body.
func TestBodyReadError(t *testing.T) {
	t.Parallel()

	for _, call := range allCalls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			err := call.run(
				context.Background(),
				newFailingBodyClient(http.StatusInternalServerError),
			)
			if err == nil {
				t.Fatal("want error, got nil")
			}

			if !errors.Is(err, errBodyRead) && !strings.Contains(err.Error(), "status 500") {
				t.Errorf("error: want %v or status 500, got %v", errBodyRead, err)
			}
		})
	}
}

// Methods parsing a response must fail on a 200 with a non-JSON body.
func TestInvalidJSON(t *testing.T) {
	t.Parallel()

	for _, call := range allCalls {
		if !call.decodes {
			continue
		}

		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"/": respond(http.StatusOK, "not json"),
			})

			err := call.run(context.Background(), api)
			assertErr(t, err, "invalid character")
		})
	}
}

// Every method must fail on a server error, reporting both status and body.
func TestHTTPErrorStatus(t *testing.T) {
	t.Parallel()

	for _, call := range allCalls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"/": respond(
					http.StatusInternalServerError,
					apiError(http.StatusInternalServerError, "INTERNAL_ERROR"),
				),
			})

			err := call.run(context.Background(), api)
			assertErr(t, err, "500")
			assertErr(t, err, "INTERNAL_ERROR")
		})
	}
}

// A 404 must map to the method's sentinel, to success for deletes, or else
// to a plain error.
func TestNotFound(t *testing.T) {
	t.Parallel()

	for _, call := range allCalls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"/": respond(http.StatusNotFound, apiError(http.StatusNotFound, "NOT_FOUND")),
			})

			err := call.run(context.Background(), api)

			switch {
			case call.notFoundOK:
				assertErr(t, err, "")
			case call.notFound != nil:
				assertErrIs(t, err, call.notFound)
			default:
				assertErr(t, err, "404")
			}
		})
	}
}

// Methods accepting more than one success status must accept each of them.
func TestAlternateSuccessStatus(t *testing.T) {
	t.Parallel()

	// Statuses besides 200 that each method treats as success.
	alternates := map[string]int{
		"GetFirewall":          http.StatusAccepted,
		"SetFirewall":          http.StatusAccepted,
		"CreateSSHKey":         http.StatusCreated,
		"CreateVSwitch":        http.StatusCreated,
		"UpdateVSwitch":        http.StatusCreated,
		"DeleteVSwitch":        http.StatusNoContent,
		"AddVSwitchServers":    http.StatusCreated,
		"RemoveVSwitchServers": http.StatusNoContent,
	}

	for _, call := range allCalls {
		alternate, ok := alternates[call.name]
		if !ok {
			continue
		}

		for _, status := range []int{http.StatusOK, alternate} {
			t.Run(fmt.Sprintf("%s/%d", call.name, status), func(t *testing.T) {
				t.Parallel()

				// Any JSON object decodes; the firewall one also ends SetFirewall's wait.
				api := newTestClient(t, map[string]http.HandlerFunc{
					"/": respond(status, testFirewallJSON),
				})

				err := call.run(context.Background(), api)
				assertErr(t, err, "")
			})
		}
	}
}

// IDs must be escaped as a single path segment, so they cannot reach
// another endpoint or inject a query.
func TestPathEscaping(t *testing.T) {
	t.Parallel()

	const (
		id   = "1/x?y"
		want = "1%2Fx%3Fy"
	)

	calls := map[string]func(ctx context.Context, api *client.HetznerRobotClient) error{
		"FetchFailover": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchFailover(ctx, id))
		},
		"SetFailover": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.SetFailover(ctx, id, "5.6.7.8")
		},
		"DeleteFailover": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteFailover(ctx, id)
		},
		"GetFirewall": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.GetFirewall(ctx, id))
		},
		"SetFirewall": func(ctx context.Context, api *client.HetznerRobotClient) error {
			firewall := testFirewall
			firewall.IP = id

			return api.SetFirewall(ctx, firewall)
		},
		"FetchServerByID": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchServerByID(ctx, id))
		},
		"RenameServer": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.RenameServer(ctx, id, "new"))
		},
		"EnableRescueMode": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.EnableRescueMode(ctx, id, "linux", nil))
		},
		"RebootServer": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RebootServer(ctx, id, "sw")
		},
		"FetchSSHKey": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchSSHKey(ctx, id))
		},
		"RenameSSHKey": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RenameSSHKey(ctx, id, "new")
		},
		"DeleteSSHKey": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteSSHKey(ctx, id)
		},
		"FetchVSwitchByID": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return errOnly(api.FetchVSwitchByID(ctx, id))
		},
		"UpdateVSwitch": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.UpdateVSwitch(ctx, id, "vsw", 4000)
		},
		"DeleteVSwitch": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.DeleteVSwitch(ctx, id, "now")
		},
		"AddVSwitchServers": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.AddVSwitchServers(ctx, id, testVSwitch(1).Servers)
		},
		"RemoveVSwitchServers": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.RemoveVSwitchServers(ctx, id, testVSwitch(1).Servers)
		},
		"WaitForVSwitchReady": func(ctx context.Context, api *client.HetznerRobotClient) error {
			return api.WaitForVSwitchReady(ctx, id)
		},
	}

	for name, run := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"/": func(writer http.ResponseWriter, req *http.Request) {
					path := req.URL.EscapedPath()
					if !strings.Contains(path, "/"+want) || req.URL.RawQuery != "" {
						t.Errorf("request: want segment %q, got %s", want, req.URL.RequestURI())
					}

					respond(http.StatusOK, testFirewallJSON)(writer, req)
				},
			})

			err := run(context.Background(), api)
			assertErr(t, err, "")
		})
	}
}
