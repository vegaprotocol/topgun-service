package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/exchange"
	"github.com/vegaprotocol/topgun-service/util"
)

type AllParties struct {
	Parties []Party `json:"parties"`
}

type Party struct {
	ID       string `json:"id"`
	Accounts []Account
}

func (p *Party) NotTraded() bool {
	return p.DeployedBTC() == 0 && p.BalanceBTC() == 10 &&
		p.BalanceUSD() == 5000 && p.DeployedUSD() == 0
}

func (p *Party) DeployedUSD() float64 {
	return p.calculateDeployed("VUSD", "Margin")
}

func (p *Party) DeployedBTC() float64 {
	return p.calculateDeployed("BTC", "Margin")
}

func (p *Party) BalanceUSD() float64 {
	return p.calculateBalance("VUSD", "General")
}

func (p *Party) BalanceBTC() float64 {
	return p.calculateBalance("BTC", "General")
}

func (p *Party) TotalUSD(btcPrice float64) float64 {
	return (p.BalanceBTC() * btcPrice) + p.BalanceUSD()
}

func (p *Party) TotalUSDWithDeployed(btcPrice float64) float64 {
	return (p.BalanceBTC() * btcPrice) + (p.DeployedBTC() * btcPrice) + p.BalanceUSD() + p.DeployedUSD()
}

func (p *Party) calculateDeployed(assetName string, assetType string) float64 {
	var total float64 = 0
	for _, acc := range p.Accounts {
		if acc.Asset.Symbol == assetName && acc.Type == assetType {
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/% balance [calculateDeployed]", assetName, assetType)
			}
			total += v / float64(100000)
		}
	}
	return total
}

func (p *Party) calculateBalance(assetName string, assetType string) float64 {
	for _, acc := range p.Accounts {
		if acc.Asset.Symbol == assetName && acc.Type == assetType {
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/% balance [calculateBalance]", assetName, assetType)
				return 0
			}
			return v / float64(100000)
		}
	}
	return 0
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
	Traders    []Participant `json:"traders"`
}

type Participant struct {
	Order                     uint64  `json:"order"`
	PublicKey                 string  `json:"publicKey"`
	BalanceUSD                float64 `json:"usdVal"`
	BalanceUSDFormatted       string  `json:"usd"`
	BalanceBTC                float64 `json:"btcVal"`
	BalanceBTCFormatted       string  `json:"btc"`
	DeployedUSD               float64 `json:"usdDeployedVal"`
	DeployedUSDFormatted      string  `json:"usdDeployed"`
	DeployedBTC               float64 `json:"btcDeployedVal"`
	DeployedBTCFormatted      string  `json:"btcDeployed"`
	TotalUSD                  float64 `json:"totalUsdVal"`
	TotalUSDFormatted         string  `json:"totalUsd"`
	TotalUSDDeployed          float64 `json:"totalUsdDeployedVal"`
	TotalUSDDeployedFormatted string  `json:"totalUsdDeployed"`
}

func NewLeaderboardService(
	endpoint string,
	vegaPoll time.Duration,
	assetPoll time.Duration,
	included map[string]byte) *Service {
	svc := &Service{
		included: included,
		endpoint: endpoint,
		poll:     vegaPoll,
		board:    Leaderboard{util.UnixTimestampUtcNowFormatted(), []Participant{}},
	}
	svc.exchange = exchange.NewExchangeService(assetPoll)
	return svc
}

type Service struct {
	endpoint string
	exchange *exchange.Service
	board    Leaderboard
	included map[string]byte
	timer    *time.Ticker
	poll     time.Duration
	mu       sync.RWMutex
}

func (s *Service) Start() {
	log.Info("Leaderboard service started")
	s.exchange.Start() // Try and get the latest asset pricing from exchange immediately
	s.timer = util.Schedule(s.update, s.poll)
}

func (s *Service) Stop() {
	s.exchange.Stop()
	s.timer.Stop()
	log.Info("Leaderboard service stopped")
}

func (s *Service) update() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load all parties with accounts from GraphQL end-point
	ctx := context.Background()
	res, err := s.performQuery(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to perform query")
		return
	}

	s.board = Leaderboard{util.UnixTimestampUtcNowFormatted(), []Participant{}}

	// Get latest BTC USD price value
	btcAsset := s.exchange.GetBtcUsdPrice()
	btcAssetLastPrice := btcAsset.LastPriceValue()

	for _, p := range res.Parties {
		// Only include whitelisted partyIDs
		if _, found := s.included[p.ID]; found {

			// Requirement from @edd to not included parties with no trading in the leaderboard at the service end
			// If in the future we want to filter these at the client we can remove this check.
			if p.NotTraded() {
				continue
			}

			s.board.Traders = append(s.board.Traders, Participant{
				PublicKey:                 p.ID,
				BalanceUSD:                p.BalanceUSD(),
				BalanceUSDFormatted:       fmt.Sprintf("%.5f", p.BalanceUSD()),
				BalanceBTC:                p.BalanceBTC(),
				BalanceBTCFormatted:       fmt.Sprintf("%.5f", p.BalanceBTC()),
				DeployedUSD:               p.DeployedUSD(),
				DeployedUSDFormatted:      fmt.Sprintf("%.5f", p.DeployedUSD()),
				DeployedBTC:               p.DeployedBTC(),
				DeployedBTCFormatted:      fmt.Sprintf("%.5f", p.DeployedBTC()),
				TotalUSD:                  p.TotalUSD(btcAssetLastPrice),
				TotalUSDFormatted:         fmt.Sprintf("%.5f", p.TotalUSD(btcAssetLastPrice)),
				TotalUSDDeployed:          p.TotalUSDWithDeployed(btcAssetLastPrice),
				TotalUSDDeployedFormatted: fmt.Sprintf("%.5f", p.TotalUSDWithDeployed(btcAssetLastPrice)),
			})
		}
	}

	// Sort the leaderboard table
	sort.Slice(s.board.Traders, func(i, j int) bool {
		return s.board.Traders[i].TotalUSDDeployed > s.board.Traders[j].TotalUSDDeployed
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
          accounts {type balance asset { symbol } }
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
