package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/vegaprotocol/topgun-service/api"
	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/leaderboard"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg := loadConfig()

	svc := leaderboard.NewLeaderboardService(cfg)
	web := api.NewAPIService(cfg, svc)

	// Run the leaderboard service in its own goroutine
	go func() {
		svc.Start()
		web.Start()
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	web.Stop()

	// Signal to stop the leaderboard service
	svc.Stop()

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("Stopping server")
	os.Exit(0)
}

func loadConfig() config.Config {
	// Command line flags
	var configName string
	flag.StringVar(&configName, "config", "", "Configuration YAML file")
	flag.Parse()
	if len(configName) == 0 {
		fmt.Println("Missing commandline argument: -config configfile.yaml")
		os.Exit(1)
	}

	var cfg config.Config
	err := configor.Load(&cfg, configName)
	// https://github.com/jinzhu/configor/issues/40
	if err != nil && !strings.Contains(err.Error(), "should be struct") {
		fmt.Printf("Failed to read config: %v", err)
		os.Exit(1)
	}

	err = config.CheckConfig(cfg)
	if err != nil && !strings.Contains(err.Error(), "should be struct") {
		fmt.Printf("Invalid config: %v", err)
		os.Exit(1)
	}

	// Logger config
	err = config.ConfigureLogging(cfg)
	if err != nil && !strings.Contains(err.Error(), "should be struct") {
		fmt.Printf("Invalid logging config: %v", err)
		os.Exit(1)
	}
	log.WithFields(cfg.LogFields()).Info("Starting server")

	return cfg
}

type ErrorObject struct {
	Error string `json:"error"`
}
