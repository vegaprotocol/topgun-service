package leaderboard

import (
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/filters"
)

func RankByAccountBalances(participants []Participant) ([]Participant, error) {
	if len(participants) == 0 {
		return nil, errors.New("No participants passed to RankByAccountBalances")
	}

	cfgAssetIds := []string{"XYZepsilon"}
	cfgAccountTypes := []string{"ACCOUNT_TYPE_GENERAL", "ACCOUNT_TYPE_MARGIN"}

	// Get asset details e.g. dp or set from config

	result := []Participant{}
	for _, p := range participants {
		if p.isBlacklisted {
			log.Infof("Blacklisted party added: %d, %s, %s",
				p.TwitterUserID, p.TwitterHandle, p.PublicKey)
		}
		//else {
		//	log.Infof("Social party added: %d, %s, %s",
		//		p.TwitterUserID, p.TwitterHandle, p.PublicKey)
		//}

		// filter 1 -- account balances
		utcNow := time.Now().UTC()
		isRanked, rankVal := filters.FilterByAccountBalance(p.PublicKey, cfgAssetIds, cfgAccountTypes)
		if isRanked {
			p.sortNum = rankVal
			p.CreatedAt = utcNow
			p.UpdatedAt = utcNow
			// todo: find max dp precision we've cheated and set to 10
			p.Data = []string{strconv.FormatFloat(rankVal, 'f', 10, 32)}
			result = append(result, p)
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return result, nil
}
