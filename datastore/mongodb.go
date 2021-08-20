package datastore

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	mu          sync.RWMutex
	cli         *mongo.Client
	ctx         context.Context
	connStr     string
	isConnected bool
}

func NewMongoDbDatastore(ctx context.Context, connStr string) *Service {
	s := Service{
		ctx:     ctx,
		connStr: connStr,
	}
	return &s
}

func (s *Service) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isConnected {
		return errors.New("MongoDB instance already connected")
	}
	cli, err := mongo.Connect(s.ctx, options.Client().ApplyURI(s.connStr))
	if err != nil {
		return errors.Wrap(err, "Could not connect to MongoDB instance")
	}
	err = cli.Ping(s.ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Could not ping the MongoDB instance")
	}
	log.Infof("Connected to MongoDB instance")
	s.cli = cli
	s.isConnected = true
	return nil
}

func (s *Service) LoadDocumentCollection(databaseName string, collectionName string) *mongo.Collection {
	database := s.cli.Database(databaseName)
	collection := database.Collection(collectionName)
	return collection
}

func (s *Service) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cli == nil || !s.isConnected {
		return nil
	}
	err := s.cli.Disconnect(s.ctx)
	s.isConnected = false
	log.Info("Disconnected from MongoDB instance")
	return err
}
