package client_test

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
		wantCode int
		wantBody string
	}{
		{
			name:     "success",
			username: testUsername,
			password: testPassword,
			wantCode: http.StatusOK,
			wantBody: `[]`,
		},
		{
			name:     "wrong username",
			username: "wrongUser",
			password: testPassword,
			wantCode: http.StatusUnauthorized,
			wantBody: apiError(http.StatusUnauthorized, "UNAUTHORIZED"),
		},
		{
			name:     "wrong password",
			username: testUsername,
			password: "wrongPass",
			wantCode: http.StatusUnauthorized,
			wantBody: apiError(http.StatusUnauthorized, "UNAUTHORIZED"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			api := newTestClient(t, map[string]http.HandlerFunc{
				"GET /server": respond(http.StatusOK, `[]`),
			})
			api.Config.Username = test.username
			api.Config.Password = test.password

			response, err := api.DoRequest(
				context.Background(),
				"GET",
				"/server",
				nil,
				"",
			)
			if err != nil {
				t.Fatalf("DoRequest: %v", err)
			}
			defer response.Body.Close()

			if test.wantCode != response.StatusCode {
				t.Errorf("status code: want %d, got %d", test.wantCode, response.StatusCode)
			}

			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Errorf("reading body: %v", err)
			}

			if test.wantBody != string(body) {
				t.Errorf("body\nwant: %s\ngot: %s", test.wantBody, string(body))
			}
		})
	}
}
