# Makefile for Event Tracker Service

APP_NAME := go-events
DOCKER_COMPOSE := docker compose
MIGRATE := migrate
LINTER := ./bin/golangci-lint
LINTER_VERSION := v1.54.2
SWAG := swag

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make install-tools   - Install all required tools (linter, swag, migrate)"
	@echo "  make install-linter  - Install golangci-lint"
	@echo "  make install-swag    - Install swag for Swagger generation"
	@echo "  make install-migrate - Install migrate (requires Go 1.16+)"
	@echo "  make swagger         - Generate Swagger documentation"
	@echo "  make build           - Build Go binary"
	@echo "  make run             - Run application locally (without Docker)"
	@echo "  make up              - Start services with docker-compose"
	@echo "  make down            - Stop services"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make migrate-up      - Apply database migrations (requires DATABASE_URL)"
	@echo "  make migrate-down    - Rollback migrations"
	@echo "  make test            - Run tests"

.PHONY: install-tools
install-tools: install-linter install-swag install-migrate

.PHONY: install-linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(LINTER_VERSION)

.PHONY: install-swag
install-swag:
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: install-migrate
install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

.PHONY: swagger
swagger:
	$(SWAG) init -g cmd/api/main.go --output docs/

.PHONY: build
build:
	go build -o bin/$(APP_NAME) ./cmd/api

.PHONY: run
run:
	go run ./cmd/api

.PHONY: up
up:
	$(DOCKER_COMPOSE) up --build

.PHONY: down
down:
	$(DOCKER_COMPOSE) down

.PHONY: lint
lint:
	$(LINTER) run ./...

.PHONY: migrate-up
migrate-up:
	$(MIGRATE) -path ./migrations -database "$$DATABASE_URL" up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) -path ./migrations -database "$$DATABASE_URL" down

.PHONY: test
test:
	go test ./... -v