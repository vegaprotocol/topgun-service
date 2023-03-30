package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
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

	// Open our jsonFile
	jsonFile, err := os.Open("initial_results.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	alreadyTraded := []Participant{}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'alreadyTraded' which we defined above
	json.Unmarshal(byteValue, &alreadyTraded)

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `{
		partiesConnection(pagination: {first: 100}) {
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
				pageInfo {
				  hasNextPage
				  hasPreviousPage
				  startCursor
				  endCursor
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
		}
	  }`
	ctx := context.Background()
	parties, err := getParties(
		ctx,
		s.cfg.VegaGraphQLURL.String(),
		gqlQueryPartiesAccounts,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	// filter parties and add social handles
	sParties := socialParties(socials, parties)
	participants := []Participant{}
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range sParties {
		transfer := 1000.0
		deposit := 1000.0
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
		percentagePnL := 0.0
		dataFormatted := ""
		dpMultiplier := math.Pow(10, decimalPlaces)
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
						PnL = (realisedPnL + unrealisedPnL)
						percentagePnL = ((PnL / dpMultiplier) / ((transfer / dpMultiplier) + (deposit / dpMultiplier))) * 100
						dataFormatted = strconv.FormatFloat(percentagePnL, 'f', 10, 32)
					}
				}

			}
		}

		if (realisedPnL != 0.0) || (unrealisedPnL != 0.0) || (openVolume != 0.0) {
			if party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
			}

			t := time.Now().UTC()
			total := 0.0
			if PnL != 0 {
				total = PnL / dpMultiplier
				for _, traded := range alreadyTraded {
					if traded.PublicKey == party.ID {
						if s, err := strconv.ParseFloat(traded.Data[0], 32); err == nil {
							percentagePnL = ((total - s) / (s + (transfer / dpMultiplier) + (deposit / dpMultiplier))) * 100
						}
					}
				}
				dataFormatted = strconv.FormatFloat(percentagePnL, 'f', 10, 32)
			}

			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{dataFormatted},
				sortNum:       percentagePnL,
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
