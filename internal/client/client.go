package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
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

func runConcurrentTasks(
	ctx context.Context,
	ids []string,
	worker func(ctx context.Context, id string) error,
) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	sem := make(chan struct{}, 10)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			err := worker(ctx, id)
			<-sem
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
		}(id)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("error fetching: %v", errs)
	}

	return nil
}
