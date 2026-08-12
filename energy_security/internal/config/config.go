package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	Country          string `json:"country"`
	RefreshMinutes   int    `json:"refresh_minutes"`
	EnableHAEntities bool   `json:"enable_ha_entities"`
	EnableWeather    bool   `json:"enable_weather"`
	AGSIKey          string `json:"agsi_key"`
	ENTSOEToken      string `json:"entsoe_token"`
}

func Defaults() Config {
	return Config{Country: "auto", RefreshMinutes: 30, EnableHAEntities: true, EnableWeather: true}
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
		if !regexp.MustCompile(`^[A-Z]{2}$`).MatchString(cfg.Country) {
			return cfg, fmt.Errorf("country must be auto or ISO 3166-1 alpha-2")
		}
	}
	return cfg, nil
}
