package client

import (
	"context"
	"interchange/config"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"go.uber.org/zap"
)

// Client is a minimal interface to allow mocking in tests.
type Client interface {
	Post(ctx context.Context, tag string, rec map[string]any) error
	Close() error
}

// FluentdClient implements Client using fluent-logger-golang.
type FluentdClient struct {
	client    *fluent.Fluent
	tagPrefix string
}

// New creates a new Fluentd forward client.
func NewFluentdClient(logger *zap.Logger, config *config.Configuration) (*FluentdClient, error) {
	prefix := "uptown"
	if config.Fluentd.TagPrefix != "" {
		prefix = config.Fluentd.TagPrefix
	}
	var timeout time.Duration
	if config.Fluentd.Timeout > 0 {
		timeout = time.Duration(config.Fluentd.Timeout) * time.Millisecond
	}

	fluent, err := fluent.New(fluent.Config{
		FluentHost: config.Fluentd.Host,
		FluentPort: config.Fluentd.Port,
		Timeout:    timeout,
		TagPrefix:  prefix,
	})
	if err != nil {
		return nil, err
	}
	return &FluentdClient{client: fluent, tagPrefix: prefix}, nil
}

func (c *FluentdClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Tag builds a tag using the configured TagPrefix and provided suffix.
// e.g. suffix="router.request" => "uptown.router.request"
func (c *FluentdClient) Tag(suffix string) string {
	if c.tagPrefix == "" {
		return suffix
	}
	return c.tagPrefix + "." + suffix
}

// Post sends a record to Fluentd with the given (possibly-suffixed) tag.
func (c *FluentdClient) Post(ctx context.Context, tag string, message any) error {
	// fluent-logger-golang doesn't support context cancellation directly;
	// we still accept ctx for API symmetry and future extension.
	return c.client.Post(tag, message)
}

// --------------------
// Noop client (disabled mode)
// --------------------

type NoopClient struct{}

func (n *NoopClient) Post(ctx context.Context, tag string, rec map[string]any) error { return nil }
func (n *NoopClient) Close() error                                                   { return nil }
