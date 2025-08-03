package persistence

import (
	"context"
	"fmt"
	"smartblox-ingestor/config"
	"smartblox-ingestor/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Persisting and retrieving running metrics
type MetricsStore interface {
	Save(metrics *types.Metrics) error
	Load() (*types.Metrics, error)
}

// Writing transaction data
type TransactionLogger interface {
	Log(ctx context.Context, tx types.Transaction) error
}

// Retrieve data from .env
var databaseName = config.GetEnvOrDefault("DATABASE_NAME", "smartblox")
var metricsCollection = config.GetEnvOrDefault("METRICS_COLLECTION", "metrics")
var txCollection = config.GetEnvOrDefault("TX_COLLECTION", "transactions")
var singletonMetricsID = config.GetEnvOrDefault("SINGLETON_METRICS_ID", "singleton_metrics_state")

// MongoDB store
type MongoStore struct {
	client *mongo.Client
	db     *mongo.Database
}

// Create a new MongoDB store
func NewMongoStore(ctx context.Context, uri string) (*MongoStore, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("could not ping mongo: %w", err)
	}

	store := &MongoStore{
		client: client,
		db:     client.Database(databaseName),
	}

	if err := store.createIndexes(ctx); err != nil {
		fmt.Printf("WARN: Could not create indexes: %v\n", err)
	}

	return store, nil
}

// Close store connection
func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// Unique index setup
func (s *MongoStore) createIndexes(ctx context.Context) error {

	// Index on transaction signature
	txIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "sig", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := s.db.Collection(txCollection).Indexes().CreateOne(ctx, txIndexModel)
	if err != nil {
		return fmt.Errorf("failed to create transaction index: %w", err)
	}
	return nil
}

// Upsert the metrics doc into the metrics collection
func (s *MongoStore) Save(ctx context.Context, metrics *types.Metrics) error {
	collection := s.db.Collection(metricsCollection)

	filter := bson.M{"_id": singletonMetricsID}
	update := bson.M{"$set": metrics}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}
	return nil
}

// Load unique metrics document
func (s *MongoStore) Load(ctx context.Context) (*types.Metrics, error) {
	collection := s.db.Collection(metricsCollection)

	filter := bson.M{"_id": singletonMetricsID}
	var metrics types.Metrics
	err := collection.FindOne(ctx, filter).Decode(&metrics)
	if err != nil {
		// For firrst run
		if err == mongo.ErrNoDocuments {
			return &types.Metrics{}, nil
		}
		return nil, fmt.Errorf("failed to load metrics: %w", err)
	}
	return &metrics, nil
}

// Add new transaction into the collection
func (s *MongoStore) Log(ctx context.Context, tx types.Transaction) error {
	collection := s.db.Collection(txCollection)
	_, err := collection.InsertOne(ctx, tx)

	if mongo.IsDuplicateKeyError(err) {
		fmt.Printf("INFO: Duplicate transaction %s found, skipping.\n", tx.Sig)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to log transaction: %w", err)
	}
	return nil

}
