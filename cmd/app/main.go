package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"interchange/config"
	"interchange/internal/command"
	"interchange/internal/log"
	"interchange/utils/path"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	_ "interchange/cmd/docs"
)

var (
	rootPath = path.RootPath()
	Version  string
	envPath  string
	yamlPath string
	conf     *config.Configuration
	logger   *zap.Logger
)

func init() {
	pflag.StringVarP(&envPath, "env", "e", "", "Environment file, e.g. --env .env")
	pflag.StringVarP(&yamlPath, "config", "c", "", "YAML config file, e.g. --config config.yaml")
	pflag.Parse()

	cobra.OnInitialize(func() {
		if envPath != "" && yamlPath != "" {
			fmt.Println("同時指定 --env 與 --config，將以 --env 優先")
		}
		initConfig()
	})
}

// @title        interchange API
// @version      1.0
// @description  這是後端 API 文件
// @host         localhost:3000
// @basePath     /
// @securityDefinitions.apikey ApiKeyAuth
// @in   header
// @name X-API-Key

// @securityDefinitions.apikey BearerAuth
// @in   header
// @name Authorization
// @description 請在欄位輸入 "Bearer {token}"
func main() {
	rootCmd := &cobra.Command{
		Use: "app",
		Run: func(cmd *cobra.Command, args []string) {
			if conf == nil {
				panic("config is nil! Check config/initConfig logic.")
			}
			// 初始化 logger
			logger, err := log.NewLogger(conf)
			if err != nil {
				panic(fmt.Errorf("init logger failed: %w", err))
			}
			defer logger.Sync()
			app, cleanup, err := wireApp(conf, logger)
			if err != nil {
				panic(err)
			}
			defer cleanup()

			logger.Info("start app ...")
			if err := app.Run(); err != nil {
				panic(err)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			logger.Info("shutdown app ...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := app.Stop(ctx); err != nil {
				panic(err)
			}
		},
	}

	command.Register(rootCmd, func() (*command.Command, func(), error) {
		return wireCommand(conf, logger)
	})

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func initConfig() {
	v := viper.NewWithOptions(viper.KeyDelimiter("__"))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	useFile := false

	if envPath != "" {
		useFile = true
		if !filepath.IsAbs(envPath) {
			envPath = filepath.Join(rootPath, envPath)
		}
		fmt.Println("load .env config:", envPath)
		v.SetConfigFile(envPath)
		v.SetConfigType("env")
	} else if yamlPath != "" {
		useFile = true
		if !filepath.IsAbs(yamlPath) {
			yamlPath = filepath.Join(rootPath, "conf", yamlPath)
		}
		fmt.Println("load yaml config:", yamlPath)
		v.SetConfigFile(yamlPath)
		v.SetConfigType("yaml")
	} else {
		fmt.Println("No configuration file specified, using environment variables only.")
	}

	if useFile {
		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Errorf("read config failed: %w", err))
		}
		v.WatchConfig()
		v.OnConfigChange(func(in fsnotify.Event) {
			fmt.Println("config file changed:", in.Name)
			if err := v.Unmarshal(&conf); err != nil {
				fmt.Println("unmarshal on change failed:", err)
			}
		})
	}

	bindEnvs(v, reflect.TypeOf(config.Configuration{}))

	if err := v.Unmarshal(&conf); err != nil {
		fmt.Println("unmarshal config failed:", err)
	}

}
func bindEnvs(v *viper.Viper, t reflect.Type, path ...string) {
	// 若遇到指標，取其 Elem
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			tag = field.Name
		}
		newPath := append(path, tag)
		if field.Type.Kind() == reflect.Struct || (field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct) {
			bindEnvs(v, field.Type, newPath...)
		} else {
			v.BindEnv(strings.Join(newPath, "__"))
		}
	}
}
