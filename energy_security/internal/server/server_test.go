package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
)

func TestNormalizeSetupPreservesBlankSecrets(t *testing.T) {
	current := config.Config{Country: "HU", RefreshMinutes: 30, EnableHAEntities: true, EnableWeather: true, AGSIKey: "agsi-secret", ENTSOEToken: "entsoe-secret"}
	next, err := normalizeSetup(setupRequest{Country: "auto", RefreshMinutes: 60, EnableHAEntities: true, EnableWeather: false}, current)
	if err != nil {
		t.Fatal(err)
	}
	if next.AGSIKey != current.AGSIKey || next.ENTSOEToken != current.ENTSOEToken {
		t.Fatal("blank dashboard secret fields must preserve existing credentials")
	}
	if next.Country != "auto" || next.RefreshMinutes != 60 || next.EnableWeather {
		t.Fatalf("unexpected normalized config: %+v", next)
	}
}

func TestNormalizeSetupCanExplicitlyClearSecrets(t *testing.T) {
	current := config.Config{Country: "HU", RefreshMinutes: 30, AGSIKey: "agsi-secret", ENTSOEToken: "entsoe-secret"}
	next, err := normalizeSetup(setupRequest{Country: "HU", RefreshMinutes: 30, ClearAGSIKey: true, ClearENTSOEToken: true}, current)
	if err != nil {
		t.Fatal(err)
	}
	if next.AGSIKey != "" || next.ENTSOEToken != "" {
		t.Fatal("explicit clear flags must remove credentials")
	}
}

func TestNormalizeSetupRejectsInvalidRefreshInterval(t *testing.T) {
	_, err := normalizeSetup(setupRequest{Country: "HU", RefreshMinutes: 5}, config.Defaults())
	if err == nil {
		t.Fatal("expected invalid refresh interval to be rejected")
	}
}

func TestNormalizeSetupRejectsInvalidCountry(t *testing.T) {
	_, err := normalizeSetup(setupRequest{Country: "HUN", RefreshMinutes: 30}, config.Defaults())
	if err == nil {
		t.Fatal("expected invalid country code to be rejected")
	}
}

func TestStandaloneHandlerAllowsExternalRequests(t *testing.T) {
	cfg := config.Config{Country: "HU", RefreshMinutes: 30, EnableWeather: false, RuntimeMode: "standalone"}
	a := app.New(cfg, t.TempDir())
	s := NewStandalone(a, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/healthz", nil)
	req.RemoteAddr = "203.0.113.10:40000"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("standalone health request returned %d", rec.Code)
	}
}

func TestHomeAssistantHandlerKeepsIngressGuard(t *testing.T) {
	a := app.New(config.Defaults(), t.TempDir())
	s := New(a)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/healthz", nil)
	req.RemoteAddr = "203.0.113.10:40000"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Home Assistant ingress guard returned %d", rec.Code)
	}
}

func TestStandaloneConfigurationIsReadOnly(t *testing.T) {
	cfg := config.Config{Country: "HU", RefreshMinutes: 30, EnableWeather: true, RuntimeMode: "standalone"}
	a := app.New(cfg, t.TempDir())
	s := NewStandalone(a, cfg)
	req := httptest.NewRequest(http.MethodPost, "http://example.test/api/v1/config", strings.NewReader(`{"country":"DE"}`))
	req.RemoteAddr = "203.0.113.10:40000"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("standalone config mutation returned %d", rec.Code)
	}
}
