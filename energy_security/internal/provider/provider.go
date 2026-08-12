package provider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type Input struct {
	Profile             country.Profile
	Latitude, Longitude float64
	HasLocation         bool
	Config              config.Config
}

type Provider interface {
	ID() string
	Name() string
	Supports(Input) bool
	Collect(context.Context, Input) ([]model.Observation, error)
}

type Group struct {
	ID        string
	Providers []Provider
	TTL       time.Duration
}

type Manager struct {
	mu     sync.Mutex
	health map[string]model.ProviderHealth
}

func NewManager() *Manager {
	return &Manager{health: map[string]model.ProviderHealth{}}
}

func (m *Manager) Health() map[string]model.ProviderHealth {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]model.ProviderHealth, len(m.health))
	for k, v := range m.health {
		out[k] = v
	}
	return out
}

func (m *Manager) allowed(p Provider, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := m.health[p.ID()]
	return h.CircuitUntil.IsZero() || !now.Before(h.CircuitUntil)
}

func cooldown(failures int) time.Duration {
	if failures < 3 {
		return 0
	}
	mins := math.Pow(2, float64(failures-3)) * 5
	if mins > 60 {
		mins = 60
	}
	return time.Duration(mins) * time.Minute
}

func (m *Manager) mark(p Provider, started time.Time, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	h := m.health[p.ID()]
	h.ID = p.ID()
	h.Name = p.Name()
	h.LatencyMS = time.Since(started).Milliseconds()

	if err == nil {
		h.State = "healthy"
		h.Failures = 0
		h.LastSuccess = now
		h.LastError = ""
		h.CircuitUntil = time.Time{}
	} else {
		h.Failures++
		h.LastFailure = now
		h.LastError = err.Error()
		if d := cooldown(h.Failures); d > 0 {
			h.State = "degraded"
			h.CircuitUntil = now.Add(d)
		} else {
			h.State = "failed"
		}
	}
	m.health[p.ID()] = h
}

func (m *Manager) CollectGroup(ctx context.Context, g Group, in Input) ([]model.Observation, string, error) {
	now := time.Now().UTC()
	var errs []error

	for _, p := range g.Providers {
		if !p.Supports(in) || !m.allowed(p, now) {
			continue
		}
		started := time.Now()
		obs, err := p.Collect(ctx, in)
		if err == nil && len(obs) == 0 {
			err = errors.New("provider returned no observations")
		}
		m.mark(p, started, err)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.ID(), err))
			continue
		}
		for i := range obs {
			if obs[i].TTLSeconds == 0 {
				obs[i].TTLSeconds = int64(g.TTL.Seconds())
			}
		}
		return obs, p.ID(), nil
	}

	if len(errs) == 0 {
		return nil, "", errors.New("no provider available")
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return nil, "", errors.Join(errs...)
}
