package config

type Fluentd struct {
	Host      string `mapstructure:"HOST" json:"host" yaml:"host"`
	Port      int    `mapstructure:"PORT" json:"port" yaml:"port"`
	TagPrefix string `mapstructure:"TAG_PREFIX" json:"tagPrefix" yaml:"tagPrefix"`
	Timeout   int64  `mapstructure:"TIMEOUT" json:"timeout" yaml:"timeout"`
}
