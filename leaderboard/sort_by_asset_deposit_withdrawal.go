package leaderboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vegaprotocol/topgun-service/datastore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Service) sortByAssetDepositWithdrawal(socials map[string]string) ([]Participant, error) {
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

	ctx := context.Background()
	parties, err := getParties(ctx, s.cfg.VegaGraphQLURL.String(), gqlQuery, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get list of parties: %w", err)
	}

	//parties = make([]Party, 0)

	//for _, p := range parties {
	//	if len(p.Deposits) > 0 {
	//		for _, d := range p.Deposits {
	//			if d.Asset.Source.Name == "ERC20" &&
	//				d.Status == "Finalized" &&
	//				d.CreatedAt.After(s.cfg.StartTime) &&
	//				d.CreatedAt.Before(s.cfg.EndTime) {
	//				log.Infof("Party %s has deposited %s ERC20 asset %s %s", p.ID, d.Amount, d.Asset.Symbol, d.Status)
	//			}
	//		}
	//	}
	//	if len(p.Withdrawals) > 0 {
	//		for _, w := range p.Withdrawals {
	//			if w.Asset.Source.Name == "ERC20" &&
	//				w.Status == "Finalized" &&
	//				w.CreatedAt.After(s.cfg.StartTime) &&
	//				w.CreatedAt.Before(s.cfg.EndTime) {
	//				log.Infof("Party %s has withdrawn %s ERC20 asset %s %s", p.ID, w.Amount, w.Asset.Symbol, w.Status)
	//			}
	//		}
	//	}
	//}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	svc := datastore.NewMongoDbDatastore(dbCtx, s.cfg.MongoConnectionString)
	err = svc.Connect()
	if err != nil {
		log.WithError(err).Error("Error starting MongoDB datastore service")
		return nil, err
	}

	log.Info("DB participants loading...")

	// Find all participants in the given collection (configurable for each incentive run)
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

	minDepositAndWithdrawals := 2
	participationCount := 0

	dbParticipantsMap := make(map[string]*Participant,len(dbParticipants))
	for _, dbParticipant := range dbParticipants {
		participationCount++
		dbParticipant.sortNum = float64(participationCount)
		dbParticipantsMap[strings.ToLower(dbParticipant.TwitterHandle)] = &dbParticipant
	}

	log.Infof("DB participants found: %d", len(dbParticipants))

	sParties := socialParties(socials, parties)
	participants := dbParticipants
	for _, party := range sParties {
		if _, found := dbParticipantsMap[strings.ToLower(party.social)]; !found {

			// Social handle not found in the database
			// - check if they've achieved the participation reward
			// - if they have then insert into mongodb collection

			if s.hasDepositedErc20Assets(minDepositAndWithdrawals, party.Deposits) &&
				s.hasWithdrawnErc20Assets(minDepositAndWithdrawals, party.Withdrawals) {
				participationCount++
				// Only users that have successfully participated in the task will be stored
				utcNow := time.Now().UTC()
				participant := Participant{
					PublicKey:     party.ID,
					TwitterHandle: party.social,
					CreatedAt:     utcNow,
					UpdatedAt:     utcNow,
					Data:          []string{"Achieved"},
					sortNum:       float64(participationCount),
				}
				participants = append(participants, participant)

				insertResult, err := dbParticipantsCollection.InsertOne(ctx, participant)
				if err != nil {
					log.WithError(err).Error("Error inserting participant")
				} else {
					log.Infof("Inserted: %s %s", participant.TwitterHandle, insertResult.InsertedID)
				}
			}
		} else {
			log.Infof("Found db participant: %s %s", party.social, party.ID)
		}
	}

	sortFunc := func(i, j int) bool {
		return participants[i].sortNum > participants[j].sortNum
	}
	sort.Slice(participants, sortFunc)

	err = svc.Disconnect()
	if err != nil {
		log.WithError(err).Warn("Error disconnecting from MongoDB datastore service")
	}
	return participants, nil
}

func (s *Service) hasDepositedErc20Assets(min int, deposits []Deposit) bool {
	totalDepositsForParty := 0
	if len(deposits) > 0 {
		for _, d := range deposits {
			if d.Asset.Source.Name == "ERC20" &&
				d.Status == "Finalized" &&
				d.CreatedAt.After(s.cfg.StartTime) &&
				d.CreatedAt.Before(s.cfg.EndTime) {
				totalDepositsForParty++
			}
			if totalDepositsForParty >= min {
				return true
			}
		}
	}
	return false
}

func (s *Service) hasWithdrawnErc20Assets(min int, withdrawals []Withdrawal) bool {
	totalWithdrawalsForParty := 0
	if len(withdrawals) > 0 {
		for _, w := range withdrawals {
			if w.Asset.Source.Name == "ERC20" &&
				w.Status == "Finalized" &&
				w.CreatedAt.After(s.cfg.StartTime) &&
				w.CreatedAt.Before(s.cfg.EndTime) {
				totalWithdrawalsForParty++
			}
			if totalWithdrawalsForParty >= min {
				return true
			}
		}
	}
	return false
}

// Loads existing participants from datastore (from the given mongodb collection)
func (s *Service) loadParticipantsFromDatastore(mongocollectionName string) (participants []Participant) {

	return nil
}

// Saves new participants to datastore (to the given mongodb collection)
func (s *Service) saveParticipantsToDatastore(collectionName string, participants []Participant) error {



	return nil
}
