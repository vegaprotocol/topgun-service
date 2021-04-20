package leaderboard

import (
	"context"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
)

func (s *Service) sortByPartyGovernanceVotes(socials map[string]string) ([]Participant, error) {
	gqlQuery := "query {parties {id votes {proposalId vote {value datetime}}}}"
	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	since, err := s.getAlgorithmConfigTime("since")
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"since ": since.String()}).Info("Algo cfg")
	for _, party := range sParties {
		voteCount := 0
		for _, v := range party.Votes {
			if v.Vote.Datetime.After(since) {
				voteCount++
			}
		}
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{fmt.Sprintf("%d", voteCount)},
			sortNum:       float64(voteCount),
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
