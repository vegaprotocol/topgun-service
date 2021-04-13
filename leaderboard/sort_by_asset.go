package leaderboard

import (
	"sort"
	"strconv"
)

func (s *Service) sortByAsset(parties []Party) []Participant {

	participants := []Participant{}
	for _, party := range parties {
		balanceGeneral := party.Balance(s.cfg.VegaAsset, "General")
		participants = append(participants, Participant{
			PublicKey:     party.ID,
			TwitterHandle: party.social,
			Data:          []string{strconv.FormatFloat(balanceGeneral, 'f', 5, 32)},
			sortNum:       balanceGeneral,
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}

	sort.Slice(participants, sortFunc)
	return participants
}
