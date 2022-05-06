package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByPartyAccountGeneralBalanceAndLP(socials map[string]verifier.Social) ([]Participant, error) {
	// Grab the market ID for the market we're targeting
	marketID, err := s.getAlgorithmConfig("marketID")
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	decimalPlaces, err := strconv.ParseInt(decimalPlacesStr, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	gqlQueryPartiesAccounts := `query($assetId: String!) {
		parties {
			id
			accounts(asset: $assetId){
				asset {
					symbol
				}
				balance
				type
			}
			liquidityProvisions {
				id
				market { id, name }
				commitmentAmount
				createdAt 
				reference
				buys {
					liquidityOrder {
						reference
						proportion
						offset
					}
				}
				sells {
					liquidityOrder {
						reference
						proportion
						offset
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
		map[string]string{"assetId": s.cfg.VegaAsset},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	// filter parties and add social handles
	sParties := socialParties(socials, parties)

	participants := []Participant{}
	for _, party := range sParties {
		// Check for matching parties who have committed LP :)
		if party.LPs != nil && len(party.LPs) > 0 {
			for _, lp := range party.LPs {
				if lp.Market.ID == marketID {
					log.WithFields(log.Fields{"partyID": party.ID, "totalLPs": len(party.LPs)}).Info("Party has LPs on correct market")

					balanceGeneral := party.Balance(s.cfg.VegaAsset, decimalPlaces, "General", "Margin")
					var sortNum float64

					balanceGeneralStr := strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
					sortNum = balanceGeneral

					utcNow := time.Now().UTC()
					participants = append(participants, Participant{
						PublicKey:     party.ID,
						TwitterHandle: party.social,
						TwitterUserID: party.twitterID,
						Data:          []string{balanceGeneralStr},
						sortNum:       sortNum,
						CreatedAt:     utcNow,
						UpdatedAt:     utcNow,
						isBlacklisted: party.blacklisted,
					})
					break
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
