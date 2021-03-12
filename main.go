package main

import (
	"flag"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/util"
)

var (
	addr      string
	includelist string
	timeout   time.Duration
	endpoint  string
	assetpoll time.Duration
	vegapoll  time.Duration
)

func init() {
	flag.StringVar(&includelist, "includelist", "", "the path to the csv file containing partyIDs to includelist for the leaderboard e.g. bots")
	flag.StringVar(&addr, "addr", "localhost:8000", "address:port to bind the service to")
	flag.StringVar(&endpoint, "endpoint", "", "endpoint url to send graphql queries to")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "the duration for which the server gracefully waits for existing connections to finish - e.g. 15s or 1m")
	flag.DurationVar(&assetpoll, "assetpoll", time.Second*30, "the duration for which the service will poll the exchange for asset price. Default: 30s")
	flag.DurationVar(&vegapoll, "vegapoll", time.Second*5, "the duration for which the service will poll the Vega API for accounts. Default: 5s")
}

func main() {
	// Logger config
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// Command line flags
	flag.Parse()
	if len(addr) <= 0 {
		log.Printf("Error: missing 'addr' flag (the address:port to bind the accounts service to)")
		return
	}
	if len(endpoint) <= 0 {
		log.Printf("Error: missing 'endpoint' flag (endpoint url to send graphql queries to)")
		return
	}
	if len(includelist) <= 0 {
		log.Printf("Error: missing 'includelist' flag (the path to the csv file containing partyIDs to includelist for the leaderboard)")
		return
	}

	//Load list of included parties e.g. bots etc
	included, err := util.LoadPartiesFromCsv(includelist)
	if err != nil {
		log.WithError(err).Fatal("Fatal error loading excluded parties from csv")
	}

	startServer(addr, timeout, endpoint, vegapoll, assetpoll, included)
}
