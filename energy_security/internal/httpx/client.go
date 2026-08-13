package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"time"
)

type Client struct {
	HTTP      *http.Client
	UserAgent string
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 15 * time.Second}, UserAgent: "ha-energy-security/0.1.3"}
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string, maxBytes int64) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json,text/html,application/xml;q=0.9,*/*;q=0.8")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		var ue *neturl.Error
		if errors.As(err, &ue) && ue.Err != nil {
			return nil, 0, fmt.Errorf("request failed: %v", ue.Err)
		}
		return nil, 0, errors.New("request failed")
	}
	defer resp.Body.Close()
	if maxBytes <= 0 {
		maxBytes = 4 << 20
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}
	return b, resp.StatusCode, nil
}
