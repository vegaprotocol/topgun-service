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
	router.HandleFunc("/", EndpointRoot)
	router.HandleFunc("/status", EndpointStatus)
	router.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		EndpointLeaderboard(w, r, svc)
	})
	router.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		EndpointLeaderboard(w, r, svc)
	}).Queries("q", "{q}")

	srv := &http.Server{
		Addr:         cfg.Listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router),
	}

	// Run the leaderboard service in its own goroutine
	go func() {
		svc.Start()
	}()
	// Run the web server in its own goroutine
	go func() {
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

type ErrorObject struct {
	Error string `json:"error"`
}

func EndpointLeaderboard(w http.ResponseWriter, r *http.Request, svc *leaderboard.Service) {
	w.Header().Set("Content-Type", "application/json")
	q := r.URL.Query().Get("q")
	payload, err := svc.MarshalLeaderboard(q)
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

func EndpointStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"success\":true}"))
}

func EndpointRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	content := `<!doctype html>
<head>
<title>Topgun Service</title>
</head>
<body>
<h1>Topgun Service</h1>
<ul>
<li><a href="/status">Status</a></li>
<li><a href="/leaderboard">Leaderboard</a></li>
</ul>
</body>
</html>`
	w.Write([]byte(content))
}
