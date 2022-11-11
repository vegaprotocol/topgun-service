package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByLPCommittedList(socials map[string]verifier.Social) ([]Participant, error) {
	// Grab the market ID for the market we're targeting
	marketID, err := s.getAlgorithmConfig("marketID")

	gqlQueryPartiesAccounts := `{
		partiesConnection {
		  edges {
			node {
			  id
			  liquidityProvisionsConnection {
				edges {
				  node {
					id
					market {
					  id
					}
					commitmentAmount
					createdAt
					reference
					buys {
					  liquidityOrder {
						reference
						proportion
						offset
					  }
					}
					sells {
					  liquidityOrder {
						reference
						proportion
						offset
					  }
					}
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
		lpCount := 0
		// Check for matching parties who have committed LP :)
		if party.LPsConnection.Edges != nil && len(party.LPsConnection.Edges) > 0 {
			for _, lpEdge := range party.LPsConnection.Edges {
				if lpEdge.LP.Market.ID == marketID {
					log.WithFields(log.Fields{"partyID": party.ID, "totalLPs": len(party.LPsConnection.Edges)}).Info("Party has LPs on correct market")
					lpCount++
				}
			}
		}

		if lpCount > 0 {
			utcNow := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterHandle: party.social,
				TwitterUserID: party.twitterID,
				Data:          []string{"Provided Liquidity"},
				CreatedAt:     utcNow,
				UpdatedAt:     utcNow,
				isBlacklisted: party.blacklisted,
			})
			break
		}

	}

	sortFunc := func(i, j int) bool {
		return participants[i].TwitterHandle < participants[j].TwitterHandle
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
