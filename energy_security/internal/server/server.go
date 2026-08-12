package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
)

//go:embed web/*
var webFiles embed.FS

type Server struct {
	app        *app.App
	mux        *http.ServeMux
	refreshing atomic.Bool
	started    time.Time
}

func New(a *app.App) *Server {
	s := &Server{app: a, mux: http.NewServeMux(), started: time.Now().UTC()}
	assets, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	s.mux.HandleFunc("GET /api/v1/status", s.status)
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("POST /api/v1/refresh", s.refresh)
	s.mux.Handle("/", http.FileServer(http.FS(assets)))
	return s
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(s.ingressGuard(s.mux))
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
	})
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
