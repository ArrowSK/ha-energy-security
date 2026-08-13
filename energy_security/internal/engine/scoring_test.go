package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

func fp(v float64) *float64 { return &v }
func measurement(key string, v, q float64) model.Observation {
	return model.Observation{Key: key, Value: fp(v), Quality: q, ObservedAt: time.Now().UTC(), TTLSeconds: 3600}
}
func agedMeasurement(key string, v, q float64, age, freshFor, ttl time.Duration) model.Observation {
	return model.Observation{
		Key:        key,
		Value:      fp(v),
		Quality:    q,
		ObservedAt: time.Now().UTC().Add(-age),
		TTLSeconds: int64(ttl.Seconds()),
		Attributes: map[string]any{"fresh_for_seconds": int64(freshFor.Seconds())},
	}
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

func TestEurostatGasProxyProducesLowerConfidenceScore(t *testing.T) {
	s := model.Snapshot{UpdatedAt: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC), Observations: map[string]model.Observation{
		"gas_stock_index_pct": measurement("gas_stock_index_pct", 80, .65),
	}}
	Score(&s)
	g := s.Domains["gas"]
	if g.Score == nil {
		t.Fatal("expected a gas score from the documented Eurostat proxy")
	}
	if g.Confidence >= 40 {
		t.Fatalf("proxy must remain data-limited and below current-horizon confidence, got %.1f", g.Confidence)
	}
	if s.Scores.Current != nil {
		t.Fatal("monthly Eurostat stock proxy must not create a current-horizon score by itself")
	}
	if !strings.Contains(strings.ToLower(g.Summary), "not physical storage-capacity fill") {
		t.Fatalf("proxy summary must disclose the limitation: %q", g.Summary)
	}
}

func TestElectricityUsesEmbeddedReferenceWhenLiveLoadMissing(t *testing.T) {
	s := model.Snapshot{
		Country:   "HU",
		UpdatedAt: time.Date(2026, time.August, 13, 8, 0, 0, 0, time.UTC),
		Observations: map[string]model.Observation{
			"electricity_generation_mw": measurement("electricity_generation_mw", 6000, .9),
		},
	}
	Score(&s)
	e := s.Domains["electricity"]
	if e.Score == nil {
		t.Fatal("expected electricity score using embedded country reference")
	}
	if s.Scores.Current == nil {
		t.Fatal("derived electricity reference should allow a partial current score")
	}
	if e.Confidence >= 50 {
		t.Fatalf("derived reference must carry discounted confidence, got %.1f", e.Confidence)
	}
	if e.Status != "Data limited" {
		t.Fatalf("derived reference must be visibly data limited, got %q", e.Status)
	}
	text := strings.ToLower(e.Summary)
	for _, want := range []string{"live consumption is unavailable", "ember", "48.73 twh", "population"} {
		if !strings.Contains(text, want) {
			t.Fatalf("derived summary must disclose %q: %q", want, e.Summary)
		}
	}
}

func TestLiveElectricityLoadPreferredOverEmbeddedReference(t *testing.T) {
	s := model.Snapshot{
		Country:   "HU",
		UpdatedAt: time.Date(2026, time.August, 13, 8, 0, 0, 0, time.UTC),
		Observations: map[string]model.Observation{
			"electricity_generation_mw": measurement("electricity_generation_mw", 6000, .9),
			"electricity_load_mw":       measurement("electricity_load_mw", 5800, .9),
		},
	}
	Score(&s)
	e := s.Domains["electricity"]
	if e.Score == nil {
		t.Fatal("expected live electricity score")
	}
	if strings.Contains(strings.ToLower(e.Summary), "derived reference") || strings.Contains(strings.ToLower(e.Summary), "live consumption is unavailable") {
		t.Fatalf("live load must be preferred over embedded reference: %q", e.Summary)
	}
	if e.Confidence < 80 {
		t.Fatalf("live load unexpectedly received fallback confidence %.1f", e.Confidence)
	}
}

func TestDelayedElectricityRemainsUsableWithReducedConfidence(t *testing.T) {
	gen := agedMeasurement("electricity_generation_mw", 4660.5, .92, 2*time.Hour, 90*time.Minute, 6*time.Hour)
	load := agedMeasurement("electricity_load_mw", 3926.8, .92, 2*time.Hour, 90*time.Minute, 6*time.Hour)
	s := model.Snapshot{
		Country:   "HU",
		UpdatedAt: time.Now().UTC(),
		Observations: map[string]model.Observation{
			"electricity_generation_mw": gen,
			"electricity_load_mw":       load,
		},
	}
	Score(&s)
	e := s.Domains["electricity"]
	if e.Score == nil {
		t.Fatal("two-hour-old Energy-Charts data should remain usable inside the six-hour hard window")
	}
	if e.Confidence >= 92 || e.Confidence < 70 {
		t.Fatalf("expected modest freshness discount, got %.1f", e.Confidence)
	}
	if !strings.Contains(strings.ToLower(e.Summary), "preferred freshness window") {
		t.Fatalf("delayed electricity summary must disclose freshness discount: %q", e.Summary)
	}
}

func TestDailyGasStorageTwoDaysOldRemainsUsable(t *testing.T) {
	fill := agedMeasurement("gas_storage_fill_pct", 63.7, .98, 50*time.Hour, 48*time.Hour, 7*24*time.Hour)
	s := model.Snapshot{
		Country:   "HU",
		UpdatedAt: time.Now().UTC(),
		Observations: map[string]model.Observation{
			"gas_storage_fill_pct": fill,
		},
	}
	Score(&s)
	g := s.Domains["gas"]
	if g.Score == nil {
		t.Fatal("daily AGSI storage level should remain usable after two days")
	}
	if g.Confidence >= 98 || g.Confidence < 80 {
		t.Fatalf("expected a small age discount for delayed daily gas data, got %.1f", g.Confidence)
	}
	if !strings.Contains(strings.ToLower(g.Summary), "older than its preferred freshness window") {
		t.Fatalf("delayed gas summary must disclose freshness discount: %q", g.Summary)
	}
}

func TestHardExpiryStillExcludesOldGasStorage(t *testing.T) {
	fill := agedMeasurement("gas_storage_fill_pct", 63.7, .98, 8*24*time.Hour, 48*time.Hour, 7*24*time.Hour)
	s := model.Snapshot{Country: "HU", UpdatedAt: time.Now().UTC(), Observations: map[string]model.Observation{"gas_storage_fill_pct": fill}}
	Score(&s)
	if s.Domains["gas"].Score != nil {
		t.Fatal("gas storage older than the seven-day hard limit must not be scored")
	}
}
