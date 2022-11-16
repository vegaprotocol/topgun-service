package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortByPartyPositionsExisting(socials map[string]verifier.Social) ([]Participant, error) {

	// Open our jsonFile
	jsonFile, err := os.Open("test.json")
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

	fmt.Println(alreadyTraded)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// Query all accounts for parties on Vega network
	gqlQueryPartiesAccounts := `{
		partiesConnection {
	      edges {
	        node {
	          id
	          positionsConnection() {
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
		for _, traded := range alreadyTraded {
			if traded.PublicKey == party.ID {
				if s, err := strconv.ParseFloat(traded.Data[0], 32); err == nil {
					PnL = s
				}
				fmt.Println(PnL)
			}
		}
		realisedPnL := 0.0
		unrealisedPnL := 0.0
		openVolume := 0.0
		marketID, err := s.getAlgorithmConfig("marketID")
		if err == nil {
			for _, acc := range party.PositionsConnection.Edges {
				if acc.Position.Market.ID == marketID {
					if s, err := strconv.ParseFloat(acc.Position.RealisedPNL, 32); err == nil {
						realisedPnL = s
					}
					if t, err := strconv.ParseFloat(acc.Position.UnrealisedPNL, 32); err == nil {
						unrealisedPnL = t
					}
					if u, err := strconv.ParseFloat(acc.Position.OpenVolume, 32); err == nil {
						openVolume = u
					}
					PnL += (realisedPnL + unrealisedPnL)
				}
			}
		}

		if (realisedPnL != 0.0) || (unrealisedPnL != 0.0) || (openVolume != 0.0) {
			if party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", party.twitterID, party.social, party.ID)
			}

			t := time.Now().UTC()
			participants = append(participants, Participant{
				PublicKey:     party.ID,
				TwitterUserID: party.twitterID,
				TwitterHandle: party.social,
				Data:          []string{strconv.FormatFloat(PnL, 'f', 10, 32)},
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
