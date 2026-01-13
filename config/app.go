package config

type App struct {
	// 當前開發環境
	Env string `mapstructure:"ENV" json:"env" yaml:"env"`
	// 服務端口
	Port uint32 `mapstructure:"PORT" json:"port" yaml:"port"`
	// 服務名稱
	Name string `mapstructure:"NAME" json:"name" yaml:"name"`
	// 服務版本
	Version string `mapstructure:"VERSION" json:"version" yaml:"version"`
	// Secret Key 用於生成 API Key
	SecretKey      string `mapstructure:"SECRET_KEY" json:"secret_key" yaml:"secret_key"`
	SwaggerEnabled bool   `mapstructure:"SWAGGER_ENABLED" json:"swagger_enabled" yaml:"swagger_enabled"`
}
