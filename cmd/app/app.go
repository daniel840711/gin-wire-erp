package main

import (
	"context"
	"interchange/config"
	"interchange/internal/cron"
	"interchange/internal/service"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RuntimeInfo struct {
	Env       string        `json:"env"`
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	GoVersion string        `json:"go_version"`
	StartAt   time.Time     `json:"start_at"`
	Uptime    time.Duration `json:"uptime"`
}

type App struct {
	conf          *config.Configuration
	logger        *zap.Logger
	cronSrv       *cron.Cron
	Router        *gin.Engine
	healthService *service.HealthService

	startAt time.Time   // 程式啟動時間（非環境變數）
	appInfo RuntimeInfo // 版本/環境快照（來源 = conf.App）
}

func newHttpServer(
	conf *config.Configuration,
	router *gin.Engine,
) *http.Server {
	return &http.Server{
		Addr:    ":" + strconv.FormatUint(uint64(conf.App.Port), 10),
		Handler: router,
	}
}

func newHttpClient() *http.Client {
	return http.DefaultClient
}

func newApp(
	conf *config.Configuration,
	logger *zap.Logger,
	router *gin.Engine,
	healthService *service.HealthService,
	cronSrv *cron.Cron,
) *App {
	startAt := time.Now()
	return &App{
		conf:          conf,
		logger:        logger,
		Router:        router,
		healthService: healthService,
		cronSrv:       cronSrv,
		startAt:       startAt,
		appInfo: RuntimeInfo{
			Env:       conf.App.Env,
			Name:      conf.App.Name,
			Version:   conf.App.Version,
			GoVersion: runtime.Version(),
			StartAt:   startAt,
		},
	}
}

func (a *App) Run() error {
	// 1) 啟動時寫入版本/環境資訊
	info := a.appInfo
	a.logger.Info("app runtime info",
		zap.String("env", info.Env),
		zap.String("name", info.Name),
		zap.String("version", info.Version),
		zap.String("go_version", info.GoVersion),
		zap.Time("start_at", info.StartAt),
	)

	// 2) 動態掛到現有 gin.Engine：加 X-App-Version 標頭與 /version API
	if a.Router != nil {
		// 全域版本標頭（只用 conf.App.Version）
		a.Router.Use(func(c *gin.Context) {
			if v := a.conf.App.Version; v != "" {
				c.Writer.Header().Set("X-App-Version", v)
			}
			c.Next()
		})
		// /version：回傳 JSON（含 uptime）
		a.Router.GET("/version", func(c *gin.Context) {
			resp := a.appInfo
			resp.Uptime = time.Since(a.startAt)
			c.JSON(http.StatusOK, resp)
		})
	}

	// 3) 啟動 cron
	if err := a.cronSrv.Run(); err != nil {
		return err
	}
	a.logger.Info("cron server started")

	return nil
}

func (a *App) Close(ctx context.Context) error {
	if a.healthService != nil {
		a.healthService.SetReady(false)
	}
	if a.cronSrv == nil {
		return nil
	}

	if err := a.cronSrv.Stop(ctx); err != nil {
		return err
	}
	a.logger.Info("cron server has been stop")

	return nil
}

func (a *App) Stop(ctx context.Context) error {
	return a.Close(ctx)
}
