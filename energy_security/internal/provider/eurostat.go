package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type EurostatOil struct{ C *httpx.Client }

func (p EurostatOil) ID() string             { return "eurostat_oil" }
func (p EurostatOil) Name() string           { return "Eurostat" }
func (p EurostatOil) Supports(in Input) bool { return in.Profile.Eurostat }

type jsCategory struct {
	Index any               `json:"index"`
	Label map[string]string `json:"label"`
}
type jsDimension struct {
	Category jsCategory `json:"category"`
}
type jsDoc struct {
	ID        []string               `json:"id"`
	Size      []int                  `json:"size"`
	Dimension map[string]jsDimension `json:"dimension"`
	Value     any                    `json:"value"`
	Label     string                 `json:"label"`
}

func catIndex(c jsCategory) map[string]int {
	out := map[string]int{}
	switch x := c.Index.(type) {
	case map[string]any:
		for k, v := range x {
			if f, ok := v.(float64); ok {
				out[k] = int(f)
			}
		}
	case []any:
		for i, v := range x {
			if s, ok := v.(string); ok {
				out[s] = i
			}
		}
	}
	return out
}

func values(v any) map[int]float64 {
	out := map[int]float64{}
	switch x := v.(type) {
	case []any:
		for i, z := range x {
			if f, ok := z.(float64); ok {
				out[i] = f
			}
		}
	case map[string]any:
		for k, z := range x {
			var i int
			if _, e := fmt.Sscanf(k, "%d", &i); e == nil {
				if f, ok := z.(float64); ok {
					out[i] = f
				}
			}
		}
	}
	return out
}

func coords(flat int, sizes []int) []int {
	out := make([]int, len(sizes))
	for i := len(sizes) - 1; i >= 0; i-- {
		if sizes[i] <= 0 {
			continue
		}
		out[i] = flat % sizes[i]
		flat /= sizes[i]
	}
	return out
}

func reverseIndex(m map[string]int) map[int]string {
	r := map[int]string{}
	for k, v := range m {
		r[v] = k
	}
	return r
}

type euroCandidate struct {
	Val     float64
	Time    string
	Codes   map[string]string
	Labels  map[string]string
	Summary string
}

func euroCandidates(d jsDoc) []euroCandidate {
	idxMaps := map[string]map[string]int{}
	rev := map[string]map[int]string{}
	for _, id := range d.ID {
		idxMaps[id] = catIndex(d.Dimension[id].Category)
		rev[id] = reverseIndex(idxMaps[id])
	}
	out := []euroCandidate{}
	for flat, v := range values(d.Value) {
		c := coords(flat, d.Size)
		codes := map[string]string{}
		labels := map[string]string{}
		parts := []string{}
		tm := ""
		for i, id := range d.ID {
			if i >= len(c) {
				continue
			}
			code := rev[id][c[i]]
			lab := d.Dimension[id].Category.Label[code]
			if lab == "" {
				lab = code
			}
			codes[id] = code
			labels[id] = lab
			parts = append(parts, strings.ToLower(id+"="+lab))
			if id == "time" {
				tm = code
			}
		}
		out = append(out, euroCandidate{Val: v, Time: tm, Codes: codes, Labels: labels, Summary: strings.Join(parts, " | ")})
	}
	return out
}

func latestFlow(candidates []euroCandidate, flow string) (euroCandidate, bool) {
	filtered := []euroCandidate{}
	for _, c := range candidates {
		if c.Codes["stk_flow"] == flow {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return euroCandidate{}, false
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Time == filtered[j].Time {
			return filtered[i].Summary < filtered[j].Summary
		}
		return filtered[i].Time > filtered[j].Time
	})
	latestTime := filtered[0].Time
	latest := []euroCandidate{}
	for _, c := range filtered {
		if c.Time != latestTime {
			break
		}
		latest = append(latest, c)
	}
	if len(latest) == 1 {
		return latest[0], true
	}
	// Some Eurostat stock tables expose product components as separate cells.
	// Prefer an explicit total when one exists; otherwise aggregate the cells
	// for the latest reporting month rather than accidentally selecting an old
	// or arbitrary component.
	for _, c := range latest {
		text := strings.ToLower(c.Summary)
		if strings.Contains(text, "=total") || strings.Contains(text, " total") || strings.Contains(text, "all products") {
			return c, true
		}
	}
	combined := latest[0]
	combined.Val = 0
	combined.Summary = "aggregated latest Eurostat components"
	for _, c := range latest {
		if c.Val > 0 {
			combined.Val += c.Val
		}
	}
	return combined, combined.Val > 0
}

func (p EurostatOil) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	q := url.Values{
		"geo":            []string{in.Profile.Code},
		"unit":           []string{"NR"},
		"stk_flow":       []string{"STK_EUE_DIR", "STK_MIN_CAL"},
		"lastTimePeriod": []string{"12"},
		"lang":           []string{"en"},
	}
	u := "https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/nrg_stk_oem?" + q.Encode()
	b, _, err := p.C.Get(ctx, u, nil, 3<<20)
	if err != nil {
		return nil, err
	}
	var d jsDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("decode Eurostat: %w", err)
	}
	if len(d.ID) == 0 {
		return nil, fmt.Errorf("Eurostat returned empty structure")
	}
	cs := euroCandidates(d)
	if len(cs) == 0 {
		return nil, fmt.Errorf("Eurostat returned no oil-stock values in the last 12 reporting periods")
	}
	actual, ok := latestFlow(cs, "STK_EUE_DIR")
	if !ok {
		return nil, fmt.Errorf("Eurostat oil dataset did not return STK_EUE_DIR in the last 12 reporting periods")
	}
	if actual.Val < 30 || actual.Val > 365 {
		return nil, fmt.Errorf("Eurostat STK_EUE_DIR returned implausible %.1f days", actual.Val)
	}
	at := time.Now().UTC()
	if actual.Time != "" {
		if t, e := time.Parse("2006-01", actual.Time); e == nil {
			at = t.UTC()
		}
	}
	o := obs("oil_emergency_stock_days", "oil", "Emergency oil stocks", actual.Val, "days equivalent", p.Name(), u, 0.88, at)
	o.Attributes = map[string]any{"series": actual.Summary, "reporting_period": actual.Time}
	o.TTLSeconds = int64((120 * 24 * time.Hour).Seconds())
	out := []model.Observation{o}
	if req, ok := latestFlow(cs, "STK_MIN_CAL"); ok && req.Val >= 30 && req.Val <= 365 {
		r := obs("oil_required_stock_days", "oil", "Minimum oil-stock obligation", req.Val, "days equivalent", p.Name(), u, 0.90, at)
		r.Attributes = map[string]any{"series": req.Summary, "reporting_period": req.Time}
		r.TTLSeconds = int64((120 * 24 * time.Hour).Seconds())
		out = append(out, r)
	}
	return out, nil
}
