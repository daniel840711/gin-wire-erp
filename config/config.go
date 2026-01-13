package config

type Configuration struct {
	App       App             `mapstructure:"APP" json:"app" yaml:"app"`
	Redis     Redis           `mapstructure:"REDIS" json:"redis" yaml:"redis"`
	Log       Log             `mapstructure:"LOG" json:"log" yaml:"log"`
	MongoDB   MongoDB         `mapstructure:"MONGODB" json:"mongodb" yaml:"mongodb"`
	Telemetry TelemetryConfig `mapstructure:"TELEMETRY" yaml:"telemetry"`
	Fluentd   Fluentd         `mapstructure:"FLUENTD" yaml:"fluentd"`
}
