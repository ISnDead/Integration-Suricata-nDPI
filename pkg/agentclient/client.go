package agentclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	http *http.Client
}

type ToggleResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`

	Changed bool `json:"changed"`
	Enabled bool `json:"enabled"`
}

type StatusResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`

	Enabled bool   `json:"enabled"`
	Line    string `json:"line"`
}

type ErrorResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
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
	return c.postToggle(ctx, "http://unix/ndpi/enable")
}

func (c *Client) DisableNDPI(ctx context.Context) (*ToggleResponse, error) {
	return c.postToggle(ctx, "http://unix/ndpi/disable")
}

func (c *Client) NDPIStatus(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/ndpi/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, decodeAgentHTTPError(resp)
	}

	var out StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return &out, fmt.Errorf("agent error: %s", out.Message)
	}

	return &out, nil
}

func (c *Client) postToggle(ctx context.Context, url string) (*ToggleResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, decodeAgentHTTPError(resp)
	}

	var out ToggleResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return &out, fmt.Errorf("agent error: %s", out.Message)
	}

	return &out, nil
}

func decodeAgentHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var er ErrorResponse
	if err := json.Unmarshal(body, &er); err == nil && strings.TrimSpace(er.Message) != "" {
		return fmt.Errorf("agent http %d: %s", resp.StatusCode, er.Message)
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("agent http %d: %s", resp.StatusCode, msg)
}
