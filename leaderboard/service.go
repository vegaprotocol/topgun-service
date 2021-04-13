package leaderboard

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/pricing"
	"github.com/vegaprotocol/topgun-service/util"
	"github.com/vegaprotocol/topgun-service/verifier"

	ppconfig "code.vegaprotocol.io/priceproxy/config"
	ppservice "code.vegaprotocol.io/priceproxy/service"
	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

// GraphQL query response structs

type Asset struct {
	Symbol string `json:"symbol"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
}

type Party struct {
	ID       string    `json:"id"`
	Accounts []Account `json:"accounts"`

	social string
}

type AllParties struct {
	Parties []Party `json:"parties"`
}

// // Guide to the base/quote/asset vars
// // ==================================
// // base => BTC (for price provider)
// // quote => USD (for price provider/calculation)
// // vegaAsset => tDAI (for asset balance calculation only)

// func (p *Party) TotalGeneral(asset string, assetPrice float64) float64 {
// 	return p.Balance(asset, "General") * assetPrice
// }

// func (p *Party) TotalMargin(asset string, assetPrice float64) float64 {
// 	return p.Balance(asset, "Margin") * assetPrice
// }

// func (p *Party) Total(asset string, assetPrice float64) float64 {
// 	return p.TotalGeneral(asset, assetPrice) + p.TotalMargin(asset, assetPrice)
// }

func (p *Party) Balance(assetName string, accountType string) float64 {
	for _, acc := range p.Accounts {
		if acc.Asset.Symbol == assetName && acc.Type == accountType {
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/%s balance [Balance]", assetName, accountType)
				return 0
			}
			return v / float64(100000)
		}
	}
	return 0
}

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
	included := s.verifier.PubKeysToTwitterHandles()
	// If no verified pub-key->social-handles found, no need to query Vega
	if len(included) == 0 {
		return
	}

	// Load all parties with accounts from GraphQL end-point
	ctx := context.Background()
	parties, err := s.getParties(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get list of parties")
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

	parties = filterParties(parties, included)

	switch s.cfg.Algorithm {
	case "ByAsset":
		newBoard.Participants = s.sortByAsset(parties)
	default:
		log.WithFields(log.Fields{"algorithm": s.cfg.Algorithm}).Warn("Invalid algorithm")
		newBoard.Participants = []Participant{}
	}

	s.board = newBoard

	log.Infof("Leaderboard updated [%s]: %d participants out of %d available",
		s.board.LastUpdate, len(s.board.Participants), len(parties))
}

func (s *Service) getParties(ctx context.Context) ([]Party, error) {
	client := graphql.NewClient(s.cfg.VegaGraphQLURL.String())
	req := graphql.NewRequest("query {parties {id accounts {type balance asset {symbol}}}}")
	req.Header.Set("Cache-Control", "no-cache")

	var response AllParties
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.Parties, nil
}

func (s *Service) GetLeaderboard() Leaderboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.board
}

// filterParties filters the parties list and returns only those listed in the
// social map, which has been fetched from the external verifier service
func filterParties(parties []Party, socials map[string]string) []Party {
	filteredParties := []Party{}
	for _, party := range parties {
		if social, found := socials[party.ID]; found {
			party.social = social
			filteredParties = append(filteredParties, party)
		}
	}
	return filteredParties
}
