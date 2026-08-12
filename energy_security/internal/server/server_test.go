package server

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
)

func TestIngressGuardAndSecurityHeaders(t *testing.T) {
	a := app.New(config.Defaults(), filepath.Join(t.TempDir(), "data"))
	h := New(a).Handler()
	r := httptest.NewRequest("GET", "http://example/healthz", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("loopback request got %d", w.Code)
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}

	r2 := httptest.NewRequest("GET", "http://example/healthz", nil)
	r2.RemoteAddr = "192.0.2.10:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != 403 {
		t.Fatalf("non-ingress request got %d", w2.Code)
	}
}
