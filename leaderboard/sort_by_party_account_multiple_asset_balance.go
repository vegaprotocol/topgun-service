package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByPartyAccountMultipleBalance(socials map[string]verifier.Social) ([]Participant, error) {
	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `query(){
		partiesConnection {
		  edges {
			node {
			  id
			  accountsConnection {
				edges {
				  node {
					asset {
					  id
					  symbol
					  decimals
					}
					balance
					type
				  }
				}
			  }
			}
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
		for _, acc := range party.AccountsConnection.Edges {
			for _, asset := range s.cfg.VegaAssets {
				if acc.Account.Asset.Id == asset {
					b := party.Balance(acc.Account.Asset.Id, acc.Account.Asset.Decimals, acc.Account.Type)
					balanceMultiAsset += b
				}
			}
		}

		if balanceMultiAsset > 0.0 {
			if party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
			}

			t := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				Data:          []string{strconv.FormatFloat(balanceMultiAsset, 'f', 10, 32)},
				sortNum:       balanceMultiAsset,
				CreatedAt:     t,
				UpdatedAt:     t,
				isBlacklisted: party.blacklisted,
			})
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
