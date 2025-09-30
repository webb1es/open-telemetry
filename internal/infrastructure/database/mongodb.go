package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
)

const (
	ConnectTimeout = 10 * time.Second
	PingTimeout    = 2 * time.Second
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(cfg *config.DatabaseConfig) (*MongoDB, error) {
	// Create client options with OpenTelemetry instrumentation
	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetMonitor(otelmongo.NewMonitor()).
		SetConnectTimeout(ConnectTimeout).
		SetServerSelectionTimeout(ConnectTimeout)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ConnectTimeout)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	pingCtx, pingCancel := context.WithTimeout(context.Background(), PingTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database name from URI or use default
	dbName := "otel_demo"

	database := client.Database(dbName)

	return &MongoDB{
		Client:   client,
		Database: database,
	}, nil
}

func (m *MongoDB) Disconnect(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

func (m *MongoDB) IsConnected(ctx context.Context) error {
	if m.Client == nil {
		return fmt.Errorf("MongoDB client is nil")
	}
	return m.Client.Ping(ctx, nil)
}

// Collection methods for easy access
func (m *MongoDB) UsersCollection() *mongo.Collection {
	return m.Database.Collection("users")
}

func (m *MongoDB) PaymentsCollection() *mongo.Collection {
	return m.Database.Collection("payments")
}

func (m *MongoDB) OrdersCollection() *mongo.Collection {
	return m.Database.Collection("orders")
}

func (m *MongoDB) RewardsCollection() *mongo.Collection {
	return m.Database.Collection("rewards")
}

func (m *MongoDB) CatalogueCollection() *mongo.Collection {
	return m.Database.Collection("catalogue")
}

// CreateIndexes creates necessary database indexes
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Users indexes
	usersIndexes := []mongo.IndexModel{
		{Keys: map[string]interface{}{"email": 1}, Options: options.Index().SetUnique(true)},
		{Keys: map[string]interface{}{"phone": 1}, Options: options.Index().SetUnique(true)},
		{Keys: map[string]interface{}{"created_at": -1}},
	}
	if _, err := m.UsersCollection().Indexes().CreateMany(ctx, usersIndexes); err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// Payments indexes
	paymentsIndexes := []mongo.IndexModel{
		{Keys: map[string]interface{}{"user_id": 1}},
		{Keys: map[string]interface{}{"order_id": 1}},
		{Keys: map[string]interface{}{"status": 1}},
		{Keys: map[string]interface{}{"reference": 1}, Options: options.Index().SetUnique(true)},
		{Keys: map[string]interface{}{"external_txn_id": 1}, Options: options.Index().SetSparse(true)},
		{Keys: map[string]interface{}{"created_at": -1}},
	}
	if _, err := m.PaymentsCollection().Indexes().CreateMany(ctx, paymentsIndexes); err != nil {
		return fmt.Errorf("failed to create payments indexes: %w", err)
	}

	// Orders indexes
	ordersIndexes := []mongo.IndexModel{
		{Keys: map[string]interface{}{"user_id": 1}},
		{Keys: map[string]interface{}{"status": 1}},
		{Keys: map[string]interface{}{"payment_id": 1}},
		{Keys: map[string]interface{}{"created_at": -1}},
	}
	if _, err := m.OrdersCollection().Indexes().CreateMany(ctx, ordersIndexes); err != nil {
		return fmt.Errorf("failed to create orders indexes: %w", err)
	}

	// Rewards indexes
	rewardsIndexes := []mongo.IndexModel{
		{Keys: map[string]interface{}{"user_id": 1}},
		{Keys: map[string]interface{}{"status": 1}},
		{Keys: map[string]interface{}{"type": 1}},
		{Keys: map[string]interface{}{"expires_at": 1}, Options: options.Index().SetSparse(true)},
		{Keys: map[string]interface{}{"created_at": -1}},
	}
	if _, err := m.RewardsCollection().Indexes().CreateMany(ctx, rewardsIndexes); err != nil {
		return fmt.Errorf("failed to create rewards indexes: %w", err)
	}

	// Catalogue indexes
	catalogueIndexes := []mongo.IndexModel{
		{Keys: map[string]interface{}{"sku": 1}, Options: options.Index().SetUnique(true)},
		{Keys: map[string]interface{}{"category": 1}},
		{Keys: map[string]interface{}{"status": 1}},
		{Keys: map[string]interface{}{"price": 1}},
	}
	if _, err := m.CatalogueCollection().Indexes().CreateMany(ctx, catalogueIndexes); err != nil {
		return fmt.Errorf("failed to create catalogue indexes: %w", err)
	}

	return nil
}
