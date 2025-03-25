# Makefile for Event Tracker Service

# Автоматическая загрузка .env файла
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

APP_NAME := go-events
DOCKER_COMPOSE := docker compose
MIGRATE := migrate
LINTER := ./bin/golangci-lint
LINTER_VERSION := v1.54.2
SWAG := swag

# ==== Тестовая среда ====
TEST_DB_CONTAINER := test-db
TEST_DB_PORT := 5433
TEST_DB_USER := postgres
TEST_DB_PASSWORD := postgres
TEST_DB_NAME := eventtracker_test
TEST_DB_URL := postgres://$(TEST_DB_USER):$(TEST_DB_PASSWORD)@localhost:$(TEST_DB_PORT)/$(TEST_DB_NAME)?sslmode=disable

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make install-tools   - Install all required tools"
	@echo "  make install-linter  - Install golangci-lint"
	@echo "  make install-swag    - Install swag for Swagger generation"
	@echo "  make install-migrate - Install migrate"
	@echo "  make swagger         - Generate Swagger documentation"
	@echo "  make build           - Build Go binary"
	@echo "  make run             - Run application locally"
	@echo "  make up              - Start services with docker-compose"
	@echo "  make down            - Stop services"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make migrate-up      - Apply database migrations"
	@echo "  make migrate-down    - Rollback migrations"
	@echo "  make test            - Run all tests (with test database)"
	@echo "  make test-only       - Run tests without restarting test database"
	@echo "  make test-db-up      - Start test database container"
	@echo "  make test-db-down    - Stop and remove test database container"
	@echo "  make test-db-migrate - Apply migrations to test database"

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
	@echo "Applying migrations..."
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back migrations..."
	$(MIGRATE) -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" down

.PHONY: test-db-up
test-db-up:
	@echo "Starting test database..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for test database to be ready..."
	@until docker compose -f docker-compose.test.yml exec test-db pg_isready -U $(TEST_DB_USER) -d $(TEST_DB_NAME); do sleep 1; done

.PHONY: test-db-down
test-db-down:
	@echo "Stopping and removing test database..."
	@docker compose -f docker-compose.test.yml down -v

.PHONY: test-db-migrate
test-db-migrate: test-db-up
	@echo "Applying migrations to test database..."
	@docker run --rm --network host -v $(PWD)/migrations:/migrations migrate/migrate \
		-path=/migrations -database "$(TEST_DB_URL)" up

.PHONY: test
test: test-db-migrate
	@echo "Running tests..."
	@DATABASE_URL_TEST="$(TEST_DB_URL)" go test ./... -v
	@echo "Tests completed."

.PHONY: test-only
test-only:
	@echo "Running tests (using existing test database)..."
	@DATABASE_URL_TEST="$(TEST_DB_URL)" go test ./... -v

.PHONY: shell
shell:
	@echo "Loading environment from .env..."
	@export $$(cat .env | xargs) && exec $$SHELL