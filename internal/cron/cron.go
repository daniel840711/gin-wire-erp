package cron

import (
	"context"

	"github.com/google/wire"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var ProviderSet = wire.NewSet(NewCron)

type Cron struct {
	logger *zap.Logger
	server *cron.Cron
}

// NewCron .
func NewCron(logger *zap.Logger) *Cron {
	server := cron.New(
		cron.WithSeconds(),
	)

	return &Cron{
		logger: logger,
		server: server,
	}
}

func (c *Cron) Run() error {
	// cron example
	//if _, err := c.server.AddFunc("*/5 * * * * *", c.exampleJob.Hello); err != nil {
	//   return err
	//}

	c.server.Start()
	return nil
}

func (c *Cron) Stop(ctx context.Context) error {
	c.server.Stop()
	return nil
}
