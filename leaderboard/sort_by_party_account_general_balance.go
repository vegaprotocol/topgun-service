package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func (s *Service) sortByPartyAccountGeneralBalance(socials map[string]string) ([]Participant, error) {
	// marketID, err := s.getAlgorithmConfig("marketID")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get algorithm config: %w", err)
	// }

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
		}
	}`
	// gqlQueryPartiesTrades := `query($marketId: ID!, $partyId: ID!) {
	// 	parties(id: $partyId) {
	// 		trades(marketId: $marketId, first: 1, last: 2) {
	// 			id
	// 			createdAt
	// 		}
	// 	}
	// }`

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

	var totalTraded int
	participants := []Participant{}
	for _, party := range sParties {
		log.WithFields(log.Fields{"partyID": party.ID}).Debug("Getting balances for party")
		var topupAssetTotal float64 = 1000000000 // xyzAsset top up amount (todo: add to config)
		balanceGeneral := party.Balance(s.cfg.VegaAsset,"General", "Margin")
		hasTraded := party.HasTraded(s.cfg.VegaAsset, topupAssetTotal)
		var sortNum float64
		var balanceGeneralStr string
		// Observed that parties that are completely wiped out can have no General balance of xyzAsset and no margin
		// So in addition to hasTraded logic we need to capture those on social list who have neither account
		// and therefore zero balance in general for asset
		if hasTraded || balanceGeneral == 0 {
			// Traded parties
			totalTraded++
			balanceGeneralStr = strconv.FormatFloat(balanceGeneral, 'f', 5, 32)
			sortNum = balanceGeneral
		} else {
			// Untraded folks have not participated in the competition.
			balanceGeneralStr = "n/a"
			sortNum = -1.0e20
		}
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{balanceGeneralStr},
			sortNum:       sortNum,
		})
	}

	fmt.Println("")
	fmt.Println("Total parties traded: ", totalTraded)

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	fmt.Println("")
	fmt.Println("-----------")
	fmt.Println("Leaderboard")
	fmt.Println("-----------")
	for _, p := range participants {
		fmt.Println(p.TwitterHandle, " - ", p.Data[0])
	}
	fmt.Println("-----------")


	return participants, nil
}
