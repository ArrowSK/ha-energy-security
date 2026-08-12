package engine

import (
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

func fp(v float64) *float64 { return &v }
func measurement(key string, v, q float64) model.Observation {
	return model.Observation{Key: key, Value: fp(v), Quality: q, ObservedAt: time.Now().UTC(), TTLSeconds: 3600}
}

func TestMissingDomainsReduceConfidenceWithoutBecomingZero(t *testing.T) {
	s := model.Snapshot{UpdatedAt: time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC), Observations: map[string]model.Observation{
		"electricity_generation_mw": measurement("electricity_generation_mw", 7000, .9),
		"electricity_load_mw":       measurement("electricity_load_mw", 6500, .9),
	}}
	Score(&s)
	if s.Scores.Headline == nil {
		t.Fatal("expected a partial headline score")
	}
	if *s.Scores.Headline <= 0 {
		t.Fatalf("missing domains must not be converted to zero, got %.1f", *s.Scores.Headline)
	}
	if s.Scores.Confidence >= 50 {
		t.Fatalf("expected partial data to reduce confidence below 50%%, got %.1f", s.Scores.Confidence)
	}
}

func TestStaleMeasurementExcluded(t *testing.T) {
	o := measurement("gas_storage_fill_pct", 95, .99)
	o.Stale = true
	s := model.Snapshot{UpdatedAt: time.Now().UTC(), Observations: map[string]model.Observation{"gas_storage_fill_pct": o}}
	Score(&s)
	if s.Domains["gas"].Score != nil {
		t.Fatal("stale gas value must not contribute a score")
	}
	if s.Domains["gas"].Status != "Unknown" {
		t.Fatalf("expected Unknown, got %q", s.Domains["gas"].Status)
	}
}

func TestLowGasTriggersAlert(t *testing.T) {
	s := model.Snapshot{UpdatedAt: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC), Observations: map[string]model.Observation{
		"gas_storage_fill_pct": measurement("gas_storage_fill_pct", 20, .98),
	}}
	Score(&s)
	if len(s.Alerts) == 0 {
		t.Fatal("expected low gas storage alert")
	}
}
