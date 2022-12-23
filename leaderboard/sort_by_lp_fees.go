package leaderboard

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByLPFees(socials map[string]verifier.Social) ([]Participant, error) {
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	decimalPlaces, err := strconv.ParseFloat(decimalPlacesStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	gqlQueryPartiesAccounts := `{
		partiesConnection {
		  edges {
			node {
			  liquidityProvisionsConnection {
				edges {
				  node {
					id
					market {
					  id
					}
					fee
					commitmentAmount
					createdAt
					reference
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
		lpFees := 0.0
		// Check for matching parties who have committed LP :)
		if party.LPsConnection.Edges != nil && len(party.LPsConnection.Edges) > 0 {
			for _, lpEdge := range party.LPsConnection.Edges {
				for _, marketID := range s.cfg.MarketIDs {
					if lpEdge.LP.Market.ID == marketID {
						if u, err := strconv.ParseFloat(lpEdge.LP.Fee, 32); err == nil {
							lpFees = u
						}
					}
				}

			}
		}

		if lpFees > 0 {
			t := time.Now().UTC()
			dataFormatted := ""

			dpMultiplier := math.Pow(10, decimalPlaces)
			total := lpFees / dpMultiplier
			dataFormatted = strconv.FormatFloat(total, 'f', 10, 32)

			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{dataFormatted},
				sortNum:       lpFees,
				CreatedAt:     t,
				UpdatedAt:     t,
				isBlacklisted: party.blacklisted,
			})
		}
		break

	}

	sortFunc := func(i, j int) bool {
		return participants[i].TwitterHandle < participants[j].TwitterHandle
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
