package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEurostatOilSelectsExplicitEmergencyStockSeries(t *testing.T) {
	fixture := `{
		"id":["freq","stk_flow","unit","geo","time"],
		"size":[1,2,1,1,1],
		"dimension":{
			"freq":{"category":{"index":{"M":0},"label":{"M":"Monthly"}}},
			"stk_flow":{"category":{"index":{"STK_EUE_DIR":0,"STK_MIN_CAL":1},"label":{"STK_EUE_DIR":"Emergency Stocks held by the MS in days equivalent","STK_MIN_CAL":"Minimum stocklevel for compliance - calculated"}}},
			"unit":{"category":{"index":{"NR":0},"label":{"NR":"Number"}}},
			"geo":{"category":{"index":{"HU":0},"label":{"HU":"Hungary"}}},
			"time":{"category":{"index":{"2026-05":0},"label":{"2026-05":"2026-05"}}}
		},
		"value":{"0":92.4,"1":90.0}
	}`
	client := &httpx.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()
		flows := q["stk_flow"]
		if len(flows) != 2 || flows[0] != "STK_EUE_DIR" || flows[1] != "STK_MIN_CAL" {
			t.Fatalf("unexpected stk_flow query: %v", flows)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fixture)), Header: make(http.Header)}, nil
	})}}
	p := EurostatOil{C: client}
	out, err := p.Collect(context.Background(), Input{Profile: country.Profile{Code: "HU", Eurostat: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Key != "oil_emergency_stock_days" || out[0].Value == nil || *out[0].Value != 92.4 {
		t.Fatalf("unexpected emergency stock observation: %+v", out)
	}
	if out[1].Key != "oil_required_stock_days" || out[1].Value == nil || *out[1].Value != 90.0 {
		t.Fatalf("unexpected minimum stock observation: %+v", out[1])
	}
}

func TestEurostatOilRejectsImplausibleDays(t *testing.T) {
	fixture := `{
		"id":["stk_flow"],"size":[1],
		"dimension":{"stk_flow":{"category":{"index":{"STK_EUE_DIR":0},"label":{"STK_EUE_DIR":"Emergency Stocks held by the MS in days equivalent"}}}},
		"value":{"0":2.0}
	}`
	client := &httpx.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fixture)), Header: make(http.Header)}, nil
	})}}
	_, err := (EurostatOil{C: client}).Collect(context.Background(), Input{Profile: country.Profile{Code: "DE", Eurostat: true}})
	if err == nil || !strings.Contains(err.Error(), "implausible") {
		t.Fatalf("expected implausible-days rejection, got %v", err)
	}
}
