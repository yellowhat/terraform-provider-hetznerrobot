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

func (c *HetznerRobotClient) FetchAllServers(ctx context.Context) ([]Server, error) {
	path := "/server"
	resp, err := c.DoRequest(ctx, "GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("FetchAllServers request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf("FetchAllServers: status %d, body %s", resp.StatusCode, data)
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

func (c *HetznerRobotClient) FetchServerByID(ctx context.Context, id int) (Server, error) {
	path := fmt.Sprintf("/server/%d", id)
	resp, err := c.DoRequest(ctx, "GET", path, nil, "")
	if err != nil {
		return Server{}, fmt.Errorf("FetchServerByID request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return Server{}, fmt.Errorf("unable to read response body: %w", err)
		}
		return Server{}, fmt.Errorf(
			"FetchServerByID %d: status %d, body %s",
			id,
			resp.StatusCode,
			data,
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

func (c *HetznerRobotClient) FetchServersByIDs(ctx context.Context, ids []int) ([]Server, error) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		servers []Server
		errs    []error
	)
	sem := make(chan struct{}, 10)
	for _, id := range ids {
		wg.Add(1)
		go func(serverID int) {
			defer wg.Done()
			sem <- struct{}{}
			srv, err := c.FetchServerByID(ctx, serverID)
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
		ctx,
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, fmt.Errorf("error renaming server %d to %s: %w", serverID, newName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, data)
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
		ctx,
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, fmt.Errorf("error enabling rescue mode for server %d: %w", serverID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, data)
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
		ctx,
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
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf("unexpected status code %d, body: %s", resp.StatusCode, data)
	}

	if resetType == "power" || resetType == "power_long" {
		// Allow some time to power off
		time.Sleep(30 * time.Second)
		if err := c.powerOnServer(ctx, serverID, endpoint); err != nil {
			return fmt.Errorf("unable to power on: %w", err)
		}
	}

	return nil
}

func (c *HetznerRobotClient) powerOnServer(
	ctx context.Context,
	serverID int,
	endpoint string,
) error {
	data := url.Values{}
	data.Set("action", "on")

	resp, err := c.DoRequest(
		ctx,
		"POST",
		endpoint,
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("error turning on server %d after power off: %w", serverID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf(
			"unexpected status code %d when turning on server, body: %s",
			resp.StatusCode,
			data,
		)
	}

	return nil
}
