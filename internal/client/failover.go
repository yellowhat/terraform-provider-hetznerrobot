package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Failover represents the routing state of a failover IP.
type Failover struct {
	IP             string `json:"ip"`
	Netmask        string `json:"netmask"`
	ServerIP       string `json:"server_ip"`
	ServerNumber   int    `json:"server_number"`
	ActiveServerIP string `json:"active_server_ip"`
}

// ErrFailoverNotFound is returned when no failover IP matches.
var ErrFailoverNotFound = errors.New("failover ip not found")

// FetchFailover returns the routing state of a failover IP.
func (c *HetznerRobotClient) FetchFailover(ctx context.Context, ip string) (Failover, error) {
	resp, err := c.DoRequest(ctx, "GET", "/failover/"+url.PathEscape(ip), nil, "")
	if err != nil {
		return Failover{}, fmt.Errorf("error fetching failover: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Failover{}, ErrFailoverNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return Failover{}, fmt.Errorf("unable to read response body: %w", err)
		}

		return Failover{}, fmt.Errorf(
			"error fetching failover: status %d, body %s",
			resp.StatusCode, body,
		)
	}

	var result struct {
		Failover Failover `json:"failover"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return Failover{}, fmt.Errorf("error decoding failover response: %w", err)
	}

	return result.Failover, nil
}

// SetFailover routes a failover IP to a different active server.
func (c *HetznerRobotClient) SetFailover(ctx context.Context, ip, activeServerIP string) error {
	form := url.Values{}
	form.Set("active_server_ip", activeServerIP)

	resp, err := c.DoRequest(
		ctx,
		"POST",
		"/failover/"+url.PathEscape(ip),
		strings.NewReader(form.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("SetFailover request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("SetFailover %s: status %d, body %s", ip, resp.StatusCode, body)
	}

	return nil
}

// DeleteFailover resets a failover IP's routing back to its primary server.
func (c *HetznerRobotClient) DeleteFailover(ctx context.Context, ip string) error {
	resp, err := c.DoRequest(ctx, "DELETE", "/failover/"+url.PathEscape(ip), nil, "")
	if err != nil {
		return fmt.Errorf("DeleteFailover request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	return fmt.Errorf("DeleteFailover %s: status %d, body %s", ip, resp.StatusCode, body)
}
