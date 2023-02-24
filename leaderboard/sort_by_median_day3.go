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

func (s *Service) sortByPartyPositionsMedianDay3(socials map[string]verifier.Social) ([]Participant, error) {

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

	// Open our jsonFile
	jsonFile1, err := os.Open("day1.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// read our opened jsonFile as a byte array.
	byteValue1, _ := ioutil.ReadAll(jsonFile1)

	day1Traded := []Participant{}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'alreadyTraded' which we defined above
	json.Unmarshal(byteValue1, &day1Traded)

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile1.Close()

	// Open our jsonFile
	jsonFile2, err := os.Open("day2.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// read our opened jsonFile as a byte array.
	byteValue2, _ := ioutil.ReadAll(jsonFile2)

	day2Traded := []Participant{}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'alreadyTraded' which we defined above
	json.Unmarshal(byteValue2, &day2Traded)

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile2.Close()

	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `{
		partiesConnection (pagination: {first: 1000000}) {
	      edges {
	        node {
	          id
	          positionsConnection {
	            edges {
	              node {
	              market{id}
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
						percentagePnL = ((PnL / dpMultiplier) / 2000) * 100
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
			day1Total := 0.0
			day2Total := 0.0
			NewTotal := 0.0
			if PnL != 0 {
				total = PnL / dpMultiplier
				for _, traded := range alreadyTraded {
					if traded.PublicKey == party.ID {
						if s, err := strconv.ParseFloat(traded.Data[0], 32); err == nil {
							percentagePnL = ((total - s) / (s + 2000)) * 100
						}
					}
				}
				for _, traded := range day1Traded {
					if traded.PublicKey == party.ID {
						if t, err := strconv.ParseFloat(traded.Data[0], 32); err == nil {
							day1Total = t
						}
					}
				}
				for _, traded := range day2Traded {
					if traded.PublicKey == party.ID {
						if u, err := strconv.ParseFloat(traded.Data[0], 32); err == nil {
							day2Total = u
						}
					}
				}
				NewTotal = median([]float64{percentagePnL, day1Total, day2Total})

				dataFormatted = strconv.FormatFloat(NewTotal, 'f', 10, 32)
			}

			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{dataFormatted},
				sortNum:       NewTotal,
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
