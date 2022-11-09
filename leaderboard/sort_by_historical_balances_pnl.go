package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	"github.com/vegaprotocol/topgun-service/verifier"
)

type BalanceChangesResponse struct {
	BalanceChangesConnection BalanceChangesConnection `json:"end"`
}

func (s *Service) sortByPartyAccountHistoricBalancesPnL(socials map[string]verifier.Social) ([]Participant, error) {

	gqlQuery := `{ 
		start: balanceChanges(
		  dateRange: {start: "1666812477000000000", end: "1666812477000000000"}
		) {
		  edges {
			node {
			  assetId
			  partyId
			  balance
			  timestamp
			}
		  }
		}
		end: balanceChanges(
		  dateRange: {start: "1666812477000000000"}
		) {
		  edges {
			node {
			  assetId
			  partyId
			  balance
			  timestamp
			}
		  }
		}
	  }`
	client := graphql.NewClient(s.cfg.VegaGraphQLURL.String())
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")

	var response BalanceChangesResponse
	ctx := context.Background()
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to get balance changes info: %w", err)
	}

	fmt.Println(response.BalanceChangesConnection.BalanceChangesEdges)

	parties := make([]Party, 0)
	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		for _, res := range response.BalanceChangesConnection.BalanceChangesEdges {
			for _, asset := range s.cfg.VegaAssets {
				if (asset == res.BalanceChanges.AssetId) && (res.BalanceChanges.PartyId == party.ID) {
					sortNum, err := strconv.ParseFloat(res.BalanceChanges.Balance, 64)
					if err != nil {
						fmt.Println(err)
					} else {
						utcNow := time.Now().UTC()
						participants = append(participants, Participant{
							PublicKey:     party.ID,
							TwitterHandle: party.social,
							TwitterUserID: party.twitterID,
							Data:          []string{res.BalanceChanges.Balance},
							sortNum:       sortNum,
							CreatedAt:     utcNow,
							UpdatedAt:     utcNow,
							isBlacklisted: party.blacklisted,
						})
					}

				}
			}
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)
	return participants, nil
}
