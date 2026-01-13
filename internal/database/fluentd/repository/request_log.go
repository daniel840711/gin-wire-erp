package repository

import (
	"context"
	"encoding/json"
	"time"

	"interchange/config"
	"interchange/internal/core"
	"interchange/internal/database/client"
	"interchange/internal/database/fluentd/model"
	"interchange/internal/telemetry"
)

// LogRepository 統一負責發送 Request/Response/Usage Log 到 Fluentd
type LogRepository struct {
	fluentdClient *client.FluentdClient
	trace         *telemetry.Trace
	version       string
}

func NewLogRepository(config *config.Configuration, client *client.FluentdClient, trace *telemetry.Trace) *LogRepository {
	version := "1.0.0"
	if config.App.Version != "" {
		version = config.App.Version
	}
	return &LogRepository{fluentdClient: client, version: version, trace: trace}
}

func (repository *LogRepository) LogRequest(ctx context.Context, req model.RequestLog) error {
	ctx, span, end := repository.trace.WithSpan(ctx, "fluentd_"+string(core.FluentdRequest))
	defer end(nil)

	if req.LoggedAt == "" {
		req.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if req.Version == "" {
		req.Version = repository.version
	}
	attributes := core.TraceRequestLogMeta{
		RequestID:   req.RequestID,
		Path:        req.Path,
		Method:      req.Method,
		ProjectName: req.ProjectName,
		Body:        req.Body,
		IPHash:      req.IPHash,
		UserAgent:   req.UserAgent,
		Version:     req.Version,
		RequestTS:   req.RequestTS,
		LoggedAt:    req.LoggedAt,
	}
	repository.trace.ApplyTraceAttributes(span, attributes)
	b, _ := json.Marshal(req)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentdRequest), fluentdMessage)
	if err != nil {
		end(err)
	}
	return err
}

func (repository *LogRepository) LogResponse(ctx context.Context, resp model.ResponseLog) error {
	ctx, span, end := repository.trace.WithSpan(ctx, "fluentd_"+string(core.FluentdResponse))
	defer end(nil)
	if resp.LoggedAt == "" {
		resp.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if resp.Version == "" {
		resp.Version = repository.version
	}
	attributes := core.TraceResponseLogMeta{
		RequestID:   resp.RequestID,
		ProjectName: resp.ProjectName,
		Code:        resp.Code,
		StatusCode:  resp.StatusCode,
		Body:        resp.Body,
		Error:       resp.Error,
		Version:     resp.Version,
		ResponseTS:  resp.ResponseTS,
		LoggedAt:    resp.LoggedAt,
	}
	repository.trace.ApplyTraceAttributes(span, attributes)
	b, _ := json.Marshal(resp)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentdResponse), fluentdMessage)
	if err != nil {
		end(err)
	}
	return err
}

func (repository *LogRepository) LogUsage(ctx context.Context, usage model.AIUsageLog) error {
	ctx, span, end := repository.trace.WithSpan(ctx, "fluentd_"+string(core.FluentUsage))
	defer end(nil)

	if usage.LoggedAt == "" {
		usage.LoggedAt = time.Now().UTC().Format("2006-01-02 15:04:05.999999 UTC")
	}
	if usage.Version == "" {
		usage.Version = repository.version
	}
	attributes := core.TraceUsageLogMeta{
		RequestID:        usage.RequestID,
		ExternalID:       usage.ExternalID,
		DisplayName:      usage.DisplayName,
		ProjectName:      usage.ProjectName,
		Provider:         usage.Provider,
		Model:            usage.Model,
		Endpoint:         usage.Endpoint,
		TokensPrompt:     usage.TokensPrompt,
		TokensCompletion: usage.TokensCompletion,
		TextToken:        usage.TextToken,
		AudioToken:       usage.AudioToken,
		ImageToken:       usage.ImageToken,
		InputToken:       usage.InputToken,
		OutputToken:      usage.OutputToken,
		TokensTotal:      usage.TokensTotal,
		Version:          usage.Version,
		LoggedAt:         usage.LoggedAt,
	}
	repository.trace.ApplyTraceAttributes(span, attributes)
	b, _ := json.Marshal(usage)
	var fluentdMessage map[string]any
	_ = json.Unmarshal(b, &fluentdMessage)
	err := repository.fluentdClient.Post(ctx, string(core.FluentUsage), fluentdMessage)
	if err != nil {
		end(err)
	}
	return err
}
