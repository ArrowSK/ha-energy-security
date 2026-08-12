package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ArrowSK/ha-energy-security/energy_security/internal/app"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/config"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/server"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "/data/options.json", "path to Home Assistant app options.json")
	dataDir := flag.String("data-dir", "/data", "persistent data directory")
	listen := flag.String("listen", ":8099", "dashboard listen address")
	selfTest := flag.Bool("self-test", false, "validate embedded configuration and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg := config.Defaults()
	if !*selfTest {
		var err error
		cfg, err = config.Load(*configPath)
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
	log.Printf("Energy Security Monitor %s listening on %s", version, *listen)
	if err := server.New(a).ListenAndServe(ctx, *listen); err != nil {
		log.Fatalf("dashboard server: %v", err)
	}
}
