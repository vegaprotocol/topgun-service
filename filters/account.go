package filters

import (
	"math"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/datanode"
)

func FilterByAccountBalance(partyID string, assetIds []string, accountTypes []string) (bool, float64) {
	// Contact datanode and retrieve asset details from assetIds
	// todo or load from config

	// Contact datanode and retrieve party details
	accounts, err := datanode.LoadAccountsForParty(partyID)
	if err != nil {
		return false, -1
	}

	//log.Infof("%+v", accounts)

	// Accumulate total balance from account types
	balanceMultiAsset := 0.0
	for _, assetId := range assetIds {
		balanceMultiAsset += getBalance(assetId,  accountTypes, accounts)
	}
	// Return balance totals
	return balanceMultiAsset > -1, balanceMultiAsset
}

func hasString(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func getBalance(assetId string, accountTypes []string, accounts []datanode.Account) float64 {
	var accu float64
	accu = 0

	decimalPlaces := 0
	for _, acc := range accounts {
		if acc.Asset == assetId && hasString(accountTypes, acc.Type) {
			decimalPlaces = 5 //acc.Asset.Decimals
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/%s balance [Balance]", assetId, accountTypes)
				return 0
			}
			accu += v
		}
	}
	if accu != 0 && decimalPlaces > 0 {
		dpMultiplier := math.Pow(10, float64(decimalPlaces))
		accu = accu / dpMultiplier
	}
	return accu
}

