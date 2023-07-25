package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/verifier"
)

var gqlQueryPartiesDepositWithdrawalPubkeys string = `query ($pagination: Pagination!) {
	partiesConnection(pagination: $pagination) {
	  edges {
		node {
		  id
		  positionsConnection {
			edges {
			  node {
				market {
				  id
				}
				openVolume
				realisedPNL
				averageEntryPrice
				unrealisedPNL
				realisedPNL
			  }
			}
		  }
		  withdrawalsConnection {
			edges {
			  node {
				amount
				createdTimestamp
				createdTimestamp
				status
				asset {
				  id
				  symbol
				  source {
					__typename
				  }
				}
			  }
			}
		  }
		  depositsConnection {
			edges {
			  node {
				amount
				createdTimestamp
				creditedTimestamp
				status
				asset {
				  id
				  symbol
				  source {
					__typename
				  }
				}
			  }
			}
		  }
		}
	  }
	  pageInfo {
		hasNextPage
		hasPreviousPage
		startCursor
		endCursor
	  }
	}
  }`

func (s *Service) sortByPartyDepositWithdrawalPubkeys(socials map[string]verifier.Social) ([]Participant, error) {

	// Grab the DP we're targeting (for the asset we're interested in for the market specified
	decimalPlacesStr, err := s.getAlgorithmConfig("decimalPlaces")
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}
	decimalPlaces, err := strconv.ParseFloat(decimalPlacesStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm config: %s", err)
	}

	pagination := Pagination{First: 50}

	ctx := context.Background()
	partyEdges := []PartiesEdge{}
	for {
		connection, err := getPartiesConnection(
			ctx,
			s.cfg.VegaGraphQLURL.String(),
			gqlQueryPartiesDepositWithdrawalPubkeys,
			map[string]interface{}{"pagination": pagination},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get list of parties in loop: %w", err)
		}

		partyEdges = append(partyEdges, connection.Edges...)

		// fmt.Println("got ", len(partyEdges), "end?", connection.PageInfo.EndCursor)

		if !connection.PageInfo.NextPage {
			// fmt.Println("done")
			break
		} else {
			pagination.After = connection.PageInfo.EndCursor
		}

	}
	participants := []Participant{}
	// if participant in JSON, PNL = json data, otherwise starting PnL 0
	for _, party := range partyEdges {
		withdrawal := 0.0
		deposit := 0.0
		if len(party.Party.WithdrawalsConnection.Edges) != 0 {
			for _, w := range party.Party.WithdrawalsConnection.Edges {
				if err != nil {
					fmt.Errorf("failed to convert Transfer amount to string", err)
				}
				if w.Withdrawal.Asset.Id == s.cfg.VegaAssets[0] &&
					w.Withdrawal.CreatedAt.After(s.cfg.StartTime) &&
					w.Withdrawal.CreatedAt.Before(s.cfg.EndTime) {
					withdrawal, err = strconv.ParseFloat(w.Withdrawal.Amount, 64)
				}
			}
		}

		for _, d := range party.Party.DepositsConnection.Edges {
			if err != nil {
				fmt.Errorf("failed to convert Withdrawal amount to string", err)
			}

			if d.Deposit.Asset.Id == s.cfg.VegaAssets[0] &&
				d.Deposit.Status == "STATUS_FINALIZED" &&
				d.Deposit.CreatedAt.After(s.cfg.StartTime) &&
				d.Deposit.CreatedAt.Before(s.cfg.EndTime) {
				deposit, err = strconv.ParseFloat(d.Deposit.Amount, 64)
			}
		}
		PnL := 0.0
		realisedPnL := 0.0
		unrealisedPnL := 0.0
		openVolume := 0.0
		if err == nil {
			for _, acc := range party.Party.PositionsConnection.Edges {
				for _, marketID := range s.cfg.MarketIDs {
					if acc.Position.Market.ID == marketID {
						if s, err := strconv.ParseFloat(acc.Position.RealisedPNL, 32); err == nil {
							realisedPnL += s
						}
						if t, err := strconv.ParseFloat(acc.Position.UnrealisedPNL, 32); err == nil {
							unrealisedPnL += t
						}
						if u, err := strconv.ParseFloat(acc.Position.OpenVolume, 32); err == nil {
							openVolume += u
						}
						PnL = (realisedPnL + unrealisedPnL) / decimalPlaces
					}
				}

			}
		}

		if (withdrawal != 0.0) && (deposit != 0.0) && (PnL != 0.0) {
			if party.Party.blacklisted {
				log.Infof("Blacklisted party added: %s", party.Party.ID)
			}

			pubkeyBlacklist := []string{"93e8077e3c0a942bd5469b3b142ffe643f4d9c5d9962a862de419bfc5f8bfeb9",
				"4af3e8fe168095cc20c9232f7b4723645a59c6717eb4e9d2121378c6002fb4bb",
				"ad0549439a1c15ebffc2ff406451eec560facfabe762b3a5401e3d20d384d5b3",
				"19e675aa6a7747ee504e31adc660aa9df4213a8ca9d1243e4a30d0974dac3705",
				"45c231260e56ba839f8cc0a4ccec16a209965f6534887ff19c3bfc9242bfc844",
				"fdab1c1c9db496f651d922e3b056a4736e3a3b0ee301cb20afa491f3656939d8",
				"1d2247dff85b9396d46545cb959e3b5c5925dd77e193ba873f01f4079481f67b",
				"937802822f779d3d14b280ffdabcd2935c8b7c708a6ca53b8c05230827c8a960",
				"86ff2c3b45be7c43202d1dc370779d070faaba1029094c46174857f1b445673f",
				"51351917f4400efe8ecdd1ecc726d174bbd87c991dc0795863cadd586b4e3865",
				"2aaeeeec54b72fcf69d89b2f5960dea1b9bca2cc71a61f421502a80fff32c139",
				"022a129df7f8360c9de598e8d0eeb06c62c9f63393b25024af89cd5bfb1c6207",
				"09a576a282cbafe3b37673949f7d563118222d4fbfa3df7879727667d4970577",
				"0b8519bb08e11dac3ab1073f4fc8cd0ff02e5b751d14c5e624c26bf65c7aaf92",
				"feae764c6615a4e9c0170a500e1d51d312b4b8e50bd59fa825843025bff4fe02",
				"4a9b97bd45af1a9d744462edcb67a249165065984667914db2d2167a0d45221d",
				"82507ccb7b6380dc36eae68bdbc5495d2b9126ce00f49bce911f6ad6a0c1359d",
				"c810aa6c86b3c367248cc41277970cf97ef95568451338d72972a41fddd079a2",
				"edae7973f562cd232ff15094509655b4d31c6d7fd5f8a45db5965572e65c4d54"}

			for _, pubkey := range pubkeyBlacklist {
				if pubkey == party.Party.ID {
					party.Party.blacklisted = true
				}
			}

			t := time.Now().UTC()
			if party.Party.blacklisted == false {
				participants = append(participants, Participant{
					PublicKey:     party.Party.ID,
					Data:          []string{"Completed"},
					sortNum:       PnL,
					CreatedAt:     t,
					UpdatedAt:     t,
					isBlacklisted: party.Party.blacklisted,
				})
			}

		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	return participants, nil
}
