package repository

import (
	"context"
	"encoding/json"
	"time"

	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/database/client"
	"interchange/internal/database/fluentd/model"
)

// LogRepository 統一負責發送 Request/Response/Usage Log 到 Fluentd
type LogRepository struct {
	fluentdClient *client.FluentdClient
	version       string
}

func NewLogRepository(config *config.Configuration, client *client.FluentdClient) *LogRepository {
	version := "1.0.0"
	if config.App.Version != "" {
		version = config.App.Version
	}
	return &LogRepository{fluentdClient: client, version: version}
}

func (repository *LogRepository) LogRequest(ctx context.Context, req model.RequestLog) error {
	if req.LoggedAt == "" {
		req.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if req.Version == "" {
		req.Version = repository.version
	}
	b, _ := json.Marshal(req)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentdRequest), fluentdMessage)
	return err
}

func (repository *LogRepository) LogResponse(ctx context.Context, resp model.ResponseLog) error {
	if resp.LoggedAt == "" {
		resp.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if resp.Version == "" {
		resp.Version = repository.version
	}
	b, _ := json.Marshal(resp)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentdResponse), fluentdMessage)
	return err
}

func (repository *LogRepository) LogUsage(ctx context.Context, usage model.AIUsageLog) error {
	if usage.LoggedAt == "" {
		usage.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if usage.Version == "" {
		usage.Version = repository.version
	}
	b, _ := json.Marshal(usage)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentUsage), fluentdMessage)
	return err
}
