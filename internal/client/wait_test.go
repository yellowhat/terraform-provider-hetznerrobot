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

// Mirrors waitDuration and waitMaxRetries (60) in client.go.
const pollInterval = 20 * time.Second

//nolint:gochecknoglobals
var (
	firewallInProcessJSON = strings.Replace(testFirewallJSON, `"active"`, `"in process"`, 1)
	vswitchProcessingJSON = strings.Replace(vswitchJSON(1), `"ready"`, `"processing"`, 1)
)

func TestWaitForFirewallActive(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON),
			"GET /firewall/1.2.3.4": sequence(
				t,
				respond(http.StatusOK, firewallInProcessJSON),
				respond(http.StatusOK, firewallInProcessJSON),
				respond(http.StatusOK, testFirewallJSON),
			),
		})

		start := time.Now()
		err := api.SetFirewall(context.Background(), testFirewall)
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
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON),
			"GET /firewall/1.2.3.4": func(writer http.ResponseWriter, req *http.Request) {
				polls++

				respond(http.StatusOK, firewallInProcessJSON)(writer, req)
			},
		})

		start := time.Now()
		err := api.SetFirewall(context.Background(), testFirewall)
		assertErr(t, err, "timeout waiting for firewall to become active on ip: 1.2.3.4")

		if polls != 60 {
			t.Errorf("polls: want 60, got %d", polls)
		}

		if got := time.Since(start); got != 60*pollInterval {
			t.Errorf("elapsed: want %v, got %v", 60*pollInterval, got)
		}
	})
}

func TestWaitForFirewallActiveCancelled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		api := newTestClient(t, map[string]http.HandlerFunc{
			"POST /firewall/1.2.3.4": respond(http.StatusOK, firewallInProcessJSON),
			"GET /firewall/1.2.3.4":  respond(http.StatusOK, firewallInProcessJSON),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		err := api.SetFirewall(ctx, testFirewall)
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
				respond(http.StatusOK, vswitchProcessingJSON),
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
		api := newTestClient(t, map[string]http.HandlerFunc{
			"GET /vswitch/1": respond(http.StatusOK, vswitchProcessingJSON),
		})

		err := api.WaitForVSwitchReady(context.Background(), "1")
		assertErr(t, err, "timeout waiting for vSwitch 1 to become ready")
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
