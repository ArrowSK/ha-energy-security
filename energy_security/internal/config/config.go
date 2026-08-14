package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)

type Config struct {
	Country          string   `json:"country"`
	RefreshMinutes   int      `json:"refresh_minutes"`
	EnableHAEntities bool     `json:"enable_ha_entities"`
	EnableWeather    bool     `json:"enable_weather"`
	AGSIKey          string   `json:"agsi_key"`
	ENTSOEToken      string   `json:"entsoe_token"`
	RuntimeMode      string   `json:"-"`
	Latitude         *float64 `json:"-"`
	Longitude        *float64 `json:"-"`
	TimeZone         string   `json:"-"`
	LocationName     string   `json:"-"`
}

func Defaults() Config {
	return Config{Country: "auto", RefreshMinutes: 30, EnableHAEntities: true, EnableWeather: true, RuntimeMode: "home_assistant"}
}

func normalize(cfg Config) (Config, error) {
	if cfg.RefreshMinutes < 10 {
		cfg.RefreshMinutes = 10
	}
	if cfg.RefreshMinutes > 180 {
		cfg.RefreshMinutes = 180
	}
	cfg.Country = strings.TrimSpace(cfg.Country)
	if cfg.Country == "" {
		cfg.Country = "auto"
	}
	if !strings.EqualFold(cfg.Country, "auto") {
		cfg.Country = strings.ToUpper(cfg.Country)
		if !countryPattern.MatchString(cfg.Country) {
			return cfg, fmt.Errorf("country must be auto or ISO 3166-1 alpha-2")
		}
	}
	return cfg, nil
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse options: %w", err)
	}
	cfg.RuntimeMode = "home_assistant"
	return normalize(cfg)
}

// LoadEnvironment builds the standalone Docker/Railway configuration. It uses
// the same Config type consumed by the Home Assistant app so provider and
// scoring behavior never fork between deployment modes.
func LoadEnvironment() (Config, error) {
	cfg := Defaults()
	cfg.RuntimeMode = "standalone"
	cfg.EnableHAEntities = false
	cfg.Country = strings.TrimSpace(os.Getenv("ENERGY_SECURITY_COUNTRY"))
	if cfg.Country == "" {
		return cfg, fmt.Errorf("ENERGY_SECURITY_COUNTRY is required for standalone mode")
	}
	if strings.EqualFold(cfg.Country, "auto") {
		return cfg, fmt.Errorf("ENERGY_SECURITY_COUNTRY cannot be auto outside Home Assistant")
	}

	if raw := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_REFRESH_MINUTES")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return cfg, fmt.Errorf("ENERGY_SECURITY_REFRESH_MINUTES must be an integer")
		}
		cfg.RefreshMinutes = v
	}
	if raw := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_ENABLE_WEATHER")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return cfg, fmt.Errorf("ENERGY_SECURITY_ENABLE_WEATHER must be true or false")
		}
		cfg.EnableWeather = v
	}
	cfg.AGSIKey = strings.TrimSpace(os.Getenv("ENERGY_SECURITY_AGSI_KEY"))
	cfg.ENTSOEToken = strings.TrimSpace(os.Getenv("ENERGY_SECURITY_ENTSOE_TOKEN"))
	cfg.TimeZone = strings.TrimSpace(os.Getenv("ENERGY_SECURITY_TIMEZONE"))
	cfg.LocationName = strings.TrimSpace(os.Getenv("ENERGY_SECURITY_LOCATION_NAME"))

	latRaw := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_LATITUDE"))
	lonRaw := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_LONGITUDE"))
	if (latRaw == "") != (lonRaw == "") {
		return cfg, fmt.Errorf("ENERGY_SECURITY_LATITUDE and ENERGY_SECURITY_LONGITUDE must be set together")
	}
	if latRaw != "" {
		lat, err := strconv.ParseFloat(latRaw, 64)
		if err != nil || lat < -90 || lat > 90 {
			return cfg, fmt.Errorf("ENERGY_SECURITY_LATITUDE must be between -90 and 90")
		}
		lon, err := strconv.ParseFloat(lonRaw, 64)
		if err != nil || lon < -180 || lon > 180 {
			return cfg, fmt.Errorf("ENERGY_SECURITY_LONGITUDE must be between -180 and 180")
		}
		cfg.Latitude = &lat
		cfg.Longitude = &lon
	}

	return normalize(cfg)
}
