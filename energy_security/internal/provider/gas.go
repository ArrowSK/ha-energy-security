package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type HungaryFGSZ struct{ C *httpx.Client }

func (p HungaryFGSZ) ID() string             { return "fgsz_hu" }
func (p HungaryFGSZ) Name() string           { return "FGSZ" }
func (p HungaryFGSZ) Supports(in Input) bool { return in.Profile.HungaryGas }
func (p HungaryFGSZ) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	u := "https://fgsz.hu/en"
	b, _, err := p.C.Get(ctx, u, nil, 5<<20)
	if err != nil {
		return nil, err
	}
	t := htmlText(b)
	at := time.Now().UTC()
	budapest, tzErr := time.LoadLocation("Europe/Budapest")
	if tzErr != nil {
		budapest = time.UTC
	}
	if m := regexp.MustCompile(`(?i)Last update on:\s*([0-9]{4})\.\s*([0-9]{1,2})\.\s*([0-9]{1,2})\.\s*([0-9]{1,2}):([0-9]{2})`).FindStringSubmatch(t); len(m) == 6 {
		if x, e := time.ParseInLocation("2006 1 2 15:04", strings.Join(m[1:], " "), budapest); e == nil {
			at = x.UTC()
		}
	}
	out := []model.Observation{}
	if m := regexp.MustCompile(`(?i)Total stock level in domestic gas storage.*?([0-9]+(?:[.,][0-9]+)?)\s*%`).FindStringSubmatch(t); len(m) > 1 {
		if v, ok := number(m[1]); ok {
			out = append(out, obs("gas_storage_fill_pct", "gas", "Gas storage fill", v, "%", p.Name(), u, 0.96, at))
		}
	}
	for _, x := range []struct{ label, key, name string }{{"Total domestic production", "gas_domestic_production_mw", "Domestic gas production"}, {"Total domestic withdrawal", "gas_storage_withdrawal_mw", "Gas storage withdrawal"}, {"Total domestic consumption", "gas_consumption_mw", "Domestic gas consumption"}, {"Total domestic injection", "gas_storage_injection_mw", "Gas storage injection"}} {
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(x.label) + `\s*([0-9][0-9., ]*)\s*kWh/h`)
		m := re.FindStringSubmatch(t)
		if len(m) > 1 {
			if v, ok := number(m[1]); ok {
				out = append(out, obs(x.key, "gas", x.name, v/1000, "MW", p.Name(), u, 0.94, at))
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("FGSZ page did not contain recognised gas measurements")
	}
	return out, nil
}

type AGSI struct{ C *httpx.Client }

func (p AGSI) ID() string             { return "gie_agsi" }
func (p AGSI) Name() string           { return "GIE AGSI" }
func (p AGSI) Supports(in Input) bool { return strings.TrimSpace(in.Config.AGSIKey) != "" }

type agsiResp struct {
	Data []map[string]any `json:"data"`
}

func strFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case string:
		return number(x)
	case float64:
		return x, true
	}
	return 0, false
}
func (p AGSI) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	u := "https://agsi.gie.eu/api?" + url.Values{"country": []string{in.Profile.Code}, "size": []string{"3"}}.Encode()
	b, _, err := p.C.Get(ctx, u, map[string]string{"x-key": in.Config.AGSIKey}, 4<<20)
	if err != nil {
		return nil, err
	}
	var r agsiResp
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("decode AGSI: %w", err)
	}
	if len(r.Data) == 0 {
		return nil, fmt.Errorf("AGSI returned no country data")
	}
	d := r.Data[0]
	at := time.Now().UTC()
	for _, k := range []string{"gasDayStart", "gasDayStartedOn"} {
		if s, ok := d[k].(string); ok {
			if tt, e := time.Parse("2006-01-02", s); e == nil {
				at = tt.UTC()
				break
			}
		}
	}
	out := []model.Observation{}
	for _, x := range []struct {
		k, key, label, unit string
		q                   float64
	}{{"full", "gas_storage_fill_pct", "Gas storage fill", "%", 0.98}, {"gasInStorage", "gas_in_storage_twh", "Gas in storage", "TWh", 0.98}, {"workingGasVolume", "gas_working_capacity_twh", "Working gas capacity", "TWh", 0.97}, {"injection", "gas_storage_injection_gwh_day", "Storage injection", "GWh/day", 0.96}, {"withdrawal", "gas_storage_withdrawal_gwh_day", "Storage withdrawal", "GWh/day", 0.96}, {"consumptionFull", "gas_storage_consumption_cover_pct", "Storage vs annual consumption", "%", 0.90}, {"trend", "gas_storage_daily_trend_pct", "Daily storage trend", "%", 0.90}} {
		if v, ok := strFloat(d[x.k]); ok {
			out = append(out, obs(x.key, "gas", x.label, v, x.unit, p.Name(), u, x.q, at))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("AGSI response lacked recognised fields")
	}
	return out, nil
}
