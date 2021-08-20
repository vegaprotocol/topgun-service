package leaderboard

import (
	"sort"

	"github.com/vegaprotocol/topgun-service/util"
	"github.com/vegaprotocol/topgun-service/verifier"
)

func (s *Service) sortBySocialRegistration(socials *verifier.Socials) ([]Participant, error) {

	// A leaderboard to show only registered social accounts, used to verify that they have
	// registered successfully with the twitter service/vega incentives

	count := 0
	participants := []Participant{}
	for _, s := range *socials {
		count++
		participants = append(participants, Participant{
			PublicKey:     s.PartyID,
			TwitterHandle: s.TwitterHandle,
			CreatedAt:     util.TimeFromUnixTimeStamp(s.CreatedAt),
			UpdatedAt:     util.TimeFromUnixTimeStamp(s.UpdatedAt),
			Data:          []string{"Registered"},
			sortNum:       float64(count),
		})
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
