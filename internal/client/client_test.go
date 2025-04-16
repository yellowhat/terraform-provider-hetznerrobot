package client_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	testUsername = "testuser"
	testPassword = "testpassword"
	testURL      = "https://robot.example.com"
)

func setupMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			auth := fmt.Sprintf("%s:%s", testUsername, testPassword)
			wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

			authHeader := request.Header.Get("Authorization")
			if authHeader != wantAuth {
				writer.WriteHeader(http.StatusUnauthorized)

				if _, err := fmt.Fprintln(writer, "Unauthorized"); err != nil {
					t.Errorf("error writing response: %v", err)
				}

				return
			}

			contentType := request.Header.Get("Content-Type")

			body, err := io.ReadAll(request.Body)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)

				if _, err := fmt.Fprintln(writer, "Error reading request body"); err != nil {
					t.Errorf("error writing response: %v", err)
				}

				return
			}
			defer request.Body.Close()

			writer.WriteHeader(http.StatusOK)

			_, err = fmt.Fprintf(
				writer,
				"Mock %s %s %s %s",
				request.Method,
				request.URL.Path,
				contentType,
				body,
			)
			if err != nil {
				t.Errorf("error writing response: %v", err)
			}
		}),
	)

	return server
}

func TestNew(t *testing.T) {
	t.Parallel()

	config := &client.ProviderConfig{
		Username: testUsername,
		Password: testPassword,
		BaseURL:  testURL,
	}

	client := client.New(config)

	if testUsername != client.Config.Username {
		t.Errorf("Incorrect username: want %s, got %s", testUsername, client.Config.Username)
	}

	if testPassword != client.Config.Password {
		t.Errorf("Incorrect password: want %s, got %s", testPassword, client.Config.Password)
	}

	if testURL != client.Config.BaseURL {
		t.Errorf("Incorrect baseurl: want %s, got %s", testURL, client.Config.BaseURL)
	}
}

func TestDoRequest(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		username    string
		password    string
		wantCode    int
	}

	testCases := []testCase{
		{
			name:        "GET success",
			method:      "GET",
			path:        "/",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusOK,
		},
		{
			name:        "POST success",
			method:      "POST",
			path:        "/post",
			body:        "test data",
			contentType: "application/json",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusOK,
		},
		{
			name:        "GET wrong username",
			method:      "GET",
			path:        "",
			body:        "",
			contentType: "",
			username:    "wrongUser",
			password:    testPassword,
			wantCode:    http.StatusUnauthorized,
		},
		{
			name:        "GET wrong password",
			method:      "GET",
			path:        "",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    "wrongPass",
			wantCode:    http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := setupMockServer(t)
			defer server.Close()

			config := &client.ProviderConfig{
				Username: tc.username,
				Password: tc.password,
				BaseURL:  server.URL,
			}

			client := client.New(config)

			ctx := context.Background()

			reqBody := bytes.NewBufferString(tc.body)

			resp, err := client.DoRequest(ctx, tc.method, tc.path, reqBody, tc.contentType)
			if err != nil {
				t.Errorf("DoRequest() errored: %v", err)
			}

			if resp.StatusCode != tc.wantCode {
				t.Errorf(
					"Incorrect status code: want %d, got %d",
					tc.wantCode,
					resp.StatusCode,
				)
			}

			if resp.StatusCode != http.StatusOK {
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("error parsing response: %v", err)
			}
			defer resp.Body.Close()

			wantBody := fmt.Sprintf("Mock %s %s %s %s", tc.method, tc.path, tc.contentType, tc.body)
			if wantBody != string(body) {
				t.Errorf("wrong body: want '%s', got '%s'", wantBody, string(body))
			}
		})
	}
}
