package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Server struct {
	IP         string `json:"server_ip"`
	IPv6Net    string `json:"server_ipv6_net"`
	Number     int    `json:"server_number"`
	ServerName string `json:"server_name"`
	Product    string `json:"product"`
	Datacenter string `json:"dc"`
	Traffic    string `json:"traffic"`
	Status     string `json:"status"`
	Cancelled  bool   `json:"cancelled"`
	PaidUntil  string `json:"paid_until"`
}

type HetznerRescueResponse struct {
	Rescue struct {
		ServerIP string `json:"server_ip"`
		Password string `json:"password"`
	} `json:"rescue"`
}

type HetznerRenameResponse struct {
	Server struct {
		ServerName string `json:"server_name"`
	} `json:"server"`
}

func (c *HetznerRobotClient) FetchAllServers() ([]Server, error) {
	path := "/server"
	resp, err := c.DoRequest("GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("FetchAllServers request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"FetchAllServers: status %d, body %s",
			resp.StatusCode,
			string(bodyBytes),
		)
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
		return Server{}, fmt.Errorf(
			"FetchServerByID %d: status %d, body %s",
			id,
			resp.StatusCode,
			string(bodyBytes),
		)
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

func (c *HetznerRobotClient) RenameServer(
	ctx context.Context,
	serverID int,
	newName string,
) (*HetznerRenameResponse, error) {
	endpoint := fmt.Sprintf("/server/%d", serverID)
	data := url.Values{}
	data.Set("server_name", newName)
	resp, err := c.DoRequest(
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, fmt.Errorf("error renaming server %d to %s: %w", serverID, newName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, string(body))
	}
	var renameResp HetznerRenameResponse
	if err := json.NewDecoder(resp.Body).Decode(&renameResp); err != nil {
		return nil, fmt.Errorf("error parsing rename response: %w", err)
	}
	return &renameResp, nil
}

func (c *HetznerRobotClient) EnableRescueMode(
	ctx context.Context,
	serverID int,
	os string,
	sshKeys []string,
) (*HetznerRescueResponse, error) {
	endpoint := fmt.Sprintf("/boot/%d/rescue", serverID)
	data := url.Values{}
	data.Set("os", os)
	for _, key := range sshKeys {
		data.Add("authorized_key[]", key)
	}
	resp, err := c.DoRequest(
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, fmt.Errorf("error enabling rescue mode for server %d: %w", serverID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, string(body))
	}
	var rescueResp HetznerRescueResponse
	if err := json.NewDecoder(resp.Body).Decode(&rescueResp); err != nil {
		return nil, fmt.Errorf("error parsing rescue response: %w", err)
	}
	return &rescueResp, nil
}

func (c *HetznerRobotClient) RebootServer(
	ctx context.Context,
	serverID int,
	resetType string,
) error {
	endpoint := fmt.Sprintf("/reset/%d", serverID)
	data := url.Values{}
	data.Set("type", resetType)

	resp, err := c.DoRequest(
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf(
			"error rebooting server %d with reset type %s: %w",
			serverID,
			resetType,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[DEBUG] Server %d reset with type: %s\n", serverID, resetType)

	if resetType == "power" || resetType == "power_long" {
		time.Sleep(30 * time.Second)
		fmt.Printf("Turning on server %d after %s reset\n", serverID, resetType)
		powerData := url.Values{}
		powerData.Set("action", "on")

		powerResp, err := c.DoRequest(
			"POST",
			endpoint,
			strings.NewReader(data.Encode()),
			"application/x-www-form-urlencoded",
		)
		if err != nil {
			return fmt.Errorf(
				"error turning on server %d after %s reset: %w",
				serverID,
				resetType,
				err,
			)
		}
		defer powerResp.Body.Close()

		if powerResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(powerResp.Body)
			return fmt.Errorf(
				"unexpected status code %d when turning on server, body: %s",
				powerResp.StatusCode,
				string(body),
			)
		}

		fmt.Printf("[DEBUG] Server %d successfully powered on\n", serverID)
	}

	return nil
}
