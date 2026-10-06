package client_test

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const testFailoverJSON = `{
  "failover": {
    "ip": "123.123.123.123",
    "netmask": "255.255.255.255",
    "server_ip": "88.99.100.1",
    "server_ipv6_net": "2a01:4f8:1:1::",
    "server_number": 1,
    "active_server_ip": "88.99.100.2"
  }
}`

func TestFetchFailover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		want      client.Failover
		wantErrIs error
		wantErr   string
	}{
		{
			name:   "success",
			status: http.StatusOK,
			body:   testFailoverJSON,
			want: client.Failover{
				IP:             "123.123.123.123",
				Netmask:        "255.255.255.255",
				ServerIP:       "88.99.100.1",
				ServerNumber:   1,
				ActiveServerIP: "88.99.100.2",
			},
			wantErrIs: nil,
			wantErr:   "",
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   apiError(http.StatusNotFound, "NOT_FOUND"),
			//exhaustruct:ignore
			want:      client.Failover{},
			wantErrIs: client.ErrFailoverNotFound,
			wantErr:   "failover ip not found",
		},
		{
			name:   "server error",
			status: http.StatusInternalServerError,
			body:   apiError(http.StatusInternalServerError, "INTERNAL_ERROR"),
			//exhaustruct:ignore
			want:      client.Failover{},
			wantErrIs: nil,
			wantErr:   "status 500",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /failover/123.123.123.123": respond(test.status, test.body),
			})

			got, err := api.FetchFailover(context.Background(), "123.123.123.123")
			assertErr(t, err, test.wantErr)

			assertErrIs(t, err, test.wantErrIs)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("failover\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestSetFailover(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /failover/123.123.123.123": expectForm(
			t,
			url.Values{"active_server_ip": []string{"88.99.100.2"}},
			http.StatusOK,
			testFailoverJSON,
		),
		"POST /failover/1.1.1.1": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "FAILOVER_ALREADY_ROUTED"),
		),
	})

	err := api.SetFailover(context.Background(), "123.123.123.123", "88.99.100.2")
	assertErr(t, err, "")

	err = api.SetFailover(context.Background(), "1.1.1.1", "88.99.100.2")
	assertErr(t, err, "SetFailover 1.1.1.1: status 409")
}

func TestDeleteFailover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{name: "success", status: http.StatusOK, wantErr: ""},
		{name: "already gone", status: http.StatusNotFound, wantErr: ""},
		{name: "locked", status: http.StatusConflict, wantErr: "status 409"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"DELETE /failover/123.123.123.123": respond(test.status, ""),
			})

			err := api.DeleteFailover(context.Background(), "123.123.123.123")
			assertErr(t, err, test.wantErr)
		})
	}
}
