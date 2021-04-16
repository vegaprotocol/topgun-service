package leaderboard

import (
	"context"
	"time"

	"github.com/machinebox/graphql"
)

type Asset struct {
	Symbol string `json:"symbol"`
}

type Account struct {
	Type    string `json:"type"`
	Balance string `json:"balance"`
	Asset   Asset  `json:"asset"`
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
	Votes    []PartyVote `json:"votes"`

	social string
}

type PartyList struct {
	Parties []Party `json:"parties"`
}

func getParties(ctx context.Context, gqlURL string, gqlQuery string) ([]Party, error) {
	client := graphql.NewClient(gqlURL)
	req := graphql.NewRequest(gqlQuery)
	req.Header.Set("Cache-Control", "no-cache")

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
				// log.WithFields(log.Fields{
				// 	"partyID":       partyID,
				// 	"social":        social,
				// 	"account_count": len(p.Accounts),
				// }).Debug("Social (found)")
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
			// log.WithFields(log.Fields{
			// 	"partyID":       partyID,
			// 	"social":        social,
			// 	"account_count": "zero",
			// }).Debug("Social (not found)")
		}
	}
	return sp
}
