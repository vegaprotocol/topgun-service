package leaderboard

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
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
	Position      int       `json:"position" bson:"position,omitempty"`
	PublicKey     string    `json:"publicKey" bson:"pub_key,omitempty"`
	TwitterHandle string    `json:"twitterHandle" bson:"twitter_handle,omitempty"`
	TwitterUserID float64    `json:"twitterUserID" bson:"twitter_userid,omitempty"`
	CreatedAt     time.Time `json:"createdAt" bson:"created,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"last_modified,omitempty"`
	Data          []string  `json:"data" bson:"data,omitempty"`

	sortNum float64
}

type Leaderboard struct {
	Version        int      `json:"version"`
	Asset          string   `json:"asset"`
	LastUpdate     string   `json:"lastUpdate"`
	Headers        []string `json:"headers"`
	Description    string   `json:"description"`
	DefaultSort    string   `json:"defaultSort"`
	DefaultDisplay string   `json:"defaultDisplay"`
	Status         string   `json:"status"`

	// Participants lists current participants in an active competition
	Participants []Participant `json:"participants"`

	// ParticipantsSnapshot lists participants at a particular time point.
	// Keys:
	// - "start": snapshot taken at the start of a competition
	// - "end": snapshot taken at the end of a competition
	// - "yyyy-mm-ddTHH:MM:SSZ": snapshot taken at given time
	ParticipantsSnapshot map[string][]Participant `json:"participantsSnapshot"`
}

func NewLeaderboardService(cfg config.Config) *Service {
	svc := &Service{
		cfg: cfg,
		pricingEngine: pricing.NewEngine(url.URL{
			Scheme: "https",
			Host:   "prices.ops.vega.xyz",
			Path:   "/prices",
		}),
		verifier:            verifier.NewVerifierService(*cfg.SocialURL),
		participantSnapshot: make(map[string][]Participant),
	}
	return svc
}

type Service struct {
	cfg config.Config

	pricingEngine       PricingEngine
	timer               *time.Ticker
	board               Leaderboard
	mu                  sync.RWMutex
	verifier            *verifier.Service
	participantSnapshot map[string][]Participant
}

func (s *Service) Start() {
	log.Info("Leaderboard service started")

	// The first time we start the service it will be
	// in a status of "loading" as it waits for first data
	// from the Vega API
	newBoard := Leaderboard{
		Version:              1,
		Asset:                s.cfg.VegaAsset,
		DefaultDisplay:       s.cfg.DefaultDisplay,
		DefaultSort:          s.cfg.DefaultSort,
		Description:          s.cfg.Description,
		Headers:              s.cfg.Headers,
		LastUpdate:           util.UnixTimestampUtcNowFormatted(),
		Status:               competitionLoading,
		ParticipantsSnapshot: s.participantSnapshot,
		Participants:         []Participant{},
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

	snapshotStart = "start"
	snapshotEnd   = "end"

	snapshotStartFilename = "snapshotStart.json"
	snapshotEndFilename   = "snapshotEnd.json"
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

func readSnapshotFile(filename string) ([]Participant, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data from %s: %w", filename, err)
	}

	var snapshot []Participant
	err = json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal file data from %s: %w", filename, err)
	}

	return snapshot, nil
}

func saveSnapshotFile(filename string, participants []Participant) error {
	payload, err := json.Marshal(participants)
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %s: %w", filename, err)
	}

	_, err = file.Write(payload)
	if err != nil {
		return fmt.Errorf("failed to write all data: %s: %w", filename, err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close file: %s: %w", filename, err)
	}
	return nil
}

func copyParticipants(src []Participant) ([]Participant, error) {
	l := len(src)
	dst := make([]Participant, l)
	count := copy(dst, src)
	if count < l {
		return nil, fmt.Errorf("failed to copy all participants (%d<%d)", count, l)
	}
	return dst, nil
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
	//timeNow := time.Now().UTC()
	//if timeNow.Before(s.cfg.StartTime) || timeNow.After(s.cfg.EndTime) {
	//	log.Info("Current date/time outside of leaderboard start/end time")
	//	return
	//}

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
	default:
		err = fmt.Errorf("invalid algorithm: %s", s.cfg.Algorithm)
	}
	if err != nil {
		log.WithError(err).Warn("Failed to sort")
		p = []Participant{}
	}
	i := 0
	for range p {
		p[i].Position = i + 1 // humans want 1-indexed lists :-|
		i++
	}
	log.Infof("Algo finish: %s", s.cfg.Algorithm)

	s.mu.Lock()
	newBoard := Leaderboard{
		Version:              1,
		Asset:                s.cfg.VegaAsset,
		DefaultDisplay:       s.cfg.DefaultDisplay,
		DefaultSort:          s.cfg.DefaultSort,
		Description:          s.cfg.Description,
		Headers:              s.cfg.Headers,
		LastUpdate:           util.UnixTimestampUtcNowFormatted(),
		Status:               status,
		ParticipantsSnapshot: s.participantSnapshot,
	}
	// Seems like sometime the participants list is empty
	// in that case we just reuse the previous
	// board participants
	if len(p) > 0 {
		newBoard.Participants = p
	} else {
		newBoard.Participants = s.board.Participants
	}

	s.board = newBoard
	s.mu.Unlock()
	log.WithFields(log.Fields{"participants": len(s.board.Participants)}).Info("Leaderboard updated")

	if s.cfg.SnapshotEnabled {

		_, startSnapshotTaken := s.participantSnapshot[snapshotStart]
		if !startSnapshotTaken && status == competitionActive {
			// First, attempt to read the start snapshot from file. This allows
			// the app to be restarted easily.
			startSnapshot, err := readSnapshotFile(snapshotStartFilename)
			if err != nil {
				// Failed to read file, so fall back to taking a snapshot and saving it.
				startSnapshot, err = copyParticipants(newBoard.Participants)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Warn("Failed to copy whole start snapshot")
					startSnapshot = []Participant{}
				}
				saveSnapshotFile(snapshotStartFilename, startSnapshot)
				log.Info("Saved start snapshot to disk")
			} else {
				log.Info("Read start snapshot from disk")
			}
			s.participantSnapshot[snapshotStart] = startSnapshot
		}

		_, endSnapshotTaken := s.participantSnapshot[snapshotEnd]
		if !endSnapshotTaken && status == competitionEnded {
			// First, attempt to read the end snapshot from file. This allows
			// the app to be restarted easily.
			endSnapshot, err := readSnapshotFile(snapshotEndFilename)
			if err != nil {
				// Failed to read file, so fall back to taking a snapshot and saving it.
				endSnapshot, err = copyParticipants(newBoard.Participants)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Warn("Failed to copy whole end snapshot")
					endSnapshot = []Participant{}
				}
				saveSnapshotFile(snapshotEndFilename, endSnapshot)
				log.Info("Saved end snapshot to disk")
			} else {
				log.Info("Read end snapshot from disk")
			}
			s.participantSnapshot[snapshotEnd] = endSnapshot
		}
	}
}

func (s *Service) MarshalLeaderboard(q string, skip int64, size int64) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	participants := []Participant{}
	if q == "" {
		// No search query filter found
		// Full data set required
		participants = s.board.Participants
	} else {
		// Search query has been passed with request
		q = strings.ToLower(q)
		for _, p := range s.board.Participants {
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
		Version:              s.board.Version,
		Asset:                s.board.Asset,
		LastUpdate:           s.board.LastUpdate,
		Headers:              s.board.Headers,
		Description:          s.board.Description,
		DefaultSort:          s.board.DefaultSort,
		DefaultDisplay:       s.board.DefaultDisplay,
		Status:               s.board.Status,
		Participants:         s.paginate(participants, skip, size),
		ParticipantsSnapshot: nil,
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
