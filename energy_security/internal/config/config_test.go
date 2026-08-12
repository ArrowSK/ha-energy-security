package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAreZeroConfiguration(t *testing.T) {
	c := Defaults()
	if c.Country != "auto" || c.RefreshMinutes != 30 || !c.EnableHAEntities || !c.EnableWeather {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoadNormalizesCountryAndBoundsRefresh(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "options.json")
	if err := os.WriteFile(p, []byte(`{"country":"hu","refresh_minutes":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Country != "HU" || c.RefreshMinutes != 10 {
		t.Fatalf("unexpected config: %+v", c)
	}
}
