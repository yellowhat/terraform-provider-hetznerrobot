package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
)

func (c *HetznerRobotClient) FetchAllServers() ([]Server, error) {
	path := "/server"
	resp, err := c.DoRequest("GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("FetchAllServers request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FetchAllServers: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	var raw []struct {
		Server Server `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("FetchAllServers decode error: %w", err)
	}
	servers := make([]Server, len(raw))
	for i, item := range raw {
		servers[i] = item.Server
	}
	return servers, nil
}

func (c *HetznerRobotClient) FetchServerByID(id int) (Server, error) {
	path := fmt.Sprintf("/server/%d", id)
	resp, err := c.DoRequest("GET", path, nil, "")
	if err != nil {
		return Server{}, fmt.Errorf("FetchServerByID request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Server{}, fmt.Errorf("FetchServerByID %d: status %d, body %s", id, resp.StatusCode, string(bodyBytes))
	}
	var result struct {
		Server Server `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Server{}, fmt.Errorf("FetchServerByID decode error: %w", err)
	}
	return result.Server, nil
}

func (c *HetznerRobotClient) FetchServersByIDs(ids []int) ([]Server, error) {
	var (
		servers []Server
		mu      sync.Mutex
		wg      sync.WaitGroup
		errs    []error
	)
	sem := make(chan struct{}, 10)
	for _, id := range ids {
		wg.Add(1)
		go func(serverID int) {
			defer wg.Done()
			sem <- struct{}{}
			srv, err := c.FetchServerByID(serverID)
			<-sem
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			servers = append(servers, srv)
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	if len(errs) > 0 {
		return nil, fmt.Errorf("FetchServersByIDs errors: %v", errs)
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Number < servers[j].Number
	})
	return servers, nil
}
