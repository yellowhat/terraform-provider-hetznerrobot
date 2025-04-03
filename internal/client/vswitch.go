package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type VSwitch struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	VLAN      int               `json:"vlan"`
	Cancelled bool              `json:"cancelled"`
	Servers   []VSwitchServer   `json:"server"`
	Subnets   []VSwitchSubnet   `json:"subnets"`
	CloudNets []VSwitchCloudNet `json:"cloud_networks"`
}

type VSwitchServer struct {
	ServerNumber  int    `json:"server_number,omitempty"`
	ServerIP      string `json:"server_ip,omitempty"`
	ServerIPv6Net string `json:"server_ipv6_net,omitempty"`
	Status        string `json:"status,omitempty"`
}

type VSwitchSubnet struct {
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

type VSwitchCloudNet struct {
	ID      int    `json:"id"`
	IP      string `json:"ip"`
	Mask    int    `json:"mask"`
	Gateway string `json:"gateway"`
}

func (c *HetznerRobotClient) FetchVSwitchByID(
	ctx context.Context,
	id string,
) (VSwitch, error) {
	resp, err := c.DoRequest(ctx, "GET", fmt.Sprintf("/vswitch/%s", id), nil, "")
	if err != nil {
		return VSwitch{}, fmt.Errorf("error fetching VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return VSwitch{}, fmt.Errorf("unable to read response body: %w", err)
		}
		return VSwitch{}, fmt.Errorf(
			"error fetching VSwitch: status %d, body %s",
			resp.StatusCode,
			data,
		)
	}

	var vswitch VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitch); err != nil {
		return VSwitch{}, fmt.Errorf("error decoding VSwitch response: %w", err)
	}
	return vswitch, nil
}

func (c *HetznerRobotClient) FetchVSwitchesByIDs(ids []string) ([]VSwitch, error) {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		vswitches []VSwitch
		errs      []error
	)
	sem := make(chan struct{}, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, id := range ids {
		wg.Add(1)
		go func(vswitchID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			vswitch, err := c.FetchVSwitchByID(ctx, vswitchID)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			vswitches = append(vswitches, vswitch)
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	if len(errs) > 0 {
		firstErrors := errs
		if len(errs) > 5 {
			firstErrors = errs[:5]
		}
		return nil, fmt.Errorf(
			"errors occurred: %v (and %d more)",
			firstErrors,
			len(errs)-len(firstErrors),
		)
	}
	sort.Slice(vswitches, func(i, j int) bool {
		return vswitches[i].ID < vswitches[j].ID
	})
	return vswitches, nil
}

func (c *HetznerRobotClient) FetchAllVSwitches(ctx context.Context) ([]VSwitch, error) {
	resp, err := c.DoRequest(ctx, "GET", "/vswitch", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error fetching all vSwitches: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf(
			"error fetching vSwitches: status %d, body %s",
			resp.StatusCode,
			data,
		)
	}

	var vswitches []VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitches); err != nil {
		return nil, fmt.Errorf("error decoding vSwitches: %w", err)
	}

	return vswitches, nil
}

func (c *HetznerRobotClient) CreateVSwitch(
	ctx context.Context,
	name string,
	vlan int,
) (*VSwitch, error) {
	data := url.Values{}
	data.Set("name", name)
	data.Set("vlan", strconv.Itoa(vlan))
	resp, err := c.DoRequest(
		ctx,
		"POST",
		"/vswitch",
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return nil, fmt.Errorf("error creating VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to read response body: %w", err)
		}
		return nil, fmt.Errorf("error creating VSwitch: status %d, body %s", resp.StatusCode, data)
	}

	var vswitch VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitch); err != nil {
		return nil, fmt.Errorf("error decoding VSwitch response: %w", err)
	}

	return &vswitch, nil
}

func (c *HetznerRobotClient) UpdateVSwitch(
	ctx context.Context,
	id, name string,
	vlan int,
) error {
	data := url.Values{}
	data.Set("name", name)
	data.Set("vlan", strconv.Itoa(vlan))

	resp, err := c.DoRequest(
		ctx,
		"POST",
		fmt.Sprintf("/vswitch/%s", id),
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("error updating VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf("error updating VSwitch: status %d, body %s", resp.StatusCode, data)
	}

	return nil
}

func (c *HetznerRobotClient) DeleteVSwitch(
	ctx context.Context,
	id string,
	cancellationDate string,
) error {
	data := url.Values{}
	data.Set("cancellation_date", cancellationDate)
	resp, err := c.DoRequest(
		ctx,
		"DELETE",
		fmt.Sprintf("/vswitch/%s", id),
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("error deleting VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf("error deleting VSwitch: status %d, body %s", resp.StatusCode, data)
	}

	return nil
}

func (c *HetznerRobotClient) AddVSwitchServers(
	ctx context.Context,
	id string,
	servers []VSwitchServer,
) error {
	data := url.Values{}
	for _, server := range servers {
		data.Add("server[]", strconv.Itoa(server.ServerNumber))
	}
	resp, err := c.DoRequest(
		ctx,
		"POST",
		fmt.Sprintf("/vswitch/%s/server", id),
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("error adding servers to VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf(
			"error adding servers to VSwitch: status %d, body %s",
			resp.StatusCode,
			data,
		)
	}

	return nil
}

func (c *HetznerRobotClient) RemoveVSwitchServers(
	ctx context.Context,
	id string,
	servers []VSwitchServer,
) error {
	data := url.Values{}
	for _, server := range servers {
		data.Add("server[]", strconv.Itoa(server.ServerNumber))
	}
	resp, err := c.DoRequest(
		ctx,
		"DELETE",
		fmt.Sprintf("/vswitch/%s/server", id),
		strings.NewReader(data.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("error removing servers from VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}
		return fmt.Errorf(
			"error removing servers from VSwitch: status %d, body %s",
			resp.StatusCode,
			data,
		)
	}

	return nil
}

func isVSwitchReady(servers []VSwitchServer) bool {
	for _, server := range servers {
		if server.Status == "processing" {
			return false
		}
	}

	return true
}

func (c *HetznerRobotClient) WaitForVSwitchReady(
	ctx context.Context,
	id string,
	maxRetries int,
	waitTime time.Duration,
) error {
	for range maxRetries {
		vsw, err := c.FetchVSwitchByID(ctx, id)
		if err != nil {
			return fmt.Errorf("error fetching VSwitch while waiting: %w", err)
		}

		if isVSwitchReady(vsw.Servers) {
			return nil
		}

		time.Sleep(waitTime)
	}

	return fmt.Errorf("timeout waiting for vSwitch %s to become ready", id)
}
