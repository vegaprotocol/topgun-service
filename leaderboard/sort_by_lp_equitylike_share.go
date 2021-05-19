package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

type PartyJustID struct {
	ID string `json:"id"`
}

type LiquidityProvision struct {
	CreatedAt        time.Time   `json:"createdAt"`
	CommitmentAmount int         `json:"commitmentAmount"`
	Fee              string      `json:"fee"` // float
	Status           string      `json:"status"`
	Party            PartyJustID `json:"party"`
}

type LiquidityProviderFeeShare struct {
	AverageEntryValuation string      `json:"averageEntryValuation"` // uint64 ** 10^DP
	EquityLikeShare       string      `json:"equityLikeShare"`       // float
	Party                 PartyJustID `json:"party"`
}

type MarketData struct {
	LPFeeShare []LiquidityProviderFeeShare `json:"liquidityProviderFeeShare"`
}

type Market struct {
	LiquidityProvisions []LiquidityProvision `json:"liquidityProvisions"`
	Data                MarketData           `json:"data"`
}

type marketResponse struct {
	Market Market `json:"market"`
}

func (s *Service) sortByLPEquitylikeShare(socials map[string]string) ([]Participant, error) {

	marketID, err := s.getAlgorithmConfig("marketID")
	if err != nil {
		return nil, err
	}

	gqlQuery := `query ($marketID: ID!) {
		market(id:$marketID) {
			liquidityProvisions {
				createdAt
				commitmentAmount
				fee
				status
				party {
					id
				}
			}
			data {
				liquidityProviderFeeShare {
					averageEntryValuation
					equityLikeShare
					party {
						id
					}
				}
			}
		}
	}`
	client := graphql.NewClient(s.cfg.VegaGraphQLURL.String())
	req := graphql.NewRequest(gqlQuery)
	req.Var("marketID", marketID)
	req.Header.Set("Cache-Control", "no-cache")

	commitmentAmount := make(map[string]int)
	averageEntryValuation := make(map[string]float64)
	equitylikeShare := make(map[string]float64)
	fee := make(map[string]float64)

	var response marketResponse
	ctx := context.Background()
	if err = client.Run(ctx, req, &response); err != nil {
		return nil, fmt.Errorf("failed to get market liquidity provider info: %w", err)
	}
	for i, lp := range response.Market.LiquidityProvisions {
		if lp.Status != "Active" {
			continue
		}
		if lp.CreatedAt.Before(s.cfg.StartTime) || lp.CreatedAt.After(s.cfg.EndTime) {
			continue
		}

		log.WithFields(log.Fields{
			"i":                i,
			"createdAt":        lp.CreatedAt,
			"commitmentAmount": lp.CommitmentAmount,
			"fee":              lp.Fee,
			"party":            lp.Party.ID,
		}).Debug("Liquidity provision")
		commitmentAmount[lp.Party.ID] = lp.CommitmentAmount
		fee[lp.Party.ID], err = strconv.ParseFloat(lp.Fee, 64)
		if err != nil {
			fee[lp.Party.ID] = -5.0
		}
	}
	for i, lpfs := range response.Market.Data.LPFeeShare {
		log.WithFields(log.Fields{
			"i":                     i,
			"averageEntryValuation": lpfs.AverageEntryValuation,
			"equityLikeShare":       lpfs.EquityLikeShare,
			"party":                 lpfs.Party.ID,
		}).Debug("Liquidity provision fee share")
		averageEntryValuation[lpfs.Party.ID], err = strconv.ParseFloat(lpfs.AverageEntryValuation, 64)
		if err != nil {
			averageEntryValuation[lpfs.Party.ID] = -6.0
		}
		equitylikeShare[lpfs.Party.ID], err = strconv.ParseFloat(lpfs.EquityLikeShare, 64)
		if err != nil {
			equitylikeShare[lpfs.Party.ID] = -7.0
		}
	}

	parties := make([]Party, 0)
	sParties := socialParties(socials, parties)
	participants := []Participant{}
	for _, party := range sParties {
		ca, found := commitmentAmount[party.ID]
		if !found {
			ca = 0
		}
		aev, found := averageEntryValuation[party.ID]
		if !found {
			aev = 0.0
		}
		els, found := equitylikeShare[party.ID]
		if !found {
			els = 0.0
		}
		f := fee[party.ID]
		if !found {
			f = 0.0
		}

		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data: []string{
				fmt.Sprintf("%d", ca/10000),      // TODO: market decimal places
				fmt.Sprintf("%.5f", aev/10000.0), // TODO: market decimal places
				fmt.Sprintf("%.6f%%", els*100.0),
				fmt.Sprintf("%.6f%%", f*100.0),
			},
			sortNum: els,
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
