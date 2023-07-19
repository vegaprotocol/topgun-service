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

func (s *Service) sortByPartyAccountGeneralBalance(socials map[string]verifier.Social) ([]Participant, error) {
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

	gqlQueryPartiesAccounts := `query ($assetId: ID) {
		partiesConnection {
		  edges {
			node {
			  id
			  accountsConnection(assetId: $assetId) {
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
	// gqlQueryPartiesTrades := `query($marketId: ID!, $partyId: ID!) {
	// 	parties(id: $partyId) {
	// 		trades(marketId: $marketId, first: 1, last: 2) {
	// 			id
	// 			createdAt
	// 		}
	// 	}
	// }`

	ctx := context.Background()

	parties, err := getParties(
		ctx,
		s.cfg.VegaGraphQLURL.String(),
		gqlQueryPartiesAccounts,
		map[string]string{"assetId": s.cfg.VegaAssets[0]},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	// filter parties and add social handles
	sParties := socialParties(socials, parties)

	participants := []Participant{}
	for _, party := range sParties {
		// Add trades for each party, one by one, to avoid GraphQL query timeouts.
		log.WithFields(log.Fields{"partyID": party.ID}).Debug("Getting trades for party")
		// partyTrades, err := getParties(
		// 	ctx,
		// 	s.cfg.VegaGraphQLURL.String(),
		// 	gqlQueryPartiesTrades,
		// 	map[string]string{
		// 		"marketId": marketID,
		// 		"partyId":  party.ID,
		// 	},
		// 	nil,
		// )
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to get list of trades for parties: %w", err)
		// }
		// if len(partyTrades) == 1 {
		// 	// Party exists on Vega
		// 	party.Trades = partyTrades[0].Trades
		// }
		// log.WithFields(log.Fields{"partyID": party.ID, "trades": len(party.Trades)}).Debug("Got trades for party")

		// Count only trades that happened during the competition.
		// tradeCount := 0
		// for _, t := range party.Trades {
		// 	if t.CreatedAt.After(s.cfg.StartTime) && t.CreatedAt.Before(s.cfg.EndTime) {
		// 		tradeCount++
		// 	}
		// }

		balanceGeneral := party.Balance(s.cfg.VegaAssets[0], int(decimalPlaces), "ACCOUNT_TYPE_GENERAL", "ACCOUNT_TYPE_MARGIN")
		var sortNum float64
		// var balanceGeneralStr string
		// if tradeCount > 0 {
		// if len(party.Trades) > 0 {
		balanceGeneralStr := strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
		sortNum = balanceGeneral
		// } else {
		// 	// Untraded folks have not participated in the competition.
		// 	balanceGeneralStr = "n/a"
		// 	sortNum = -1.0e20
		// }
		if balanceGeneral > 0 {
			utcNow := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				Data:          []string{balanceGeneralStr},
				sortNum:       sortNum,
				CreatedAt:     utcNow,
				UpdatedAt:     utcNow,
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
