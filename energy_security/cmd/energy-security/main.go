package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/server"
)

var version = "dev"

func main() {
	defaultMode := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_MODE"))
	if defaultMode == "" {
		defaultMode = "home_assistant"
	}
	defaultDataDir := strings.TrimSpace(os.Getenv("ENERGY_SECURITY_DATA_DIR"))
	if defaultDataDir == "" {
		defaultDataDir = "/data"
	}
	defaultListen := ":8099"
	if strings.EqualFold(defaultMode, "standalone") {
		port := strings.TrimSpace(os.Getenv("PORT"))
		if port == "" {
			port = "8080"
		}
		defaultListen = "0.0.0.0:" + port
	}

	mode := flag.String("mode", defaultMode, "runtime mode: home_assistant or standalone")
	configPath := flag.String("config", "/data/options.json", "path to Home Assistant app options.json")
	dataDir := flag.String("data-dir", defaultDataDir, "persistent data directory")
	listen := flag.String("listen", defaultListen, "dashboard listen address")
	selfTest := flag.Bool("self-test", false, "validate embedded configuration and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg := config.Defaults()
	runtimeMode := strings.ToLower(strings.TrimSpace(*mode))
	if !*selfTest {
		var err error
		switch runtimeMode {
		case "home_assistant", "ha":
			runtimeMode = "home_assistant"
			cfg, err = config.Load(*configPath)
		case "standalone", "docker", "railway":
			runtimeMode = "standalone"
			cfg, err = config.LoadEnvironment()
		default:
			log.Fatalf("configuration: unsupported runtime mode %q", *mode)
		}
		if err != nil {
			log.Fatalf("configuration: %v", err)
		}
	}

	a := app.New(cfg, *dataDir)
	if err := a.SelfTest(); err != nil {
		log.Fatalf("self-test: %v", err)
	}
	if *selfTest {
		fmt.Printf("energy-security %s: self-test OK\n", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go a.Run(ctx)
	log.Printf("Energy Security Monitor %s (%s) listening on %s", version, runtimeMode, *listen)
	srv := server.New(a)
	if runtimeMode == "standalone" {
		srv = server.NewStandalone(a, cfg)
	}
	if err := srv.ListenAndServe(ctx, *listen); err != nil {
		log.Fatalf("dashboard server: %v", err)
	}
}
