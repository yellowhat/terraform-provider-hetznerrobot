package client_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/client"
)

const (
	testUsername = "foo"
	testPassword = "bar"
)

func authValidator() openapi3filter.AuthenticationFunc {
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		request := input.RequestValidationInput.Request

		auth := request.Header.Get("Authorization")
		if auth == "" {
			return errors.New("missing authorization header")
		}

		if !strings.HasPrefix(auth, "Basic ") {
			return errors.New("invalid authorization type")
		}

		decoded, err := base64.StdEncoding.DecodeString(auth[6:]) // Remove "Basic " prefix
		if err != nil {
			return errors.New("invalid base64 encoding")
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid credentials format")
		}

		username := parts[0]
		password := parts[1]

		if username != testUsername || password != testPassword {
			return errors.New("invalid credentials")
		}

		return nil
	}
}

func mockServer() *httptest.Server {
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile("mock.yaml")
	if err != nil {
		log.Fatalf("doc load file: %s", err)
	}

	err = doc.Validate(loader.Context)
	if err != nil {
		log.Fatalf("doc validation: %s", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		log.Fatalf("error creating router: %s", err)
	}

	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			route, pathParams, err := router.FindRoute(req)
			if err != nil {
				http.Error(writer, "Error validating request", http.StatusBadRequest)

				return
			}

			//exhaustruct:ignore
			requestValidationInput := &openapi3filter.RequestValidationInput{
				Request:    req,
				PathParams: pathParams,
				Route:      route,
				Options: &openapi3filter.Options{
					AuthenticationFunc: authValidator(),
				},
			}

			err = openapi3filter.ValidateRequest(req.Context(), requestValidationInput)
			if err != nil {
				http.Error(writer, "Error validating request", http.StatusBadRequest)

				return
			}

			responses := route.PathItem.GetOperation(route.Method).Responses

			// Check if 200 response exists
			responseSpec := responses.Map()["200"]
			if responseSpec == nil || responseSpec.Value == nil {
				http.Error(writer, "No 200 response defined", http.StatusInternalServerError)

				return
			}

			// Check if JSON content exists
			content := responseSpec.Value.Content["application/json"]
			if content == nil {
				http.Error(writer, "No JSON content defined", http.StatusInternalServerError)

				return
			}

			// Check if example is available
			if content.Example == nil {
				http.Error(writer, "No Example defined", http.StatusInternalServerError)

				return
			}

			responseJSON, err := json.Marshal(content.Example)
			if err != nil {
				http.Error(writer, "Error marshalling json", http.StatusInternalServerError)

				return
			}

			_, err = writer.Write(responseJSON)
			if err != nil {
				http.Error(writer, "Error writing response", http.StatusInternalServerError)
			}
		}),
	)

	return server
}

func TestAuth(t *testing.T) {
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
		wantBody    string
	}

	testCases := []testCase{
		{
			name:        "GET success",
			method:      "GET",
			path:        "/server",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusOK,
			wantBody: `[
				{
					"server_name": "server-1",
					"server_number": 1
				},
				{
					"server_name": "server-2",
					"server_number": 2
				}
			]`,
		},
		{
			name:        "GET wrong username",
			method:      "GET",
			path:        "/server",
			body:        "",
			contentType: "",
			username:    "wrongUser",
			password:    testPassword,
			wantCode:    http.StatusBadRequest,
			wantBody:    "",
		},
		{
			name:        "GET wrong password",
			method:      "GET",
			path:        "/server",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    "wrongPass",
			wantCode:    http.StatusBadRequest,
			wantBody:    "",
		},
		{
			name:        "GET no path",
			method:      "GET",
			path:        "/",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusBadRequest,
			wantBody:    "",
		},
		{
			name:        "POST no method",
			method:      "POST",
			path:        "/server",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusBadRequest,
			wantBody:    "",
		},
		{
			name:        "POST no path",
			method:      "POST",
			path:        "/post",
			body:        "",
			contentType: "",
			username:    testUsername,
			password:    testPassword,
			wantCode:    http.StatusBadRequest,
			wantBody:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := mockServer()
			defer server.Close()

			client := client.New(&client.ProviderConfig{
				Username: tc.username,
				Password: tc.password,
				BaseURL:  server.URL,
			})

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

			if tc.wantBody != string(body) {
				t.Errorf("wrong body: want '%s', got '%s'", tc.wantBody, string(body))
			}
		})
	}
}
