package config

type Redis struct {
	Host     string `mapstructure:"HOST" json:"host" yaml:"host"`
	Port     int    `mapstructure:"PORT" json:"port" yaml:"port"`
	Password string `mapstructure:"PASSWORD" json:"password" yaml:"password"`
	DB       int    `mapstructure:"DB" json:"db" yaml:"db"`
}
