package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByPartyGovernanceVotedList(socials map[string]verifier.Social) ([]Participant, error) {
	gqlQuery := "query {parties {id votes {proposalId vote {value datetime}}}}"
	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		for _, v := range party.Votes {
			if v.Vote.Datetime.After(s.cfg.StartTime) && v.Vote.Datetime.Before(s.cfg.EndTime) {
				utcNow := time.Now().UTC()
				participants = append(participants, Participant{
					PublicKey:     party.ID,
					TwitterHandle: party.social,
					TwitterUserID: party.twitterID,
					Data:          []string{"Voted"},
					sortNum:       float64(1),
					CreatedAt:     utcNow,
					UpdatedAt:     utcNow,
					isBlacklisted: party.blacklisted,
				})
			}
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].UpdatedAt.Unix() > participants[j].UpdatedAt.Unix()
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
