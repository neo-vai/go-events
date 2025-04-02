# Makefile for Event Tracker Service

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

APP_NAME := go-events
DOCKER_COMPOSE := docker compose

export DOCKER_BUILDKIT := 1
export COMPOSE_DOCKER_CLI_BUILD := 1

TEST_DB_PORT := 5433
TEST_DB_USER := postgres
TEST_DB_PASSWORD := postgres
TEST_DB_NAME := eventtracker_test
TEST_DB_URL := postgres://$(TEST_DB_USER):$(TEST_DB_PASSWORD)@localhost:$(TEST_DB_PORT)/$(TEST_DB_NAME)?sslmode=disable

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make admin-build   - Build admin panel static files"
	@echo "  make rebuild-admin - Force rebuild admin panel"
	@echo "  make build         - Build Go binaries (api + worker)"
	@echo "  make build-api     - Build only API binary"
	@echo "  make build-worker  - Build only worker binary"
	@echo "  make clean         - Remove Go binaries"
	@echo "  make up            - Start all services (auto-builds admin if missing)"
	@echo "  make down          - Stop all services"
	@echo "  make logs          - Follow logs of all services"
	@echo "  make run           - Run API locally (without Docker)"
	@echo "  make test          - Run all tests (with test database)"
	@echo "  make migrate-up    - Apply database migrations"
	@echo "  make migrate-down  - Rollback migrations"

.PHONY: admin-build
admin-build:
	@echo "Building admin panel..."
	cd admin && npm ci && npm run build

.PHONY: rebuild-admin
rebuild-admin: admin-build

.PHONY: build-api
build-api:
	@echo "Building API binary..."
	go build -o app ./cmd/api

.PHONY: build-worker
build-worker:
	@echo "Building worker binary..."
	go build -o worker ./cmd/worker

.PHONY: build
build: build-api build-worker

.PHONY: clean
clean:
	@echo "Removing Go binaries..."
	rm -f app worker

.PHONY: up
up:
	@if [ ! -f admin/dist/index.html ]; then \
		echo "Admin panel not built or missing. Building..."; \
		$(MAKE) admin-build; \
	else \
		echo "Admin panel already built. Skipping build."; \
	fi
	$(DOCKER_COMPOSE) up -d
	@echo "Services started. Admin panel available at http://localhost/admin"

.PHONY: down
down:
	$(DOCKER_COMPOSE) down

.PHONY: logs
logs:
	$(DOCKER_COMPOSE) logs -f

.PHONY: run
run:
	go run ./cmd/api

.PHONY: test
test:
	@echo "Starting test database..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for test database..."
	@until docker compose -f docker-compose.test.yml exec test-db pg_isready -U $(TEST_DB_USER) -d $(TEST_DB_NAME); do sleep 1; done
	@echo "Applying migrations to test database..."
	@docker run --rm --network host -v $(PWD)/migrations:/migrations migrate/migrate \
		-path=/migrations -database "$(TEST_DB_URL)" up
	@echo "Running tests..."
	@DATABASE_URL_TEST="$(TEST_DB_URL)" go test -p 1 ./... -v
	@echo "Stopping test database..."
	@docker compose -f docker-compose.test.yml down -v

.PHONY: migrate-up
migrate-up:
	@echo "Applying migrations..."
	migrate -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" down