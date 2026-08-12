package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
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
	if m := regexp.MustCompile(`(?i)Total stock level in domestic gas storage[^%]{0,240}?([0-9]+(?:[.,][0-9]+)?)\s*%`).FindStringSubmatch(t); len(m) > 1 {
		if v, ok := number(m[1]); ok {
			out = append(out, obs("gas_storage_fill_pct", "gas", "Gas storage fill", v, "%", p.Name(), u, 0.96, at))
		}
	}
	for _, x := range []struct{ label, key, name string }{{"Total domestic production", "gas_domestic_production_mw", "Domestic gas production"}, {"Total domestic withdrawal", "gas_storage_withdrawal_mw", "Gas storage withdrawal"}, {"Total domestic consumption", "gas_consumption_mw", "Domestic gas consumption"}, {"Total domestic injection", "gas_storage_injection_mw", "Gas storage injection"}} {
		// FGSZ has changed the amount of markup/text inserted between the label
		// and value several times. Allow a bounded gap but never scan across the
		// whole page, which could associate a label with an unrelated number.
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(x.label) + `.{0,180}?([0-9][0-9., ]*)\s*kWh/h`)
		m := re.FindStringSubmatch(t)
		if len(m) > 1 {
			if v, ok := number(m[1]); ok {
				out = append(out, obs(x.key, "gas", x.name, v/1000, "MW", p.Name(), u, 0.94, at))
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("FGSZ public page contained no server-rendered live gas values; falling back")
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

// EurostatGasStocks is the keyless, lower-frequency fallback for EU profiles.
// It deliberately does not pretend that monthly national closing stock is the
// same thing as a live physical storage-capacity fill percentage. Instead it
// exposes the latest stock and an index against the country's own trailing
// 36-month maximum. The scoring engine lowers confidence when this proxy is
// used.
type EurostatGasStocks struct{ C *httpx.Client }

func (p EurostatGasStocks) ID() string             { return "eurostat_gas" }
func (p EurostatGasStocks) Name() string           { return "Eurostat gas stocks" }
func (p EurostatGasStocks) Supports(in Input) bool { return in.Profile.Eurostat }

func (p EurostatGasStocks) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	q := url.Values{
		"geo":            []string{in.Profile.Code},
		"siec":           []string{"G3000"},
		"unit":           []string{"TJ_GCV"},
		"stk_flow":       []string{"STKCL_NAT"},
		"lastTimePeriod": []string{"36"},
		"lang":           []string{"en"},
	}
	u := "https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/nrg_stk_gasm?" + q.Encode()
	b, _, err := p.C.Get(ctx, u, nil, 3<<20)
	if err != nil {
		return nil, err
	}
	var d jsDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("decode Eurostat gas stocks: %w", err)
	}
	cs := euroCandidates(d)
	if len(cs) == 0 {
		return nil, fmt.Errorf("Eurostat returned no national natural-gas closing-stock values")
	}

	// Filters above normally leave one value per month. Keep the maximum when
	// a duplicate dimension remains, rather than summing categories and risking
	// double-counting an aggregate together with its components.
	byMonth := map[string]float64{}
	for _, c := range cs {
		if c.Time == "" || c.Val <= 0 {
			continue
		}
		if c.Codes["stk_flow"] != "" && c.Codes["stk_flow"] != "STKCL_NAT" {
			continue
		}
		if c.Codes["siec"] != "" && c.Codes["siec"] != "G3000" {
			continue
		}
		if c.Val > byMonth[c.Time] {
			byMonth[c.Time] = c.Val
		}
	}
	if len(byMonth) == 0 {
		return nil, fmt.Errorf("Eurostat gas-stock response contained no usable monthly values")
	}
	months := make([]string, 0, len(byMonth))
	var peak float64
	for month, v := range byMonth {
		months = append(months, month)
		if v > peak {
			peak = v
		}
	}
	sort.Strings(months)
	latestMonth := months[len(months)-1]
	latest := byMonth[latestMonth]
	if peak <= 0 || latest <= 0 {
		return nil, fmt.Errorf("Eurostat gas-stock response had non-positive latest/peak values")
	}
	index := latest / peak * 100
	if index > 105 {
		return nil, fmt.Errorf("Eurostat gas-stock index is implausible: %.1f%%", index)
	}
	if index > 100 {
		index = 100
	}
	at := time.Now().UTC()
	if t, e := time.Parse("2006-01", latestMonth); e == nil {
		at = t.UTC()
	}
	stock := obs("gas_national_stock_twh", "gas", "National natural-gas closing stock", latest/3600, "TWh", p.Name(), u, 0.70, at)
	stock.Attributes = map[string]any{"reporting_period": latestMonth, "eurostat_flow": "STKCL_NAT", "eurostat_product": "G3000"}
	stock.TTLSeconds = int64((120 * 24 * time.Hour).Seconds())
	proxy := obs("gas_stock_index_pct", "gas", "Gas stock index", index, "% of trailing 36-month maximum", p.Name(), u, 0.65, at)
	proxy.Attributes = map[string]any{
		"reporting_period":  latestMonth,
		"basis":             "latest national closing stock divided by maximum reported monthly closing stock in the returned 36-month window",
		"not_capacity_fill": true,
	}
	proxy.TTLSeconds = int64((120 * 24 * time.Hour).Seconds())
	return []model.Observation{stock, proxy}, nil
}
