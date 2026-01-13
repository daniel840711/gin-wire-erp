package database

import (
	client "interchange/internal/database/client"
	fluentdRepo "interchange/internal/database/fluentd/repository"
	mongoRepo "interchange/internal/database/mongodb/repository"
	redisRepo "interchange/internal/database/redis/repository"

	"github.com/google/wire"
)

// ProviderSet 定義所有 DB Client 的依賴
var ProviderSet = wire.NewSet(
	client.NewMongoClient,
	client.NewRedisClient,
	client.NewFluentdClient,
	mongoRepo.ProviderSet,
	redisRepo.ProviderSet,
	fluentdRepo.ProviderSet,
)
