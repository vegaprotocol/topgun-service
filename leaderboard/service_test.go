package leaderboard_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/vegaprotocol/topgun-service/config"
	"github.com/vegaprotocol/topgun-service/leaderboard"

	"github.com/golang/mock/gomock"
	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testService struct {
	*leaderboard.Service

	ctrl *gomock.Controller
}

func getTestService(t *testing.T, cfg *config.Config) *testService {
	ctrl := gomock.NewController(t)

	s := leaderboard.NewLeaderboardService(*cfg)
	return &testService{
		Service: s,
		ctrl:    ctrl,
	}
}

func TestServiceStatusNotStarted(t *testing.T) {
	now := time.Now()
	cfg := config.Config{
		StartTime:       now.Add(time.Hour),
		EndTime:         now.Add(time.Hour * 2),
		AlgorithmConfig: map[string]string{},
		SocialURL:       &url.URL{},
	}
	s := getTestService(t, &cfg)
	defer s.ctrl.Finish()
	require.Equal(t, "notStarted", s.Status())
}

func TestServiceStatusActive(t *testing.T) {
	now := time.Now()
	cfg := config.Config{
		StartTime:       now.Add(time.Hour * -1),
		EndTime:         now.Add(time.Hour),
		AlgorithmConfig: map[string]string{},
		SocialURL:       &url.URL{},
	}
	s := getTestService(t, &cfg)
	defer s.ctrl.Finish()
	require.Equal(t, "active", s.Status())
}

func TestServiceStatusEnded(t *testing.T) {
	now := time.Now()
	cfg := config.Config{
		StartTime:       now.Add(time.Hour * -2),
		EndTime:         now.Add(time.Hour * -1),
		AlgorithmConfig: map[string]string{},
		SocialURL:       &url.URL{},
	}
	s := getTestService(t, &cfg)
	defer s.ctrl.Finish()
	require.Equal(t, "ended", s.Status())
}
