package database

import (
	client "interchange/internal/database/client"
	mongoRepo "interchange/internal/database/mongodb/repository"

	"github.com/google/wire"
)

// ProviderSet 定義所有 DB Client 的依賴
var ProviderSet = wire.NewSet(
	client.NewMongoClient,
	mongoRepo.ProviderSet,
)
