package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type EnergyCharts struct{ C *httpx.Client }

func (p EnergyCharts) ID() string             { return "energy_charts" }
func (p EnergyCharts) Name() string           { return "Energy-Charts (Fraunhofer ISE)" }
func (p EnergyCharts) Supports(in Input) bool { return in.Profile.EnergyCharts != "" }

type ecSeries struct {
	Name string     `json:"name"`
	Data []*float64 `json:"data"`
}
type ecPower struct {
	Unix       []int64    `json:"unix_seconds"`
	Production []ecSeries `json:"production_types"`
	Deprecated bool       `json:"deprecated"`
}

func latest(series ecSeries, idx int) (float64, bool) {
	if idx < 0 || idx >= len(series.Data) || series.Data[idx] == nil {
		return 0, false
	}
	v := *series.Data[idx]
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func energyChartsRange(now time.Time) (string, string) {
	// Energy-Charts defaults to the current local day when no timestamps are
	// supplied. Shortly after midnight that day can legitimately have no
	// published data yet and the API returns 404. A short explicit lookback
	// makes collection independent of publication timing while keeping the
	// payload small.
	end := now.UTC().Format("2006-01-02")
	start := now.UTC().AddDate(0, 0, -2).Format("2006-01-02")
	return start, end
}

func (p EnergyCharts) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	start, end := energyChartsRange(time.Now())
	u := "https://api.energy-charts.info/public_power?" + url.Values{
		"country": []string{in.Profile.EnergyCharts},
		"start":   []string{start},
		"end":     []string{end},
	}.Encode()
	b, _, err := p.C.Get(ctx, u, nil, 8<<20)
	if err != nil {
		return nil, err
	}
	var r ecPower
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("decode Energy-Charts: %w", err)
	}
	if len(r.Unix) == 0 {
		return nil, fmt.Errorf("Energy-Charts returned no timestamps")
	}
	idx := len(r.Unix) - 1
	for idx >= 0 {
		ok := false
		for _, s := range r.Production {
			if _, yes := latest(s, idx); yes {
				ok = true
				break
			}
		}
		if ok {
			break
		}
		idx--
	}
	if idx < 0 {
		return nil, fmt.Errorf("Energy-Charts returned no usable values")
	}
	at := time.Unix(r.Unix[idx], 0).UTC()
	out := []model.Observation{}
	var generation, nuclear, solar, wind, hydro, gas, coal, oil, biomass float64
	var load, renShare, cross float64
	var hasLoad, hasRen, hasCross bool
	mix := map[string]float64{}
	for _, s := range r.Production {
		v, ok := latest(s, idx)
		if !ok {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(s.Name))
		switch {
		case name == "load":
			load = v
			hasLoad = true
		case strings.Contains(name, "renewable") && strings.Contains(name, "share"):
			renShare = v
			hasRen = true
		case strings.Contains(name, "cross border"):
			cross = v
			hasCross = true
		case strings.Contains(name, "residual load"):
		default:
			if v > 0 {
				generation += v
				mix[s.Name] = v
			}
			if strings.Contains(name, "nuclear") {
				nuclear += v
			}
			if strings.Contains(name, "solar") {
				solar += v
			}
			if strings.Contains(name, "wind") {
				wind += v
			}
			if strings.Contains(name, "hydro") || strings.Contains(name, "water reservoir") || strings.Contains(name, "run-of-river") {
				hydro += v
			}
			if strings.Contains(name, "gas") {
				gas += v
			}
			if strings.Contains(name, "coal") || strings.Contains(name, "lignite") {
				coal += v
			}
			if strings.Contains(name, "oil") {
				oil += v
			}
			if strings.Contains(name, "biomass") {
				biomass += v
			}
		}
	}
	q := 0.92
	if r.Deprecated {
		q = 0.75
	}
	if generation > 0 {
		o := obs("electricity_generation_mw", "electricity", "Electricity generation", generation, "MW", p.Name(), u, q, at)
		o.Attributes = map[string]any{"mix_mw": mix}
		out = append(out, o)
	}
	if hasLoad && load > 0 {
		out = append(out, obs("electricity_load_mw", "electricity", "Electricity load", load, "MW", p.Name(), u, q, at))
	}
	if hasCross {
		out = append(out, obs("electricity_cross_border_mw", "electricity", "Cross-border electricity trading", cross, "MW", p.Name(), u, q*0.95, at))
	}
	if hasRen {
		if renShare > 0 && renShare <= 1.5 {
			renShare *= 100
		}
		out = append(out, obs("renewable_share_pct", "renewables", "Renewable share of load", renShare, "%", p.Name(), u, q, at))
	}
	if nuclear > 0 {
		out = append(out, obs("nuclear_output_mw", "nuclear", "Nuclear output", nuclear, "MW", p.Name(), u, q, at))
	}
	if solar > 0 {
		out = append(out, obs("solar_output_mw", "renewables", "Solar output", solar, "MW", p.Name(), u, q, at))
	}
	if wind > 0 {
		out = append(out, obs("wind_output_mw", "renewables", "Wind output", wind, "MW", p.Name(), u, q, at))
	}
	if hydro > 0 {
		out = append(out, obs("hydro_output_mw", "water", "Hydro output", hydro, "MW", p.Name(), u, q, at))
	}
	if gas > 0 {
		out = append(out, obs("gas_power_output_mw", "electricity", "Gas-fired output", gas, "MW", p.Name(), u, q, at))
	}
	if coal > 0 {
		out = append(out, obs("coal_power_output_mw", "electricity", "Coal/lignite output", coal, "MW", p.Name(), u, q, at))
	}
	if oil > 0 {
		out = append(out, obs("oil_power_output_mw", "electricity", "Oil-fired output", oil, "MW", p.Name(), u, q, at))
	}
	if biomass > 0 {
		out = append(out, obs("biomass_output_mw", "renewables", "Biomass output", biomass, "MW", p.Name(), u, q, at))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("Energy-Charts response had no recognised measurements")
	}
	return out, nil
}
