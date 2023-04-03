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

func (s *Service) sortByPartyPositionsWithTransfers(socials map[string]verifier.Social) ([]Participant, error) {

	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	decimalPlaces, err := strconv.ParseFloat(decimalPlacesStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `{
		partiesConnection(pagination: {first: 50 }) {
		  edges {
			node {
			  id
			  positionsConnection {
				edges {
				  node {
					market {
					  id
					}
					openVolume
					realisedPNL
					averageEntryPrice
					unrealisedPNL
					realisedPNL
				  }
				}
			  }
			  transfersConnection(direction: To) {
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
	ctx := context.Background()

	connection, err := getPartiesConnection(
		ctx,
		s.cfg.VegaGraphQLURL.String(),
		gqlQueryPartiesAccounts,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	parties := connection.Edges
	endCursor := connection.PageInfo.EndCursor

	for {
		gqlQueryPartiesAccounts2 := `query ($endCursor: String!) {
			partiesConnection(pagination: {first: 50, after: $endCursor}) {
			  edges {
				node {
				  id
				  positionsConnection {
					edges {
					  node {
						market {
						  id
						}
						openVolume
						realisedPNL
						averageEntryPrice
						unrealisedPNL
						realisedPNL
					  }
					}
				  }
				  transfersConnection(direction: To) {
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
		connection2, err := getPartiesConnection(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryPartiesAccounts2,
			map[string]string{"endCursor": endCursor},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get list of parties in loop: %w", err)
		}

		parties = append(parties, connection2.Edges...)
		pageInfo := connection2.PageInfo

		pageInfo, err = getPageInfo(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryPartiesAccounts2,
			map[string]string{"endCursor": endCursor},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get page info: %w", err)
		}

		fmt.Println("got ", len(parties), "end?", pageInfo.EndCursor)

		if pageInfo.NextPage == false {
			fmt.Println("done")
			break
		} else {
			endCursor = pageInfo.EndCursor
		}

	}

	fmt.Println(len(parties))

	// filter parties and add social handles
	sParties := socialParties(socials, parties)
	participants := []Participant{}
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range sParties {
		transfer := 1000.0
		deposit := 0.0
		if len(party.TransfersConnection.Edges) != 0 {
			for _, w := range party.TransfersConnection.Edges {
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				if w.Transfer.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Transfer.Timestamp.After(s.cfg.StartTime) &&
					w.Transfer.Timestamp.Before(s.cfg.EndTime) {
					transfer, err = strconv.ParseFloat(w.Transfer.Amount, 64)
				}
			}
		}

		for _, d := range party.DepositsConnection.Edges {
			if err != nil {
				fmt.Errorf("failed to convert Withdrawal amount to string", err)
			}

			if d.Deposit.Asset.Id == s.cfg.VegaAssets[0] &&
				d.Deposit.Status == "STATUS_FINALIZED" &&
				d.Deposit.CreatedAt.After(s.cfg.StartTime) &&
				d.Deposit.CreatedAt.Before(s.cfg.EndTime) {
				deposit, err = strconv.ParseFloat(d.Deposit.Amount, 64)
			}
		}
		PnL := 0.0
		realisedPnL := 0.0
		unrealisedPnL := 0.0
		openVolume := 0.0
		dataFormatted := ""
		if err == nil {
			for _, acc := range party.PositionsConnection.Edges {
				for _, marketID := range s.cfg.MarketIDs {
					if acc.Position.Market.ID == marketID {
						if s, err := strconv.ParseFloat(acc.Position.RealisedPNL, 32); err == nil {
							realisedPnL += s
						}
						if t, err := strconv.ParseFloat(acc.Position.UnrealisedPNL, 32); err == nil {
							unrealisedPnL += t
						}
						if u, err := strconv.ParseFloat(acc.Position.OpenVolume, 32); err == nil {
							openVolume += u
						}
						PnL = (realisedPnL + unrealisedPnL) - transfer + deposit
						dataFormatted = strconv.FormatFloat(PnL, 'f', 10, 32)
					}
				}

			}
		}

		if (realisedPnL != 0.0) || (unrealisedPnL != 0.0) || (openVolume != 0.0) {
			if party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
			}

			t := time.Now().UTC()
			if PnL != 0 {
				dpMultiplier := math.Pow(10, decimalPlaces)
				total := PnL / dpMultiplier
				dataFormatted = strconv.FormatFloat(total, 'f', 10, 32)
			}

			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{dataFormatted},
				sortNum:       PnL,
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
