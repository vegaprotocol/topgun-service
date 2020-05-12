package leaderboard

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/machinebox/graphql"

	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/exchange"
	"github.com/vegaprotocol/topgun-service/util"
)

type Response struct {
	Parties []Party `json:"parties"`
}

type Party struct {
	ID       string `json:"id"`
	Accounts []Account
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
	var total float64
	total = 0
	for _, acc := range p.Accounts {
		if acc.Asset == assetName && acc.Type == assetType {
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
		if acc.Asset == assetName && acc.Type == assetType {
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

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   string `json:"asset"`
}

type Participant struct {
	PartyID              string
	BalanceUSD           float64
	BalanceBTC           float64
	DeployedUSD          float64
	DeployedBTC          float64
	TotalUSD             float64
	TotalUSDWithDeployed float64
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
		table:    []Participant{},
	}
	svc.exchange = exchange.NewExchangeService(assetPoll)
	return svc
}

type Service struct {
	endpoint string
	exchange *exchange.Service
	table    []Participant
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

	s.table = []Participant{}

	// Get latest BTC USD price value
	btcAsset := s.exchange.GetBtcUsdPrice()
	btcAssetLastPrice := btcAsset.LastPriceValue()

	for _, p := range res.Parties {
		// Only include whitelisted partyIDs
		if _, found := s.included[p.ID]; found {
			s.table = append(s.table, Participant{
				PartyID:              p.ID,
				BalanceUSD:           p.BalanceUSD(),
				BalanceBTC:           p.BalanceBTC(),
				DeployedUSD:          p.DeployedUSD(),
				DeployedBTC:          p.DeployedBTC(),
				TotalUSD:             p.TotalUSD(btcAssetLastPrice),
				TotalUSDWithDeployed: p.TotalUSDWithDeployed(btcAssetLastPrice),
			})
		}
	}

	// Sort the leaderboard table
	sort.Slice(s.table, func(i, j int) bool {
		return s.table[i].TotalUSDWithDeployed > s.table[j].TotalUSDWithDeployed
	})

	log.Infof("Leaderboard updated: %d participants included out of %d available", len(s.table), len(res.Parties))
}

func (s *Service) performQuery(ctx context.Context) (*Response, error) {
	client := graphql.NewClient(s.endpoint)
	req := graphql.NewRequest(`
    query {
       parties {
          id
          accounts { type, balance, asset }
       }
    }
`)
	req.Header.Set("Cache-Control", "no-cache")

	var resp Response
	if err := client.Run(ctx, req, &resp); err != nil {
		return &Response{}, err
	}
	return &resp, nil
}

func (s *Service) GetLeaderboard() []Participant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.table
}
