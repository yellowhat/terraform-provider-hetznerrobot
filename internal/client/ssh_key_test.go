package client_test

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	testFingerprint = "56:29:99:a4:5d:ed:ac:95:c1:f5:88:82:90:5d:dd:10"
	testSSHKeyJSON  = `{
  "key": {
    "name": "key1",
    "fingerprint": "56:29:99:a4:5d:ed:ac:95:c1:f5:88:82:90:5d:dd:10",
    "type": "ED25519",
    "size": 256,
    "data": "ssh-ed25519 AAAA",
    "created_at": "2026-01-01 00:00:00"
  }
}`
)

//nolint:gochecknoglobals
var testSSHKey = client.SSHKey{
	Name:        "key1",
	Fingerprint: testFingerprint,
	Type:        "ED25519",
	Size:        256,
	Data:        "ssh-ed25519 AAAA",
	CreatedAt:   "2026-01-01 00:00:00",
}

func TestFetchSSHKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		want      client.SSHKey
		wantErrIs error
		wantErr   string
	}{
		{
			name:      "success",
			status:    http.StatusOK,
			body:      testSSHKeyJSON,
			want:      testSSHKey,
			wantErrIs: nil,
			wantErr:   "",
		},
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   apiError(http.StatusNotFound, "NOT_FOUND"),
			//exhaustruct:ignore
			want:      client.SSHKey{},
			wantErrIs: client.ErrSSHKeyNotFound,
			wantErr:   "ssh key not found",
		},
		{
			name:   "server error",
			status: http.StatusInternalServerError,
			body:   apiError(http.StatusInternalServerError, "INTERNAL_ERROR"),
			//exhaustruct:ignore
			want:      client.SSHKey{},
			wantErrIs: nil,
			wantErr:   "status 500",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /key/" + testFingerprint: respond(test.status, test.body),
			})

			got, err := api.FetchSSHKey(context.Background(), testFingerprint)
			assertErr(t, err, test.wantErr)

			assertErrIs(t, err, test.wantErrIs)

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("key\nwant: %+v\ngot:  %+v", test.want, got)
			}
		})
	}
}

func TestCreateSSHKey(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /key": expectForm(
			t,
			url.Values{"name": []string{"key1"}, "data": []string{"ssh-ed25519 AAAA"}},
			http.StatusCreated,
			testSSHKeyJSON,
		),
	})

	got, err := api.CreateSSHKey(context.Background(), "key1", "ssh-ed25519 AAAA")
	assertErr(t, err, "")

	if !reflect.DeepEqual(testSSHKey, got) {
		t.Errorf("key\nwant: %+v\ngot:  %+v", testSSHKey, got)
	}
}

func TestCreateSSHKeyConflict(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /key": respond(
			http.StatusConflict,
			apiError(http.StatusConflict, "KEY_ALREADY_EXISTS"),
		),
	})

	_, err := api.CreateSSHKey(context.Background(), "key1", "ssh-ed25519 AAAA")
	assertErr(t, err, "CreateSSHKey: status 409")
}

func TestRenameSSHKey(t *testing.T) {
	t.Parallel()

	api := newTestClient(t, map[string]http.HandlerFunc{
		"POST /key/" + testFingerprint: expectForm(
			t,
			url.Values{"name": []string{"renamed"}},
			http.StatusOK,
			testSSHKeyJSON,
		),
		"POST /key/missing": respond(
			http.StatusNotFound,
			apiError(http.StatusNotFound, "NOT_FOUND"),
		),
	})

	err := api.RenameSSHKey(context.Background(), testFingerprint, "renamed")
	assertErr(t, err, "")

	err = api.RenameSSHKey(context.Background(), "missing", "renamed")
	assertErr(t, err, "RenameSSHKey missing: status 404")
}

func TestDeleteSSHKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{name: "success", status: http.StatusOK, wantErr: ""},
		{name: "already gone", status: http.StatusNotFound, wantErr: ""},
		{name: "server error", status: http.StatusInternalServerError, wantErr: "status 500"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"DELETE /key/" + testFingerprint: respond(test.status, ""),
			})

			err := api.DeleteSSHKey(context.Background(), testFingerprint)
			assertErr(t, err, test.wantErr)
		})
	}
}
