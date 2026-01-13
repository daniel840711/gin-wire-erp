package repository

import (
	"github.com/google/wire"
)

// 統一管理所有 Redis repository
type RedisRepository struct {
	rateLimitRepo *RateLimiterRepository
}

// 建立 Redis repository 物件
func NewRedisRepository(
	rateLimitRepo *RateLimiterRepository,
) *RedisRepository {
	return &RedisRepository{
		rateLimitRepo: rateLimitRepo,
	}
}

// Wire 依賴提供
var ProviderSet = wire.NewSet(
	NewRateLimiterRepository,
	NewRedisRepository)
