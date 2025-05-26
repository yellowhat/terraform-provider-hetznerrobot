// Package client provides a client for interacting with the Hetzner Robot API.
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	waitMaxRetries = 60
	waitDuration   = 20 * time.Second
)

// ProviderConfig provides a client for interacting with the Hetzner Robot API.
type ProviderConfig struct {
	Username string
	Password string
	BaseURL  string
}

// HetznerRobotClient represents the Hetzner Robot client.
type HetznerRobotClient struct {
	Config *ProviderConfig
	Client *http.Client
}

// New creates a new Hetzner Robot client.
func New(config *ProviderConfig) *HetznerRobotClient {
	return &HetznerRobotClient{
		Config: config,
		Client: &http.Client{},
	}
}

// DoRequest executes a request to the Hetzner Robot API.
func (c *HetznerRobotClient) DoRequest(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
	contentType string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		fmt.Sprintf("%s%s", c.Config.BaseURL, path),
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.SetBasicAuth(c.Config.Username, c.Config.Password)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return resp, nil
}
