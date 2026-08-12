package provider

import "testing"

func TestParseDanubeStation(t *testing.T) {
	text := "442027 Budapest Danube 28 36 40 12 880 25.9 // 442030 Paks Danube -128 -127 -121 7 797 26.9 //"
	b, ok := parseDanubeStation(text, "442027", "Budapest")
	if !ok || b.LevelCM == nil || *b.LevelCM != 40 || b.ChangeCM == nil || *b.ChangeCM != 12 || b.Discharge == nil || *b.Discharge != 880 || b.Temperature == nil || *b.Temperature != 25.9 {
		t.Fatalf("unexpected Budapest parse: %+v ok=%v", b, ok)
	}
	p, ok := parseDanubeStation(text, "442030", "Paks")
	if !ok || p.LevelCM == nil || *p.LevelCM != -121 || p.Temperature == nil || *p.Temperature != 26.9 {
		t.Fatalf("unexpected Paks parse: %+v ok=%v", p, ok)
	}
}

func TestParseDanubeStationMissingFields(t *testing.T) {
	text := "442027 Budapest Danube 28 36 40 12 // // //"
	d, ok := parseDanubeStation(text, "442027", "Budapest")
	if !ok || d.LevelCM == nil || *d.LevelCM != 40 || d.Discharge != nil || d.Temperature != nil {
		t.Fatalf("unexpected parse: %+v ok=%v", d, ok)
	}
}
