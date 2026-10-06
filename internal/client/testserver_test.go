package client_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	testUsername = "foo"
	testPassword = "bar"
)

// newTestClient returns a client wired to a fake Robot API serving only the
// given routes. Route keys are http.ServeMux patterns, e.g. "GET /server/{id}".
// Requests without valid basic auth get a 401, unknown routes fail the test.
//
// Requests are served in-process, without network or goroutines, so tests can
// run inside a synctest bubble where waits advance fake time instantly.
func newTestClient(
	t *testing.T,
	routes map[string]http.HandlerFunc,
) *client.HetznerRobotClient {
	t.Helper()

	mux := http.NewServeMux()
	for pattern, handler := range routes {
		mux.HandleFunc(pattern, handler)
	}

	handler := http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		username, password, ok := req.BasicAuth()
		if !ok || username != testUsername || password != testPassword {
			respond(
				http.StatusUnauthorized,
				apiError(http.StatusUnauthorized, "UNAUTHORIZED"),
			)(writer, req)

			return
		}

		_, pattern := mux.Handler(req)
		if pattern == "" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(writer, req)

			return
		}

		mux.ServeHTTP(writer, req)
	})

	api := client.New(&client.ProviderConfig{
		Username: testUsername,
		Password: testPassword,
		BaseURL:  "http://robot.test",
	})
	api.Client.Transport = handlerTransport{handler: handler}

	return api
}

// handlerTransport serves requests by calling handler directly.
type handlerTransport struct {
	handler http.Handler
}

func (h handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := req.Context().Err()
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}

	req = req.Clone(req.Context())
	if req.Body == nil {
		req.Body = http.NoBody
	}

	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, req)

	return recorder.Result(), nil
}

// respond returns a handler writing a fixed status and JSON body.
func respond(status int, body string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = io.WriteString(writer, body)
	}
}

// sequence returns a handler serving each handler once, in order.
// Extra requests fail the test.
func sequence(t *testing.T, handlers ...http.HandlerFunc) http.HandlerFunc {
	t.Helper()

	return func(writer http.ResponseWriter, req *http.Request) {
		if len(handlers) == 0 {
			t.Errorf("unexpected extra request: %s %s", req.Method, req.URL.Path)
			http.NotFound(writer, req)

			return
		}

		handler := handlers[0]
		handlers = handlers[1:]

		handler(writer, req)
	}
}

// apiError returns a Robot API error body.
func apiError(status int, code string) string {
	return fmt.Sprintf(
		`{"error":{"status":%d,"code":%q,"message":"test error"}}`,
		status,
		code,
	)
}

// expectForm returns a handler asserting the url-encoded request body,
// then writing status and body. Works for DELETE too, unlike req.ParseForm.
func expectForm(t *testing.T, want url.Values, status int, body string) http.HandlerFunc {
	t.Helper()

	return func(writer http.ResponseWriter, req *http.Request) {
		contentType := req.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type: want form, got %q", contentType)
		}

		raw, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}

		got, err := url.ParseQuery(string(raw))
		if err != nil {
			t.Errorf("parsing body: %v", err)
		}

		if !reflect.DeepEqual(want, got) {
			t.Errorf("form\nwant: %v\ngot:  %v", want, got)
		}

		respond(status, body)(writer, req)
	}
}

// assertErrIs checks errors.Is(err, target), skipped when target is nil.
func assertErrIs(t *testing.T, err, target error) {
	t.Helper()

	if target != nil && !errors.Is(err, target) {
		t.Errorf("error: want %v, got %v", target, err)
	}
}

// assertErr checks err contains wantErr, or is nil when wantErr is empty.
func assertErr(t *testing.T, err error, wantErr string) {
	t.Helper()

	if wantErr == "" {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		return
	}

	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Errorf("error: want containing %q, got %v", wantErr, err)
	}
}
