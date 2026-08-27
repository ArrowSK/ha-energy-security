package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/ha"
)

const (
	// RetentionDays is deliberately fixed: persisted topology evidence may never
	// contain more than the latest 100 calendar days.
	RetentionDays       = 100
	maxRelationships    = 200
	analysisDelay       = 22 * time.Second
	contaminationWindow = 6 * time.Second
	registryRescanEvery = 6 * time.Hour
	minChildDeltaW      = 5.0
)

type Snapshot struct {
	Enabled                bool           `json:"enabled"`
	Status                 string         `json:"status"`
	RetentionDays          int            `json:"retention_days"`
	PhysicalPowerSensors   int            `json:"physical_power_sensors"`
	MeteredSwitches        int            `json:"metered_switches"`
	ObservedTransitions    int64          `json:"observed_transitions"`
	IgnoredTransitions     int64          `json:"ignored_transitions"`
	CandidateRelationships int            `json:"candidate_relationships"`
	SuspectedRelationships int            `json:"suspected_relationships"`
	StrongRelationships    int            `json:"strong_relationships"`
	ConfirmedRelationships int            `json:"confirmed_relationships"`
	LastObservation        time.Time      `json:"last_observation,omitempty"`
	LastError              string         `json:"last_error,omitempty"`
	Relationships          []Relationship `json:"relationships"`
}

type Relationship struct {
	ParentDeviceID  string    `json:"parent_device_id"`
	ParentEntityID  string    `json:"parent_entity_id"`
	ParentName      string    `json:"parent_name"`
	ChildDeviceID   string    `json:"child_device_id"`
	ChildEntityID   string    `json:"child_entity_id"`
	ChildSwitchID   string    `json:"child_switch_id"`
	ChildName       string    `json:"child_name"`
	Status          string    `json:"status"`
	Confidence      float64   `json:"confidence"`
	Matches         int       `json:"matches"`
	Contradictions  int       `json:"contradictions"`
	OnMatches       int       `json:"on_matches"`
	OffMatches      int       `json:"off_matches"`
	CurrentStreak   int       `json:"current_streak"`
	BestStreak      int       `json:"best_streak"`
	LearnedFactor   float64   `json:"learned_factor"`
	Direct          bool      `json:"direct"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
}

type persistedStore struct {
	Version         int                      `json:"version"`
	UpdatedAt       time.Time                `json:"updated_at"`
	LastObservation time.Time                `json:"last_observation,omitempty"`
	Relations       map[string]*relationData `json:"relations"`
}

type relationData struct {
	ParentDeviceID string                   `json:"parent_device_id"`
	ParentEntityID string                   `json:"parent_entity_id"`
	ParentName     string                   `json:"parent_name"`
	ChildDeviceID  string                   `json:"child_device_id"`
	ChildEntityID  string                   `json:"child_entity_id"`
	ChildSwitchID  string                   `json:"child_switch_id"`
	ChildName      string                   `json:"child_name"`
	CurrentStreak  int                      `json:"current_streak"`
	BestStreak     int                      `json:"best_streak"`
	LastResult     string                   `json:"last_result"`
	FirstSeen      time.Time                `json:"first_seen"`
	LastSeen       time.Time                `json:"last_seen"`
	Days           map[string]*dailyEvidence `json:"days"`
}

type dailyEvidence struct {
	Matches        int     `json:"matches"`
	Contradictions int     `json:"contradictions"`
	OnMatches      int     `json:"on_matches"`
	OffMatches     int     `json:"off_matches"`
	RatioSum       float64 `json:"ratio_sum"`
	RatioCount     int     `json:"ratio_count"`
}

type sample struct {
	At    time.Time
	Value float64
}

type switchTransition struct {
	At       time.Time
	EntityID string
}

type pendingTransition struct {
	At           time.Time
	SwitchEntity string
	DeviceID     string
	Direction    string
	Pre          map[string]float64
}

type runtimeState struct {
	entityMeta       map[string]ha.EntityRegistryEntry
	deviceNames      map[string]string
	entityNames      map[string]string
	powerByDevice    map[string][]string
	powerDevice      map[string]string
	switchDevice     map[string]string
	currentPower     map[string]float64
	currentPowerAt   map[string]time.Time
	samples          map[string][]sample
	transitions      []switchTransition
	observed         int64
	ignored          int64
	lastError        string
}

type Learner struct {
	client  *ha.Client
	path    string
	publish bool

	mu       sync.RWMutex
	store    persistedStore
	runtime  runtimeState
	snapshot Snapshot
}

func New(client *ha.Client, path string, publishHAEntity bool) *Learner {
	l := &Learner{client: client, path: path, publish: publishHAEntity}
	l.store = persistedStore{Version: 1, Relations: map[string]*relationData{}}
	l.resetRuntimeLocked()
	if err := l.load(); err != nil {
		l.runtime.lastError = err.Error()
		log.Printf("power topology learner cache: %v", err)
	}
	l.mu.Lock()
	l.pruneLocked(time.Now().UTC())
	l.updateSnapshotLocked()
	l.mu.Unlock()
	return l
}

func (l *Learner) resetRuntimeLocked() {
	l.runtime.entityMeta = map[string]ha.EntityRegistryEntry{}
	l.runtime.deviceNames = map[string]string{}
	l.runtime.entityNames = map[string]string{}
	l.runtime.powerByDevice = map[string][]string{}
	l.runtime.powerDevice = map[string]string{}
	l.runtime.switchDevice = map[string]string{}
	l.runtime.currentPower = map[string]float64{}
	l.runtime.currentPowerAt = map[string]time.Time{}
	l.runtime.samples = map[string][]sample{}
	l.runtime.transitions = nil
}

func (l *Learner) Snapshot() Snapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := l.snapshot
	out.Relationships = append([]Relationship(nil), l.snapshot.Relationships...)
	return out
}

func (l *Learner) Run(ctx context.Context) {
	backoff := 3 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		sessionCtx, cancel := context.WithTimeout(ctx, registryRescanEvery)
		started := time.Now()
		err := l.client.WatchStates(sessionCtx, l.handleReady, l.handleStateChange)
		cancel()
		if ctx.Err() != nil {
			return
		}
		plannedRescan := errors.Is(sessionCtx.Err(), context.DeadlineExceeded)
		if err != nil && !plannedRescan {
			l.setError(err)
			log.Printf("power topology learner websocket: %v", err)
		}
		if plannedRescan || time.Since(started) > 2*time.Minute {
			backoff = 3 * time.Second
		}
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
		if backoff < time.Minute {
			backoff *= 2
			if backoff > time.Minute {
				backoff = time.Minute
			}
		}
	}
}

func (l *Learner) setError(err error) {
	l.mu.Lock()
	if err == nil {
		l.runtime.lastError = ""
	} else {
		l.runtime.lastError = err.Error()
	}
	l.updateSnapshotLocked()
	l.mu.Unlock()
}

func virtualPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "template", "powercalc", "integration", "statistics", "utility_meter", "derivative", "filter", "min_max", "threshold", "history_stats", "group":
		return true
	default:
		return false
	}
}

func stateName(s ha.State) string {
	if s.Attributes != nil {
		if v, ok := s.Attributes["friendly_name"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return s.EntityID
}

func (l *Learner) handleReady(reg ha.RegistrySnapshot) {
	now := time.Now().UTC()
	l.mu.Lock()
	observed := l.runtime.observed
	ignored := l.runtime.ignored
	l.resetRuntimeLocked()
	l.runtime.observed = observed
	l.runtime.ignored = ignored
	l.runtime.lastError = ""

	for _, e := range reg.Entities {
		l.runtime.entityMeta[e.EntityID] = e
	}
	for _, d := range reg.Devices {
		name := strings.TrimSpace(d.Name)
		if d.NameByUser != nil && strings.TrimSpace(*d.NameByUser) != "" {
			name = strings.TrimSpace(*d.NameByUser)
		}
		if name != "" {
			l.runtime.deviceNames[d.ID] = name
		}
	}

	stateByID := make(map[string]ha.State, len(reg.States))
	for _, s := range reg.States {
		stateByID[s.EntityID] = s
		l.runtime.entityNames[s.EntityID] = stateName(s)
	}

	for _, s := range reg.States {
		if !strings.HasPrefix(s.EntityID, "sensor.") {
			continue
		}
		meta, ok := l.runtime.entityMeta[s.EntityID]
		if !ok || meta.DeviceID == "" || meta.DisabledBy != nil || virtualPlatform(meta.Platform) {
			continue
		}
		deviceClass, _ := s.Attributes["device_class"].(string)
		if !strings.EqualFold(deviceClass, "power") {
			continue
		}
		v, ok := powerWatts(s)
		if !ok {
			continue
		}
		l.runtime.powerByDevice[meta.DeviceID] = append(l.runtime.powerByDevice[meta.DeviceID], s.EntityID)
		l.runtime.powerDevice[s.EntityID] = meta.DeviceID
		at := s.LastUpdated
		if at.IsZero() {
			at = now
		}
		l.runtime.currentPower[s.EntityID] = v
		l.runtime.currentPowerAt[s.EntityID] = at
		l.runtime.samples[s.EntityID] = []sample{{At: at, Value: v}}
	}

	for _, s := range reg.States {
		if !strings.HasPrefix(s.EntityID, "switch.") {
			continue
		}
		meta, ok := l.runtime.entityMeta[s.EntityID]
		if !ok || meta.DeviceID == "" || meta.DisabledBy != nil || virtualPlatform(meta.Platform) {
			continue
		}
		if len(l.runtime.powerByDevice[meta.DeviceID]) == 0 {
			continue
		}
		l.runtime.switchDevice[s.EntityID] = meta.DeviceID
	}

	for key, rel := range l.store.Relations {
		if name := l.runtime.deviceNames[rel.ParentDeviceID]; name != "" {
			rel.ParentName = name
		}
		if name := l.runtime.deviceNames[rel.ChildDeviceID]; name != "" {
			rel.ChildName = name
		}
		if len(l.runtime.powerByDevice[rel.ParentDeviceID]) > 0 && !contains(l.runtime.powerByDevice[rel.ParentDeviceID], rel.ParentEntityID) {
			rel.ParentEntityID = l.runtime.powerByDevice[rel.ParentDeviceID][0]
		}
		if len(l.runtime.powerByDevice[rel.ChildDeviceID]) > 0 && !contains(l.runtime.powerByDevice[rel.ChildDeviceID], rel.ChildEntityID) {
			rel.ChildEntityID = l.runtime.powerByDevice[rel.ChildDeviceID][0]
		}
		l.store.Relations[key] = rel
	}
	l.pruneLocked(now)
	l.updateSnapshotLocked()
	l.mu.Unlock()
	l.publishSummary()
}

func contains(items []string, want string) bool {
	for _, x := range items {
		if x == want {
			return true
		}
	}
	return false
}

func powerWatts(s ha.State) (float64, bool) {
	if s.State == "" || s.State == "unknown" || s.State == "unavailable" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s.State), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	unit, _ := s.Attributes["unit_of_measurement"].(string)
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "w", "watt", "watts":
		return v, true
	case "kw", "kilowatt", "kilowatts":
		return v * 1000, true
	default:
		return 0, false
	}
}

func (l *Learner) handleStateChange(ev ha.StateChange) {
	at := ev.TimeFired.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}

	l.mu.Lock()
	if ev.NewState != nil {
		l.runtime.entityNames[ev.EntityID] = stateName(*ev.NewState)
	}
	if _, ok := l.runtime.powerDevice[ev.EntityID]; ok && ev.NewState != nil {
		if v, valid := powerWatts(*ev.NewState); valid {
			l.addPowerSampleLocked(ev.EntityID, at, v)
		}
		l.mu.Unlock()
		return
	}

	deviceID, monitored := l.runtime.switchDevice[ev.EntityID]
	if !monitored || ev.OldState == nil || ev.NewState == nil {
		l.mu.Unlock()
		return
	}
	oldState := strings.ToLower(strings.TrimSpace(ev.OldState.State))
	newState := strings.ToLower(strings.TrimSpace(ev.NewState.State))
	direction := ""
	if oldState == "off" && newState == "on" {
		direction = "on"
	} else if oldState == "on" && newState == "off" {
		direction = "off"
	}
	if direction == "" {
		l.mu.Unlock()
		return
	}

	l.runtime.observed++
	l.runtime.transitions = append(l.runtime.transitions, switchTransition{At: at, EntityID: ev.EntityID})
	cut := at.Add(-2 * time.Minute)
	i := 0
	for i < len(l.runtime.transitions) && l.runtime.transitions[i].At.Before(cut) {
		i++
	}
	if i > 0 {
		l.runtime.transitions = append([]switchTransition(nil), l.runtime.transitions[i:]...)
	}
	pre := make(map[string]float64, len(l.runtime.currentPower))
	for entity := range l.runtime.currentPower {
		if v, ok := l.preValueLocked(entity, at); ok {
			pre[entity] = v
		}
	}
	pending := pendingTransition{At: at, SwitchEntity: ev.EntityID, DeviceID: deviceID, Direction: direction, Pre: pre}
	l.updateSnapshotLocked()
	l.mu.Unlock()

	go func() {
		t := time.NewTimer(analysisDelay)
		defer t.Stop()
		<-t.C
		l.evaluate(pending)
	}()
}

func (l *Learner) addPowerSampleLocked(entity string, at time.Time, value float64) {
	l.runtime.currentPower[entity] = value
	l.runtime.currentPowerAt[entity] = at
	items := append(l.runtime.samples[entity], sample{At: at, Value: value})
	cut := at.Add(-90 * time.Second)
	i := 0
	for i < len(items) && items[i].At.Before(cut) {
		i++
	}
	if i > 0 {
		items = append([]sample(nil), items[i:]...)
	}
	if len(items) > 48 {
		items = items[len(items)-48:]
	}
	l.runtime.samples[entity] = items
}

func median(values []float64) float64 {
	sort.Float64s(values)
	n := len(values)
	if n%2 == 1 {
		return values[n/2]
	}
	return (values[n/2-1] + values[n/2]) / 2
}

func (l *Learner) medianWindowLocked(entity string, from, to time.Time) (float64, bool) {
	items := l.runtime.samples[entity]
	values := make([]float64, 0, len(items))
	for _, s := range items {
		if (s.At.Equal(from) || s.At.After(from)) && (s.At.Equal(to) || s.At.Before(to)) {
			values = append(values, s.Value)
		}
	}
	if len(values) == 0 {
		return 0, false
	}
	return median(values), true
}

func (l *Learner) preValueLocked(entity string, at time.Time) (float64, bool) {
	if v, ok := l.medianWindowLocked(entity, at.Add(-20*time.Second), at.Add(-750*time.Millisecond)); ok {
		return v, true
	}
	v, ok := l.runtime.currentPower[entity]
	if !ok {
		return 0, false
	}
	stamp := l.runtime.currentPowerAt[entity]
	if stamp.After(at.Add(-500 * time.Millisecond)) {
		return 0, false
	}
	return v, true
}

func (l *Learner) postValueLocked(entity string, at time.Time) (float64, bool) {
	if v, ok := l.medianWindowLocked(entity, at.Add(1*time.Second), time.Now().UTC()); ok {
		return v, true
	}
	v, ok := l.runtime.currentPower[entity]
	return v, ok
}

func sameSign(a, b float64) bool {
	return (a > 0 && b > 0) || (a < 0 && b < 0)
}

func relationKey(parentDevice, childDevice string) string {
	return parentDevice + "|" + childDevice
}

func (l *Learner) aggregate(rel *relationData) (matches, contradictions, onMatches, offMatches, ratioCount int, ratioSum float64) {
	for _, d := range rel.Days {
		matches += d.Matches
		contradictions += d.Contradictions
		onMatches += d.OnMatches
		offMatches += d.OffMatches
		ratioCount += d.RatioCount
		ratioSum += d.RatioSum
	}
	return
}

func relationshipStats(rel *relationData) Relationship {
	matches, contradictions, onMatches, offMatches, ratioCount, ratioSum := 0, 0, 0, 0, 0, 0.0
	for _, d := range rel.Days {
		matches += d.Matches
		contradictions += d.Contradictions
		onMatches += d.OnMatches
		offMatches += d.OffMatches
		ratioCount += d.RatioCount
		ratioSum += d.RatioSum
	}
	total := matches + contradictions
	support := 0.0
	if total > 0 {
		support = float64(matches) / float64(total)
	}
	strength := math.Min(1, float64(matches)/8)
	directionFactor := 0.90
	if onMatches > 0 && offMatches > 0 {
		directionFactor = 1
	}
	confidence := 100 * (0.35 + 0.65*strength) * support * directionFactor
	if confidence > 100 {
		confidence = 100
	}
	factor := 1.0
	if ratioCount > 0 {
		factor = ratioSum / float64(ratioCount)
	}
	status := "learning"
	if matches >= 3 && support >= 0.75 && (rel.CurrentStreak >= 3 || rel.BestStreak >= 3) {
		status = "suspected"
	}
	if matches >= 5 && support >= 0.85 {
		status = "strong"
	}
	if matches >= 8 && support >= 0.90 && onMatches > 0 && offMatches > 0 {
		status = "confirmed"
	}
	return Relationship{
		ParentDeviceID: rel.ParentDeviceID, ParentEntityID: rel.ParentEntityID, ParentName: rel.ParentName,
		ChildDeviceID: rel.ChildDeviceID, ChildEntityID: rel.ChildEntityID, ChildSwitchID: rel.ChildSwitchID, ChildName: rel.ChildName,
		Status: status, Confidence: math.Round(confidence*10) / 10, Matches: matches, Contradictions: contradictions,
		OnMatches: onMatches, OffMatches: offMatches, CurrentStreak: rel.CurrentStreak, BestStreak: rel.BestStreak,
		LearnedFactor: math.Round(factor*1000) / 1000, Direct: true, FirstSeen: rel.FirstSeen, LastSeen: rel.LastSeen,
	}
}

func (l *Learner) evaluate(p pendingTransition) {
	now := time.Now().UTC()
	l.mu.Lock()

	for _, tr := range l.runtime.transitions {
		if tr.EntityID == p.SwitchEntity {
			continue
		}
		d := tr.At.Sub(p.At)
		if d < 0 {
			d = -d
		}
		if d <= contaminationWindow {
			l.runtime.ignored++
			l.updateSnapshotLocked()
			l.mu.Unlock()
			return
		}
	}

	childEntity := ""
	childDelta := 0.0
	for _, entity := range l.runtime.powerByDevice[p.DeviceID] {
		pre, ok := p.Pre[entity]
		if !ok {
			continue
		}
		post, ok := l.postValueLocked(entity, p.At)
		if !ok {
			continue
		}
		delta := post - pre
		if math.Abs(delta) > math.Abs(childDelta) {
			childDelta = delta
			childEntity = entity
		}
	}
	if childEntity == "" || math.Abs(childDelta) < minChildDeltaW || (p.Direction == "on" && childDelta <= 0) || (p.Direction == "off" && childDelta >= 0) {
		l.runtime.ignored++
		l.updateSnapshotLocked()
		l.mu.Unlock()
		return
	}

	type parentCandidate struct {
		device string
		entity string
		delta  float64
		ratio  float64
		valid  bool
		match  bool
	}
	candidates := map[string]parentCandidate{}
	matchedCount := 0
	for parentDevice, entities := range l.runtime.powerByDevice {
		if parentDevice == p.DeviceID {
			continue
		}
		key := relationKey(parentDevice, p.DeviceID)
		existing := l.store.Relations[key]
		target := 1.0
		if existing != nil {
			_, _, _, _, ratioCount, ratioSum := l.aggregate(existing)
			if ratioCount > 0 {
				target = ratioSum / float64(ratioCount)
			}
		}
		best := parentCandidate{device: parentDevice}
		bestDistance := math.MaxFloat64
		for _, entity := range entities {
			pre, ok := p.Pre[entity]
			if !ok {
				continue
			}
			post, ok := l.postValueLocked(entity, p.At)
			if !ok {
				continue
			}
			delta := post - pre
			best.valid = true
			if !sameSign(delta, childDelta) || math.Abs(delta) < 1 {
				continue
			}
			ratio := math.Abs(delta / childDelta)
			distance := math.Abs(ratio - target)
			if distance < bestDistance {
				bestDistance = distance
				best.entity = entity
				best.delta = delta
				best.ratio = ratio
			}
		}
		if best.entity != "" {
			if existing == nil {
				best.match = best.ratio >= 0.75 && best.ratio <= 1.30
			} else {
				tolerance := math.Max(0.15, target*0.15)
				best.match = best.ratio >= 0.70 && best.ratio <= 1.35 && math.Abs(best.ratio-target) <= tolerance
			}
		}
		if best.match {
			matchedCount++
		}
		candidates[parentDevice] = best
	}
	if matchedCount > 6 {
		l.runtime.ignored++
		l.updateSnapshotLocked()
		l.mu.Unlock()
		return
	}

	day := p.At.UTC().Format("2006-01-02")
	for parentDevice, cand := range candidates {
		key := relationKey(parentDevice, p.DeviceID)
		rel := l.store.Relations[key]
		if rel == nil && !cand.match {
			continue
		}
		if rel == nil {
			rel = &relationData{
				ParentDeviceID: parentDevice, ParentEntityID: cand.entity, ParentName: l.nameForDeviceLocked(parentDevice, cand.entity),
				ChildDeviceID: p.DeviceID, ChildEntityID: childEntity, ChildSwitchID: p.SwitchEntity, ChildName: l.nameForDeviceLocked(p.DeviceID, p.SwitchEntity),
				FirstSeen: p.At, Days: map[string]*dailyEvidence{},
			}
			l.store.Relations[key] = rel
		}
		if rel.Days == nil {
			rel.Days = map[string]*dailyEvidence{}
		}
		if cand.entity != "" {
			rel.ParentEntityID = cand.entity
		}
		rel.ChildEntityID = childEntity
		rel.ChildSwitchID = p.SwitchEntity
		rel.ParentName = l.nameForDeviceLocked(parentDevice, rel.ParentEntityID)
		rel.ChildName = l.nameForDeviceLocked(p.DeviceID, p.SwitchEntity)
		d := rel.Days[day]
		if d == nil {
			d = &dailyEvidence{}
			rel.Days[day] = d
		}
		if cand.match {
			d.Matches++
			d.RatioSum += cand.ratio
			d.RatioCount++
			if p.Direction == "on" {
				d.OnMatches++
			} else {
				d.OffMatches++
			}
			rel.CurrentStreak++
			if rel.CurrentStreak > rel.BestStreak {
				rel.BestStreak = rel.CurrentStreak
			}
			rel.LastResult = "match"
			rel.LastSeen = p.At
		} else if cand.valid {
			d.Contradictions++
			rel.CurrentStreak = 0
			rel.LastResult = "contradiction"
			rel.LastSeen = p.At
		}
	}
	l.store.LastObservation = p.At
	l.store.UpdatedAt = now
	l.pruneLocked(now)
	if err := l.saveLocked(); err != nil {
		l.runtime.lastError = err.Error()
		log.Printf("power topology learner save: %v", err)
	} else {
		l.runtime.lastError = ""
	}
	l.updateSnapshotLocked()
	l.mu.Unlock()
	l.publishSummary()
}

func (l *Learner) nameForDeviceLocked(deviceID, fallbackEntity string) string {
	if name := strings.TrimSpace(l.runtime.deviceNames[deviceID]); name != "" {
		return name
	}
	if name := strings.TrimSpace(l.runtime.entityNames[fallbackEntity]); name != "" {
		return name
	}
	if fallbackEntity != "" {
		return fallbackEntity
	}
	return deviceID
}

func (l *Learner) pruneLocked(now time.Time) {
	cutoffDay := now.UTC().AddDate(0, 0, -(RetentionDays - 1)).Format("2006-01-02")
	for key, rel := range l.store.Relations {
		for day := range rel.Days {
			if day < cutoffDay {
				delete(rel.Days, day)
			}
		}
		if len(rel.Days) == 0 {
			delete(l.store.Relations, key)
			continue
		}
		firstDay := "9999-99-99"
		lastDay := ""
		for day := range rel.Days {
			if day < firstDay {
				firstDay = day
			}
			if day > lastDay {
				lastDay = day
			}
		}
		if t, err := time.Parse("2006-01-02", firstDay); err == nil && rel.FirstSeen.Before(t) {
			rel.FirstSeen = t
		}
		if t, err := time.Parse("2006-01-02", lastDay); err == nil && rel.LastSeen.Before(t) {
			rel.LastSeen = t
		}
	}
	if len(l.store.Relations) <= maxRelationships {
		return
	}
	type ranked struct {
		key     string
		matches int
		last    time.Time
	}
	items := make([]ranked, 0, len(l.store.Relations))
	for key, rel := range l.store.Relations {
		matches, _, _, _, _, _ := l.aggregate(rel)
		items = append(items, ranked{key: key, matches: matches, last: rel.LastSeen})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].matches != items[j].matches {
			return items[i].matches > items[j].matches
		}
		return items[i].last.After(items[j].last)
	})
	for _, item := range items[maxRelationships:] {
		delete(l.store.Relations, item.key)
	}
}

func (l *Learner) updateSnapshotLocked() {
	rels := make([]Relationship, 0, len(l.store.Relations))
	for _, rel := range l.store.Relations {
		rels = append(rels, relationshipStats(rel))
	}
	markTransitiveAncestors(rels)
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].Confidence != rels[j].Confidence {
			return rels[i].Confidence > rels[j].Confidence
		}
		if rels[i].Matches != rels[j].Matches {
			return rels[i].Matches > rels[j].Matches
		}
		return rels[i].LastSeen.After(rels[j].LastSeen)
	})
	suspected, strong, confirmed := 0, 0, 0
	for _, rel := range rels {
		switch rel.Status {
		case "suspected":
			suspected++
		case "strong":
			strong++
		case "confirmed":
			confirmed++
		}
	}
	status := "learning"
	if len(l.runtime.powerDevice) == 0 || len(l.runtime.switchDevice) == 0 {
		status = "waiting"
	}
	if suspected > 0 {
		status = "suspected"
	}
	if strong > 0 {
		status = "strong"
	}
	if confirmed > 0 {
		status = "confirmed"
	}
	l.snapshot = Snapshot{
		Enabled: true, Status: status, RetentionDays: RetentionDays,
		PhysicalPowerSensors: len(l.runtime.powerDevice), MeteredSwitches: len(l.runtime.switchDevice),
		ObservedTransitions: l.runtime.observed, IgnoredTransitions: l.runtime.ignored,
		CandidateRelationships: len(rels), SuspectedRelationships: suspected, StrongRelationships: strong, ConfirmedRelationships: confirmed,
		LastObservation: l.store.LastObservation, LastError: l.runtime.lastError, Relationships: rels,
	}
}

func markTransitiveAncestors(rels []Relationship) {
	edges := map[string][]string{}
	for _, rel := range rels {
		if rel.Status == "confirmed" {
			edges[rel.ChildDeviceID] = append(edges[rel.ChildDeviceID], rel.ParentDeviceID)
		}
	}
	for i := range rels {
		if rels[i].Status != "confirmed" {
			continue
		}
		child, parent := rels[i].ChildDeviceID, rels[i].ParentDeviceID
		for _, via := range edges[child] {
			if via == parent {
				continue
			}
			if pathExists(via, parent, edges, map[string]bool{}) {
				rels[i].Direct = false
				break
			}
		}
	}
}

func pathExists(current, target string, edges map[string][]string, seen map[string]bool) bool {
	if current == target {
		return true
	}
	if seen[current] {
		return false
	}
	seen[current] = true
	for _, next := range edges[current] {
		if pathExists(next, target, edges, seen) {
			return true
		}
	}
	return false
}

func (l *Learner) load() error {
	b, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var s persistedStore
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("parse topology evidence: %w", err)
	}
	if s.Version != 1 {
		return fmt.Errorf("unsupported topology evidence version %d", s.Version)
	}
	if s.Relations == nil {
		s.Relations = map[string]*relationData{}
	}
	l.store = s
	return nil
}

func (l *Learner) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(l.store, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, l.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (l *Learner) publishSummary() {
	if !l.publish || l.client == nil {
		return
	}
	s := l.Snapshot()
	top := make([]map[string]any, 0, 5)
	for _, rel := range s.Relationships {
		if rel.Status == "learning" {
			continue
		}
		top = append(top, map[string]any{
			"parent": rel.ParentName, "child": rel.ChildName, "status": rel.Status,
			"confidence": rel.Confidence, "factor": rel.LearnedFactor, "direct": rel.Direct,
		})
		if len(top) == 5 {
			break
		}
	}
	attrs := map[string]any{
		"friendly_name": "Energy Security Power Topology", "icon": "mdi:transmission-tower-import",
		"retention_days": RetentionDays, "physical_power_sensors": s.PhysicalPowerSensors, "metered_switches": s.MeteredSwitches,
		"candidate_relationships": s.CandidateRelationships, "suspected_relationships": s.SuspectedRelationships,
		"strong_relationships": s.StrongRelationships, "confirmed_relationships": s.ConfirmedRelationships,
		"observed_transitions": s.ObservedTransitions, "ignored_transitions": s.IgnoredTransitions,
		"read_only": true, "affects_energy_totals": false, "top_relationships": top,
	}
	if !s.LastObservation.IsZero() {
		attrs["last_observation"] = s.LastObservation.Format(time.RFC3339)
	}
	if s.LastError != "" {
		attrs["last_error"] = s.LastError
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := l.client.SetState(ctx, "sensor.energy_security_power_topology", s.Status, attrs); err != nil {
		log.Printf("power topology learner entity publish: %v", err)
	}
}
