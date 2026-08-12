package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type fakeProvider struct {
	id    string
	fail  bool
	value float64
}

func (p fakeProvider) ID() string          { return p.id }
func (p fakeProvider) Name() string        { return p.id }
func (p fakeProvider) Supports(Input) bool { return true }
func (p fakeProvider) Collect(context.Context, Input) ([]model.Observation, error) {
	if p.fail {
		return nil, errors.New("broken")
	}
	return []model.Observation{obs("x", "test", "X", p.value, "", "fake", "", 1, time.Now().UTC())}, nil
}

func TestManagerFallsBack(t *testing.T) {
	m := NewManager()
	g := Group{ID: "test", TTL: time.Hour, Providers: []Provider{fakeProvider{id: "primary", fail: true}, fakeProvider{id: "fallback", value: 7}}}
	o, used, err := m.CollectGroup(context.Background(), g, Input{})
	if err != nil {
		t.Fatal(err)
	}
	if used != "fallback" || len(o) != 1 || o[0].Value == nil || *o[0].Value != 7 {
		t.Fatalf("unexpected fallback result: used=%s obs=%+v", used, o)
	}
	h := m.Health()
	if h["primary"].Failures != 1 || h["fallback"].State != "healthy" {
		t.Fatalf("unexpected health: %+v", h)
	}
}

func TestCircuitBreakerOpensAfterThreeFailures(t *testing.T) {
	m := NewManager()
	p := fakeProvider{id: "p", fail: true}
	g := Group{ID: "test", TTL: time.Hour, Providers: []Provider{p}}
	for i := 0; i < 3; i++ {
		_, _, _ = m.CollectGroup(context.Background(), g, Input{})
	}
	h := m.Health()["p"]
	if h.CircuitUntil.IsZero() || h.State != "degraded" {
		t.Fatalf("expected open circuit after three failures: %+v", h)
	}
}
