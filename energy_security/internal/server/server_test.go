package server

import (
	"testing"

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
