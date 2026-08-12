package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed")
}

func TestTransportErrorDoesNotEchoRequestURL(t *testing.T) {
	c := New()
	c.HTTP = &http.Client{Transport: failingTransport{}}
	_, _, err := c.Get(context.Background(), "https://example.invalid/api?securityToken=super-secret", nil, 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "super-secret") || strings.Contains(err.Error(), "securityToken") {
		t.Fatalf("request URL leaked into error: %v", err)
	}
}
