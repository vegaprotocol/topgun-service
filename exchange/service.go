package exchange

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vegaprotocol/topgun-service/util"
)

const BtcUsdAssetId = "BTC:USD"

func NewExchangeService(assetPoll time.Duration) *Service {
	return &Service{
		btcUsdPrice: fallBackBtcUsdAsset(),
		poll:        assetPoll,
	}
}

func (s *Service) GetBtcUsdPrice() AssetPrice {
	return s.btcUsdPrice
}

type Service struct {
	timer       *time.Ticker
	mu          sync.Mutex
	poll        time.Duration
	btcUsdPrice AssetPrice
}

type AssetPrice struct {
	Asset     string `json:"asset"`
	High      string `json:"high"`
	Last      string `json:"last"`
	Timestamp string `json:"timestamp"`
	Bid       string `json:"bid"`
	Vwap      string `json:"vwap"`
	Volume    string `json:"volume"`
	Low       string `json:"low"`
	Ask       string `json:"ask"`
	Open      string `json:"open"`
}

func (a *AssetPrice) LastPriceValue() float64 {
	v, err := strconv.ParseFloat(a.Last, 64)
	if err != nil {
		log.WithError(err).Errorf(
			"Failed to parse Last Asset Price for %s", a.Asset)
		return 0
	}
	log.Debug("LastPriceValue: ", v)
	return v
}

func (s *Service) Start() {
	log.Info("Exchange price polling started")
	s.updateBtcPrice()
	s.timer = util.Schedule(s.updateBtcPrice, s.poll)
}

func (s *Service) Stop() {
	s.timer.Stop()
	log.Info("Exchange price polling stopped")
}

func (s *Service) updateBtcPrice() {
	s.mu.Lock()
	defer s.mu.Unlock()

	url := "https://www.bitstamp.net/api/v2/ticker/btcusd"

	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithError(err).Error("Failed to create new http request")
	}

	req.Header.Set("User-Agent", "topgun-maverick")

	res, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("Failed to load data from bitstamp price feed")
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.WithError(err).Error("Failed to read the response body from price feed")
	}

	price := AssetPrice{}
	err = json.Unmarshal(body, &price)
	if err != nil {
		log.WithError(err).Error("Failed to decode/unmarshal asset price information from response body")
	}
	price.Asset = BtcUsdAssetId
	s.btcUsdPrice = price

	log.Infof("Asset price updated: [%s] %s", price.Asset, price.Last)
}

func fallBackBtcUsdAsset() AssetPrice {
	// Fallback price from 11 May 2020
	return AssetPrice{
		Asset:     BtcUsdAssetId,
		High:      "8899.00",
		Last:      "8845.08",
		Timestamp: "1589196841",
		Bid:       "8843.42",
		Vwap:      "8641.64",
		Volume:    "19380.67700168",
		Low:       "8267.91",
		Ask:       "8852.58",
		Open:      "8740.88",
	}
}
