package leaderboard

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

type Asset struct {
	Id     string      `json:"id"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	Symbol string      `json:"symbol"`
	Source AssetSource `json:"source"`
}

type AssetSource struct {
	Name string `json:"__typename"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
}

type Deposit struct {
	Id         string    `json:"id"`
	Amount     string    `json:"amount"`
	Asset      Asset     `json:"asset"`
	CreatedAt  time.Time `json:"createdTimestamp"`
	CreditedAt time.Time `json:"creditedTimestamp"`
	Status     string    `json:"status"`
}

type Withdrawal struct {
	Amount     string    `json:"amount"`
	Asset      Asset     `json:"asset"`
	CreatedAt  time.Time `json:"createdTimestamp"`
	CreditedAt time.Time `json:"creditedTimestamp"`
	Status     string    `json:"status"`
}

type Order struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type Trade struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type Vote struct {
	Value    string    `json:"value"`
	Datetime time.Time `json:"datetime"`
}

type PartyVote struct {
	ProposalID string `json:"proposalId"`
	Vote       Vote   `json:"vote"`
}

type LiquidityOrder struct {
	Reference  string `json:"reference"`
	Proportion int    `json:"proportion"`
	Offset     string `json:"offset"`
}

type Buys struct {
	LiquidityOrder LiquidityOrder `json:"liquidityOrder"`
}

type Sells struct {
	LiquidityOrder LiquidityOrder `json:"liquidityOrder"`
}

type LiquidityProvision struct {
	ID               string    `json:"id"`
	Market           Market    `json:"market"`
	CommitmentAmount string    `json:"commitmentAmount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Status           string    `json:"status"`
	Fee              string    `json:"fee"`
	Version          string    `json:"version"`
	Reference        string    `json:"reference"`
	Buys             []Buys    `json:"buys"`
	Sells            []Sells   `json:"sells"`
}

type Party struct {
	ID          string               `json:"id"`
	Accounts    []Account            `json:"accounts"`
	Deposits    []Deposit            `json:"deposits"`
	Orders      []Order              `json:"orders"`
	Trades      []Trade              `json:"trades"`
	Votes       []PartyVote          `json:"votes"`
	Withdrawals []Withdrawal         `json:"withdrawals"`
	LPs         []LiquidityProvision `json:"liquidityProvisions"`
	social      string
}

type Market struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func hasString(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func (p *Party) Balance(assetName string, decimalPlaces int64, accountTypes ...string) float64 {
	var accu float64
	for _, acc := range p.Accounts {
		if acc.Asset.Id == assetName && hasString(accountTypes, acc.Type) {
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/%s balance [Balance]", assetName, accountTypes)
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

func (p *Party) CalculateTotalDeposits(asset string, decimalPlaces int64) float64 {
	// Total deposits made in asset
	var total float64
	total = 0
	for _, d := range p.Deposits {
		if d.Asset.Id == asset && d.Status == "Finalized" {
			amount, err := strconv.ParseFloat(d.Amount, 10)
			if err != nil {
				log.WithError(err).Error("Cannot parse the found epoch in delegation")
			}
			//log.Infof("Amount raw %s, converted: %f", d.Amount, amount)
			total += amount
		}
	}
	if total != 0 && decimalPlaces > 0 {
		dpMultiplier := math.Pow(10, float64(decimalPlaces))
		total = total / dpMultiplier
		log.Infof("Amount total %f, dpMultiplier: %f", total, dpMultiplier)
	}
	return total
}

type PartyList struct {
	Parties []Party `json:"parties"`
}

func getParties(
	ctx context.Context,
	gqlURL string,
	gqlQuery string,
	vars map[string]string,
	cli *http.Client,
) ([]Party, error) {

	if cli == nil {
		cli = &http.Client{Timeout: time.Second * 180}
	}
	client := graphql.NewClient(gqlURL, graphql.WithHTTPClient(cli))
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")
	for key, value := range vars {
		req.Var(key, value)
	}

	var response PartyList
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.Parties, nil
}

func socialParties(socials map[string]string, parties []Party) []Party {
	// Must show in the leaderboard ALL parties registered in the socials list, regardless of whether they exist in Vega
	sp := make([]Party, 0, len(socials))
	for partyID, social := range socials {
		found := false
		for _, p := range parties {
			if p.ID == partyID {
				log.WithFields(log.Fields{
					"partyID":       partyID,
					"social":        social,
					"account_count": len(p.Accounts),
				}).Debug("Social (found)")
				p.social = social
				sp = append(sp, p)
				found = true
				break
			}
		}
		if !found {
			sp = append(sp, Party{
				ID:     partyID,
				social: social,
			})
			log.WithFields(log.Fields{
				"partyID":       partyID,
				"social":        social,
				"account_count": "zero",
			}).Debug("Social (not found)")
		}
	}
	return sp
}
