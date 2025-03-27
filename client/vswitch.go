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

func (c *HetznerRobotClient) FetchVSwitchByIDWithContext(ctx context.Context, id string) (*VSwitch, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/vswitch/%s", id), nil, "")
	if err != nil {
		return nil, fmt.Errorf("error fetching VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching VSwitch: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}

	var vswitch VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitch); err != nil {
		return nil, fmt.Errorf("error decoding VSwitch response: %w", err)
	}
	return &vswitch, nil
}

func (c *HetznerRobotClient) FetchVSwitchesByIDs(ids []string) ([]VSwitch, error) {
	var (
		vswitches []VSwitch
		mu        sync.Mutex
		wg        sync.WaitGroup
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
			vswitch, err := c.FetchVSwitchByIDWithContext(ctx, vswitchID)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			vswitches = append(vswitches, *vswitch)
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	if len(errs) > 0 {
		firstErrors := errs
		if len(errs) > 5 {
			firstErrors = errs[:5]
		}
		return nil, fmt.Errorf("errors occurred: %v (and %d more)", firstErrors, len(errs)-len(firstErrors))
	}
	sort.Slice(vswitches, func(i, j int) bool {
		return vswitches[i].ID < vswitches[j].ID
	})
	return vswitches, nil
}

func (c *HetznerRobotClient) FetchAllVSwitches(ctx context.Context) ([]VSwitch, error) {
	resp, err := c.DoRequest("GET", "/vswitch", nil, "")
	if err != nil {
		return nil, fmt.Errorf("error fetching all vSwitches: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching vSwitches: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	var vswitches []VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitches); err != nil {
		return nil, fmt.Errorf("error decoding vSwitches: %w", err)
	}
	return vswitches, nil
}

func (c *HetznerRobotClient) CreateVSwitch(ctx context.Context, name string, vlan int) (*VSwitch, error) {
	data := url.Values{}
	data.Set("name", name)
	data.Set("vlan", strconv.Itoa(vlan))
	resp, err := c.DoRequest("POST", "/vswitch", strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, fmt.Errorf("error creating VSwitch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating VSwitch: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	var vswitch VSwitch
	if err := json.NewDecoder(resp.Body).Decode(&vswitch); err != nil {
		return nil, fmt.Errorf("error decoding VSwitch response: %w", err)
	}
	return &vswitch, nil
}

func (c *HetznerRobotClient) UpdateVSwitch(ctx context.Context, id, name string, vlan int, oldVlan int) error {
	data := url.Values{}
	data.Set("name", name)

	if vlan != oldVlan {
		data.Set("vlan", strconv.Itoa(vlan))
		fmt.Printf("VLAN changed, including in update request: %d -> %d\n", oldVlan, vlan)
	} else {
		fmt.Printf("VLAN has not changed, sending only name update\n")
	}

	resp, err := c.DoRequest("POST", fmt.Sprintf("/vswitch/%s", id), strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error updating VSwitch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		switch resp.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("error updating VSwitch: INVALID_INPUT - %s", bodyStr)
		case http.StatusNotFound:
			return fmt.Errorf("error updating VSwitch: NOT_FOUND - %s", bodyStr)
		case http.StatusConflict:
			if strings.Contains(bodyStr, "VSWITCH_IN_PROCESS") {
				return fmt.Errorf("error updating VSwitch: VSWITCH_IN_PROCESS - %s", bodyStr)
			}
			if strings.Contains(bodyStr, "VSWITCH_VLAN_NOT_UNIQUE") {
				return fmt.Errorf("error updating VSwitch: VSWITCH_VLAN_NOT_UNIQUE - %s", bodyStr)
			}
		default:
			return fmt.Errorf("error updating VSwitch: status %d, body %s", resp.StatusCode, bodyStr)
		}
	}

	return nil
}

func (c *HetznerRobotClient) DeleteVSwitch(ctx context.Context, id string, cancellationDate string) error {
	data := url.Values{}
	data.Set("cancellation_date", cancellationDate)
	resp, err := c.DoRequest("DELETE", fmt.Sprintf("/vswitch/%s", id), strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error deleting VSwitch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting VSwitch: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *HetznerRobotClient) AddVSwitchServers(ctx context.Context, id string, servers []VSwitchServer) error {
	data := url.Values{}
	for _, server := range servers {
		data.Add("server[]", strconv.Itoa(server.ServerNumber))
	}
	resp, err := c.DoRequest("POST", fmt.Sprintf("/vswitch/%s/server", id), strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error adding servers to VSwitch: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error adding servers to VSwitch: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *HetznerRobotClient) RemoveVSwitchServers(ctx context.Context, id string, servers []VSwitchServer) error {
	data := url.Values{}
	for _, server := range servers {
		data.Add("server[]", strconv.Itoa(server.ServerNumber))
	}
	resp, err := c.DoRequest("DELETE", fmt.Sprintf("/vswitch/%s/server", id), strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error removing servers from VSwitch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error removing servers from VSwitch: status %d, body %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *HetznerRobotClient) SetVSwitchCancellation(ctx context.Context, id, cancellationDate string) error {
	data := url.Values{}
	data.Set("cancellation_date", cancellationDate)
	resp, err := c.DoRequest("POST", fmt.Sprintf("/vswitch/%s/cancel", id), strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error setting cancellation date: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *HetznerRobotClient) WaitForVSwitchReady(ctx context.Context, id string, maxRetries int, waitTime time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		vsw, err := c.FetchVSwitchByIDWithContext(ctx, id)
		if err != nil {
			return fmt.Errorf("error fetching VSwitch while waiting: %w", err)
		}

		if vsw == nil {
			return fmt.Errorf("vSwitch with ID %s not found", id)
		}

		allReady := true
		for _, server := range vsw.Servers {
			fmt.Printf("Checking server %d status: %s\n", server.ServerNumber, server.Status)
			if server.Status == "processing" {
				allReady = false
				break
			}
		}

		if allReady {
			fmt.Println("vSwitch is now ready.")
			return nil
		}

		fmt.Printf("vSwitch is still processing, retrying in %v seconds (%d/%d)...\n", waitTime.Seconds(), i+1, maxRetries)
		time.Sleep(waitTime)
	}

	return fmt.Errorf("timeout waiting for vSwitch %s to become ready", id)
}
