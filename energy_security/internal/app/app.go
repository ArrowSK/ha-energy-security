package app

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/cache"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/engine"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/ha"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/provider"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/topology"
)

type App struct {
	cfg             config.Config
	cache           cache.Store
	ha              *ha.Client
	pm              *provider.Manager
	topologyLearner *topology.Learner
	mu              sync.RWMutex
	refreshMu       sync.Mutex
	snapshot        model.Snapshot
	groups          []provider.Group
	locationReady   bool
}

func New(cfg config.Config, dataDir string) *App {
	hc := httpx.New()
	a := &App{cfg: cfg, cache: cache.Store{Path: dataDir + "/state.json"}, ha: ha.New(), pm: provider.NewManager()}
	a.groups = []provider.Group{
		{ID: "electricity", TTL: 90 * time.Minute, Providers: []provider.Provider{provider.EnergyCharts{C: hc}, provider.ENTSOE{C: hc}}},
		{ID: "gas", TTL: 36 * time.Hour, Providers: []provider.Provider{provider.HungaryFGSZ{C: hc}, provider.AGSI{C: hc}, provider.EurostatGasStocks{C: hc}}},
		{ID: "oil", TTL: 45 * 24 * time.Hour, Providers: []provider.Provider{provider.EurostatOil{C: hc}}},
		{ID: "water", TTL: 36 * time.Hour, Providers: []provider.Provider{provider.HungaryHydroinfo{C: hc}}},
		{ID: "weather", TTL: 6 * time.Hour, Providers: []provider.Provider{provider.OpenMeteo{C: hc}}},
	}
	if cfg.EnableTopologyLearner && !strings.EqualFold(cfg.RuntimeMode, "standalone") {
		a.topologyLearner = topology.New(a.ha, dataDir+"/topology.json", cfg.EnableHAEntities)
	}
	if old, err := a.cache.Load(); err == nil {
		a.snapshot = old
		markStale(&a.snapshot, time.Now().UTC())
	}
	return a
}

func (a *App) Snapshot() model.Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneSnapshot(a.snapshot)
}

func (a *App) TopologySnapshot() topology.Snapshot {
	if a.topologyLearner == nil {
		return topology.Snapshot{Enabled: false, Status: "disabled", RetentionDays: topology.RetentionDays}
	}
	return a.topologyLearner.Snapshot()
}

func cloneSnapshot(s model.Snapshot) model.Snapshot {
	out := s
	out.Domains = map[string]model.DomainScore{}
	for k, v := range s.Domains {
		out.Domains[k] = v
	}
	out.Observations = map[string]model.Observation{}
	for k, v := range s.Observations {
		out.Observations[k] = v
	}
	out.Providers = map[string]model.ProviderHealth{}
	for k, v := range s.Providers {
		out.Providers[k] = v
	}
	out.Alerts = append([]string(nil), s.Alerts...)
	out.Notes = append([]string(nil), s.Notes...)
	out.History = append([]model.HistoryPoint(nil), s.History...)
	return out
}

func markStale(s *model.Snapshot, now time.Time) {
	for k, o := range s.Observations {
		ttl := time.Duration(o.TTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 2 * time.Hour
		}
		if o.ObservedAt.IsZero() {
			o.Stale = true
		} else {
			o.Stale = now.After(o.ObservedAt.Add(ttl))
		}
		s.Observations[k] = o
	}
}

func (a *App) resolve(ctx context.Context) (country.Profile, float64, float64, string, string, bool, error) {
	if strings.EqualFold(a.cfg.RuntimeMode, "standalone") {
		p := country.Resolve(a.cfg.Country)
		hasLoc := a.cfg.Latitude != nil && a.cfg.Longitude != nil
		if hasLoc {
			return p, *a.cfg.Latitude, *a.cfg.Longitude, a.cfg.TimeZone, a.cfg.LocationName, true, nil
		}
		return p, 0, 0, a.cfg.TimeZone, a.cfg.LocationName, false, nil
	}

	hcfg, err := a.ha.Config(ctx)
	if err != nil {
		if strings.EqualFold(a.cfg.Country, "auto") {
			return country.Profile{}, 0, 0, "", "", false, fmt.Errorf("read Home Assistant location: %w", err)
		}
		p := country.Resolve(a.cfg.Country)
		return p, 0, 0, "", "", false, nil
	}
	code := hcfg.Country
	if !strings.EqualFold(a.cfg.Country, "auto") {
		code = a.cfg.Country
	}
	p := country.Resolve(code)
	return p, hcfg.Latitude, hcfg.Longitude, hcfg.TimeZone, hcfg.LocationName, true, nil
}

func (a *App) Refresh(ctx context.Context) error {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	p, lat, lon, tz, loc, hasLoc, err := a.resolve(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	a.mu.RLock()
	prev := cloneSnapshot(a.snapshot)
	a.mu.RUnlock()
	if prev.Country != "" && prev.Country != p.Code {
		prev = model.Snapshot{}
	}
	s := prev
	s.Version = 1
	s.Country = p.Code
	s.CountryName = p.Name
	s.LocationName = loc
	s.Latitude = lat
	s.Longitude = lon
	s.Timezone = tz
	s.UpdatedAt = now
	if s.Observations == nil {
		s.Observations = map[string]model.Observation{}
	}
	in := provider.Input{Profile: p, Latitude: lat, Longitude: lon, HasLocation: hasLoc, Config: a.cfg}
	notes := []string{}
	for _, g := range a.groups {
		gctx, cancel := context.WithTimeout(ctx, 35*time.Second)
		obs, used, e := a.pm.CollectGroup(gctx, g, in)
		cancel()
		if e != nil {
			notes = append(notes, fmt.Sprintf("%s: using cached data where available (%v)", g.ID, e))
			continue
		}
		for k, o := range s.Observations {
			if o.Attributes != nil {
				if pg, ok := o.Attributes["provider_group"].(string); ok && pg == g.ID {
					delete(s.Observations, k)
				}
			}
		}
		for _, o := range obs {
			if o.Attributes == nil {
				o.Attributes = map[string]any{}
			}
			o.Attributes["provider_group"] = g.ID
			o.Attributes["provider_id"] = used
			s.Observations[o.Key] = o
		}
	}
	markStale(&s, now)
	s.Providers = a.pm.Health()
	s.Notes = notes
	engine.Score(&s)
	appendHistory(&s)
	if err := a.cache.Save(s); err != nil {
		notes = append(notes, "cache write failed: "+err.Error())
		s.Notes = notes
	}
	a.mu.Lock()
	a.snapshot = s
	a.locationReady = hasLoc
	a.mu.Unlock()
	if a.cfg.EnableHAEntities {
		if err := a.publishHA(ctx, s); err != nil {
			log.Printf("home assistant entity publish: %v", err)
		}
	}
	return nil
}

func appendHistory(s *model.Snapshot) {
	p := model.HistoryPoint{Time: s.UpdatedAt, Headline: s.Scores.Headline, Confidence: s.Scores.Confidence}
	if len(s.History) > 0 {
		last := s.History[len(s.History)-1]
		if p.Time.Sub(last.Time) < 5*time.Minute {
			return
		}
	}
	s.History = append(s.History, p)
	cut := time.Now().UTC().Add(-7 * 24 * time.Hour)
	i := 0
	for i < len(s.History) && s.History[i].Time.Before(cut) {
		i++
	}
	if i > 0 {
		s.History = append([]model.HistoryPoint(nil), s.History[i:]...)
	}
	if len(s.History) > 700 {
		s.History = s.History[len(s.History)-700:]
	}
}

func stateNumber(v *float64) string {
	if v == nil {
		return "unknown"
	}
	if math.IsNaN(*v) || math.IsInf(*v, 0) {
		return "unknown"
	}
	return strconv.FormatFloat(*v, 'f', 1, 64)
}
func (a *App) publishHA(ctx context.Context, s model.Snapshot) error {
	base := map[string]any{"country": s.Country, "country_name": s.CountryName, "updated_at": s.UpdatedAt.Format(time.RFC3339), "confidence": math.Round(s.Scores.Confidence*10) / 10, "attribution": "Energy Security Monitor; source details are available in the app dashboard", "icon": "mdi:shield-home"}
	if err := a.ha.SetState(ctx, "sensor.energy_security_score", stateNumber(s.Scores.Headline), merge(base, map[string]any{"friendly_name": "Energy Security Score", "unit_of_measurement": "score", "state_class": "measurement", "status": s.Scores.Status})); err != nil {
		return err
	}
	_ = a.ha.SetState(ctx, "sensor.energy_security_confidence", strconv.FormatFloat(s.Scores.Confidence, 'f', 1, 64), merge(base, map[string]any{"friendly_name": "Energy Security Confidence", "unit_of_measurement": "%", "state_class": "measurement", "icon": "mdi:database-check"}))
	_ = a.ha.SetState(ctx, "sensor.energy_security_status", s.Scores.Status, merge(base, map[string]any{"friendly_name": "Energy Security Status", "icon": "mdi:shield-check"}))
	for _, k := range []string{"electricity", "gas", "oil", "water", "weather"} {
		d := s.Domains[k]
		_ = a.ha.SetState(ctx, "sensor.energy_security_"+k, stateNumber(d.Score), merge(base, map[string]any{"friendly_name": "Energy Security " + title(k), "unit_of_measurement": "score", "state_class": "measurement", "domain_status": d.Status, "summary": d.Summary, "domain_confidence": d.Confidence}))
	}
	for _, x := range []struct{ k, e, f, u, i string }{
		{"gas_storage_fill_pct", "sensor.energy_security_gas_storage_fill", "Gas Storage Fill", "%", "mdi:storage-tank"},
		{"gas_stock_index_pct", "sensor.energy_security_gas_stock_index", "Gas Stock Index", "%", "mdi:storage-tank"},
		{"gas_national_stock_twh", "sensor.energy_security_gas_national_stock", "Gas National Stock", "TWh", "mdi:storage-tank"},
		{"nuclear_output_mw", "sensor.energy_security_nuclear_output", "Nuclear Output", "MW", "mdi:atom"},
		{"renewable_share_pct", "sensor.energy_security_renewable_share", "Renewable Share", "%", "mdi:leaf"},
		{"electricity_load_mw", "sensor.energy_security_electricity_load", "Electricity Load", "MW", "mdi:transmission-tower"},
		{"electricity_generation_mw", "sensor.energy_security_electricity_generation", "Electricity Generation", "MW", "mdi:lightning-bolt"},
	} {
		if o, ok := s.Observations[x.k]; ok && o.Value != nil {
			attrs := merge(base, map[string]any{"friendly_name": x.f, "unit_of_measurement": x.u, "state_class": "measurement", "icon": x.i, "source": o.Source, "observed_at": o.ObservedAt.Format(time.RFC3339), "stale": o.Stale})
			if o.Attributes != nil {
				if proxy, ok := o.Attributes["not_capacity_fill"].(bool); ok && proxy {
					attrs["not_capacity_fill"] = true
					attrs["basis"] = o.Attributes["basis"]
				}
			}
			_ = a.ha.SetState(ctx, x.e, strconv.FormatFloat(*o.Value, 'f', 1, 64), attrs)
		}
	}
	return nil
}
func merge(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func (a *App) Run(ctx context.Context) {
	if a.topologyLearner != nil {
		go a.topologyLearner.Run(ctx)
	}
	if err := a.Refresh(ctx); err != nil {
		log.Printf("initial refresh failed: %v", err)
	}
	t := time.NewTicker(time.Duration(a.cfg.RefreshMinutes) * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := a.Refresh(ctx); err != nil {
				log.Printf("refresh failed: %v", err)
			}
		}
	}
}

func (a *App) SelfTest() error {
	if err := country.Validate(); err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, g := range a.groups {
		if g.ID == "" {
			return fmt.Errorf("empty provider group")
		}
		for _, p := range g.Providers {
			if ids[p.ID()] {
				return fmt.Errorf("duplicate provider id %s", p.ID())
			}
			ids[p.ID()] = true
		}
	}
	names := make([]string, 0, len(ids))
	for id := range ids {
		names = append(names, id)
	}
	sort.Strings(names)
	return nil
}

func title(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}
