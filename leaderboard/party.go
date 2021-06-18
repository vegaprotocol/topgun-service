package leaderboard

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
)

type Asset struct {
	Symbol string `json:"symbol"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
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

type Party struct {
	ID       string      `json:"id"`
	Accounts []Account   `json:"accounts"`
	Orders   []Order     `json:"orders"`
	Trades   []Trade     `json:"trades"`
	Votes    []PartyVote `json:"votes"`

	social string
}

func hasString(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// this is not the optimal solution but with the
// game being restarted we have no other way to know
// than checking if the general account balance is exactly
// 10K, there's also quite few changes for the party to
// have traded and close the position to 10K exactly
// again, so we should be OK.
func (p *Party) HasTraded(assetName string, topupAssetTotal float64) bool {
	var hasAsset, hasMargin, hasGeneral bool
	for _, acc := range p.Accounts {
		if acc.Asset.Symbol == assetName {
			hasAsset = true
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/%s balance [Balance]", assetName, acc.Type)
				return false
			}
			if acc.Type == "General" && v != topupAssetTotal {
				hasGeneral = true
			}
			if acc.Type == "Margin" && v != 0 {
				hasMargin = true
			}
		}
	}

	if !hasAsset {
		// requested asset not found in party accounts
		return false
	}

	return hasGeneral || hasMargin
}

func (p *Party) Balance(assetName string, accountTypes ...string) float64 {
	var accu float64
	for _, acc := range p.Accounts {
		if acc.Asset.Symbol == assetName && hasString(accountTypes, acc.Type) {
			v, err := strconv.ParseFloat(acc.Balance, 64)
			if err != nil {
				log.WithError(err).Errorf(
					"Failed to parse %s/%s balance [Balance]", assetName, accountTypes)
				return 0
			}
			accu += v
		}
	}

	if len(p.Accounts) == 0 {
		fmt.Println(p.social," has no accounts - ", p.ID)
	}

	if accu != 0 {
		accu = accu / float64(100000)
	}

	return accu
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
		cli = &http.Client{Timeout: time.Second * 10}
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
