package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/leaderboard"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func startServer(
	addr string,
	wait time.Duration,
	endpoint string,
	vegaPoll time.Duration,
	base string,
	quote string,
	vegaAsset string,
	verifierUrl string,
) {
	log.Info("Starting up API server")

	vfs := verifier.NewVerifierService(verifierUrl)
	svc := leaderboard.NewLeaderboardService(endpoint, vegaPoll, base, quote, vegaAsset, vfs)

	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		lb := svc.GetLeaderboard()
		payload, err := json.Marshal(lb)
		if err != nil {
			log.WithError(err).Error("Error marshaling response")
		} else {
			w.Write(payload)
		}
	})

	srv := &http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(router),
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		svc.Start()
		if err := srv.ListenAndServe(); err != nil {
			log.Warn(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Signal to stop the leaderboard service
	svc.Stop()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("Shutting down API server")
	os.Exit(0)

	log.Fatal(srv.ListenAndServe())
}
