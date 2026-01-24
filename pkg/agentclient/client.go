package agentclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	http *http.Client
}

type ToggleResponse struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
	Enabled bool   `json:"enabled,omitempty"`
}

func New(sockPath string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "unix", sockPath)
		},
	}

	return &Client{
		http: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}

func (c *Client) EnableNDPI(ctx context.Context) (*ToggleResponse, error) {
	return c.post(ctx, "http://unix/ndpi/enable")
}

func (c *Client) DisableNDPI(ctx context.Context) (*ToggleResponse, error) {
	return c.post(ctx, "http://unix/ndpi/disable")
}

func (c *Client) post(ctx context.Context, url string) (*ToggleResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out ToggleResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 || !out.OK {
		return &out, fmt.Errorf("agent error: %s", out.Message)
	}

	return &out, nil
}
