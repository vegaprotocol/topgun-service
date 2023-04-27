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

var gqlQueryPartiesAccountsMakerPaid string = `query ($pagination: Pagination!) {
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

func (s *Service) sortByPartyRewardsMakerPaid(socials map[string]verifier.Social) ([]Participant, error) {

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
			gqlQueryPartiesAccountsMakerPaid,
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
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range sParties {
		rewards := 0.0
		dataFormatted := "0.0"
		if len(party.RewardsConnection.Edges) != 0 {
			for _, w := range party.RewardsConnection.Edges {
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				if w.Reward.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Reward.ReceivedAt.After(s.cfg.StartTime) &&
					w.Reward.ReceivedAt.Before(s.cfg.EndTime) &&
					w.Reward.RewardType == "ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES" {
					rewards, err = strconv.ParseFloat(w.Reward.Amount, 64)
				}
			}
		}

		if rewards != 0.0 {
			if party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
			}

			t := time.Now().UTC()
			if rewards != 0 {
				dpMultiplier := math.Pow(10, decimalPlaces)
				total := rewards / dpMultiplier
				dataFormatted = strconv.FormatFloat(total, 'f', 10, 32)
			}

			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{dataFormatted},
				sortNum:       rewards,
				CreatedAt:     t,
				UpdatedAt:     t,
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
