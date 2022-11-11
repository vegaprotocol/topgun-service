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

func (s *Service) sortByPartyPositions(socials map[string]verifier.Social) ([]Participant, error) {
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
	for _, party := range sParties {
		PnL := 0.0
		realisedPnL := 0.0
		unrealisedPnL := 0.0
		marketID, err := s.getAlgorithmConfig("marketID")
		if err == nil {
			for _, acc := range party.PositionsConnection.Edges {
				fmt.Println(acc.Position)
				if acc.Position.Market.ID == marketID {
					if s, err := strconv.ParseFloat(acc.Position.realisedPNL, 32); err == nil {
						realisedPnL = s
					}
					if t, err := strconv.ParseFloat(acc.Position.unrealisedPNL, 32); err == nil {
						unrealisedPnL = t
					}
					PnL = realisedPnL + unrealisedPnL
					fmt.Println(PnL)
				}
			}
		}

		if (realisedPnL != 0.0) || (unrealisedPnL != 0.0) {
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
