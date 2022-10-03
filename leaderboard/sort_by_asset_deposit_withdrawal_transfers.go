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
	minDepositThreshold := 2000
	minTransferThreshold := 2000

	gqlQuery := `query {
		parties {
			 depositsConnection {
			edges {
			  node {
				amount
				id
				party { id }
				asset { id, name }
				createdTimestamp
				creditedTimestamp
				status
			  }
			}
		  }
		  withdrawalsConnection {
			edges {
			  node {
				amount
				id
				party { id }
				asset { id, name }
				createdTimestamp
				withdrawnTimestamp
				status
			  }
			}
		  }
		  transfersConnection {
			edges {
			  node {
				id
				fromAccountType 
				toAccountType
				from
				  amount
				timestamp
				asset {id, name}
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
		for _, w := range party.Withdrawals {
			// string to int
			amount, err := strconv.Atoi(w.Amount)
			if err != nil {
				fmt.Errorf("failed to convert Withdrawal amount to string", err)
			}

			if w.Asset.Id == s.cfg.VegaAssets[0] &&
				w.Status == "STATUS_FINALIZED" &&
				amount >= minWithdrawalThreshold &&
				w.CreatedAt.After(s.cfg.StartTime) &&
				w.CreatedAt.Before(s.cfg.EndTime) {
				withdrawalCount++
				fmt.Println(withdrawalCount)
			}
		}

		for _, party := range sParties {
			depositCount := 0
			for _, w := range party.Deposits {
				// string to int
				amount, err := strconv.Atoi(w.Amount)
				if err != nil {
					fmt.Errorf("failed to convert deposit amount to string", err)
				}

				if w.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Status == "STATUS_FINALIZED" &&
					amount >= minDepositThreshold &&
					w.CreatedAt.After(s.cfg.StartTime) &&
					w.CreatedAt.Before(s.cfg.EndTime) {
					depositCount++
					fmt.Println(depositCount)
				}
			}
		}

		for _, party := range sParties {
			transferCount := 0
			for _, w := range party.transfers {
				// string to int
				amount, err := strconv.Atoi(w.Amount)
				if err != nil {
					fmt.Errorf("failed to convert transfer amount to string", err)
				}

				if w.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Status == "STATUS_FINALIZED" &&
					amount >= minTransferThreshold &&
					w.CreatedAt.After(s.cfg.StartTime) &&
					w.CreatedAt.Before(s.cfg.EndTime) {
					transferCount++
					fmt.Println(transferCount)
				}
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
