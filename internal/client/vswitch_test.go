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

// Robot vSwitch VLANs must be 4000-4091.
func vswitchJSON(id int) string {
	return fmt.Sprintf(`{
  "id": %[1]d,
  "name": "vswitch-%[1]d",
  "vlan": %[2]d,
  "cancelled": false,
  "server": [
    {"server_number": 1, "server_ip": "88.99.100.1", "server_ipv6_net": "2a01:4f8:1:1::", "status": "ready"}
  ],
  "subnets": [{"ip": "10.0.0.0", "mask": 24, "gateway": "10.0.0.1"}],
  "cloud_networks": [{"id": 9, "ip": "10.1.0.0", "mask": 24, "gateway": "10.1.0.1"}]
}`, id, 4000+id)
}

func testVSwitch(id int) client.VSwitch {
	return client.VSwitch{
		ID:        id,
		Name:      fmt.Sprintf("vswitch-%d", id),
		VLAN:      4000 + id,
		Cancelled: false,
		Servers: []client.VSwitchServer{{
			ServerNumber:  1,
			ServerIP:      "88.99.100.1",
			ServerIPv6Net: "2a01:4f8:1:1::",
			Status:        "ready",
		}},
		Subnets: []client.VSwitchSubnet{{IP: "10.0.0.0", Mask: 24, Gateway: "10.0.0.1"}},
		CloudNets: []client.VSwitchCloudNet{
			{ID: 9, IP: "10.1.0.0", Mask: 24, Gateway: "10.1.0.1"},
		},
	}
}

func TestFetchVSwitchByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		want    client.VSwitch
		wantErr string
	}{
		{
			name:    "success",
			status:  http.StatusOK,
			body:    vswitchJSON(1),
			want:    testVSwitch(1),
			wantErr: "",
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   apiError(http.StatusNotFound, "NOT_FOUND"),
			//exhaustruct:ignore
			want:    client.VSwitch{},
			wantErr: "error fetching VSwitch: status 404",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /vswitch/1": respond(test.status, test.body),
			})

			got, err := api.FetchVSwitchByID(context.Background(), "1")
			assertErr(t, err, test.wantErr)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("vswitch\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestFetchVSwitchesByIDs(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"GET /vswitch/1": respond(http.StatusOK, vswitchJSON(1)),
		"GET /vswitch/2": respond(http.StatusOK, vswitchJSON(2)),
	})

	got, err := api.FetchVSwitchesByIDs(context.Background(), []string{"2", "1"})
	assertErr(t, err, "")

	want := []client.VSwitch{testVSwitch(1), testVSwitch(2)}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("vswitches\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestFetchAllVSwitches(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"GET /vswitch": respond(
			http.StatusOK,
			"["+vswitchJSON(1)+","+vswitchJSON(2)+"]",
		),
	})

	got, err := api.FetchAllVSwitches(context.Background())
	assertErr(t, err, "")

	want := []client.VSwitch{testVSwitch(1), testVSwitch(2)}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("vswitches\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestCreateVSwitch(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /vswitch": expectForm(
			t,
			url.Values{"name": []string{"vswitch-1"}, "vlan": []string{"4001"}},
			http.StatusCreated,
			vswitchJSON(1),
		),
	})

	got, err := api.CreateVSwitch(context.Background(), "vswitch-1", 4001)
	assertErr(t, err, "")

	want := testVSwitch(1)
	if got == nil || !reflect.DeepEqual(want, *got) {
		t.Errorf("vswitch\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestCreateVSwitchLimitExceeded(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /vswitch": respond(
			http.StatusForbidden,
			apiError(http.StatusForbidden, "VSWITCH_LIMIT_REACHED"),
		),
	})

	_, err := api.CreateVSwitch(context.Background(), "vswitch-1", 4001)
	assertErr(t, err, "error creating VSwitch: status 403")
}

func TestUpdateVSwitch(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /vswitch/1": expectForm(
			t,
			url.Values{"name": []string{"renamed"}, "vlan": []string{"4010"}},
			http.StatusCreated,
			"",
		),
		"POST /vswitch/2": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "VSWITCH_IN_PROCESS"),
		),
	})

	err := api.UpdateVSwitch(context.Background(), "1", "renamed", 4010)
	assertErr(t, err, "")

	err = api.UpdateVSwitch(context.Background(), "2", "renamed", 4010)
	assertErr(t, err, "error updating VSwitch: status 409")
}

func TestDeleteVSwitch(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"DELETE /vswitch/1": expectForm(
			t,
			url.Values{"cancellation_date": []string{"now"}},
			http.StatusNoContent,
			"",
		),
		"DELETE /vswitch/2": respond(
			http.StatusNotFound,
			apiError(http.StatusNotFound, "NOT_FOUND"),
		),
	})

	err := api.DeleteVSwitch(context.Background(), "1", "now")
	assertErr(t, err, "")

	err = api.DeleteVSwitch(context.Background(), "2", "now")
	assertErr(t, err, "error deleting VSwitch: status 404")
}

func TestVSwitchServers(t *testing.T) {
	t.Parallel()

	//exhaustruct:ignore
	servers := []client.VSwitchServer{{ServerNumber: 1}, {ServerNumber: 2}}
	wantForm := url.Values{"server[]": []string{"1", "2"}}

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /vswitch/1/server":   expectForm(t, wantForm, http.StatusCreated, ""),
		"DELETE /vswitch/1/server": expectForm(t, wantForm, http.StatusOK, ""),
		"POST /vswitch/2/server": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "VSWITCH_SERVER_LIMIT_REACHED"),
		),
		"DELETE /vswitch/2/server": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "VSWITCH_IN_PROCESS"),
		),
	})

	err := api.AddVSwitchServers(context.Background(), "1", servers)
	assertErr(t, err, "")

	err = api.RemoveVSwitchServers(context.Background(), "1", servers)
	assertErr(t, err, "")

	err = api.AddVSwitchServers(context.Background(), "2", servers)
	assertErr(t, err, "error adding servers to VSwitch: status 409")

	err = api.RemoveVSwitchServers(context.Background(), "2", servers)
	assertErr(t, err, "error removing servers from VSwitch: status 409")
}

func TestWaitForVSwitchReady(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"GET /vswitch/1": respond(http.StatusOK, vswitchJSON(1)),
		"GET /vswitch/2": respond(
			http.StatusNotFound,
			apiError(http.StatusNotFound, "NOT_FOUND"),
		),
	})

	err := api.WaitForVSwitchReady(context.Background(), "1")
	assertErr(t, err, "")

	err = api.WaitForVSwitchReady(context.Background(), "2")
	assertErr(t, err, "error fetching VSwitch while waiting")
}
