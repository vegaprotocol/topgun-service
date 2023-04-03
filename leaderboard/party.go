package leaderboard

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/machinebox/graphql"
	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

type Asset struct {
	Id       string      `json:"id"`
	Name     string      `json:"name"`
	Decimals int         `json:"decimals"`
	Symbol   string      `json:"symbol"`
	Source   AssetSource `json:"source"`
}

type AssetSource struct {
	Name string `json:"__typename"`
}

type AccountsConnection struct {
	Edges []AccountsEdge `json:"edges"`
}

type AccountsEdge struct {
	Account Account `json:"node"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
}

type DepositsConnection struct {
	Edges []DepositsEdge `json:"edges"`
}

type DepositsEdge struct {
	Deposit Deposit `json:"node"`
}

type Deposit struct {
	Id         string    `json:"id"`
	Amount     string    `json:"amount"`
	Asset      Asset     `json:"asset"`
	CreatedAt  time.Time `json:"createdTimestamp"`
	CreditedAt time.Time `json:"creditedTimestamp"`
	Status     string    `json:"status"`
}

type WithdrawalsConnection struct {
	Edges []WithdrawalsEdge `json:"edges"`
}

type WithdrawalsEdge struct {
	Withdrawal Withdrawal `json:"node"`
}

type Withdrawal struct {
	Amount     string    `json:"amount"`
	Asset      Asset     `json:"asset"`
	CreatedAt  time.Time `json:"createdTimestamp"`
	CreditedAt time.Time `json:"creditedTimestamp"`
	Status     string    `json:"status"`
}

type TransfersConnection struct {
	Edges []TransfersEdge `json:"edges"`
}

type TransfersEdge struct {
	Transfer Transfer `json:"node"`
}

type Transfer struct {
	Id        string    `json:"id"`
	Amount    string    `json:"amount"`
	Asset     Asset     `json:"asset"`
	Timestamp time.Time `json:"timestamp"`
}

type OrdersConnection struct {
	Edges []OrdersEdge `json:"edges"`
}

type OrdersEdge struct {
	Order Order `json:"node"`
}

type Order struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type PositionsResponse struct {
	PositionsConnection PositionsConnection `json:"positions"`
}

type PositionsConnection struct {
	Edges []PositionsEdge `json:"edges"`
}

type PositionsEdge struct {
	Position Position `json:"node"`
}

type Position struct {
	Market            Market `json:"market"`
	OpenVolume        string `json:"openVolume"`
	AverageEntryPrice string `json:"averageEntryPrice"`
	UnrealisedPNL     string `json:"unrealisedPNL"`
	RealisedPNL       string `json:"realisedPNL"`
	Party             Party  `json:"party"`
	PartyID           string
	Partysocial       string
	PartytwitterID    int64
	Partyblacklisted  bool
}

type TradesConnection struct {
	Edges []TradesEdge `json:"edges"`
}

type TradesEdge struct {
	Trade Trade `json:"node"`
}

type Trade struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type VotesConnection struct {
	Edges []VotesEdge `json:"edges"`
}

type VotesEdge struct {
	Vote Vote `json:"node"`
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

type LiquidityProvisionsConnection struct {
	Edges []LiquidityProvisionsEdge `json:"edges"`
}

type LiquidityProvisionsEdge struct {
	LP LiquidityProvision `json:"node"`
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

type PartiesConnection struct {
	Edges    []PartiesEdge `json:"edges"`
	PageInfo PageInfo      `json:"pageInfo"`
}

type PartiesEdge struct {
	Party Party `json:"node"`
}

type Party struct {
	ID                    string                        `json:"id"`
	AccountsConnection    AccountsConnection            `json:"accountsConnection"`
	DepositsConnection    DepositsConnection            `json:"depositsConnection"`
	OrdersConnection      OrdersConnection              `json:"ordersConnection"`
	TradesConnection      TradesConnection              `json:"tradesConnection"`
	TransfersConnection   TransfersConnection           `json:"transfersConnection"`
	VotesConnection       VotesConnection               `json:"votesConnection"`
	WithdrawalsConnection WithdrawalsConnection         `json:"withdrawalsConnection"`
	LPsConnection         LiquidityProvisionsConnection `json:"liquidityProvisionsConnection"`
	PositionsConnection   PositionsConnection           `json:"positionsConnection"`
	social                string
	twitterID             int64
	blacklisted           bool
}

type Market struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PageInfo struct {
	NextPage     bool   `json:"hasNextPage"`
	PreviousPage bool   `json:"hasPreviousPage"`
	StartCursor  string `json:"startCursor"`
	EndCursor    string `json:"endCursor"`
}

func hasString(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func (p *Party) Balance(assetId string, decimalPlaces int, accountTypes ...string) float64 {
	var accu float64
	accu = 0

	for _, acc := range p.AccountsConnection.Edges {
		if acc.Account.Asset.Id == assetId && hasString(accountTypes, acc.Account.Type) {
			v, err := strconv.ParseFloat(acc.Account.Balance, 64)
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

func (p *Party) CalculateTotalDeposits(asset string, decimalPlaces int) float64 {
	// Total deposits made in asset
	var total float64
	total = 0
	for _, d := range p.DepositsConnection.Edges {
		if d.Deposit.Asset.Id == asset && d.Deposit.Status == "Finalized" {
			amount, err := strconv.ParseFloat(d.Deposit.Amount, 10)
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

type PartiesResponse struct {
	PartiesConnection PartiesConnection `json:"partiesConnection"`
}

func getParties(
	ctx context.Context,
	gqlURL string,
	gqlQuery string,
	vars map[string]string,
	cli *http.Client,
) ([]PartiesEdge, error) {

	if cli == nil {
		cli = &http.Client{Timeout: time.Second * 180}
	}
	client := graphql.NewClient(gqlURL, graphql.WithHTTPClient(cli))
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")
	for key, value := range vars {
		req.Var(key, value)
	}
	var response PartiesResponse
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.PartiesConnection.Edges, nil
}

func getPageInfo(
	ctx context.Context,
	gqlURL string,
	gqlQuery string,
	vars map[string]string,
	cli *http.Client,
) (PageInfo, error) {

	if cli == nil {
		cli = &http.Client{Timeout: time.Second * 180}
	}
	client := graphql.NewClient(gqlURL, graphql.WithHTTPClient(cli))
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")
	for key, value := range vars {
		req.Var(key, value)
	}
	var response PartiesResponse
	if err := client.Run(ctx, req, &response); err != nil {
		return PageInfo{}, err
	}
	return response.PartiesConnection.PageInfo, nil
}

func getPositions(
	ctx context.Context,
	gqlURL string,
	gqlQuery string,
	vars map[string]string,
	cli *http.Client,
) ([]PositionsEdge, error) {

	if cli == nil {
		cli = &http.Client{Timeout: time.Second * 180}
	}
	client := graphql.NewClient(gqlURL, graphql.WithHTTPClient(cli))
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")
	for key, value := range vars {
		req.Var(key, value)
	}
	var response PositionsResponse
	if err := client.Run(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.PositionsConnection.Edges, nil
}

func socialParties(socials map[string]verifier.Social, parties []PartiesEdge) []Party {
	// Must show in the leaderboard ALL parties registered in the socials list, regardless of whether they exist in Vega
	sp := make([]Party, 0, len(socials))
	for partyID, social := range socials {
		found := false
		for _, p := range parties {
			if p.Party.ID == partyID {
				log.WithFields(log.Fields{
					"partyID":       partyID,
					"social":        social,
					"account_count": len(p.Party.AccountsConnection.Edges),
				}).Debug("Social (found)")
				p.Party.social = social.TwitterHandle
				p.Party.twitterID = social.TwitterUserID
				p.Party.blacklisted = social.IsBlacklisted
				sp = append(sp, p.Party)
				found = true
				break
			}
		}
		if !found {
			sp = append(sp, Party{
				ID:          partyID,
				social:      social.TwitterHandle,
				twitterID:   social.TwitterUserID,
				blacklisted: social.IsBlacklisted,
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

func socialPositions(socials map[string]verifier.Social, positions []PositionsEdge) []Position {
	// Must show in the leaderboard ALL parties registered in the socials list, regardless of whether they exist in Vega
	sp := make([]Position, 0, len(socials))
	for partyID, social := range socials {
		found := false
		for _, p := range positions {
			if p.Position.Party.ID == partyID {
				log.WithFields(log.Fields{
					"partyID":       partyID,
					"social":        social,
					"account_count": len(p.Position.Party.AccountsConnection.Edges),
				}).Debug("Social (found)")
				p.Position.Party.social = social.TwitterHandle
				p.Position.Party.twitterID = social.TwitterUserID
				p.Position.Party.blacklisted = social.IsBlacklisted
				sp = append(sp, p.Position)
				found = true
				break
			}
		}
		if !found {
			sp = append(sp, Position{
				PartyID:          partyID,
				Partysocial:      social.TwitterHandle,
				PartytwitterID:   social.TwitterUserID,
				Partyblacklisted: social.IsBlacklisted,
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
