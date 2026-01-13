package config

type TelemetryConfig struct {
	Metric struct {
		Enabled bool      `yaml:"enabled" mapstructure:"ENABLED" json:"enabled"`
		Buckets []float64 `yaml:"buckets" mapstructure:"BUCKETS" json:"buckets"`
	} `yaml:"metric" mapstructure:"METRIC" json:"metric"`
	Trace struct {
		Enabled     bool   `yaml:"enabled" mapstructure:"ENABLED" json:"enabled"`
		EndpointUrl string `yaml:"endpointUrl" mapstructure:"ENDPOINT_URL" json:"endpointUrl"`
	} `yaml:"trace" mapstructure:"TRACE" json:"trace"`
}
