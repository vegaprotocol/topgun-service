package leaderboard

import (
	"fmt"
	"time"
)

func (s *Service) getAlgorithmConfig(key string) (string, error) {
	value, found := s.cfg.AlgorithmConfig[key]
	if !found {
		return "", fmt.Errorf("missing algorithmConfig: %s", key)
	}
	return value, nil
}

func (s *Service) getAlgorithmConfigTime(key string) (time.Time, error) {
	valueStr, err := s.getAlgorithmConfig(key)
	if err != nil {
		return time.Time{}, err
	}
	return parseTime(valueStr)
}

func parseTime(timeStr string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05Z", timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse datetime: %w", err)
	}
	return t, nil
}
