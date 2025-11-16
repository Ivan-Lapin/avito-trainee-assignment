APP_NAME=pr-reviewer
GO=go

.PHONY: all build run test e2e lint generate migrate-up migrate-down seed up down k6

all: build

build:
	$(GO) build -o bin/$(APP_NAME) ./cmd/server

run:
	ENV_FILE=.env $(GO) run ./cmd/server

# Юнит-тесты и пакетные тесты
test:
	$(GO) test ./... -cover

# Интеграционные/E2E (ожидается, что сервис уже запущен на 8095)
e2e:
	$(GO) test ./app/test/integration -v

lint:
	golangci-lint run

generate:
	oapi-codegen -generate types,chi-server -package openapi -o internal/openapi/generated.go openapi/openapi.yml

migrate-up:
	goose -dir internal/repository/migrations postgres "$$DB_DSN" up

migrate-down:
	goose -dir internal/repository/migrations postgres "$$DB_DSN" down

seed:
	psql "$$DB_DSN" -f internal/repository/seeds/seed.sql

up:
	docker-compose up --build

down:
	docker-compose down -v

k6:
	BASE=http://localhost:8095/api k6 run k6/load.js
