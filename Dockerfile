FROM golang:1.24-alpine AS builder
RUN apk update && apk add --no-cache git curl make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go install github.com/google/wire/cmd/wire@latest \
  && go install github.com/swaggo/swag/cmd/swag@latest \
  && swag init --parseDependency --parseInternal -g /cmd/app/main.go --output ../docs
WORKDIR /app/cmd/app
RUN wire && go build -o /app/bin/app .

FROM alpine:latest

WORKDIR /app

# 複製執行檔與 docs
COPY --from=builder /app/bin/app .
COPY --from=builder /app/cmd/docs ./docs

# 複製設定與資源（如有）
COPY --from=builder /app/conf ./conf

# 開放埠口
EXPOSE 3000

# 啟動程式
CMD ["./app"]
