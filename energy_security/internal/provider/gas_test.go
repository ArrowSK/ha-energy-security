package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
)

func TestEurostatGasStocksBuildsLowFrequencyProxy(t *testing.T) {
	fixture := `{
		"id":["freq","siec","stk_flow","unit","geo","time"],
		"size":[1,1,1,1,1,3],
		"dimension":{
			"freq":{"category":{"index":{"M":0},"label":{"M":"Monthly"}}},
			"siec":{"category":{"index":{"G3000":0},"label":{"G3000":"Natural gas"}}},
			"stk_flow":{"category":{"index":{"STKCL_NAT":0},"label":{"STKCL_NAT":"Closing stock - national territory"}}},
			"unit":{"category":{"index":{"TJ_GCV":0},"label":{"TJ_GCV":"Terajoule (gross calorific value)"}}},
			"geo":{"category":{"index":{"HU":0},"label":{"HU":"Hungary"}}},
			"time":{"category":{"index":{"2026-03":0,"2026-04":1,"2026-05":2},"label":{"2026-03":"2026-03","2026-04":"2026-04","2026-05":"2026-05"}}}
		},
		"value":{"0":180000,"1":216000,"2":198000}
	}`
	client := &httpx.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()
		if q.Get("siec") != "G3000" || q.Get("stk_flow") != "STKCL_NAT" || q.Get("unit") != "TJ_GCV" || q.Get("lastTimePeriod") != "36" {
			t.Fatalf("unexpected Eurostat gas query: %s", r.URL.RawQuery)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fixture)), Header: make(http.Header)}, nil
	})}}
	out, err := (EurostatGasStocks{C: client}).Collect(context.Background(), Input{Profile: country.Profile{Code: "HU", Eurostat: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("unexpected observations: %+v", out)
	}
	if out[0].Key != "gas_national_stock_twh" || out[0].Value == nil || *out[0].Value != 55 {
		t.Fatalf("unexpected stock observation: %+v", out[0])
	}
	if out[1].Key != "gas_stock_index_pct" || out[1].Value == nil || *out[1].Value < 91.6 || *out[1].Value > 91.7 {
		t.Fatalf("unexpected proxy observation: %+v", out[1])
	}
}

func TestAGSIAssignsMetricAppropriateFreshnessWindows(t *testing.T) {
	fixture := `{"data":[{"gasDayStart":"2026-08-11","full":"63.7","gasInStorage":"43.1","workingGasVolume":"67.7","injection":"178.8","withdrawal":"0","consumptionFull":"45.7","trend":"0.3"}]}`
	client := &httpx.Client{HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("x-key") != "test-key" {
			t.Fatalf("AGSI key not sent")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fixture)), Header: make(http.Header)}, nil
	})}}
	out, err := (AGSI{C: client}).Collect(context.Background(), Input{
		Profile: country.Profile{Code: "HU"},
		Config:  config.Config{AGSIKey: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 7 {
		t.Fatalf("expected seven AGSI observations, got %+v", out)
	}
	for _, o := range out {
		fresh, ok := o.Attributes["fresh_for_seconds"].(int64)
		if !ok {
			t.Fatalf("missing freshness metadata on %s: %+v", o.Key, o.Attributes)
		}
		switch o.Key {
		case "gas_storage_fill_pct", "gas_in_storage_twh", "gas_working_capacity_twh", "gas_storage_consumption_cover_pct":
			if o.TTLSeconds != int64((7*24*time.Hour).Seconds()) || fresh != int64((48*time.Hour).Seconds()) {
				t.Fatalf("unexpected storage freshness policy for %s: ttl=%d fresh=%d", o.Key, o.TTLSeconds, fresh)
			}
		default:
			if o.TTLSeconds != int64((4*24*time.Hour).Seconds()) || fresh != int64((36*time.Hour).Seconds()) {
				t.Fatalf("unexpected flow freshness policy for %s: ttl=%d fresh=%d", o.Key, o.TTLSeconds, fresh)
			}
		}
	}
}
