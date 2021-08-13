package leaderboard

import (
	"context"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
)

func (s *Service) sortByAssetDepositWithdrawal(socials map[string]string) ([]Participant, error) {
	gqlQuery := `query {
	  parties{
		id
		deposits {
		  amount
		  createdTimestamp
		  creditedTimestamp
          asset {
			id
			symbol
            source { __typename }
		  }
		}
		withdrawals{
		  amount
		  createdTimestamp
		  createdTimestamp
		  status
		  asset {
			id
			symbol
			source { __typename }
		  }
		}
	  }
	}`

	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		participationCount := 0
		minDepositAndWithdrawals := 3
		if s.hasDepositedErc20Assets(minDepositAndWithdrawals, party.Deposits) &&
			s.hasWithdrawnErc20Assets(minDepositAndWithdrawals, party.Withdrawals) {
			participationCount++
		}
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{fmt.Sprintf("%d", participationCount)},
			sortNum:       float64(participationCount),
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}

func (s *Service) hasDepositedErc20Assets(min int, deposits []Deposit) bool {
	totalDepositsForParty := 0
	if len(deposits) > 0 {
		for _, d := range deposits {
			if d.Asset.Source.Name == "ERC20" &&
				d.Status == "Finalized" &&
				d.CreatedAt.After(s.cfg.StartTime) &&
				d.CreatedAt.Before(s.cfg.EndTime) {
				totalDepositsForParty++
			}
			if totalDepositsForParty >= min {
				return true
			}
		}
	}
	return false
}

func (s *Service) hasWithdrawnErc20Assets(min int, withdrawals []Withdrawal) bool {
	totalWithdrawalsForParty := 0
	if len(withdrawals) > 0 {
		log.Info("party has withdrawals")
		for _, w := range withdrawals {
			if w.Asset.Source.Name == "ERC20" &&
				w.Status == "Finalized" &&
				w.CreatedAt.After(s.cfg.StartTime) &&
				w.CreatedAt.Before(s.cfg.EndTime) {
				totalWithdrawalsForParty++
			}
			if totalWithdrawalsForParty >= min {
				return true
			}
		}
	}
	return false
}
