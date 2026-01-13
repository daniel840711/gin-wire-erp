package telemetry

import (
	"interchange/config"
	"interchange/internal/core"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metric struct
type Metric struct {
	HttpRequestsTotal   *prometheus.CounterVec
	HttpRequestDuration *prometheus.HistogramVec
	ProxySuccessTotal   *prometheus.CounterVec
	ProxyFailTotal      *prometheus.CounterVec
	config              *config.Configuration
}

// NewMetric 建立所有指標
func NewMetric(config *config.Configuration) *Metric {
	if config == nil || !config.Telemetry.Metric.Enabled {
		return &Metric{}
	}
	buckets := prometheus.DefBuckets
	if len(config.Telemetry.Metric.Buckets) > 0 {
		buckets = config.Telemetry.Metric.Buckets
	}
	return &Metric{
		config: config,
		HttpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.App.Name + "_" + string(core.MetricHttpRequestsTotal),
				Help: "Total received API requests (all proxy calls)",
			},
			labelNames(core.MetricLabelEndpoint, core.MetricLabelStatus),
		),
		HttpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    config.App.Name + "_" + string(core.MetricHttpRequestDuration),
				Help:    "Request duration to provider (seconds)",
				Buckets: buckets,
			},
			labelNames(core.MetricLabelEndpoint),
		),
		ProxySuccessTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.App.Name + "_" + string(core.MetricProxySuccessTotal),
				Help: "Proxy forward success count",
			},
			labelNames(core.MetricLabelEndpoint, core.MetricLabelStatus),
		),
		ProxyFailTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: config.App.Name + "_" + string(core.MetricProxyFailTotal),
				Help: "Proxy forward failed count",
			},
			labelNames(core.MetricLabelReason),
		),
	}
}

// labelNames helper: LabelName slice 轉成 []string
func labelNames(labels ...core.MetricLabelName) []string {
	strs := make([]string, len(labels))
	for i, l := range labels {
		strs[i] = string(l)
	}
	return strs
}
