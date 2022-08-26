package leaderboard

import "github.com/vegaprotocol/topgun-service/leaderboard/filters"

func GetAccountBalance() bool {
	return filters.GetFilter()
}