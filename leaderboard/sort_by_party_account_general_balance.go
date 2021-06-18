package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func (s *Service) sortByPartyAccountGeneralBalance(socials map[string]string) ([]Participant, error) {

	// Warning the change to use accounts only removes the ability to check that
	// a user has traded inside (or outside) the time window via checking their trades.

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

	// xyzAsset top up amount formatted to x DP of market e.g. 1000000000
	// Please set to 0 before asset has been
	topupAssetTotal, err := strconv.ParseFloat(s.cfg.TopupAssetTotal, 64)
	if err != nil {
		log.WithError(err).Fatal(
			"Failed to parse top-up asset total from config, is it missing?")
	}

	var totalTraded, totalNoAccounts int
	participants := []Participant{}
	for _, party := range sParties {
		log.WithFields(log.Fields{"partyID": party.ID}).Debug("Getting balances for party")

		// Observed that parties that are completely wiped out can have no General balance of xyzAsset and no margin
		// So in addition to hasTraded logic we need to capture those on social list who have neither account
		// and therefore zero balance in general for asset
		var balanceGeneral float64
		var hasNoAccounts, hasTraded bool
		if topupAssetTotal > 0 && !party.HasAccounts(s.cfg.VegaAsset,"General", "Margin") {
			fmt.Println(party.social, "has no accounts")
			totalNoAccounts++
			balanceGeneral = 0
			hasNoAccounts = true
		} else {
			balanceGeneral = party.Balance(s.cfg.VegaAsset,"General", "Margin")
			hasTraded = party.HasTraded(s.cfg.VegaAsset, topupAssetTotal)
		}

		var sortNum float64
		var balanceGeneralStr string
		if  hasNoAccounts || hasTraded {
			// Parties that have traded or have no xyzAsset accounts any more
			totalTraded++
			balanceGeneralStr = strconv.FormatFloat(balanceGeneral, 'f', 5, 32)
			sortNum = balanceGeneral
		} else {
			// Parties that have not participated in the competition
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
