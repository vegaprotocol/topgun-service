package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByAssetTransfers(socials map[string]verifier.Social) ([]Participant, error) {
	// The minimum number of unique withdrawals needed to achieve this reward
	minTransferThreshold := 0

	gqlQueryPartiesAccounts := `query{
		parties {
		  id
		  transfersConnection {
			edges {
			  node {
				id
				fromAccountType
				toAccountType
				from
				amount
				timestamp
				asset {
				  id
				  name
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

	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		transferCount := 0
		if len(party.TransfersConnection.Edges) != 0 {
			fmt.Println(party.TransfersConnection)
			for _, w := range party.TransfersConnection.Edges {
				// string to int
				amount, err := strconv.Atoi(w.Node.Amount)
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				fmt.Println(w.Node.Asset.Id)
				fmt.Println(amount)
				fmt.Println(w.Node.Timestamp)
				if w.Node.Asset.Id == s.cfg.VegaAssets[0] &&
					amount >= minTransferThreshold &&
					w.Node.Timestamp.After(s.cfg.StartTime) &&
					w.Node.Timestamp.Before(s.cfg.EndTime) {
					transferCount++
				}
			}

		}

		var sortNum float64
		transferCountStr := strconv.FormatFloat(float64(transferCount), 'f', int(0), 32)
		sortNum = float64(transferCount)

		if transferCount > 0 {
			utcNow := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterHandle: party.social,
				TwitterUserID: party.twitterID,
				Data:          []string{transferCountStr},
				sortNum:       sortNum,
				CreatedAt:     utcNow,
				UpdatedAt:     utcNow,
				isBlacklisted: party.blacklisted,
			})
		}

	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
