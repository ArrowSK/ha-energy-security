package country

import "testing"

func TestProfilesValidateAndHungaryIsFull(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
	p := Resolve("hu")
	if p.Code != "HU" || p.Support != "full" || !p.HungaryGas || !p.HungaryWater {
		t.Fatalf("unexpected Hungary profile: %+v", p)
	}
}

func TestUnknownCountryGetsSafePartialProfile(t *testing.T) {
	p := Resolve("zz")
	if p.Code != "ZZ" || p.Support == "full" {
		t.Fatalf("unexpected fallback profile: %+v", p)
	}
}
