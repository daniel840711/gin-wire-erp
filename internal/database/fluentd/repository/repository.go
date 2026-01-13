package repository

import (
	"github.com/google/wire"
)

// 統一管理所有 Redis repository
type FluentdRepository struct {
	logRepository *LogRepository
}

// 建立 Redis repository 物件
func NewFluentdRepository(
	logRepository *LogRepository,
) *FluentdRepository {
	return &FluentdRepository{
		logRepository: logRepository,
	}
}

// Wire 依賴提供
var ProviderSet = wire.NewSet(
	NewLogRepository,
	NewFluentdRepository)
