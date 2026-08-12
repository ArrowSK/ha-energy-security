package provider

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type ENTSOE struct{ C *httpx.Client }

func (p ENTSOE) ID() string   { return "entsoe" }
func (p ENTSOE) Name() string { return "ENTSO-E Transparency Platform" }
func (p ENTSOE) Supports(in Input) bool {
	return strings.TrimSpace(in.Config.ENTSOEToken) != "" && in.Profile.ENTSOEEIC != ""
}

type entsoeDoc struct {
	TimeSeries []entsoeSeries `xml:"TimeSeries"`
}
type entsoeSeries struct {
	PSR struct {
		Type string `xml:"psrType"`
	} `xml:"MktPSRType"`
	Periods []entsoePeriod `xml:"Period"`
}
type entsoePeriod struct {
	Interval struct {
		Start string `xml:"start"`
	} `xml:"timeInterval"`
	Resolution string `xml:"resolution"`
	Points     []struct {
		Position int     `xml:"position"`
		Quantity float64 `xml:"quantity"`
	} `xml:"Point"`
}

func isoStep(s string) time.Duration {
	switch s {
	case "PT15M":
		return 15 * time.Minute
	case "PT30M":
		return 30 * time.Minute
	case "PT60M", "PT1H":
		return time.Hour
	}
	return time.Hour
}
func parseENTSOETime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04Z", strings.TrimSpace(s))
}
func psrLabel(code string) string {
	m := map[string]string{"B01": "Biomass", "B02": "Lignite", "B03": "Coal-derived gas", "B04": "Fossil gas", "B05": "Hard coal", "B06": "Fossil oil", "B09": "Geothermal", "B10": "Hydro pumped storage", "B11": "Hydro run-of-river", "B12": "Hydro reservoir", "B14": "Nuclear", "B15": "Other renewable", "B16": "Solar", "B18": "Wind offshore", "B19": "Wind onshore", "B20": "Other"}
	if v := m[code]; v != "" {
		return v
	}
	return code
}

func (p ENTSOE) fetch(ctx context.Context, in Input, docType string) (entsoeDoc, string, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	q := url.Values{"securityToken": []string{in.Config.ENTSOEToken}, "documentType": []string{docType}, "processType": []string{"A16"}, "periodStart": []string{start.Format("200601021504")}, "periodEnd": []string{end.Format("200601021504")}}
	if docType == "A75" {
		q.Set("in_Domain", in.Profile.ENTSOEEIC)
	} else {
		q.Set("outBiddingZone_Domain", in.Profile.ENTSOEEIC)
	}
	u := "https://web-api.tp.entsoe.eu/api?" + q.Encode()
	const publicURL = "https://transparency.entsoe.eu/"
	b, _, err := p.C.Get(ctx, u, nil, 8<<20)
	if err != nil {
		return entsoeDoc{}, publicURL, err
	}
	if strings.Contains(string(b), "Acknowledgement_MarketDocument") {
		return entsoeDoc{}, publicURL, fmt.Errorf("ENTSO-E returned acknowledgement/error document")
	}
	var d entsoeDoc
	if err := xml.Unmarshal(b, &d); err != nil {
		return d, publicURL, fmt.Errorf("decode ENTSO-E XML: %w", err)
	}
	if len(d.TimeSeries) == 0 {
		return d, publicURL, fmt.Errorf("ENTSO-E returned no time series")
	}
	return d, publicURL, nil
}

type pointBucket map[string]float64

func buckets(d entsoeDoc) map[time.Time]pointBucket {
	out := map[time.Time]pointBucket{}
	for _, s := range d.TimeSeries {
		label := psrLabel(s.PSR.Type)
		for _, p := range s.Periods {
			st, err := parseENTSOETime(p.Interval.Start)
			if err != nil {
				continue
			}
			step := isoStep(p.Resolution)
			for _, pt := range p.Points {
				if pt.Position <= 0 {
					continue
				}
				t := st.Add(time.Duration(pt.Position-1) * step)
				if out[t] == nil {
					out[t] = pointBucket{}
				}
				out[t][label] += pt.Quantity
			}
		}
	}
	return out
}
func newestBucket(m map[time.Time]pointBucket) (time.Time, pointBucket, bool) {
	if len(m) == 0 {
		return time.Time{}, nil, false
	}
	times := make([]time.Time, 0, len(m))
	for t := range m {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	now := time.Now().UTC().Add(30 * time.Minute)
	for i := len(times) - 1; i >= 0; i-- {
		if !times[i].After(now) {
			return times[i], m[times[i]], true
		}
	}
	t := times[len(times)-1]
	return t, m[t], true
}

func (p ENTSOE) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	genDoc, genURL, err := p.fetch(ctx, in, "A75")
	if err != nil {
		return nil, err
	}
	loadDoc, loadURL, loadErr := p.fetch(ctx, in, "A65")
	at, mix, ok := newestBucket(buckets(genDoc))
	if !ok {
		return nil, fmt.Errorf("no usable ENTSO-E generation points")
	}
	var total, nuclear, solar, wind, hydro, gas, coal, oil, biomass float64
	for n, v := range mix {
		if v > 0 {
			total += v
		}
		l := strings.ToLower(n)
		if strings.Contains(l, "nuclear") {
			nuclear += v
		}
		if strings.Contains(l, "solar") {
			solar += v
		}
		if strings.Contains(l, "wind") {
			wind += v
		}
		if strings.Contains(l, "hydro") {
			hydro += v
		}
		if strings.Contains(l, "gas") {
			gas += v
		}
		if strings.Contains(l, "coal") || strings.Contains(l, "lignite") {
			coal += v
		}
		if strings.Contains(l, "oil") {
			oil += v
		}
		if strings.Contains(l, "biomass") {
			biomass += v
		}
	}
	out := []model.Observation{}
	q := 0.98
	if total > 0 {
		o := obs("electricity_generation_mw", "electricity", "Electricity generation", total, "MW", p.Name(), genURL, q, at)
		o.Attributes = map[string]any{"mix_mw": mix}
		out = append(out, o)
	}
	if loadErr == nil {
		lat, lmix, lok := newestBucket(buckets(loadDoc))
		if lok {
			var load float64
			for _, v := range lmix {
				load += v
			}
			if load > 0 {
				out = append(out, obs("electricity_load_mw", "electricity", "Electricity load", load, "MW", p.Name(), loadURL, q, lat))
			}
		}
	}
	for _, x := range []struct {
		k, d, l string
		v       float64
	}{{"nuclear_output_mw", "nuclear", "Nuclear output", nuclear}, {"solar_output_mw", "renewables", "Solar output", solar}, {"wind_output_mw", "renewables", "Wind output", wind}, {"hydro_output_mw", "water", "Hydro output", hydro}, {"gas_power_output_mw", "electricity", "Gas-fired output", gas}, {"coal_power_output_mw", "electricity", "Coal/lignite output", coal}, {"oil_power_output_mw", "electricity", "Oil-fired output", oil}, {"biomass_output_mw", "renewables", "Biomass output", biomass}} {
		if x.v > 0 {
			out = append(out, obs(x.k, x.d, x.l, x.v, "MW", p.Name(), genURL, q, at))
		}
	}
	if loadErr != nil {
		for i := range out {
			if out[i].Attributes == nil {
				out[i].Attributes = map[string]any{}
			}
			out[i].Attributes["load_fallback_error"] = loadErr.Error()
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ENTSO-E data contained no recognised measurements")
	}
	return out, nil
}

var _ = strconv.Itoa
