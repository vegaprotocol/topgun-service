package main

import (
	"flag"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	addr        string
	base        string
	quote       string
	vegaasset   string
	verify      string
	timeout     time.Duration
	endpoint    string
	vegapoll    time.Duration
)

func init() {
	flag.StringVar(&verify, "verifyurl", "", "the http/web URL for the 3rd party social handle to pubkey verifier API service")
	flag.StringVar(&addr, "addr", "localhost:8000", "address:port to bind the service to")
	flag.StringVar(&endpoint, "endpoint", "", "endpoint url to send graphql queries to")
	flag.DurationVar(&timeout, "timeout", time.Second*15, "the duration for which the server gracefully waits for existing connections to finish - e.g. 15s or 1m")
	flag.DurationVar(&vegapoll, "vegapoll", time.Second*5, "the duration for which the service will poll the Vega API for accounts. Default: 5s")
	flag.StringVar(&vegaasset, "vegaasset", "", "Vega asset, e.g. tDAI")
	flag.StringVar(&base, "base", "", "base for prices")
	flag.StringVar(&quote, "quote", "", "quote for prices")
}

func main() {
	// Logger config
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// Command line flags
	flag.Parse()
	if len(base) <= 0 {
		log.Printf("Error: missing 'base' flag")
		return
	}
	if len(quote) <= 0 {
		log.Printf("Error: missing 'quote' flag")
		return
	}
	if len(vegaasset) <= 0 {
		log.Printf("Error: missing 'vegaasset' flag")
		return
	}
	if len(addr) <= 0 {
		log.Printf("Error: missing 'addr' flag (the address:port to bind the accounts service to)")
		return
	}
	if len(endpoint) <= 0 {
		log.Printf("Error: missing 'endpoint' flag (endpoint url to send graphql queries to)")
		return
	}
	if len(verify) <= 0 {
		log.Printf("Error: missing 'verifier' flag (url to download verified social->pubkey mapping from)")
		return
	}

	startServer(addr, timeout, endpoint, vegapoll, base, quote, vegaasset, verify)
}
