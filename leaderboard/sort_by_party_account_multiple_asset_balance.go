package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
)

func (s *Service) sortByPartyAccountMultipleBalance(socials map[string]string) ([]Participant, error) {
	// marketID, err := s.getAlgorithmConfig("marketID")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get algorithm config: %w", err)
	// }

	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	decimalPlaces, err := strconv.ParseInt(decimalPlacesStr, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	gqlQueryPartiesAccounts := `query {
		parties {
			id
			accounts {
				asset {
					symbol
				}
				balance
				type
			}
		}
	}`

	ctx := context.Background()
	parties, err := getParties(
		ctx,
		s.cfg.VegaGraphQLURL.String(),
		gqlQueryPartiesAccounts,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	// filter parties and add social handles
	sParties := socialParties(socials, parties)

	participants := []Participant{}
	for _, party := range sParties {

		balanceGeneral := party.Balance(s.cfg.VegaAssets, decimalPlaces, "General", "Margin")
		var sortNum float64

		balanceGeneralStr := strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
		sortNum = balanceGeneral

		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{balanceGeneralStr},
			sortNum:       sortNum,
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
