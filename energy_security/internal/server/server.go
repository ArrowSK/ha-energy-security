package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/country"
)

//go:embed web/*
var webFiles embed.FS

type Server struct {
	app        *app.App
	mux        *http.ServeMux
	refreshing atomic.Bool
	started    time.Time
	mode       string
	cfg        config.Config
	configPath string
}

type setupRequest struct {
	Country          string `json:"country"`
	RefreshMinutes   int    `json:"refresh_minutes"`
	EnableHAEntities bool   `json:"enable_ha_entities"`
	EnableWeather    bool   `json:"enable_weather"`
	AGSIKey          string `json:"agsi_key"`
	ENTSOEToken      string `json:"entsoe_token"`
	ClearAGSIKey     bool   `json:"clear_agsi_key"`
	ClearENTSOEToken bool   `json:"clear_entsoe_token"`
}

type setupCountry struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Support string `json:"support"`
}

func New(a *app.App) *Server {
	return newServer(a, "home_assistant", config.Config{}, "/data/options.json")
}

func NewStandalone(a *app.App, cfg config.Config) *Server {
	return newServer(a, "standalone", cfg, "")
}

func newServer(a *app.App, mode string, cfg config.Config, configPath string) *Server {
	s := &Server{app: a, mux: http.NewServeMux(), started: time.Now().UTC(), mode: mode, cfg: cfg, configPath: configPath}
	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	s.mux.HandleFunc("GET /api/v1/status", s.status)
	s.mux.HandleFunc("GET /api/v1/config", s.getConfig)
	s.mux.HandleFunc("POST /api/v1/config", s.setConfig)
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("POST /api/v1/refresh", s.refresh)
	s.mux.Handle("/", http.FileServer(http.FS(assets)))
	return s
}

func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	if s.mode != "standalone" {
		h = s.ingressGuard(h)
	}
	return securityHeaders(h)
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Snapshot())
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	snap := s.app.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"country":      snap.Country,
		"updated_at":   snap.UpdatedAt,
		"uptime_sec":   int64(time.Since(s.started).Seconds()),
		"refreshing":   s.refreshing.Load(),
		"has_snapshot": !snap.UpdatedAt.IsZero(),
		"mode":         s.mode,
	})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := s.cfg
	editable := false
	if s.mode != "standalone" {
		var err error
		cfg, err = config.Load(s.configPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "unable to read app configuration"})
			return
		}
		editable = true
	}
	profiles := country.All()
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	countries := make([]setupCountry, 0, len(profiles))
	for _, p := range profiles {
		countries = append(countries, setupCountry{Code: p.Code, Name: p.Name, Support: p.Support})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"country":                 cfg.Country,
		"refresh_minutes":         cfg.RefreshMinutes,
		"enable_ha_entities":      cfg.EnableHAEntities,
		"enable_weather":          cfg.EnableWeather,
		"agsi_key_configured":     strings.TrimSpace(cfg.AGSIKey) != "",
		"entsoe_token_configured": strings.TrimSpace(cfg.ENTSOEToken) != "",
		"countries":               countries,
		"deployment_mode":         s.mode,
		"editable":                editable,
	})
}

func normalizeSetup(req setupRequest, current config.Config) (config.Config, error) {
	next := current
	c := strings.TrimSpace(req.Country)
	if c == "" || strings.EqualFold(c, "auto") {
		c = "auto"
	} else {
		c = strings.ToUpper(c)
		if len(c) != 2 || c[0] < 'A' || c[0] > 'Z' || c[1] < 'A' || c[1] > 'Z' {
			return current, fmt.Errorf("country must be HOME/auto or a two-letter ISO code")
		}
	}
	if req.RefreshMinutes < 10 || req.RefreshMinutes > 180 {
		return current, fmt.Errorf("refresh interval must be between 10 and 180 minutes")
	}
	next.Country = c
	next.RefreshMinutes = req.RefreshMinutes
	next.EnableHAEntities = req.EnableHAEntities
	next.EnableWeather = req.EnableWeather

	if req.ClearAGSIKey {
		next.AGSIKey = ""
	} else if strings.TrimSpace(req.AGSIKey) != "" {
		next.AGSIKey = strings.TrimSpace(req.AGSIKey)
	}
	if req.ClearENTSOEToken {
		next.ENTSOEToken = ""
	} else if strings.TrimSpace(req.ENTSOEToken) != "" {
		next.ENTSOEToken = strings.TrimSpace(req.ENTSOEToken)
	}
	return next, nil
}

func (s *Server) setConfig(w http.ResponseWriter, r *http.Request) {
	if s.mode == "standalone" {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "standalone configuration is controlled by environment variables; change them in Docker or Railway and redeploy"})
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]any{"error": "configuration must be sent as JSON"})
		return
	}
	var req setupRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid configuration payload"})
		return
	}
	current, err := config.Load(s.configPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "unable to read current configuration"})
		return
	}
	next, err := normalizeSetup(req, current)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	options := map[string]any{
		"country":            next.Country,
		"refresh_minutes":    next.RefreshMinutes,
		"enable_ha_entities": next.EnableHAEntities,
		"enable_weather":     next.EnableWeather,
		"agsi_key":           next.AGSIKey,
		"entsoe_token":       next.ENTSOEToken,
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	if err := supervisorPost(ctx, "/addons/self/options", map[string]any{"options": options}); err != nil {
		log.Printf("dashboard setup save failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Home Assistant Supervisor rejected or could not save the configuration"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"saved": true, "restarting": true})
	go func() {
		time.Sleep(900 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		if err := supervisorPost(ctx, "/addons/self/restart", nil); err != nil {
			log.Printf("dashboard requested restart failed: %v", err)
		}
	}()
}

func supervisorPost(ctx context.Context, path string, payload any) error {
	token := strings.TrimSpace(os.Getenv("SUPERVISOR_TOKEN"))
	if token == "" {
		return errors.New("Supervisor API token is unavailable")
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://supervisor"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return errors.New("Supervisor API request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Supervisor API returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Result  string `json:"result"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("decode Supervisor response: %w", err)
	}
	if result.Result != "" && result.Result != "ok" {
		return errors.New("Supervisor API returned an error")
	}
	return nil
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.refreshing.CompareAndSwap(false, true) {
		writeJSON(w, http.StatusAccepted, map[string]any{"accepted": false, "reason": "refresh already running"})
		return
	}
	go func() {
		defer s.refreshing.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if err := s.app.Refresh(ctx); err != nil {
			log.Printf("manual refresh failed: %v", err)
		}
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'self'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ingressGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(strings.TrimSpace(host))
		allowed := ip != nil && (ip.IsLoopback() || ip.String() == "172.30.32.2")
		if !allowed {
			http.Error(w, "ingress only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
