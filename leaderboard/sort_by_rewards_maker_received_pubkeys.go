package leaderboard

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

var gqlQueryPartiesAccountsMakerReceivedPubkeys string = `query ($pagination: Pagination!) {
	partiesConnection(pagination: $pagination) {
	  edges {
		node {
		  id
		  rewardsConnection {
			edges {
			  node {
				amount
				asset {
				  id
				}
				marketId
				rewardType
				receivedAt
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

func (s *Service) sortByPartyRewardsMakerReceivedPubkeys(socials map[string]verifier.Social) ([]Participant, error) {

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
			gqlQueryPartiesAccountsMakerReceived,
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
	participants := []Participant{}
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range partyEdges {
		rewards := 0.0
		dataFormatted := "0.0"
		if len(party.Party.RewardsConnection.Edges) != 0 {
			for _, w := range party.Party.RewardsConnection.Edges {
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				if w.Reward.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Reward.ReceivedAt.After(s.cfg.StartTime) &&
					w.Reward.ReceivedAt.Before(s.cfg.EndTime) &&
					w.Reward.RewardType == "ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES" {
					rewards1, err := strconv.ParseFloat(w.Reward.Amount, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to convert reward amount into float: %w", err)
					}
					rewards += rewards1
					fmt.Println(rewards)
				}
			}
		}

		if rewards != 0.0 {
			if party.Party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.Party.ID)
			}

			t := time.Now().UTC()
			if rewards != 0 {
				dpMultiplier := math.Pow(10, decimalPlaces)
				total := rewards / dpMultiplier
				fmt.Println("Total is")
				fmt.Println(total)
				dataFormatted = strconv.FormatFloat(rewards, 'f', 10, 32)
			}

			participants = append(participants, Participant{
				PublicKey:     party.Party.ID,
				Data:          []string{dataFormatted},
				sortNum:       rewards,
				CreatedAt:     t,
				UpdatedAt:     t,
				isBlacklisted: party.Party.blacklisted,
			})
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
