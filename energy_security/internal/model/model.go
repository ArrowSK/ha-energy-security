package model

import "time"

type Observation struct {
	Key         string         `json:"key"`
	Domain      string         `json:"domain"`
	Label       string         `json:"label"`
	Value       *float64       `json:"value,omitempty"`
	Text        string         `json:"text,omitempty"`
	Unit        string         `json:"unit,omitempty"`
	Source      string         `json:"source"`
	SourceURL   string         `json:"source_url,omitempty"`
	Quality     float64        `json:"quality"`
	ObservedAt  time.Time      `json:"observed_at"`
	RetrievedAt time.Time      `json:"retrieved_at"`
	TTLSeconds  int64          `json:"ttl_seconds"`
	Stale       bool           `json:"stale"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

type ProviderHealth struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	State        string    `json:"state"`
	Failures     int       `json:"failures"`
	LastSuccess  time.Time `json:"last_success,omitempty"`
	LastFailure  time.Time `json:"last_failure,omitempty"`
	CircuitUntil time.Time `json:"circuit_until,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
	LatencyMS    int64     `json:"latency_ms,omitempty"`
}

type DomainScore struct {
	Score      *float64 `json:"score,omitempty"`
	Status     string   `json:"status"`
	Confidence float64  `json:"confidence"`
	Summary    string   `json:"summary"`
	Trend      string   `json:"trend,omitempty"`
}

type Scores struct {
	Current    *float64 `json:"current,omitempty"`
	Outlook7D  *float64 `json:"outlook_7d,omitempty"`
	Strategic  *float64 `json:"strategic,omitempty"`
	Headline   *float64 `json:"headline,omitempty"`
	Confidence float64  `json:"confidence"`
	Status     string   `json:"status"`
}

type HistoryPoint struct {
	Time       time.Time `json:"time"`
	Headline   *float64  `json:"headline,omitempty"`
	Confidence float64   `json:"confidence"`
}

type Snapshot struct {
	Version      int                       `json:"version"`
	Country      string                    `json:"country"`
	CountryName  string                    `json:"country_name"`
	LocationName string                    `json:"location_name,omitempty"`
	Latitude     float64                   `json:"latitude,omitempty"`
	Longitude    float64                   `json:"longitude,omitempty"`
	Timezone     string                    `json:"timezone,omitempty"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	Scores       Scores                    `json:"scores"`
	Domains      map[string]DomainScore    `json:"domains"`
	Observations map[string]Observation    `json:"observations"`
	Providers    map[string]ProviderHealth `json:"providers"`
	Alerts       []string                  `json:"alerts"`
	Notes        []string                  `json:"notes,omitempty"`
	History      []HistoryPoint            `json:"history,omitempty"`
}
