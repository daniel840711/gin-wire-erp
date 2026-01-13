package client

import (
	"context"
	"interchange/config"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// MongoClient 連接 MongoDB
type MongoClient struct {
	client *mongo.Client
	logger *zap.Logger
}

func NewMongoClient(logger *zap.Logger, config *config.Configuration) (*MongoClient, func(), error) {
	mongoClient := &MongoClient{logger: logger}
	client, err := mongoClient.connectDB(config)
	if err != nil {
		logger.Error("failed to connect to MongoDB", zap.Error(err))
		return nil, nil, err
	}
	logger.Info("Connected to MongoDB")
	mongoClient.client = client

	cleanup := func() {
		logger.Info("closing the MongoDB resources")
		if err := mongoClient.Close(); err != nil {
			logger.Error("failed to close MongoDB client", zap.Error(err))
		}
	}

	return mongoClient, cleanup, nil
}

func (client *MongoClient) connectDB(config *config.Configuration) (*mongo.Client, error) {
	uri := buildMongoURI(config.MongoDB.URI, config.MongoDB.Options)
	return mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
}
func buildMongoURI(baseURI, optionStr string) string {
	if optionStr == "" {
		return baseURI
	}
	if strings.Contains(baseURI, "?") {
		return baseURI + "&" + optionStr
	}
	return baseURI + "?" + optionStr
}

// Close 關閉 MongoDB 連線
func (m *MongoClient) Close() error {
	return m.client.Disconnect(context.Background())
}

// Client 回傳 MongoDB 連線
func (m *MongoClient) Client() *mongo.Client {
	return m.client
}
