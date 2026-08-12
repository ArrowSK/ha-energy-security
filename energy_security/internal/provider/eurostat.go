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
func (p EurostatOil) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	q := url.Values{"geo": []string{in.Profile.Code}, "unit": []string{"NR"}, "stk_flow": []string{"STK_EUE_DIR", "STK_MIN_CAL"}, "lastTimePeriod": []string{"1"}, "lang": []string{"en"}}
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
	idxMaps := map[string]map[string]int{}
	rev := map[string]map[int]string{}
	for _, id := range d.ID {
		idxMaps[id] = catIndex(d.Dimension[id].Category)
		rev[id] = reverseIndex(idxMaps[id])
	}
	vals := values(d.Value)
	if len(vals) == 0 {
		return nil, fmt.Errorf("Eurostat returned no oil-stock values")
	}
	type cand struct {
		val     float64
		labels  string
		time    string
		stkFlow string
	}
	cs := []cand{}
	for flat, v := range vals {
		c := coords(flat, d.Size)
		parts := []string{}
		tm := ""
		stkFlow := ""
		for i, id := range d.ID {
			code := rev[id][c[i]]
			lab := d.Dimension[id].Category.Label[code]
			if lab == "" {
				lab = code
			}
			parts = append(parts, strings.ToLower(id+"="+lab))
			if id == "time" {
				tm = code
			}
			if id == "stk_flow" {
				stkFlow = code
			}
		}
		cs = append(cs, cand{val: v, labels: strings.Join(parts, " | "), time: tm, stkFlow: stkFlow})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].val > cs[j].val })
	var actual *cand
	var req *cand
	for i := range cs {
		switch cs[i].stkFlow {
		case "STK_EUE_DIR":
			if actual == nil {
				actual = &cs[i]
			}
		case "STK_MIN_CAL":
			if req == nil {
				req = &cs[i]
			}
		}
	}
	if actual == nil {
		return nil, fmt.Errorf("Eurostat oil dataset did not return STK_EUE_DIR")
	}
	if actual.val < 30 || actual.val > 365 {
		return nil, fmt.Errorf("Eurostat STK_EUE_DIR returned implausible %.1f days", actual.val)
	}
	at := time.Now().UTC()
	if actual.time != "" {
		if t, e := time.Parse("2006-01", actual.time); e == nil {
			at = t.UTC()
		}
	}
	o := obs("oil_emergency_stock_days", "oil", "Emergency oil stocks", actual.val, "days equivalent", p.Name(), u, 0.88, at)
	o.Attributes = map[string]any{"series": actual.labels}
	out := []model.Observation{o}
	if req != nil {
		r := obs("oil_required_stock_days", "oil", "Minimum oil-stock obligation", req.val, "days equivalent", p.Name(), u, 0.90, at)
		r.Attributes = map[string]any{"series": req.labels}
		out = append(out, r)
	}
	return out, nil
}
