package leaderboard

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

var gqlQueryPartiesELS string = `query ($pagination: Pagination!) {
	partiesConnection(pagination: $pagination) {
		edges {
			node {
			  id
			  liquidityProvisionsConnection {
				edges {
				  node {
					market {
					  id
					  data {
						liquidityProviderFeeShare {
						  equityLikeShare
						  averageEntryValuation
						  averageScore
						}
					  }
					}
				  }
				}
			  }
			}
		  }
		}
  }`

func (s *Service) sortByELS() ([]Participant, error) {

	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	decimalPlaces, err := strconv.ParseFloat(decimalPlacesStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	pagination := Pagination{First: 50}

	ctx := context.Background()
	partyEdges := []PartiesEdge{}
	for {
		connection, err := getPartiesConnection(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryPartiesELS,
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
	participants := []Participant{}
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range partyEdges {
		ELS := 0.0
		dataFormatted := ""
		if len(party.Party.LPsConnection.Edges) != 0 {
			for _, w := range party.Party.LPsConnection.Edges {
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				ELS = w.LP.Market.Data.EquitLikeShare
			}
		}



		if (ELS != 0.0) {

			t := time.Now().UTC()
				dpMultiplier := math.Pow(10, decimalPlaces)
				total := ELS / dpMultiplier
				dataFormatted = strconv.FormatFloat(total, 'f', 10, 32)
		

			participants = append(participants, Participant{
				PublicKey: party.Party.ID,
				Data:      []string{dataFormatted},
				sortNum:   ELS,
				CreatedAt: t,
				UpdatedAt: t,
			})
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
