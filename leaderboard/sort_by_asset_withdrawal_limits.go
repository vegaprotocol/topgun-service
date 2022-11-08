package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByAssetWithdrawalLimit(socials map[string]verifier.Social) ([]Participant, error) {
	// The minimum number of unique withdrawals needed to achieve this reward
	minWithdrawalThreshold := 2000

	gqlQuery := `{
		partiesConnection {
		  edges {
			node {
			  id
			  withdrawalsConnection {
				edges {
				  node {
					amount
					createdTimestamp
					createdTimestamp
					status
					asset {
					  id
					  symbol
					  source {
						__typename
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
		gqlQuery,
		map[string]string{"assetId": s.cfg.VegaAssets[0]},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		withdrawalCount := 0
		for _, w := range party.WithdrawalsConnection.Edges {
			// string to int
			amount, err := strconv.Atoi(w.Withdrawal.Amount)
			if err != nil {
				fmt.Errorf("failed to convert Withdrawal amount to string", err)
			}

			if w.Withdrawal.Asset.Id == s.cfg.VegaAssets[0] &&
				w.Withdrawal.Status == "STATUS_FINALIZED" &&
				amount >= minWithdrawalThreshold &&
				w.Withdrawal.CreatedAt.After(s.cfg.StartTime) &&
				w.Withdrawal.CreatedAt.Before(s.cfg.EndTime) {
				withdrawalCount++
				fmt.Println(withdrawalCount)
			}
		}

		if withdrawalCount > 0 {
			utcNow := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterHandle: party.social,
				TwitterUserID: party.twitterID,
				Data:          []string{"Withdrawal Completed"},
				CreatedAt:     utcNow,
				UpdatedAt:     utcNow,
				isBlacklisted: party.blacklisted,
			})
		}

	}

	sortFunc := func(i, j int) bool {
		return participants[i].TwitterHandle < participants[j].TwitterHandle
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
