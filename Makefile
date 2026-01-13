GOPATH := $(shell go env GOPATH)
APP_MAIN_DIR = cmd/app
MAIN        = ./cmd/app/main.go
SWAG_DIR    = ./cmd/docs
CONFIG_TYPE ?= env

# ENV 變數從第二參數帶入（如 make run dev），預設 dev
ENV ?= dev

# 根據 ENV 判斷 .env 檔名
define ENV_FILENAME
$(if $(findstring prod,$(1)),.env.prod,\
  $(if $(findstring test,$(1)),.env.test,.env))
endef

# 根據 CONFIG_TYPE 產生對應 flag
define CONFIG_FLAG
$(if $(findstring env,$(CONFIG_TYPE)),-e $(call ENV_FILENAME,$(1)),\
  $(if $(findstring config,$(CONFIG_TYPE)),\
    $(if $(findstring dev,$(1)),-c ./config.yaml,-c ./config.$(1).yaml),\
  $(error ❌ Invalid CONFIG_TYPE: $(CONFIG_TYPE). Allowed values: env, config)))
endef

.PHONY: run dev test prod

# 通用入口：使用 make run dev/config/test + CONFIG_TYPE
run:
	cd $(APP_MAIN_DIR) && go run main.go wire_gen.go app.go $(call CONFIG_FLAG,$(ENV))

# 指定環境別：設定 ENV 給 run
dev:  ENV=dev
dev:  run

test: ENV=test
test: run

prod: ENV=prod
prod: run

# 可選項目
.PHONY: build
build:
	mkdir -p bin/ && go build -o ./bin/ ./...

.PHONY: generate
generate:
	go mod tidy
	go get github.com/google/wire/cmd/wire@latest
	go generate ./...
	swag init --parseDependency --parseInternal -g $(MAIN) -o $(SWAG_DIR)
	cd $(APP_MAIN_DIR) && wire

.PHONY: wire
wire:
	cd $(APP_MAIN_DIR) && wire

.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make run dev CONFIG_TYPE=env    # 使用 .env'
	@echo '  make run prod CONFIG_TYPE=config # 使用 conf/config.prod.yaml'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z0-9_-]+:/ { \
		if (match(lastLine, /^# (.*)/, arr)) { \
			helpCommand = substr($$1, 1, index($$1, ":")-1); \
			helpMessage = arr[1]; \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
