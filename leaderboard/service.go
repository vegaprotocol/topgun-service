package leaderboard

import (
	"sync"
	"time"

	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/util"
	"github.com/vegaprotocol/topgun-service/verifier"

	log "github.com/sirupsen/logrus"
)

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
		verifier: verifier.NewVerifierService(*cfg.SocialURL, cfg.TwitterBlacklist),
	}
	return svc
}

type Service struct {
	cfg config.Config

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

	s.mu.Unlock()
	log.WithFields(log.Fields{"participants": len(s.board.Participants)}).Info("Leaderboard updated")
}
