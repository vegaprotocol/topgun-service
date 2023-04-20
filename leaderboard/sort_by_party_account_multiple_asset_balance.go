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

var gqlQueryMultiAssets string = `query ($pagination: Pagination!) {
	partiesConnection(pagination: $pagination) {
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

func (s *Service) sortByPartyAccountMultipleBalance(socials map[string]verifier.Social) ([]Participant, error) {

	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	// decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	// }
	// decimalPlaces, err := strconv.ParseFloat(decimalPlacesStr, 64)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	// }

	pagination := Pagination{First: 50}

	ctx := context.Background()
	partyEdges := []PartiesEdge{}
	for {
		connection, err := getPartiesConnection(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryMultiAssets,
			map[string]interface{}{"pagination": pagination},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get list of parties in loop: %w", err)
		}

		partyEdges = append(partyEdges, connection.Edges...)

		// fmt.Println("got ", len(partyEdges), "end?", connection.PageInfo.EndCursor)

		if !connection.PageInfo.NextPage {
			// fmt.Println("done")
			break
		} else {
			pagination.After = connection.PageInfo.EndCursor
		}

	}

	// filter parties and add social handles
	sParties := socialParties(socials, partyEdges)
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
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
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
