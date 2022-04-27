package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func (s *Service) sortByPartyAccountGeneralProfit(socials map[string]string, hasCommittedLP bool) ([]Participant, error) {
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
				deposits{
					id
					asset { 
						id
						name
						decimals 
						symbol 
					}
					status
					createdTimestamp
					creditedTimestamp
					amount
				}
			}
		}`

	if hasCommittedLP {
		gqlQueryPartiesAccounts = `query($assetId: String!) {
			parties {
				id
				accounts(asset: $assetId){
					asset {
						symbol
					}
					balance
					type
				}
				deposits{
					id
					asset { 
						id
						name
						decimals 
						symbol 
					}
					status
					createdTimestamp
					creditedTimestamp
					amount
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
	}

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

		// Default is to rank by asset profit for all traders, inc directional (non LP)
		// Optional flag to filter for parties that have LP on the configured market only
		calculateBalance := false
		if hasCommittedLP && party.LPs != nil && len(party.LPs) > 0 {
			for _, lp := range party.LPs {
				if lp.Market.ID == marketID {
					calculateBalance = true
					log.WithFields(log.Fields{"partyID": party.ID, "totalLPs": len(party.LPs)}).Info("Party has LPs on correct market")
					break
				}
			}
		}

		if calculateBalance || !hasCommittedLP {
			balanceGeneral := party.Balance(s.cfg.VegaAsset, decimalPlaces, "General", "Margin")
			depositTotal := party.CalculateTotalDeposits(s.cfg.VegaAsset, decimalPlaces)
			if depositTotal == 0 {
				continue
			}

			log.Infof("[[ Balance %f | TotalDeposits %f | Balance-TotalDeposits/TotalDeposits: %f ]]",
				balanceGeneral, depositTotal, balanceGeneral/depositTotal)

			var sortNum float64
			profit := (balanceGeneral - depositTotal)/depositTotal
			sortNum = profit

			balanceGeneralStr := fmt.Sprintf("%f", balanceGeneral) //strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
			totalDepositStr := fmt.Sprintf("%f", depositTotal) //strconv.FormatFloat(depositTotal, 'f', int(decimalPlaces), 32)
			partyProfitStr := strconv.FormatFloat(profit, 'f', 6, 32)
			if profit > 0 {
				partyProfitStr = fmt.Sprintf("+%s", partyProfitStr)
			}
			formattedBalancePosition := fmt.Sprintf("%s (%s)", balanceGeneralStr, partyProfitStr)

			// Only include participants who have non-zero positions
			if balanceGeneral != depositTotal {
				t := time.Now().UTC()
				participants = append(participants, Participant{
					PublicKey:     party.ID,
					TwitterHandle: party.social,
					Data:          []string{formattedBalancePosition, balanceGeneralStr, totalDepositStr, partyProfitStr},
					sortNum:       sortNum,
					CreatedAt:     t,
					UpdatedAt:     t,
				})
			}
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}


