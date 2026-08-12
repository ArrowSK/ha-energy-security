package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	st := Store{Path: filepath.Join(t.TempDir(), "state.json")}
	want := model.Snapshot{Version: 1, Country: "HU", CountryName: "Hungary", UpdatedAt: time.Now().UTC(), Observations: map[string]model.Observation{}, Domains: map[string]model.DomainScore{}}
	if err := st.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Country != want.Country || got.Version != want.Version {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}
