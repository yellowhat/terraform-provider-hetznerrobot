package client_test

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// Wait loops sleep between polls. Inside a synctest bubble time is fake:
// sleeps return instantly and time.Since reports the simulated duration.
const (
	pollInterval = 20 * time.Second // mirrors waitDuration in client.go
	maxPolls     = 60               // mirrors waitMaxRetries in client.go
)

func firewallInProcessJSON() string {
	return strings.Replace(testFirewallJSON, `"active"`, `"in process"`, 1)
}

func vswitchProcessingJSON() string {
	return strings.Replace(vswitchJSON(1), `"ready"`, `"processing"`, 1)
}

func TestWaitForFirewallActive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON()),
			"GET /firewall/1.2.3.4": sequence(
				t,
				respond(http.StatusOK, firewallInProcessJSON()),
				respond(http.StatusOK, firewallInProcessJSON()),
				respond(http.StatusOK, testFirewallJSON),
			),
		})

		start := time.Now()
		err := api.SetFirewall(context.Background(), testFirewall())
		assertErr(t, err, "")

		if got := time.Since(start); got != 2*pollInterval {
			t.Errorf("elapsed: want %v, got %v", 2*pollInterval, got)
		}
	})
}

func TestWaitForFirewallActiveTimeout(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		polls := 0
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON()),
			"GET /firewall/1.2.3.4": func(writer http.ResponseWriter, req *http.Request) {
				polls++

				respond(http.StatusOK, firewallInProcessJSON())(writer, req)
			},
		})

		start := time.Now()
		err := api.SetFirewall(context.Background(), testFirewall())
		assertErr(t, err, "timeout waiting for firewall to become active on ip: 1.2.3.4")

		if polls != maxPolls {
			t.Errorf("polls: want %d, got %d", maxPolls, polls)
		}

		if got := time.Since(start); got != maxPolls*pollInterval {
			t.Errorf("elapsed: want %v, got %v", maxPolls*pollInterval, got)
		}
	})
}

func TestWaitForFirewallActiveCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON()),
			"GET /firewall/1.2.3.4":  respond(http.StatusOK, firewallInProcessJSON()),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		err := api.SetFirewall(ctx, testFirewall())
		assertErr(t, err, "wait interrupted: context deadline exceeded")

		// Returns at the deadline, not at the end of the second sleep (40s).
		if got := time.Since(start); got != 30*time.Second {
			t.Errorf("elapsed: want 30s, got %v", got)
		}
	})
}

func TestWaitForVSwitchReadyPolling(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"GET /vswitch/1": sequence(
				t,
				respond(http.StatusOK, vswitchProcessingJSON()),
				respond(http.StatusOK, vswitchJSON(1)),
			),
		})

		start := time.Now()
		err := api.WaitForVSwitchReady(context.Background(), "1")
		assertErr(t, err, "")

		if got := time.Since(start); got != pollInterval {
			t.Errorf("elapsed: want %v, got %v", pollInterval, got)
		}
	})
}

func TestWaitForVSwitchReadyTimeout(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		polls := 0
		api := newTestClient(t, map[string]http.HandlerFunc{
			"GET /vswitch/1": func(writer http.ResponseWriter, req *http.Request) {
				polls++

				respond(http.StatusOK, vswitchProcessingJSON())(writer, req)
			},
		})

		start := time.Now()
		err := api.WaitForVSwitchReady(context.Background(), "1")
		assertErr(t, err, "timeout waiting for vSwitch 1 to become ready")

		if polls != maxPolls {
			t.Errorf("polls: want %d, got %d", maxPolls, polls)
		}

		if got := time.Since(start); got != maxPolls*pollInterval {
			t.Errorf("elapsed: want %v, got %v", maxPolls*pollInterval, got)
		}
	})
}

func TestRebootServerPower(t *testing.T) {
	t.Parallel()

	for _, resetType := range []string{"power", "power_long"} {
		t.Run(resetType, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				var calls []time.Duration

				start := time.Now()
				record := func(handler http.HandlerFunc) http.HandlerFunc {
					return func(writer http.ResponseWriter, req *http.Request) {
						calls = append(calls, time.Since(start))

						handler(writer, req)
					}
				}

				api := newTestClient(t, map[string]http.HandlerFunc{
					"POST /reset/1": sequence(
						t,
						record(expectForm(
							t,
							url.Values{"type": []string{resetType}},
							http.StatusOK,
							"",
						)),
						record(expectForm(
							t,
							url.Values{"action": []string{"on"}},
							http.StatusOK,
							"",
						)),
					),
				})

				err := api.RebootServer(context.Background(), "1", resetType)
				assertErr(t, err, "")

				// RebootServer waits 30s between power-off and power-on.
				want := []time.Duration{0, 30 * time.Second}
				if !slices.Equal(want, calls) {
					t.Errorf("call times: want %v, got %v", want, calls)
				}
			})
		})
	}
}

func TestRebootServerPowerOnFails(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /reset/1": sequence(
				t,
				respond(http.StatusOK, ""),
				respond(http.StatusConflict, apiError(http.StatusConflict, "RESET_MANUAL_ACTIVE")),
			),
		})

		err := api.RebootServer(context.Background(), "1", "power")
		assertErr(t, err, "unable to power on: unexpected status code 409")
	})
}

func TestWaitForFirewallActiveFetchFails(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON()),
		"GET /firewall/1.2.3.4": respond(
			http.StatusInternalServerError,
			apiError(http.StatusInternalServerError, "INTERNAL_ERROR"),
		),
	})

	err := api.SetFirewall(context.Background(), testFirewall())
	assertErr(t, err, "error checking firewall status")
}

func TestWaitForVSwitchReadyCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"GET /vswitch/1": respond(http.StatusOK, vswitchProcessingJSON()),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := api.WaitForVSwitchReady(ctx, "1")
		assertErr(t, err, "wait interrupted")
		assertErrIs(t, err, context.DeadlineExceeded)
	})
}

func TestRebootServerPowerCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		// Only the power-off request is expected: power-on is never sent.
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /reset/1": sequence(t, respond(http.StatusOK, "")),
		})

		// Expires during the 30s power-off wait.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := api.RebootServer(ctx, "1", "power")
		assertErr(t, err, "wait interrupted")
		assertErrIs(t, err, context.DeadlineExceeded)
	})
}

func TestRebootServerPowerOnRequestFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		powerOn func(req *http.Request) (*http.Response, error)
		wantIs  error
	}{
		{
			name: "transport error",
			powerOn: func(*http.Request) (*http.Response, error) {
				return nil, errTransport
			},
			wantIs: errTransport,
		},
		{
			name:    "unreadable body",
			powerOn: failingBodyTransport{status: http.StatusInternalServerError}.RoundTrip,
			wantIs:  errBodyRead,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				api := newTestClient(t, map[string]http.HandlerFunc{
					"POST /reset/1": respond(http.StatusOK, ""),
				})
				powerOff := api.Client.Transport.RoundTrip
				responses := []roundTripFunc{powerOff, test.powerOn}
				api.Client.Transport = roundTripFunc(
					func(req *http.Request) (*http.Response, error) {
						next := responses[0]
						responses = responses[1:]

						return next(req)
					},
				)

				err := api.RebootServer(context.Background(), "1", "power")
				assertErr(t, err, "unable to power on")
				assertErrIs(t, err, test.wantIs)
			})
		})
	}
}
