package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/baseline"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func attrSeconds(o model.Observation, key string) time.Duration {
	if o.Attributes == nil {
		return 0
	}
	v, ok := o.Attributes[key]
	if !ok {
		return 0
	}
	var seconds float64
	switch x := v.(type) {
	case float64:
		seconds = x
	case float32:
		seconds = float64(x)
	case int:
		seconds = float64(x)
	case int64:
		seconds = float64(x)
	case int32:
		seconds = float64(x)
	case uint:
		seconds = float64(x)
	case uint64:
		seconds = float64(x)
	case uint32:
		seconds = float64(x)
	default:
		return 0
	}
	if seconds <= 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func observationUsable(o model.Observation) bool {
	if o.Stale || o.ObservedAt.IsZero() {
		return false
	}
	if o.TTLSeconds > 0 && time.Now().UTC().After(o.ObservedAt.Add(time.Duration(o.TTLSeconds)*time.Second)) {
		return false
	}
	return true
}

func freshnessFactor(o model.Observation) float64 {
	if !observationUsable(o) {
		return 0
	}
	freshFor := attrSeconds(o, "fresh_for_seconds")
	if freshFor <= 0 {
		return 1
	}
	hard := time.Duration(o.TTLSeconds) * time.Second
	if hard <= freshFor {
		return 1
	}
	age := time.Since(o.ObservedAt)
	if age <= freshFor {
		return 1
	}
	if age >= hard {
		return 0
	}
	progress := float64(age-freshFor) / float64(hard-freshFor)
	// A delayed-but-still-usable observation fades from full source quality to
	// half quality at the hard expiry boundary. It then becomes unusable. This
	// avoids arbitrary score disappearance at normal publication lag while still
	// making older evidence visibly less authoritative through confidence.
	return clamp(1-0.5*progress, 0.5, 1)
}

func val(obs map[string]model.Observation, k string) (float64, bool) {
	o, ok := obs[k]
	if !ok || o.Value == nil || !observationUsable(o) {
		return 0, false
	}
	return *o.Value, true
}

func quality(obs map[string]model.Observation, keys ...string) float64 {
	var s float64
	var n int
	for _, k := range keys {
		if o, ok := obs[k]; ok && observationUsable(o) {
			s += o.Quality * freshnessFactor(o)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return s / float64(n)
}

func status(score *float64, conf float64) string {
	if score == nil {
		return "Unknown"
	}
	if conf < 40 {
		return "Data limited"
	}
	s := *score
	switch {
	case s >= 85:
		return "Very good"
	case s >= 70:
		return "Good"
	case s >= 55:
		return "Watch"
	case s >= 40:
		return "Stressed"
	default:
		return "Critical"
	}
}

func weighted(items []struct{ s, c, w float64 }, expectedWeight float64) (*float64, float64) {
	var sw, ss, cw float64
	for _, x := range items {
		if x.w <= 0 || x.c <= 0 {
			continue
		}
		sw += x.w
		ss += x.s * x.w
		cw += x.c * x.w
	}
	if sw == 0 {
		return nil, 0
	}
	v := ss / sw
	coverage := 1.0
	if expectedWeight > 0 {
		coverage = clamp(sw/expectedWeight, 0, 1)
	}
	return &v, clamp((cw/sw)*coverage, 0, 100)
}

func electricity(obs map[string]model.Observation, countryCode string) (model.DomainScore, *float64, float64, bool) {
	gen, gok := val(obs, "electricity_generation_mw")
	load, lok := val(obs, "electricity_load_mw")
	cross, cok := val(obs, "electricity_cross_border_mw")
	q := quality(obs, "electricity_generation_mw", "electricity_load_mw") * 100
	estimatedLoad := false
	var ref baseline.ElectricityReference

	if gok && (!lok || load <= 0) {
		if r, ok := baseline.Electricity(countryCode); ok {
			if average := r.AverageLoadMW(); average > 0 {
				ref = r
				load = average
				lok = true
				estimatedLoad = true
				q = quality(obs, "electricity_generation_mw") * 100 * 0.45
			}
		}
	}

	var cur *float64
	summary := "No usable electricity generation measurement is available."
	if gok {
		summary = "Generation data available; live load is unavailable and no embedded country reference is available."
	}
	if gok && lok && load > 0 {
		imports := 0.0
		if cok && cross < 0 {
			imports = -cross
		}
		coverage := (gen + imports) / load
		cs := clamp(20+(coverage-0.70)*200, 20, 96)
		cur = &cs
		if estimatedLoad {
			summary = fmt.Sprintf(
				"Generation %.0f MW against derived reference load %.0f MW. Live consumption is unavailable; the load estimate uses the embedded %d Ember electricity-demand reference (%.2f TWh/year, population about %.2f million, %.2f MWh/person/year).",
				gen, load, ref.Year, ref.DemandTWh, ref.PopulationMillions, ref.DemandMWhPerCapita,
			)
		} else {
			summary = fmt.Sprintf("Generation %.0f MW against load %.0f MW", gen, load)
		}
		if cok {
			if cross < 0 {
				summary += fmt.Sprintf(" Net trading implies %.0f MW imports.", -cross)
			} else {
				summary += fmt.Sprintf(" Net trading implies %.0f MW exports.", cross)
			}
		} else if !strings.HasSuffix(summary, ".") {
			summary += "."
		}
		if o, ok := obs["electricity_generation_mw"]; ok && freshnessFactor(o) < 0.999 && observationUsable(o) {
			summary += " The newest source sample is beyond the preferred freshness window, so confidence is reduced rather than dropping the electricity score entirely."
		}
	}

	var strategic *float64
	if o, ok := obs["electricity_generation_mw"]; ok && o.Attributes != nil {
		if raw, ok := o.Attributes["mix_mw"].(map[string]float64); ok {
			var total float64
			for _, v := range raw {
				if v > 0 {
					total += v
				}
			}
			if total > 0 {
				var hhi float64
				var n int
				for _, v := range raw {
					if v <= 0 {
						continue
					}
					p := v / total
					hhi += p * p
					n++
				}
				ds := 60.0
				if n > 1 {
					ds = clamp((1-hhi)/(1-1/float64(n))*100, 0, 100)
				}
				strategic = &ds
			}
		} else if raw, ok := o.Attributes["mix_mw"].(map[string]any); ok {
			var total float64
			vals := []float64{}
			for _, z := range raw {
				if v, ok := z.(float64); ok && v > 0 {
					vals = append(vals, v)
					total += v
				}
			}
			if total > 0 && len(vals) > 1 {
				var hhi float64
				for _, v := range vals {
					p := v / total
					hhi += p * p
				}
				ds := clamp((1-hhi)/(1-1/float64(len(vals)))*100, 0, 100)
				strategic = &ds
			}
		}
	}
	if cur == nil && strategic != nil {
		q *= 0.65
	}
	domainStatus := status(cur, q)
	if estimatedLoad && cur != nil {
		domainStatus = "Data limited"
	}
	return model.DomainScore{Score: cur, Status: domainStatus, Confidence: q, Summary: summary}, strategic, q, estimatedLoad
}

func gas(obs map[string]model.Observation, month time.Month) model.DomainScore {
	fill, live := val(obs, "gas_storage_fill_pct")
	q := quality(obs, "gas_storage_fill_pct") * 100
	proxy := false
	if !live {
		var ok bool
		fill, ok = val(obs, "gas_stock_index_pct")
		if !ok {
			return model.DomainScore{Status: "Unknown", Confidence: 0, Summary: "No usable gas-storage measurement or defensible stock proxy."}
		}
		q = quality(obs, "gas_stock_index_pct") * 100
		proxy = true
	}
	if proxy {
		q *= 0.55
		s := clamp(fill, 20, 95)
		summary := fmt.Sprintf("Eurostat monthly national stock is %.1f%% of the trailing 36-month maximum. This is a low-confidence strategic stock proxy, not physical storage-capacity fill.", fill)
		return model.DomainScore{Score: &s, Status: status(&s, q), Confidence: q, Summary: summary}
	}
	targets := map[time.Month]float64{
		time.January: 55, time.February: 40, time.March: 30, time.April: 28,
		time.May: 38, time.June: 50, time.July: 62, time.August: 72,
		time.September: 82, time.October: 90, time.November: 82, time.December: 68,
	}
	target := targets[month]
	s := clamp(75+(fill-target)*1.25, 20, 98)
	if fill < 25 {
		s = clamp(s, 0, 35)
	}
	summary := fmt.Sprintf("Storage %.1f%%; seasonal reference %.0f%%.", fill, target)
	if o, ok := obs["gas_storage_fill_pct"]; ok && freshnessFactor(o) < 0.999 && observationUsable(o) {
		summary += " The latest storage report is older than its preferred freshness window, so confidence is reduced while the slower-moving storage level remains usable."
	}
	return model.DomainScore{Score: &s, Status: status(&s, q), Confidence: q, Summary: summary}
}

func oil(obs map[string]model.Observation) model.DomainScore {
	days, ok := val(obs, "oil_emergency_stock_days")
	q := quality(obs, "oil_emergency_stock_days") * 100
	if !ok {
		return model.DomainScore{Status: "Unknown", Summary: "No current emergency oil-stock measure."}
	}
	if days < 30 || days > 365 {
		return model.DomainScore{Status: "Data limited", Confidence: q * 0.4, Summary: fmt.Sprintf("Oil dataset returned %.1f days-equivalent, but the series could not be safely interpreted as national cover.", days)}
	}
	s := clamp(35+(days-45)*1.1, 20, 98)
	return model.DomainScore{Score: &s, Status: status(&s, q), Confidence: q, Summary: fmt.Sprintf("Emergency stocks %.1f days equivalent.", days)}
}

func water(obs map[string]model.Observation) model.DomainScore {
	s, ok := val(obs, "water_security_proxy")
	q := quality(obs, "water_security_proxy") * 100
	if !ok {
		return model.DomainScore{Status: "Unknown", Summary: "No national hydrological security indicator available."}
	}
	txt := obs["water_security_proxy"].Text
	if txt == "" {
		txt = "hydrological conditions"
	}
	return model.DomainScore{Score: &s, Status: status(&s, q), Confidence: q, Summary: "Current hydrological assessment: " + txt + "."}
}

func weather(obs map[string]model.Observation) model.DomainScore {
	mx, mxok := val(obs, "forecast_max_temperature_c")
	mn, mnok := val(obs, "forecast_min_temperature_c")
	wind, wok := val(obs, "forecast_max_wind_kmh")
	prec, pok := val(obs, "forecast_precipitation_7d_mm")
	if !mxok && !mnok && !wok {
		return model.DomainScore{Status: "Unknown", Summary: "No seven-day weather stress forecast."}
	}
	s := 92.0
	reasons := []string{}
	if mxok {
		if mx >= 40 {
			s = 45
			reasons = append(reasons, "extreme heat")
		} else if mx >= 35 {
			s = math.Min(s, 65)
			reasons = append(reasons, "heat stress")
		}
	}
	if mnok {
		if mn <= -20 {
			s = math.Min(s, 45)
			reasons = append(reasons, "extreme cold")
		} else if mn <= -12 {
			s = math.Min(s, 65)
			reasons = append(reasons, "cold stress")
		}
	}
	if wok {
		if wind >= 110 {
			s = math.Min(s, 45)
			reasons = append(reasons, "severe wind")
		} else if wind >= 80 {
			s = math.Min(s, 65)
			reasons = append(reasons, "strong wind")
		}
	}
	if pok && prec >= 100 {
		s = math.Min(s, 65)
		reasons = append(reasons, "very heavy precipitation")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "no major weather stress signal")
	}
	q := quality(obs, "forecast_max_temperature_c", "forecast_min_temperature_c", "forecast_max_wind_kmh") * 100
	return model.DomainScore{Score: &s, Status: status(&s, q), Confidence: q, Summary: "7-day outlook: " + strings.Join(reasons, ", ") + "."}
}

func Score(s *model.Snapshot) {
	if s.Domains == nil {
		s.Domains = map[string]model.DomainScore{}
	}
	e, estrat, _, electricityEstimated := electricity(s.Observations, s.Country)
	g := gas(s.Observations, s.UpdatedAt.Month())
	o := oil(s.Observations)
	w := water(s.Observations)
	wx := weather(s.Observations)
	s.Domains["electricity"] = e
	s.Domains["gas"] = g
	s.Domains["oil"] = o
	s.Domains["water"] = w
	s.Domains["weather"] = wx

	if n, ok := val(s.Observations, "nuclear_output_mw"); ok {
		s.Domains["nuclear"] = model.DomainScore{Status: "Observed", Confidence: quality(s.Observations, "nuclear_output_mw") * 100, Summary: fmt.Sprintf("Current nuclear output %.0f MW.", n)}
	} else {
		s.Domains["nuclear"] = model.DomainScore{Status: "Unknown", Summary: "No nuclear output reported for this country/source."}
	}
	if r, ok := val(s.Observations, "renewable_share_pct"); ok {
		s.Domains["renewables"] = model.DomainScore{Status: "Observed", Confidence: quality(s.Observations, "renewable_share_pct") * 100, Summary: fmt.Sprintf("Renewables currently cover %.1f%% of load.", r)}
	} else {
		s.Domains["renewables"] = model.DomainScore{Status: "Unknown", Summary: "Renewable share unavailable."}
	}

	items := []struct{ s, c, w float64 }{}
	add := func(d model.DomainScore, wgt float64) {
		if d.Score != nil {
			items = append(items, struct{ s, c, w float64 }{*d.Score, d.Confidence, wgt})
		}
	}
	electricityWeight := 0.50
	if electricityEstimated {
		electricityWeight = 0.25
	}
	add(e, electricityWeight)
	if g.Confidence >= 40 {
		add(g, 0.30)
	}
	add(w, 0.10)
	add(wx, 0.10)
	cur, cc := weighted(items, 1.0)

	outItems := []struct{ s, c, w float64 }{}
	if cur != nil {
		outItems = append(outItems, struct{ s, c, w float64 }{*cur, cc, 0.7})
	}
	if wx.Score != nil {
		outItems = append(outItems, struct{ s, c, w float64 }{*wx.Score, wx.Confidence, 0.3})
	}
	out, oc := weighted(outItems, 1.0)

	stItems := []struct{ s, c, w float64 }{}
	if estrat != nil {
		stItems = append(stItems, struct{ s, c, w float64 }{*estrat, e.Confidence, 0.45})
	}
	if g.Score != nil {
		stItems = append(stItems, struct{ s, c, w float64 }{*g.Score, g.Confidence, 0.30})
	}
	if o.Score != nil {
		stItems = append(stItems, struct{ s, c, w float64 }{*o.Score, o.Confidence, 0.15})
	}
	if w.Score != nil {
		stItems = append(stItems, struct{ s, c, w float64 }{*w.Score, w.Confidence, 0.10})
	}
	st, sc := weighted(stItems, 1.0)

	hItems := []struct{ s, c, w float64 }{}
	if cur != nil {
		hItems = append(hItems, struct{ s, c, w float64 }{*cur, cc, 0.55})
	}
	if out != nil {
		hItems = append(hItems, struct{ s, c, w float64 }{*out, oc, 0.25})
	}
	if st != nil {
		hItems = append(hItems, struct{ s, c, w float64 }{*st, sc, 0.20})
	}
	head, hc := weighted(hItems, 1.0)
	s.Scores = model.Scores{Current: cur, Outlook7D: out, Strategic: st, Headline: head, Confidence: hc, Status: status(head, hc)}

	alerts := []string{}
	if e.Score != nil && *e.Score < 55 && !electricityEstimated {
		alerts = append(alerts, "Electricity supply indicators are under stress.")
	}
	if g.Score != nil && *g.Score < 50 {
		alerts = append(alerts, "Gas storage or stock indicators are weak for the season.")
	}
	if w.Score != nil && *w.Score < 50 {
		alerts = append(alerts, "Hydrological conditions may constrain energy infrastructure.")
	}
	if wx.Score != nil && *wx.Score < 60 {
		alerts = append(alerts, "Severe weather may stress the energy system during the next seven days.")
	}
	sort.Strings(alerts)
	s.Alerts = alerts
}
