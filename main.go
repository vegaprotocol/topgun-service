package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/leaderboard"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"
)

func main() {
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

	svc := leaderboard.NewLeaderboardService(cfg)

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		lb := svc.GetLeaderboard()
		payload, err := json.Marshal(lb)
		if err != nil {
			log.WithError(err).Error("Error marshaling response")
			return
		}
		w.Write(payload)
	})

	srv := &http.Server{
		Addr:         cfg.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router),
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		svc.Start()
		if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.WithError(err).Warn("Failed to serve")
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)
	defer cancel()

	// Signal to stop the leaderboard service
	svc.Stop()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("Stopping server")
	os.Exit(0)
}
