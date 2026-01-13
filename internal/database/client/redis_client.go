package client

import (
	"context"
	"fmt"
	"interchange/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient 連接 Redis
type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisClient(logger *zap.Logger, config *config.Configuration) (*RedisClient, func(), error) {
	redisClient := &RedisClient{logger: logger}
	client, err := redisClient.connectDB(config)
	if err != nil {
		logger.Error("failed to connect to Redis", zap.Error(err))
		return nil, nil, err
	}
	logger.Info("Connected to Redis")
	redisClient.client = client

	cleanup := func() {
		logger.Info("closing the Redis resources")
		if err := redisClient.Close(); err != nil {
			logger.Error("failed to close Redis client", zap.Error(err))
		}
	}

	return redisClient, cleanup, nil
}

func (client *RedisClient) connectDB(config *config.Configuration) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		DB:   config.Redis.DB,
	})
	if _, err := r.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return r, nil
}

// Close 關閉 Redis 連線
func (redisClient *RedisClient) Close() error {
	return redisClient.client.Close()
}

// Client 回傳 Redis 連線
func (redisClient *RedisClient) Client() *redis.Client {
	return redisClient.client
}
