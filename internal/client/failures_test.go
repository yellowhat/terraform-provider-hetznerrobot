package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

// apiCall wraps one client method and how it reacts to error statuses.
type apiCall struct {
	name       string
	decodes    bool  // parses a JSON body on 200
	notFound   error // sentinel returned on 404, nil for a generic error
	notFoundOK bool  // 404 is treated as success
	run        func(ctx context.Context, api *client.HetznerRobotClient) error
}

// allCalls lists every exported endpoint method (DoRequest is tested separately).
func allCalls() []apiCall {
	return slices.Concat(
		failoverCalls(),
		firewallCalls(),
		serverCalls(),
		sshKeyCalls(),
		vswitchReadCalls(),
		vswitchWriteCalls(),
	)
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

	for _, call := range allCalls() {
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

	for _, call := range allCalls() {
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

	for _, call := range allCalls() {
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

	for _, call := range allCalls() {
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

	for _, call := range allCalls() {
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

	// Non-200 success statuses with realistic bodies; decoded values are
	// checked in the per-method tests.
	alternates := map[string]struct {
		status int
		body   string
	}{
		"GetFirewall":          {http.StatusAccepted, testFirewallJSON},
		"SetFirewall":          {http.StatusAccepted, testFirewallJSON},
		"CreateSSHKey":         {http.StatusCreated, testSSHKeyJSON},
		"CreateVSwitch":        {http.StatusCreated, vswitchJSON(1)},
		"UpdateVSwitch":        {http.StatusCreated, ""},
		"DeleteVSwitch":        {http.StatusNoContent, ""},
		"AddVSwitchServers":    {http.StatusCreated, ""},
		"RemoveVSwitchServers": {http.StatusNoContent, ""},
	}

	for _, call := range allCalls() {
		alternate, ok := alternates[call.name]
		if !ok {
			continue
		}

		for _, status := range []int{http.StatusOK, alternate.status} {
			t.Run(fmt.Sprintf("%s/%d", call.name, status), func(t *testing.T) {
				t.Parallel()

				// "/" also serves SetFirewall's follow-up GET, ending its wait.
				api := newTestClient(t, map[string]http.HandlerFunc{
					"/": respond(status, alternate.body),
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
			firewall := testFirewall()
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
