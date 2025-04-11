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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := fmt.Sprintf("%s:%s", testUsername, testPassword)
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

		authHeader := r.Header.Get("Authorization")
		if authHeader != wantAuth {
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := fmt.Fprintln(w, "Unauthorized"); err != nil {
				t.Errorf("error writing response: %v", err)
			}
			return
		}

		contentType := r.Header.Get("Content-Type")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := fmt.Fprintln(w, "Error reading request body"); err != nil {
				t.Errorf("error writing response: %v", err)
			}
			return
		}
		defer r.Body.Close()

		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintf(w, "Mock %s %s %s %s", r.Method, r.URL.Path, contentType, body)
		if err != nil {
			t.Errorf("error writing response: %v", err)
		}
	}))
	return server
}

func TestNew(t *testing.T) {
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
			name:     "GET success",
			method:   "GET",
			path:     "/",
			username: testUsername,
			password: testPassword,
			wantCode: http.StatusOK,
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
			name:     "GET wrong username",
			method:   "GET",
			username: "wrongUser",
			password: testPassword,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "GET wrong password",
			method:   "GET",
			username: testUsername,
			password: "wrongPass",
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := setupMockServer(t)
			defer server.Close()

			config := &client.ProviderConfig{
				Username: tc.username,
				Password: tc.password,
				BaseURL:  server.URL,
			}

			client := client.New(config)

			ctx := context.Background()

			reqBody := bytes.NewBuffer([]byte(tc.body))
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
