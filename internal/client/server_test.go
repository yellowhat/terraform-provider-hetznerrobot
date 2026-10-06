package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

func serverJSON(number int) string {
	return fmt.Sprintf(`{
  "server": {
    "server_ip": "88.99.100.%[1]d",
    "server_ipv6_net": "2a01:4f8:1:%[1]d::",
    "server_number": %[1]d,
    "server_name": "server-%[1]d",
    "product": "AX41",
    "dc": "FSN1-DC14",
    "traffic": "unlimited",
    "status": "ready",
    "cancelled": false,
    "paid_until": "2026-12-31"
  }
}`, number)
}

func testServer(number int) client.Server {
	return client.Server{
		IP:         fmt.Sprintf("88.99.100.%d", number),
		IPv6Net:    fmt.Sprintf("2a01:4f8:1:%d::", number),
		Number:     number,
		ServerName: fmt.Sprintf("server-%d", number),
		Product:    "AX41",
		Datacenter: "FSN1-DC14",
		Traffic:    "unlimited",
		Status:     "ready",
		Cancelled:  false,
		PaidUntil:  "2026-12-31",
	}
}

func TestFetchServerByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		want    client.Server
		wantErr string
	}{
		{
			name:    "success",
			status:  http.StatusOK,
			body:    serverJSON(1),
			want:    testServer(1),
			wantErr: "",
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   apiError(http.StatusNotFound, "SERVER_NOT_FOUND"),
			//exhaustruct:ignore
			want:    client.Server{},
			wantErr: "FetchServerByID 1: status 404",
		},
		{
			name:   "invalid json",
			status: http.StatusOK,
			body:   `{`,
			//exhaustruct:ignore
			want:    client.Server{},
			wantErr: "FetchServerByID decode error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /server/1": respond(test.status, test.body),
			})

			got, err := api.FetchServerByID(context.Background(), "1")
			assertErr(t, err, test.wantErr)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("server\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestFetchServersByIDs(t *testing.T) {
	t.Parallel()

	routes := map[string]http.HandlerFunc{
		"GET /server/1": respond(http.StatusOK, serverJSON(1)),
		"GET /server/2": respond(http.StatusOK, serverJSON(2)),
		"GET /server/3": respond(
			http.StatusNotFound,
			apiError(http.StatusNotFound, "SERVER_NOT_FOUND"),
		),
	}

	t.Run("sorted by number", func(t *testing.T) {
		t.Parallel()

		got, err := newTestClient(t, routes).
			FetchServersByIDs(context.Background(), []string{"2", "1"})
		assertErr(t, err, "")

		want := []client.Server{testServer(1), testServer(2)}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("servers\nwant: %+v\ngot:  %+v", want, got)
		}
	})

	t.Run("one fails", func(t *testing.T) {
		t.Parallel()

		got, err := newTestClient(t, routes).
			FetchServersByIDs(context.Background(), []string{"1", "3"})
		assertErr(t, err, "FetchServerByID 3: status 404")

		if got != nil {
			t.Errorf("servers: want nil, got %+v", got)
		}
	})
}

func TestFetchAllServers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		want    []client.Server
		wantErr string
	}{
		{
			name:    "success",
			status:  http.StatusOK,
			body:    "[" + serverJSON(1) + "," + serverJSON(2) + "]",
			want:    []client.Server{testServer(1), testServer(2)},
			wantErr: "",
		},
		{
			name:    "empty",
			status:  http.StatusOK,
			body:    `[]`,
			want:    []client.Server{},
			wantErr: "",
		},
		{
			name:    "no servers",
			status:  http.StatusNotFound,
			body:    apiError(http.StatusNotFound, "SERVER_NOT_FOUND"),
			want:    nil,
			wantErr: "FetchAllServers: status 404",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /server": respond(test.status, test.body),
			})

			got, err := api.FetchAllServers(context.Background())
			assertErr(t, err, test.wantErr)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("servers\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestRenameServer(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /server/1": expectForm(
			t,
			url.Values{"server_name": []string{"renamed"}},
			http.StatusOK,
			`{"server":{"server_name":"renamed"}}`,
		),
		"POST /server/2": respond(
			http.StatusBadRequest,
			apiError(http.StatusBadRequest, "INVALID_INPUT"),
		),
	})

	got, err := api.RenameServer(context.Background(), "1", "renamed")
	assertErr(t, err, "")

	if got == nil || got.Server.ServerName != "renamed" {
		t.Errorf("server_name: want renamed, got %+v", got)
	}

	_, err = api.RenameServer(context.Background(), "2", "renamed")
	assertErr(t, err, "unexpected status code 400")
}

func TestEnableRescueMode(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /boot/1/rescue": expectForm(
			t,
			url.Values{
				"os":               []string{"linux"},
				"authorized_key[]": []string{"aa:bb", "cc:dd"},
			},
			http.StatusOK,
			`{"rescue":{"server_ip":"88.99.100.1","password":"secret"}}`,
		),
		"POST /boot/2/rescue": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "BOOT_ALREADY_ENABLED"),
		),
	})

	got, err := api.EnableRescueMode(
		context.Background(),
		"1",
		"linux",
		[]string{"aa:bb", "cc:dd"},
	)
	assertErr(t, err, "")

	if got == nil || got.Rescue.ServerIP != "88.99.100.1" || got.Rescue.Password != "secret" {
		t.Errorf("rescue: got %+v", got)
	}

	_, err = api.EnableRescueMode(context.Background(), "2", "linux", nil)
	assertErr(t, err, "unexpected status code 409")
}

func TestRebootServer(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /reset/1": expectForm(
			t,
			url.Values{"type": []string{"hw"}},
			http.StatusOK,
			`{"reset":{"server_ip":"88.99.100.1","type":"hw"}}`,
		),
		"POST /reset/2": expectForm(
			t,
			url.Values{"type": []string{"sw"}},
			http.StatusConflict,
			apiError(http.StatusConflict, "RESET_MANUAL_ACTIVE"),
		),
	})

	err := api.RebootServer(context.Background(), "1", "hw")
	assertErr(t, err, "")

	err = api.RebootServer(context.Background(), "2", "sw")
	assertErr(t, err, "unexpected status code 409")
}
