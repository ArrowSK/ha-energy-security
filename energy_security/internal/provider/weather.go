package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/httpx"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
)

type OpenMeteo struct{ C *httpx.Client }

func (p OpenMeteo) ID() string   { return "open_meteo" }
func (p OpenMeteo) Name() string { return "Open-Meteo" }
func (p OpenMeteo) Supports(in Input) bool {
	return in.Config.EnableWeather && in.HasLocation && in.Latitude >= -90 && in.Latitude <= 90 && in.Longitude >= -180 && in.Longitude <= 180
}

type omResp struct {
	Current struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		Wind        float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Daily struct {
		Time   []string  `json:"time"`
		Max    []float64 `json:"temperature_2m_max"`
		Min    []float64 `json:"temperature_2m_min"`
		Precip []float64 `json:"precipitation_sum"`
		Wind   []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

func maxf(v []float64) (float64, bool) {
	if len(v) == 0 {
		return 0, false
	}
	m := v[0]
	for _, x := range v[1:] {
		if x > m {
			m = x
		}
	}
	return m, true
}
func minf(v []float64) (float64, bool) {
	if len(v) == 0 {
		return 0, false
	}
	m := v[0]
	for _, x := range v[1:] {
		if x < m {
			m = x
		}
	}
	return m, true
}
func sumf(v []float64) float64 {
	var s float64
	for _, x := range v {
		s += x
	}
	return s
}
func (p OpenMeteo) Collect(ctx context.Context, in Input) ([]model.Observation, error) {
	if p.C == nil {
		p.C = httpx.New()
	}
	q := url.Values{"latitude": []string{strconv.FormatFloat(in.Latitude, 'f', 4, 64)}, "longitude": []string{strconv.FormatFloat(in.Longitude, 'f', 4, 64)}, "current": []string{"temperature_2m,wind_speed_10m"}, "daily": []string{"temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max"}, "forecast_days": []string{"7"}, "timezone": []string{"auto"}, "wind_speed_unit": []string{"kmh"}}
	u := "https://api.open-meteo.com/v1/forecast?" + q.Encode()
	b, _, err := p.C.Get(ctx, u, nil, 3<<20)
	if err != nil {
		return nil, err
	}
	var r omResp
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("decode Open-Meteo: %w", err)
	}
	at := time.Now().UTC()
	out := []model.Observation{obs("weather_temperature_c", "weather", "Current temperature", r.Current.Temperature, "°C", p.Name(), u, 0.82, at), obs("weather_wind_kmh", "weather", "Current wind speed", r.Current.Wind, "km/h", p.Name(), u, 0.82, at)}
	if v, ok := maxf(r.Daily.Max); ok {
		out = append(out, obs("forecast_max_temperature_c", "weather", "7-day maximum temperature", v, "°C", p.Name(), u, 0.82, at))
	}
	if v, ok := minf(r.Daily.Min); ok {
		out = append(out, obs("forecast_min_temperature_c", "weather", "7-day minimum temperature", v, "°C", p.Name(), u, 0.82, at))
	}
	if v, ok := maxf(r.Daily.Wind); ok {
		out = append(out, obs("forecast_max_wind_kmh", "weather", "7-day maximum wind", v, "km/h", p.Name(), u, 0.82, at))
	}
	if len(r.Daily.Precip) > 0 {
		out = append(out, obs("forecast_precipitation_7d_mm", "weather", "7-day precipitation", sumf(r.Daily.Precip), "mm", p.Name(), u, 0.78, at))
	}
	return out, nil
}
