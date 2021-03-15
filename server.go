package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/leaderboard"
)

func startServer(
	addr string,
	wait time.Duration,
	endpoint string,
	vegaPoll time.Duration,
	assetPoll time.Duration,
	included map[string]byte,
	base, quote, vegaAsset string,
) {
	log.Info("Starting up API server")

	svc := leaderboard.NewLeaderboardService(endpoint, vegaPoll, assetPoll, included, base, quote, vegaAsset)
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		quotes := []string{
			"Sorry, Goose, but it's time to buzz the tower.",
			"You’re everyone’s problem. That’s because every time you go up in the air you’re unsafe." +
				" I don’t like you because you’re dangerous.",
			"That's right, Iceman. I am dangerous.",
			"It's classified. I could tell you, but then I'd have to kill you.",
			"Because I was inverted.",
			"Son, your ego is writing checks your body can't cash.",
			"I feel the need -- the need for speed.",
			"You can be my wing-man any time.",
			"This could be complicated. You know on the first one I crashed and burned.",
		}
		rand.Seed(time.Now().Unix())
		w.Write([]byte(quotes[rand.Intn(len(quotes))]))
	})
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
