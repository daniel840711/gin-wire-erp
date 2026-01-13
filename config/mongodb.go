package config

type MongoDB struct {
	URI     string `mapstructure:"URI" json:"uri" yaml:"uri"`
	Options string `mapstructure:"OPTIONS" json:"options" yaml:"options"`
}
