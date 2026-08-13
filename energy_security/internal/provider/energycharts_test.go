package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
)

func TestEnergyChartsUsesLookbackWindow(t *testing.T) {
	fixture := `{
		"unix_seconds":[1786406400,1786407300],
		"production_types":[
			{"name":"Nuclear","data":[1500,1510]},
			{"name":"Solar","data":[600,620]},
			{"name":"Load","data":[4300,4350]}
		]
	}`
	client := &httpx.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()
		if q.Get("country") != "hu" {
			t.Fatalf("unexpected country: %q", q.Get("country"))
		}
		if q.Get("start") == "" || q.Get("end") == "" {
			t.Fatalf("Energy-Charts request must use an explicit lookback window: %s", r.URL.String())
		}
		start, err := time.Parse("2006-01-02", q.Get("start"))
		if err != nil {
			t.Fatal(err)
		}
		end, err := time.Parse("2006-01-02", q.Get("end"))
		if err != nil {
			t.Fatal(err)
		}
		if end.Sub(start) < 48*time.Hour {
			t.Fatalf("lookback too short: %s to %s", start, end)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fixture)), Header: make(http.Header)}, nil
	})}}
	out, err := (EnergyCharts{C: client}).Collect(context.Background(), Input{Profile: country.Profile{Code: "HU", EnergyCharts: "hu"}})
	if err != nil {
		t.Fatal(err)
	}
	seenGeneration, seenLoad := false, false
	for _, o := range out {
		seenGeneration = seenGeneration || o.Key == "electricity_generation_mw"
		seenLoad = seenLoad || o.Key == "electricity_load_mw"
		if o.TTLSeconds != int64((6 * time.Hour).Seconds()) {
			t.Fatalf("Energy-Charts observation should have six-hour hard expiry, got %d", o.TTLSeconds)
		}
		if o.Attributes == nil {
			t.Fatalf("Energy-Charts observation missing freshness attributes: %+v", o)
		}
		fresh, ok := o.Attributes["fresh_for_seconds"].(int64)
		if !ok || fresh != int64((90*time.Minute).Seconds()) {
			t.Fatalf("unexpected preferred freshness metadata: %#v", o.Attributes["fresh_for_seconds"])
		}
	}
	if !seenGeneration || !seenLoad {
		t.Fatalf("expected generation and load observations, got %+v", out)
	}
}
