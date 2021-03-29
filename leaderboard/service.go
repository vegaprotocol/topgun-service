package leaderboard

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/vegaprotocol/topgun-service/pricing"
	"github.com/vegaprotocol/topgun-service/util"
	"github.com/vegaprotocol/topgun-service/verifier"

	ppconfig "code.vegaprotocol.io/priceproxy/config"
	ppservice "code.vegaprotocol.io/priceproxy/service"
	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

type AllParties struct {
	Parties []Party `json:"parties"`
}

type Party struct {
	ID       string `json:"id"`
	Accounts []Account
}

// Guide to the base/quote/asset vars
// ==================================
// base => BTC (for price provider)
// quote => USD (for price provider/calculation)
// vegaAsset => tDAI (for asset balance calculation only)

func (p *Party) TotalGeneral(asset string, assetPrice float64) float64 {
	return p.Balance(asset, "General") * assetPrice
}

func (p *Party) TotalMargin(asset string, assetPrice float64) float64 {
	return p.Balance(asset, "Margin") * assetPrice
}

func (p *Party) Total(asset string, assetPrice float64) float64 {
	return p.TotalGeneral(asset, assetPrice) + p.TotalMargin(asset, assetPrice)
}

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

type Asset struct {
	Symbol string `json:"symbol"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
}

type Leaderboard struct {
	LastUpdate string        `json:"lastUpdate"`
	Base       string        `json:"base"`
	Quote      string        `json:"quote"`
	Asset      string        `json:"asset"`
	Traders    []Participant `json:"traders"`
}

type Participant struct {
	Order                   uint64  `json:"order"`
	PublicKey               string  `json:"publicKey"`
	TwitterHandle           string  `json:"twitterHandle"`
	BalanceGeneral          float64 `json:"balanceGeneral"`
	BalanceMargin           float64 `json:"balanceMargin"`
	BalanceTotal            float64 `json:"balanceTotal"`
	QuoteGeneral            float64 `json:"quoteGeneral"`
	QuoteMargin             float64 `json:"quoteMargin"`
	QuoteTotal              float64 `json:"quoteTotal"`
}

func NewLeaderboardService(
	endpoint string,
	vegaPoll time.Duration,
	base string,
	quote string,
	vegaAsset string,
	verifier *verifier.Service,
) *Service {
	svc := &Service{
		base:      base,
		quote:     quote,
		vegaAsset: vegaAsset,
		endpoint:  endpoint,
		poll:      vegaPoll,
		verifier:  verifier,
		board: Leaderboard{
			Base:       base,
			Quote:      quote,
			Asset:      vegaAsset,
			LastUpdate: util.UnixTimestampUtcNowFormatted(),
			Traders:    []Participant{},
		},
	}
	u := url.URL{
		Scheme: "https",
		Host:   "prices.ops.vega.xyz",
		Path:   "/prices",
	}
	svc.pricingEngine = pricing.NewEngine(u)
	return svc
}

type Service struct {
	base          string
	quote         string
	vegaAsset     string
	endpoint      string
	pricingEngine PricingEngine
	timer         *time.Ticker
	board         Leaderboard
	poll          time.Duration
	mu            sync.RWMutex
	verifier      *verifier.Service
}

func (s *Service) Start() {
	log.Info("Leaderboard service started")
	s.update()
	s.timer = util.Schedule(s.update, s.poll)
}

func (s *Service) Stop() {
	s.timer.Stop()
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
	res, err := s.performQuery(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to perform query")
		return
	}

	s.board = Leaderboard{
		LastUpdate: util.UnixTimestampUtcNowFormatted(),
		Base:       s.base,
		Quote:      s.quote,
		Asset:      s.vegaAsset,
		Traders:    []Participant{},
	}

	// Get latest Base Quote price value
	pc := ppconfig.PriceConfig{
		Base:   s.base,
		Quote:  s.quote,
		Wander: true,
	}
	response, err := s.pricingEngine.GetPrice(pc)
	if err != nil {
		log.Warnf("Failed to update leaderboard: %s", err.Error())
	}
	lastPrice := response.Price

	for _, p := range res.Parties {
		// Only include verified pub-keys from the external verifier API service
		// Return their twitter handle if they exist
		if twitterHandle, found := included[p.ID]; found {

			balanceGeneral := p.Balance(s.vegaAsset, "General")
			balanceMargin := p.Balance(s.vegaAsset, "Margin")

			s.board.Traders = append(s.board.Traders, Participant{
				PublicKey:      p.ID,
				TwitterHandle:  twitterHandle,
				BalanceGeneral: balanceGeneral,
				BalanceMargin:  balanceMargin,
				QuoteGeneral:   p.TotalGeneral( s.vegaAsset, lastPrice),
				QuoteMargin:    p.TotalMargin(s.vegaAsset, lastPrice),
				QuoteTotal:     p.Total(s.vegaAsset, lastPrice),
				BalanceTotal:   balanceMargin + balanceGeneral,
			})
		}
	}

	// Sort the leaderboard table
	sort.Slice(s.board.Traders, func(i, j int) bool {
		return s.board.Traders[i].BalanceTotal > s.board.Traders[j].BalanceTotal
	})

	// Set order value
	var rank uint64 = 1
	for i := range s.board.Traders {
		s.board.Traders[i].Order = rank
		rank++
	}

	log.Infof("Leaderboard updated [%s]: %d participants out of %d available",
		s.board.LastUpdate, len(s.board.Traders), len(res.Parties))
}

func (s *Service) performQuery(ctx context.Context) (*AllParties, error) {
	client := graphql.NewClient(s.endpoint)
	req := graphql.NewRequest(`
    query {
       parties {
          id
          accounts { type balance asset { symbol } }
       }
    }
`)
	req.Header.Set("Cache-Control", "no-cache")

	var resp AllParties
	if err := client.Run(ctx, req, &resp); err != nil {
		return &AllParties{}, err
	}
	return &resp, nil
}

func (s *Service) GetLeaderboard() Leaderboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.board
}
