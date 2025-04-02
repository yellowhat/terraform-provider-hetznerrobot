package client

import (
	"fmt"
	"io"
	"net/http"
)

type ProviderConfig struct {
	Username string
	Password string
	BaseURL  string
}

type HetznerRobotClient struct {
	Config *ProviderConfig
	Client *http.Client
}

func New(config *ProviderConfig) *HetznerRobotClient {
	return &HetznerRobotClient{
		Config: config,
		Client: &http.Client{},
	}
}

func (c *HetznerRobotClient) DoRequest(
	method, path string,
	body io.Reader,
	contentType string,
) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.Config.BaseURL, path), body)
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
