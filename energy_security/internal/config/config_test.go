package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAreZeroConfiguration(t *testing.T) {
	c := Defaults()
	if c.Country != "auto" || c.RefreshMinutes != 30 || !c.EnableHAEntities || !c.EnableWeather || c.RuntimeMode != "home_assistant" {
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
	if c.Country != "HU" || c.RefreshMinutes != 10 || c.RuntimeMode != "home_assistant" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestLoadEnvironmentUsesSharedConfigWithoutHAEntities(t *testing.T) {
	t.Setenv("ENERGY_SECURITY_COUNTRY", "hu")
	t.Setenv("ENERGY_SECURITY_REFRESH_MINUTES", "45")
	t.Setenv("ENERGY_SECURITY_ENABLE_WEATHER", "true")
	t.Setenv("ENERGY_SECURITY_LATITUDE", "47.4979")
	t.Setenv("ENERGY_SECURITY_LONGITUDE", "19.0402")
	t.Setenv("ENERGY_SECURITY_TIMEZONE", "Europe/Budapest")
	t.Setenv("ENERGY_SECURITY_LOCATION_NAME", "Budapest")
	t.Setenv("ENERGY_SECURITY_AGSI_KEY", "agsi")
	t.Setenv("ENERGY_SECURITY_ENTSOE_TOKEN", "entsoe")

	c, err := LoadEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if c.Country != "HU" || c.RefreshMinutes != 45 || c.RuntimeMode != "standalone" || c.EnableHAEntities {
		t.Fatalf("unexpected standalone config: %+v", c)
	}
	if c.Latitude == nil || c.Longitude == nil || *c.Latitude != 47.4979 || *c.Longitude != 19.0402 {
		t.Fatalf("standalone coordinates were not loaded: %+v", c)
	}
	if c.TimeZone != "Europe/Budapest" || c.LocationName != "Budapest" || c.AGSIKey != "agsi" || c.ENTSOEToken != "entsoe" {
		t.Fatalf("standalone metadata was not loaded: %+v", c)
	}
}

func TestLoadEnvironmentRequiresExplicitCountry(t *testing.T) {
	t.Setenv("ENERGY_SECURITY_COUNTRY", "")
	if _, err := LoadEnvironment(); err == nil {
		t.Fatal("expected standalone mode to require an explicit country")
	}
}

func TestLoadEnvironmentRejectsAutoCountry(t *testing.T) {
	t.Setenv("ENERGY_SECURITY_COUNTRY", "auto")
	if _, err := LoadEnvironment(); err == nil {
		t.Fatal("expected standalone mode to reject auto country")
	}
}

func TestLoadEnvironmentRequiresCoordinatePair(t *testing.T) {
	t.Setenv("ENERGY_SECURITY_COUNTRY", "HU")
	t.Setenv("ENERGY_SECURITY_LATITUDE", "47.5")
	if _, err := LoadEnvironment(); err == nil {
		t.Fatal("expected standalone coordinates to be provided as a pair")
	}
}
