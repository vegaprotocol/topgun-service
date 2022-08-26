package leaderboard

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vegaprotocol/topgun-service/filters"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func RankByAccountBalances(participants []Participant) (result []Participant, err error) {
	if len(participants) == 0 {
		return nil, errors.New("No participants passed to RankByAccountBalances")
	}

	cfgAssetIds := []string{"XYZepsilon"}
	cfgAccountTypes := []string{"ACCOUNT_TYPE_GENERAL", "ACCOUNT_TYPE_MARGIN"}

	for _, p := range participants {
		// filter 1 -- account balances
		isRanked, rankVal := filters.FilterByAccountBalance(p.PublicKey, cfgAssetIds, cfgAccountTypes)
		if isRanked {
			p.sortNum = rankVal
		}
	}

	return nil, nil
}

func (s *Service) sortByPartyAccountGeneralBalance(socials map[string]verifier.Social) ([]Participant, error) {
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	// filter parties and add social handles
	sParties := socialParties(socials, parties)

	participants := []Participant{}
	for _, party := range sParties {

		balanceGeneral := party.Balance(s.cfg.VegaAssets[0], int(decimalPlaces), "ACCOUNT_TYPE_GENERAL", "ACCOUNT_TYPE_MARGIN")
		var sortNum float64

		balanceGeneralStr := strconv.FormatFloat(balanceGeneral, 'f', int(decimalPlaces), 32)
		sortNum = balanceGeneral
		utcNow := time.Now().UTC()
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			TwitterUserID: party.twitterID,
			Data:          []string{balanceGeneralStr},
			sortNum:       sortNum,
			CreatedAt:     utcNow,
			UpdatedAt:     utcNow,
			isBlacklisted: party.blacklisted,
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
