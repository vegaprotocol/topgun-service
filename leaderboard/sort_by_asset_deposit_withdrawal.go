package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByAssetDepositWithdrawal(socials map[string]verifier.Social) ([]Participant, error) {

	// The minimum number of unique deposits and withdrawals needed to achieve this reward
	minDepositAndWithdrawals := 1
	minWithdrawalThreshold := 2000
	minDepositThreshold := 2000

	// Default: 1 unique asset deposit and 1 unique withdrawal1 from the erc20 bridge

	gqlQuery := `{
		partiesConnection {
		  edges {
			node {
			  id
			  depositsConnection {
				edges {
				  node {
					amount
					createdTimestamp
					creditedTimestamp
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
		depositCount := 0
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
			}
		}

		for _, d := range party.DepositsConnection.Edges {
			// string to int
			amount, err := strconv.Atoi(d.Deposit.Amount)
			if err != nil {
				fmt.Errorf("failed to convert Withdrawal amount to string", err)
			}

			if d.Deposit.Asset.Id == s.cfg.VegaAssets[0] &&
				d.Deposit.Status == "STATUS_FINALIZED" &&
				amount >= minDepositThreshold &&
				d.Deposit.CreatedAt.After(s.cfg.StartTime) &&
				d.Deposit.CreatedAt.Before(s.cfg.EndTime) {
				depositCount++
			}
		}

		totalCount := withdrawalCount + depositCount

		if totalCount > (minDepositAndWithdrawals - 1) {
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
