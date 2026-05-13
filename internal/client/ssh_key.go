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

// SSHKey represents an SSH key registered in the Hetzner Robot account.
type SSHKey struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Data        string `json:"data"`
	CreatedAt   string `json:"created_at"`
}

// ErrSSHKeyNotFound is returned when no key matches the requested fingerprint.
var ErrSSHKeyNotFound = errors.New("ssh key not found")

// FetchSSHKey returns the SSH key entry for a fingerprint.
func (c *HetznerRobotClient) FetchSSHKey(ctx context.Context, fingerprint string) (SSHKey, error) {
	resp, err := c.DoRequest(ctx, "GET", "/key/"+url.PathEscape(fingerprint), nil, "")
	if err != nil {
		return SSHKey{}, fmt.Errorf("FetchSSHKey request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return SSHKey{}, ErrSSHKeyNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return SSHKey{}, fmt.Errorf(
			"FetchSSHKey %s: status %d, body %s",
			fingerprint, resp.StatusCode, body,
		)
	}

	var result struct {
		Key SSHKey `json:"key"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return SSHKey{}, fmt.Errorf("FetchSSHKey decode error: %w", err)
	}

	return result.Key, nil
}

// CreateSSHKey uploads a new SSH key. The fingerprint is computed by Hetzner.
func (c *HetznerRobotClient) CreateSSHKey(ctx context.Context, name, data string) (SSHKey, error) {
	form := url.Values{}
	form.Set("name", name)
	form.Set("data", data)

	resp, err := c.DoRequest(
		ctx,
		"POST",
		"/key",
		strings.NewReader(form.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return SSHKey{}, fmt.Errorf("CreateSSHKey request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)

		return SSHKey{}, fmt.Errorf("CreateSSHKey: status %d, body %s", resp.StatusCode, body)
	}

	var result struct {
		Key SSHKey `json:"key"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return SSHKey{}, fmt.Errorf("CreateSSHKey decode error: %w", err)
	}

	return result.Key, nil
}

// RenameSSHKey updates the display name of an SSH key without changing its data.
func (c *HetznerRobotClient) RenameSSHKey(ctx context.Context, fingerprint, name string) error {
	form := url.Values{}
	form.Set("name", name)

	resp, err := c.DoRequest(
		ctx,
		"POST",
		"/key/"+url.PathEscape(fingerprint),
		strings.NewReader(form.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return fmt.Errorf("RenameSSHKey request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"RenameSSHKey %s: status %d, body %s",
			fingerprint, resp.StatusCode, body,
		)
	}

	return nil
}

// DeleteSSHKey removes an SSH key. A 404 is treated as success.
func (c *HetznerRobotClient) DeleteSSHKey(ctx context.Context, fingerprint string) error {
	resp, err := c.DoRequest(ctx, "DELETE", "/key/"+url.PathEscape(fingerprint), nil, "")
	if err != nil {
		return fmt.Errorf("DeleteSSHKey request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	return fmt.Errorf("DeleteSSHKey %s: status %d, body %s", fingerprint, resp.StatusCode, body)
}
