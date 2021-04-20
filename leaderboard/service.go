package leaderboard

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/pricing"
	"github.com/vegaprotocol/topgun-service/util"
	"github.com/vegaprotocol/topgun-service/verifier"

	ppconfig "code.vegaprotocol.io/priceproxy/config"
	ppservice "code.vegaprotocol.io/priceproxy/service"
	log "github.com/sirupsen/logrus"
)

// PricingEngine is the source of price information from the price proxy.
type PricingEngine interface {
	GetPrice(pricecfg ppconfig.PriceConfig) (pi ppservice.PriceResponse, err error)
}

type Participant struct {
	PublicKey     string   `json:"publicKey"`
	TwitterHandle string   `json:"twitterHandle"`
	Data          []string `json:"data"`

	sortNum float64
}

type Leaderboard struct {
	Version        int           `json:"version"`
	Base           string        `json:"base"`
	Quote          string        `json:"quote"`
	Asset          string        `json:"asset"`
	LastUpdate     string        `json:"lastUpdate"`
	Headers        []string      `json:"headers"`
	Description    string        `json:"description"`
	DefaultSort    string        `json:"defaultSort"`
	DefaultDisplay string        `json:"defaultDisplay"`
	Participants   []Participant `json:"participants"`
}

func NewLeaderboardService(cfg config.Config) *Service {
	svc := &Service{
		cfg: cfg,
		pricingEngine: pricing.NewEngine(url.URL{
			Scheme: "https",
			Host:   "prices.ops.vega.xyz",
			Path:   "/prices",
		}),
		verifier: verifier.NewVerifierService(*cfg.SocialURL),
	}
	return svc
}

type Service struct {
	cfg config.Config

	pricingEngine PricingEngine
	timer         *time.Ticker
	board         Leaderboard
	mu            sync.RWMutex
	verifier      *verifier.Service
}

func (s *Service) Start() {
	log.Info("Leaderboard service started")
	s.update()
	s.timer = util.Schedule(s.update, s.cfg.VegaPoll)
}

func (s *Service) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
	log.Info("Leaderboard service stopped")
}

func (s *Service) update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Attempt to update parties from external social verifier service
	// Safe approach, will only overwrite internal collection if successful
	s.verifier.UpdateVerifiedParties()
	// Grab a map of the verified pub-key->twitter-handle for leaderboard
	socials := s.verifier.PubKeysToTwitterHandles()
	// If no verified pub-key->social-handles found, no need to query Vega
	if len(socials) == 0 {
		return
	}

	newBoard := Leaderboard{
		Version:        1,
		Base:           s.cfg.Base,
		Quote:          s.cfg.Quote,
		Asset:          s.cfg.VegaAsset,
		DefaultDisplay: s.cfg.DefaultDisplay,
		DefaultSort:    s.cfg.DefaultSort,
		Description:    s.cfg.Description,
		Headers:        s.cfg.Headers,
		LastUpdate:     util.UnixTimestampUtcNowFormatted(),
	}

	// Get latest Base Quote price value
	// pc := ppconfig.PriceConfig{
	// 	Base:   s.cfg.Base,
	// 	Quote:  s.cfg.Quote,
	// 	Wander: true,
	// }
	// response, err := s.pricingEngine.GetPrice(pc)
	// if err != nil {
	// 	log.Warnf("Failed to update leaderboard: %s", err.Error())
	// }
	// lastPrice := response.Price

	var p []Participant
	var err error
	switch s.cfg.Algorithm {
	case "ByPartyAccountGeneralBalance":
		p, err = s.sortByPartyAccountGeneralBalance(socials)
	case "ByPartyGovernanceVotes":
		p, err = s.sortByPartyGovernanceVotes(socials)
	case "ByLPEquitylikeShare":
		p, err = s.sortByLPEquitylikeShare(socials)
	default:
		err = fmt.Errorf("invalid algorithm: %s", s.cfg.Algorithm)
	}
	if err != nil {
		log.WithError(err).Warn("Failed to sort")
		p = []Participant{}
	}
	newBoard.Participants = p
	s.board = newBoard
	log.WithFields(log.Fields{"participants": len(s.board.Participants)}).Info("Leaderboard updated")
}

func (s *Service) GetLeaderboard() Leaderboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.board
}
