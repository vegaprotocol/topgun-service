package leaderboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vegaprotocol/topgun-service/datastore"
	"github.com/vegaprotocol/topgun-service/verifier"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Service) sortByAssetDepositWithdrawal(socials map[string]verifier.Social) ([]Participant, error) {

	// The minimum number of unique deposits and withdrawals needed to achieve this reward
	minDepositAndWithdrawals := 2
	// Default: 2 unique asset deposits and 2 unique withdrawals from the erc20 bridge

	// Total number of participants awarded
	maxAwarded := 5000

	gqlQuery := `query {
	  parties{
		id
		deposits {
		  amount
		  createdTimestamp
		  creditedTimestamp
		  status
          asset {
			id
			symbol
            source { __typename }
		  }
		}
		withdrawals{
		  amount
		  createdTimestamp
		  createdTimestamp
		  status
		  asset {
			id
			symbol
			source { __typename }
		  }
		}
	  }
	}`

	log.Info("Vega query starting...")

	startedAt := time.Now()
	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	completedAt := time.Now()
	diff := completedAt.Sub(startedAt)
	log.Infof("Vega query completed in %s", diff)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	svc := datastore.NewMongoDbDatastore(dbCtx, s.cfg.MongoConnectionString)
	err = svc.Connect()
	if err != nil {
		log.WithError(err).Error("Error starting MongoDB datastore service")
		return nil, err
	}

	startedAt = time.Now()
	log.Info("DB participants loading...")

	// Find all participants in the given collection (configurable for each incentive)
	dbParticipantsCollection := svc.LoadDocumentCollection(s.cfg.MongoDatabaseName, s.cfg.MongoCollectionName)

	var dbParticipants []Participant
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"created", -1}})
	cursor, err := dbParticipantsCollection.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		log.WithError(err).Error("Error querying MongoDB datastore [1]")
	} else if err = cursor.All(ctx, &dbParticipants); err != nil {
		log.WithError(err).Error("Error querying MongoDB datastore [2]")
	}

	participationCount := 0
	dbParticipantsMap := make(map[string]*Participant, len(dbParticipants))
	for _, dbParticipant := range dbParticipants {
		participationCount++
		dbParticipant.sortNum = float64(participationCount)
		dbParticipantsMap[strings.ToLower(dbParticipant.TwitterHandle)] = &dbParticipant
	}

	completedAt = time.Now()
	diff = completedAt.Sub(startedAt)
	log.Infof("DB participants found %d in %s", len(dbParticipants), diff)

	sParties := socialParties(socials, parties)
	participants := dbParticipants

	if participationCount < maxAwarded {
		for _, party := range sParties {
			if _, found := dbParticipantsMap[strings.ToLower(party.social)]; !found {

				// Social handle not found in the database
				// - check if they've achieved the participation reward
				// - if they have then insert into mongodb collection

				if participationCount == maxAwarded {
					log.Infof("Reached maximum awarded: %d", participationCount)
					break
				}

				if s.hasDepositedErc20Assets(minDepositAndWithdrawals, party.Deposits) &&
					s.hasWithdrawnErc20Assets(minDepositAndWithdrawals, party.Withdrawals) {
					participationCount++

					// Only users that have successfully participated in the task will be stored
					utcNow := time.Now().UTC()
					participant := Participant{
						PublicKey:     party.ID,
						TwitterHandle: party.social,
						TwitterUserID: party.twitterID,
						CreatedAt:     utcNow,
						UpdatedAt:     utcNow,
						Data:          []string{"Achieved"},
						sortNum:       float64(participationCount),
						isBlacklisted: party.blacklisted,
					}

					// Push newly found participant to the top of the list
					// existing participants from db are  in created-at desc order
					participants = append([]Participant{participant}, participants...)

					insertResult, err := dbParticipantsCollection.InsertOne(ctx, participant)
					if err != nil {
						log.WithError(err).Error("Error inserting participant")
					} else {
						log.Infof("Inserted: %s %s", participant.TwitterHandle, insertResult.InsertedID)
					}
				}
			} else {
				log.Debugf("Found db participant: %s %s", party.social, party.ID)
			}
		}
	}

	err = svc.Disconnect()
	if err != nil {
		log.WithError(err).Warn("Error disconnecting from MongoDB datastore service")
	}
	return participants, nil
}

func (s *Service) hasDepositedErc20Assets(min int, deposits []Deposit) bool {
	totalDepositsForParty := 0
	if len(deposits) > 0 {
		foundAssets := make(map[string]bool, 0)
		for _, d := range deposits {
			if d.Asset.Source.Name == "ERC20" &&
				d.Status == "Finalized" &&
				d.CreatedAt.After(s.cfg.StartTime) &&
				d.CreatedAt.Before(s.cfg.EndTime) {
				foundAssets[d.Asset.Symbol] = true
				totalDepositsForParty++
			}
			if totalDepositsForParty >= min && len(foundAssets) >= min {
				return true
			}
		}
	}
	return false
}

func (s *Service) hasWithdrawnErc20Assets(min int, withdrawals []Withdrawal) bool {
	totalWithdrawalsForParty := 0
	if len(withdrawals) > 0 {
		foundAssets := make(map[string]bool, 0)
		for _, w := range withdrawals {
			if w.Asset.Source.Name == "ERC20" &&
				w.Status == "Finalized" &&
				w.CreatedAt.After(s.cfg.StartTime) &&
				w.CreatedAt.Before(s.cfg.EndTime) {
				foundAssets[w.Asset.Symbol] = true
				totalWithdrawalsForParty++
			}
			if totalWithdrawalsForParty >= min && len(foundAssets) >= min {
				return true
			}
		}
	}
	return false
}
