package leaderboard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
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
	Position      int       `json:"position" bson:"position,omitempty"`
	PublicKey     string    `json:"publicKey" bson:"pub_key,omitempty"`
	TwitterHandle string    `json:"twitterHandle" bson:"twitter_handle,omitempty"`
	TwitterUserID int64     `json:"twitterUserID" bson:"twitter_userid,omitempty"`
	CreatedAt     time.Time `json:"createdAt" bson:"created,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"last_modified,omitempty"`
	Data          []string  `json:"data" bson:"data,omitempty"`

	isBlacklisted bool
	sortNum       float64
}

type Leaderboard struct {
	Version        int      `json:"version"`
	Assets         []string `json:"assets"`
	LastUpdate     string   `json:"lastUpdate"`
	Headers        []string `json:"headers"`
	Description    string   `json:"description"`
	DefaultSort    string   `json:"defaultSort"`
	DefaultDisplay string   `json:"defaultDisplay"`
	Status         string   `json:"status"`

	// Participants is the filtered list of participants in an active incentive
	Participants []Participant `json:"participants"`

	// Blacklisted is the list of participants in an active
	// incentive including excluded/blacklisted socials e.g. team/bots
	blacklisted []Participant
}

func NewLeaderboardService(cfg config.Config) *Service {
	svc := &Service{
		cfg: cfg,
		pricingEngine: pricing.NewEngine(url.URL{
			Scheme: "https",
			Host:   "prices.ops.vega.xyz",
			Path:   "/prices",
		}),
		verifier: verifier.NewVerifierService(*cfg.SocialURL, cfg.TwitterBlacklist),
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

	// The first time we start the service it will be
	// in a status of "loading" as it waits for first data
	// from the Vega API
	newBoard := Leaderboard{
		Version:        1,
		Assets:         s.cfg.VegaAssets,
		DefaultDisplay: s.cfg.DefaultDisplay,
		DefaultSort:    s.cfg.DefaultSort,
		Description:    s.cfg.Description,
		Headers:        s.cfg.Headers,
		LastUpdate:     util.UnixTimestampUtcNowFormatted(),
		Status:         competitionLoading,
		Participants:   []Participant{},
		blacklisted:    []Participant{},
	}
	s.board = newBoard

	s.update()
	s.timer = util.Schedule(s.update, s.cfg.VegaPoll)
}

func (s *Service) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
	log.Info("Leaderboard service stopped")
}

const (
	competitionLoading    = "loading"
	competitionNotStarted = "notStarted"
	competitionActive     = "active"
	competitionEnded      = "ended"
)

func (s *Service) Status() string {
	now := time.Now()
	if now.Before(s.cfg.StartTime) {
		// Competition has not yet started
		return competitionNotStarted
	}
	if now.Before(s.cfg.EndTime) {
		// Competition is active
		return competitionActive
	}
	// Competition has ended
	return competitionEnded
}

func (s *Service) update() {
	status := s.Status()

	// Attempt to update parties from external social verifier service
	// Safe approach, will only overwrite internal collection if successful
	s.verifier.UpdateVerifiedParties()
	// Grab a map of the verified pub-key->twitter-handle for leaderboard
	socials := s.verifier.PubKeysToSocials()
	// If no verified pub-key->social-handles found, no need to query Vega
	if len(socials) == 0 {
		return
	}

	// Only process leaderboard between competition start and end times
	timeNow := time.Now().UTC()
	if timeNow.Before(s.cfg.StartTime) {
		log.Info("This incentive has not started yet. The leaderboard will update when the incentive begins")
		return
	} else if timeNow.After(s.cfg.EndTime) {
		log.Info("This incentive has now ended")
		return
	}

	log.Infof("Algo start: %s", s.cfg.Algorithm)
	var p []Participant
	var err error
	switch s.cfg.Algorithm {
	case "ByPartyAccountGeneralBalance":
		p, err = s.sortByPartyAccountGeneralBalance(socials)
	case "ByPartyAccountGeneralBalanceLP":
		p, err = s.sortByPartyAccountGeneralBalanceAndLP(socials)
	case "ByPartyAccountGeneralProfit":
		p, err = s.sortByPartyAccountGeneralProfit(socials, false)
	case "ByPartyAccountGeneralProfitLP":
		p, err = s.sortByPartyAccountGeneralProfit(socials, true)
	case "ByPartyGovernanceVotes":
		p, err = s.sortByPartyGovernanceVotes(socials)
	case "ByLPEquitylikeShare":
		p, err = s.sortByLPEquitylikeShare(socials)
	case "ByAssetDepositWithdrawal":
		p, err = s.sortByAssetDepositWithdrawal(socials)
	case "BySocialRegistration":
		p, err = s.sortBySocialRegistration(s.verifier.List())
	case "ByPartyAccountMultipleBalance":
		p, err = s.sortByPartyAccountMultipleBalance(socials)
	case "ByPartyAccountGeneralLoser":
		p, err = s.sortByPartyAccountGeneralLoser(socials)
	default:
		err = fmt.Errorf("invalid algorithm: %s", s.cfg.Algorithm)
	}
	if err != nil {
		log.WithError(err).Warn("Failed to sort")
		p = []Participant{}
	}

	// Filter into two sets to separate blacklisted users
	include := []Participant{}
	exclude := []Participant{}
	for _, ppt := range p {
		if ppt.isBlacklisted {
			exclude = append(exclude, ppt)
		} else {
			include = append(include, ppt)
		}
	}
	include = s.AllocatePositions(include)
	exclude = s.AllocatePositions(exclude)

	log.Infof("Algo finish: %s", s.cfg.Algorithm)

	s.mu.Lock()
	newBoard := Leaderboard{
		Version:        1,
		Assets:         s.cfg.VegaAssets,
		DefaultDisplay: s.cfg.DefaultDisplay,
		DefaultSort:    s.cfg.DefaultSort,
		Description:    s.cfg.Description,
		Headers:        s.cfg.Headers,
		LastUpdate:     util.UnixTimestampUtcNowFormatted(),
		Status:         status,
	}
	// Seems like sometime the participants list is empty
	// in that case we just reuse the previous
	// board participants
	if len(include) > 0 {
		newBoard.Participants = include
	} else {
		newBoard.Participants = s.board.Participants
	}
	if len(exclude) > 0 {
		newBoard.blacklisted = exclude
	} else {
		newBoard.blacklisted = s.board.blacklisted
	}

	s.board = newBoard
	s.mu.Unlock()
	log.WithFields(log.Fields{"participants": len(s.board.Participants)}).Info("Leaderboard updated")
}

func (s *Service) CsvLeaderboard(q string, skip int64, size int64, blacklisted bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter based on blacklisted or regular leaderboard participants
	target := s.board.Participants
	if blacklisted {
		target = s.board.blacklisted
	}

	participants := []Participant{}
	if q == "" {
		// No search query filter found
		// Full data set required
		participants = target
	} else {
		// Search query has been passed with request
		q = strings.ToLower(q)
		for _, p := range target {
			pubKey := strings.ToLower(p.PublicKey)
			twitterHandle := strings.ToLower(p.TwitterHandle)
			// case insensitive comparison
			if pubKey == q || twitterHandle == q || strings.Contains(pubKey, q) || strings.Contains(twitterHandle, q) {
				participants = append(participants, p)
			}
		}
		// Filtered data set with search query
	}

	board := Leaderboard{
		Version:        s.board.Version,
		Assets:         s.board.Assets,
		LastUpdate:     s.board.LastUpdate,
		Headers:        s.board.Headers,
		Description:    s.board.Description,
		DefaultSort:    s.board.DefaultSort,
		DefaultDisplay: s.board.DefaultDisplay,
		Status:         s.board.Status,
		Participants:   s.paginate(participants, skip, size),
	}

	return s.WriteParticipantsToCsvBytes(board.Participants)
}

func (s *Service) JsonLeaderboard(q string, skip int64, size int64, blacklisted bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter based on blacklisted or regular leaderboard participants
	target := s.board.Participants
	if blacklisted {
		target = s.board.blacklisted
	}

	participants := []Participant{}
	if q == "" {
		// No search query filter found
		participants = target
	} else {
		// Search query has been passed with request
		q = strings.ToLower(q)
		for _, p := range target {
			pubKey := strings.ToLower(p.PublicKey)
			twitterHandle := strings.ToLower(p.TwitterHandle)
			// case insensitive comparison
			if pubKey == q || twitterHandle == q || strings.Contains(pubKey, q) || strings.Contains(twitterHandle, q) {
				participants = append(participants, p)
			}
		}
		// Filtered data set with search query
	}

	board := Leaderboard{
		Version:        s.board.Version,
		Assets:         s.board.Assets,
		LastUpdate:     s.board.LastUpdate,
		Headers:        s.board.Headers,
		Description:    s.board.Description,
		DefaultSort:    s.board.DefaultSort,
		DefaultDisplay: s.board.DefaultDisplay,
		Status:         s.board.Status,
		Participants:   s.paginate(participants, skip, size),
	}

	return json.Marshal(board)
}

func (s *Service) paginate(p []Participant, skip int64, size int64) []Participant {
	if skip < 1 {
		skip = 0
	}
	if size < 1 {
		size = 999999
	}
	if skip > int64(len(p)) {
		skip = int64(len(p))
	}
	end := skip + size
	if end > int64(len(p)) {
		end = int64(len(p))
	}
	return p[skip:end]
}

func (s *Service) WriteParticipantsToCsvBytes(participants []Participant) (result []byte, err error) {
	if len(participants) > 0 {
		csvData := make([]util.ParticipantCsvEntry, 0)
		for _, p := range participants {
			csv := util.ParticipantCsvEntry{
				Position:      p.Position,
				TwitterHandle: p.TwitterHandle,
				TwitterID:     p.TwitterUserID,
				VegaPubKey:    p.PublicKey,
				CreatedAt:     p.CreatedAt,
				UpdatedAt:     p.UpdatedAt,
			}
			for i, d := range p.Data {
				if i > 0 {
					csv.VegaData += "|"
				}
				csv.VegaData += d
			}
			csvData = append(csvData, csv)
		}
		res, err := gocsv.MarshalBytes(&csvData)
		if err != nil {
			log.WithError(err).Error("Error marshaling Participant data to CSV bytes")
			return make([]byte, 0), err
		}
		return res, nil
	}
	return make([]byte, 0), nil
}

func (s *Service) AllocatePositions(p []Participant) []Participant {
	i := 0
	for range p {
		p[i].Position = i + 1 // humans want 1-indexed lists :-|
		i++
	}
	return p
}
