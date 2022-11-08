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

func (s *Service) sortByPartyAccountGeneralProfit(socials map[string]verifier.Social, hasCommittedLP bool) ([]Participant, error) {
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

	gqlQueryPartiesAccounts := `query ($assetId: ID) {
		partiesConnection {
		  edges {
			node {
			  id
			  accountsConnection(assetId: $assetId) {
				edges {
				  node {
					asset {
					  id
					  symbol
					}
					balance
					type
				  }
				}
			  }
			  depositsConnection {
				edges {
				  node {
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
			  }
			}
		  }
		}
	  }`

	if hasCommittedLP {
		gqlQueryPartiesAccounts = `query ($assetId: ID) {
			partiesConnection {
			  edges {
				node {
				  id
				  accountsConnection(assetId: $assetId) {
					edges {
					  node {
						asset {
						  id
						  symbol
						}
						balance
						type
					  }
					}
				  }
				  depositsConnection {
					edges {
					  node {
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
				  }
				  liquidityProvisionsConnection {
					edges {
					  node {
						id
						market {
						  id
						}
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
		map[string]string{"assetId": s.cfg.VegaAssets[0]},
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
		if hasCommittedLP && party.LPsConnection.Edges != nil && len(party.LPsConnection.Edges) > 0 {
			for _, lp := range party.LPsConnection.Edges {
				if lp.LP.Market.ID == marketID {
					calculateBalance = true
					log.WithFields(log.Fields{"partyID": party.ID, "totalLPs": len(party.LPsConnection.Edges)}).Info("Party has LPs on correct market")
					break
				}
			}
		}

		if calculateBalance || !hasCommittedLP {
			balanceGeneral := party.Balance(s.cfg.VegaAssets[0], int(decimalPlaces), "ACCOUNT_TYPE_GENERAL", "ACCOUNT_TYPE_MARGIN")
			depositTotal := party.CalculateTotalDeposits(s.cfg.VegaAssets[0], int(decimalPlaces))
			if depositTotal == 0 {
				continue
			}

			log.Infof("[[ Balance %f | TotalDeposits %f | Balance-TotalDeposits/TotalDeposits: %f ]]",
				balanceGeneral, depositTotal, balanceGeneral/depositTotal)

			var sortNum float64
			profit := (balanceGeneral - depositTotal) / depositTotal
			sortNum = profit

			balanceGeneralStr := fmt.Sprintf("%f", balanceGeneral) //strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
			totalDepositStr := fmt.Sprintf("%f", depositTotal)     //strconv.FormatFloat(depositTotal, 'f', int(decimalPlaces), 32)
			partyProfitStr := strconv.FormatFloat(profit, 'f', 6, 32)
			if profit > 0 {
				partyProfitStr = fmt.Sprintf("+%s", partyProfitStr)
			}
			formattedBalancePosition := fmt.Sprintf("%s (%s)", balanceGeneralStr, partyProfitStr)

			// Only include participants who have non-zero positions
			if balanceGeneral != depositTotal {
				if party.blacklisted {
					log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
				}

				t := time.Now().UTC()
				participants = append(participants, Participant{
					PublicKey:     party.ID,
					TwitterHandle: party.social,
					TwitterUserID: party.twitterID,
					Data:          []string{formattedBalancePosition, balanceGeneralStr, totalDepositStr, partyProfitStr},
					sortNum:       sortNum,
					CreatedAt:     t,
					UpdatedAt:     t,
					isBlacklisted: party.blacklisted,
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
