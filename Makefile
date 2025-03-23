# Makefile for Event Tracker Service

# Variables
APP_NAME := go-events
DOCKER_COMPOSE := docker-compose
MIGRATE := migrate
LINTER := golangci-lint
LINTER_VERSION := v1.54.2 # укажи нужную версию

# Default target
.PHONY: help
help:
	@echo "Makefile commands:"
	@echo "  make up              - Start services with docker-compose"
	@echo "  make down            - Stop services"
	@echo "  make build           - Build Go binary"
	@echo "  make install-linter  - Download and install golangci-lint"
	@echo "  make lint            - Run golangci-lint on the code"
	@echo "  make migrate-up      - Apply migrations"
	@echo "  make migrate-down    - Rollback migrations"
	@echo "  make test            - Run Go tests"

# Docker
.PHONY: up
up:
	$(DOCKER_COMPOSE) up --build

.PHONY: down
down:
	$(DOCKER_COMPOSE) down

# Go build
.PHONY: build
build:
	go build -o bin/$(APP_NAME) ./cmd/api

# Install golangci-lint automatically
.PHONY: install-linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(LINTER_VERSION)

# Linter
.PHONY: lint
lint:
	$(LINTER) run ./...

# Migrations
.PHONY: migrate-up
migrate-up:
	$(MIGRATE) -path ./migrations -database $$DATABASE_URL up

.PHONY: migrate-down
migrate-down:
	$(MIGRATE) -path ./migrations -database $$DATABASE_URL down

# Tests
.PHONY: test
test:
	go test ./... -v