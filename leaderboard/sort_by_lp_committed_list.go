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
	// Algo for single market only
	// marketID, err := s.getAlgorithmConfig("marketID")

	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get market ID: %w", err)
	// }

	gqlQueryPartiesAccounts := `query ($pagination: Pagination!) {
		partiesConnection(pagination: $pagination) {
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
		  pageInfo {
			hasNextPage
			hasPreviousPage
			startCursor
			endCursor
		  }
		}
	  }`

	pagination := Pagination{First: 50}

	ctx := context.Background()
	partyEdges := []PartiesEdge{}
	for {
		connection, err := getPartiesConnection(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryPartiesAccounts,
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
		lpCount := 0
		// Check for matching parties who have committed LP :)
		if party.LPsConnection.Edges != nil && len(party.LPsConnection.Edges) > 0 {
			for _, lpEdge := range party.LPsConnection.Edges {
				for _, marketID := range s.cfg.MarketIDs {
					if lpEdge.LP.Market.ID == marketID {
						log.WithFields(log.Fields{"partyID": party.ID, "totalLPs": len(party.LPsConnection.Edges)}).Info("Party has LPs on correct market")
						lpCount++
					}
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
