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
	StartBalance BalanceChangesConnection `json:"start"`
	EndBalance   BalanceChangesConnection `json:"end"`
}

func (s *Service) sortByPartyAccountHistoricBalancesPnL(socials map[string]verifier.Social) ([]Participant, error) {

	gqlQuery := `query($startTime: Timestamp){ 
		start: balanceChanges(
		  dateRange: {start: $startTime, end: "1666812477000000000"}
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
		  dateRange: {start: $startTime}
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
	vars := map[string]string{"startTime": strconv.FormatInt(s.cfg.StartTime.Unix(), 10)}
	for key, value := range vars {
		req.Var(key, value)
	}
	var response BalanceChangesResponse
	ctx := context.Background()
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to get balance changes info: %w", err)
	}

	fmt.Println(response.EndBalance.BalanceChangesEdges)

	parties := make([]Party, 0)
	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		for _, resEnd := range response.EndBalance.BalanceChangesEdges {
			for _, resStart := range response.StartBalance.BalanceChangesEdges {
				if (s.cfg.VegaAssets[0] == resEnd.BalanceChanges.AssetId) && (resEnd.BalanceChanges.PartyId == party.ID) {
					PnL := (resEnd.BalanceChanges.Balance - resStart.BalanceChanges.Balance)
					sortNum, err := strconv.ParseFloat(resEnd.BalanceChanges.Balance, 64)
					if err != nil {
						fmt.Println(err)
					} else {
						utcNow := time.Now().UTC()
						participants = append(participants, Participant{
							PublicKey:     party.ID,
							TwitterHandle: party.social,
							TwitterUserID: party.twitterID,
							Data:          []string{resEnd.BalanceChanges.Balance},
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

func typeOf(key string) {
	panic("unimplemented")
}
