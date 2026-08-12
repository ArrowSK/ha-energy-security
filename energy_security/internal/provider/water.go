package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type HungaryHydroinfo struct{ C *httpx.Client }

func (p HungaryHydroinfo) ID() string { return "hydroinfo_hu" }
func (p HungaryHydroinfo) Name() string {
	return "Hungarian Hydrological Forecasting Service (HYDROINFO)"
}
func (p HungaryHydroinfo) Supports(in Input) bool { return in.Profile.HungaryWater }

type danubeStation struct {
	LevelCM     *float64
	ChangeCM    *float64
	Discharge   *float64
	Temperature *float64
}

func optionalNumber(s string) *float64 {
	if strings.TrimSpace(s) == "//" || strings.TrimSpace(s) == "" {
		return nil
	}
	v, ok := number(s)
	if !ok {
		return nil
	}
	return &v
}

func parseDanubeStation(text, code, station string) (danubeStation, bool) {
	field := `(-?[0-9]+(?:[.,][0-9]+)?|//)`
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(code) + `\s+` + regexp.QuoteMeta(station) + `\s+Danube\s+` + field + `\s+` + field + `\s+` + field + `\s+` + field + `\s+` + field + `\s+` + field)
	m := re.FindStringSubmatch(text)
	if len(m) != 7 {
		return danubeStation{}, false
	}
	return danubeStation{
		LevelCM:     optionalNumber(m[3]),
		ChangeCM:    optionalNumber(m[4]),
		Discharge:   optionalNumber(m[5]),
		Temperature: optionalNumber(m[6]),
	}, true
}

func appendDanubeStation(out []model.Observation, stationKey, label string, d danubeStation, source, sourceURL string, at time.Time) []model.Observation {
	if d.LevelCM != nil {
		o := obs("danube_"+stationKey+"_level_cm", "water", "Danube "+label+" water level", *d.LevelCM, "cm", source, sourceURL, 0.94, at)
		if d.ChangeCM != nil {
			o.Attributes = map[string]any{"change_24h_cm": *d.ChangeCM}
		}
		out = append(out, o)
	}
	if d.Discharge != nil {
		out = append(out, obs("danube_"+stationKey+"_discharge_m3s", "water", "Danube "+label+" discharge", *d.Discharge, "m³/s", source, sourceURL, 0.94, at))
	}
	if d.Temperature != nil {
		out = append(out, obs("danube_"+stationKey+"_water_temperature_c", "water", "Danube "+label+" water temperature", *d.Temperature, "°C", source, sourceURL, 0.94, at))
	}
	return out
}
func (p HungaryHydroinfo) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	u := "https://www.hydroinfo.hu/mobil/en/hydroinfo.php"
	b, _, err := p.C.Get(ctx, u, nil, 3<<20)
	if err != nil {
		return nil, err
	}
	t := htmlText(b)
	low := strings.ToLower(t)
	at := time.Now().UTC()
	budapest, tzErr := time.LoadLocation("Europe/Budapest")
	if tzErr != nil {
		budapest = time.UTC
	}
	if m := regexp.MustCompile(`(?i)([0-9]{1,2})\.([0-9]{1,2})\.([0-9]{4})\s+([0-9]{1,2}):([0-9]{2})`).FindStringSubmatch(t); len(m) == 6 {
		if tt, e := time.ParseInLocation("2.1.2006 15:04", m[1]+"."+m[2]+"."+m[3]+" "+m[4]+":"+m[5], budapest); e == nil {
			at = tt.UTC()
		}
	}
	score := 75.0
	state := "normal"
	if strings.Contains(low, "below the ever observed minima") || strings.Contains(low, "extremely low") {
		score = 30
		state = "extremely low"
	} else if strings.Contains(low, "very low") {
		score = 45
		state = "very low"
	} else if strings.Contains(low, "low water") {
		score = 55
		state = "low"
	}
	if (strings.Contains(low, "flood alert") || strings.Contains(low, "flood level")) && !strings.Contains(low, "not expected") {
		if score > 40 {
			score = 40
		}
		state = "flood risk"
	}
	o := obs("water_security_proxy", "water", "Hydrological security", score, "score", p.Name(), u, 0.87, at)
	o.Text = state
	o.Attributes = map[string]any{"report": t}
	out := []model.Observation{o, textObs("hydrology_summary", "water", "Hydrological outlook", t, p.Name(), u, 0.87, at)}

	du := "https://www.hydroinfo.hu/tables/eng/dunhif.html"
	if db, _, de := p.C.Get(ctx, du, nil, 4<<20); de == nil {
		dt := htmlText(db)
		if d, ok := parseDanubeStation(dt, "442027", "Budapest"); ok {
			out = appendDanubeStation(out, "budapest", "Budapest", d, p.Name(), du, at)
		}
		if d, ok := parseDanubeStation(dt, "442030", "Paks"); ok {
			out = appendDanubeStation(out, "paks", "Paks", d, p.Name(), du, at)
		}
	}

	lu := "https://www.hydroinfo.hu/tables/ENG/tohid.html"
	lb, _, le := p.C.Get(ctx, lu, nil, 3<<20)
	if le == nil {
		lt := htmlText(lb)
		re := regexp.MustCompile(`(?i)Balaton average\s+Balaton\s+([0-9-]+|//)\s+([0-9-]+|//)\s+([0-9-]+|//)`)
		if m := re.FindStringSubmatch(lt); len(m) >= 4 {
			if v, ok := number(m[3]); ok {
				out = append(out, obs("balaton_level_cm", "water", "Lake Balaton level", v, "cm", p.Name(), lu, 0.90, at))
			}
		}
	}
	return out, nil
}

var _ = fmt.Sprintf
