package ha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	base, token string
	http        *http.Client
}

type CoreConfig struct {
	Country      string  `json:"country"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	TimeZone     string  `json:"time_zone"`
	LocationName string  `json:"location_name"`
}

func New() *Client {
	return &Client{base: "http://supervisor/core/api", token: os.Getenv("SUPERVISOR_TOKEN"), http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("home assistant api %s %s: http %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, nil
}

func (c *Client) Config(ctx context.Context) (CoreConfig, error) {
	var out CoreConfig
	b, err := c.request(ctx, "GET", "/config", nil)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}

func (c *Client) SetState(ctx context.Context, entityID, state string, attrs map[string]any) error {
	_, err := c.request(ctx, "POST", "/states/"+entityID, map[string]any{"state": state, "attributes": attrs})
	return err
}
