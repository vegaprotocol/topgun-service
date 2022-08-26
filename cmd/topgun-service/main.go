package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)
	defer cancel()

	web.Stop()

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

func GetQuery(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func GetQueryInt(r *http.Request, key string) int64 {
	q := GetQuery(r, key)
	if len(q) > 0 {
		i, err := strconv.ParseInt(q, 10, 64)
		if err != nil {
			log.Warnf("Could not parse query string param %s %s to int", key, q)
			return -1
		}
		return i
	}
	return -1
}

func EndpointLeaderboard(w http.ResponseWriter, r *http.Request, svc *leaderboard.Service) {
	responseType := GetQuery(r, "type")
	if strings.ToLower(responseType) == "csv" {
		w.Header().Set("Content-Type", "text/plain")
		q := GetQuery(r, "q")
		skip := GetQueryInt(r, "skip")
		size := GetQueryInt(r, "size")
		blacklisted := strings.ToLower(GetQuery(r, "blacklisted")) == "true"
		payload, err := svc.CsvLeaderboard(q, skip, size, blacklisted)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error marshaling leaderboard")
			payload = []byte(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Write(payload)
	} else {
		w.Header().Set("Content-Type", "application/json")
		q := GetQuery(r, "q")
		skip := GetQueryInt(r, "skip")
		size := GetQueryInt(r, "size")
		blacklisted := strings.ToLower(GetQuery(r, "blacklisted")) == "true"
		payload, err := svc.JsonLeaderboard(q, skip, size, blacklisted)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error marshaling leaderboard")

			payload, err = json.Marshal(ErrorObject{Error: err.Error()})
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Error marshaling error message during marshaling of leaderboard")
				payload = []byte("{\"error\":\"\"}")
			}
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Write(payload)
	}
}


