package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByPartyAccountMultipleBalance(socials map[string]verifier.Social) ([]Participant, error) {
	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `query {
		parties {
			id
			accounts {
				asset {
					symbol
					decimals
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
		balanceMultiAsset := 0.0
		for _, acc := range party.Accounts {
			for _, asset := range s.cfg.VegaAssets {
				if acc.Asset.Symbol == asset {
					balanceMultiAsset += party.Balance(acc.Asset.Id, acc.Asset.Decimals, "General", "Margin")
				}
			}
		}
		if balanceMultiAsset > 0.0 {
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterHandle: party.social,
				Data:          []string{strconv.FormatFloat(balanceMultiAsset, 'f', 5, 32)},
				sortNum:       balanceMultiAsset,
			})
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
