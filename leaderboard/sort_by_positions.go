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

func (s *Service) sortByPartyPositions(socials map[string]verifier.Social) ([]Participant, error) {
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
	gqlQueryPositionsParties := `{
	positions {
		edges {
		node {
			market {
			id
			}
			party {
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
	}`
	ctx := context.Background()
	positions, err := getPositions(
		ctx,
		s.cfg.VegaGraphQLURL.String(),
		gqlQueryPositionsParties,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of positions: %w", err)
	}

	// filter parties and add social handles
	sPositions := socialPositions(socials, positions)
	participants := []Participant{}
	for _, position := range sPositions {
		PnL := 0.0
		realisedPnL := 0.0
		unrealisedPnL := 0.0
		openVolume := 0.0
		if err == nil {
			for _, marketID := range s.cfg.MarketIDs {
				if position.Market.ID == marketID {
					if s, err := strconv.ParseFloat(position.RealisedPNL, 32); err == nil {
						realisedPnL += s
					}
					if t, err := strconv.ParseFloat(position.UnrealisedPNL, 32); err == nil {
						unrealisedPnL += t
					}
					if u, err := strconv.ParseFloat(position.OpenVolume, 32); err == nil {
						openVolume += u
					}
					PnL = realisedPnL + unrealisedPnL
				}
			}
		}

		if (realisedPnL != 0.0) || (unrealisedPnL != 0.0) || (openVolume != 0.0) {
			if position.Party.blacklisted {
				log.Infof("Blacklisted party added: %d, %s, %s", position.PartytwitterID, position.Partysocial, position.PartyID)
			}
			t := time.Now().UTC()
			dataFormatted := ""
			if PnL != 0 {
				dpMultiplier := math.Pow(10, decimalPlaces)
				total := PnL / dpMultiplier
				dataFormatted = strconv.FormatFloat(total, 'f', 10, 32)
			}
			participants = append(participants, Participant{
				PublicKey:     position.Party.ID,
				TwitterUserID: position.Party.twitterID,
				TwitterHandle: position.Party.social,
				Data:          []string{dataFormatted},
				sortNum:       PnL,
				CreatedAt:     t,
				UpdatedAt:     t,
				isBlacklisted: position.Party.blacklisted,
			})
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
