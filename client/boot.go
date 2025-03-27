package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *HetznerRobotClient) RenameServer(ctx context.Context, serverID int, newName string) (*HetznerRenameResponse, error) {
	endpoint := fmt.Sprintf("/server/%d", serverID)
	data := url.Values{}
	data.Set("server_name", newName)
	resp, err := c.DoRequest("POST", endpoint, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
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

func (c *HetznerRobotClient) EnableRescueMode(ctx context.Context, serverID int, os string, sshKeys []string) (*HetznerRescueResponse, error) {
	endpoint := fmt.Sprintf("/boot/%d/rescue", serverID)
	data := url.Values{}
	data.Set("os", os)
	for _, key := range sshKeys {
		data.Add("authorized_key[]", key)
	}
	resp, err := c.DoRequest("POST", endpoint, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
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

func (c *HetznerRobotClient) RebootServer(ctx context.Context, serverID int, resetType string) error {
	endpoint := fmt.Sprintf("/reset/%d", serverID)
	data := url.Values{}
	data.Set("type", resetType)

	resp, err := c.DoRequest("POST", endpoint, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return fmt.Errorf("error rebooting server %d with reset type %s: %w", serverID, resetType, err)
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

		powerResp, err := c.DoRequest("POST", endpoint, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			return fmt.Errorf("error turning on server %d after %s reset: %w", serverID, resetType, err)
		}
		defer powerResp.Body.Close()

		if powerResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(powerResp.Body)
			return fmt.Errorf("unexpected status code %d when turning on server, body: %s", powerResp.StatusCode, string(body))
		}

		fmt.Printf("[DEBUG] Server %d successfully powered on\n", serverID)
	}

	return nil
}
