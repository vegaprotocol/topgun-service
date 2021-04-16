package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	// log "github.com/sirupsen/logrus"
)

func (s *Service) sortByPartyAccountGeneralBalance(socials map[string]string) ([]Participant, error) {
	// Load all parties with accounts from GraphQL end-point
	gqlQuery := "query {parties {id accounts {type balance asset {symbol}}}}"
	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		balanceGeneral := party.Balance(s.cfg.VegaAsset, "General")
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{strconv.FormatFloat(balanceGeneral, 'f', 5, 32)},
			sortNum:       balanceGeneral,
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
