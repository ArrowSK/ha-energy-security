package baseline

import (
	"math"
	"testing"
)

func TestElectricityLibraryCoversEmbeddedCountryProfiles(t *testing.T) {
	if got := ElectricityCount(); got != 39 {
		t.Fatalf("expected 39 embedded electricity references, got %d", got)
	}
}

func TestHungaryElectricityReference(t *testing.T) {
	r, ok := Electricity("hu")
	if !ok {
		t.Fatal("expected Hungary electricity reference")
	}
	if r.Year != 2024 {
		t.Fatalf("expected Hungary reference year 2024, got %d", r.Year)
	}
	if math.Abs(r.PopulationMillions-9.669) > 0.001 {
		t.Fatalf("unexpected Hungary population reference %.3f", r.PopulationMillions)
	}
	if math.Abs(r.AverageLoadMW()-5563.0) > 2.0 {
		t.Fatalf("unexpected Hungary annual-average load %.1f MW", r.AverageLoadMW())
	}
}

func TestUkraineReferenceRetainsItsActualSourceYear(t *testing.T) {
	r, ok := Electricity("UA")
	if !ok {
		t.Fatal("expected Ukraine electricity reference")
	}
	if r.Year != 2022 {
		t.Fatalf("Ukraine reference must disclose source year 2022, got %d", r.Year)
	}
}
