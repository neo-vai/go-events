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
	@echo "  make build         - Build Docker images"
	@echo "  make up            - Start all services"
	@echo "  make down          - Stop all services"
	@echo "  make logs          - Follow logs of all services"
	@echo "  make dev           - Run API locally (without Docker)"
	@echo "  make test          - Run all tests (with test database)"
	@echo "  make migrate-up    - Apply database migrations"
	@echo "  make migrate-down  - Rollback migrations"
	@echo "  make swagger-install - Install swag CLI"
	@echo "  make swagger         - Generate Swagger documentation"

.PHONY: admin-build
admin-build:
	@echo "Building admin panel..."
	cd admin && npm ci --verbose && npm run build

.PHONY: build
build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

.PHONY: up
up:
	@if [ ! -f admin/dist/index.html ]; then \
		echo "Admin panel not built or missing. Building..."; \
		$(MAKE) admin-build; \
	else \
		echo "Admin panel already built. Skipping build."; \
	fi
	$(DOCKER_COMPOSE) up -d

.PHONY: down
down:
	$(DOCKER_COMPOSE) down

.PHONY: logs
logs:
	$(DOCKER_COMPOSE) logs -f

.PHONY: dev
dev:
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

.PHONY: migrate-install
migrate-install:
	@echo "Installing golang-migrate CLI..."
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Migrate CLI installed to $$(go env GOPATH)/bin/migrate"
	@echo "Make sure $$(go env GOPATH)/bin is in your PATH"
	@echo "Run 'export PATH=\$$HOME/go/bin:\$$PATH' if needed"

.PHONY: migrate-up
migrate-up:
	@echo "Applying migrations..."
	migrate -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path ./migrations -database "$(DATABASE_URL_LOCALHOST)" down

.PHONY: swagger-install
swagger-install:
	@echo "Installing swagger tools..."
	go install github.com/swaggo/swag/cmd/swag@latest

.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	swag init \
		-d ./cmd/api,./internal/handler \
		-g main.go \
		-o ./docs \
		--parseDependency \
		--parseInternal \
		--parseDepth 3
	@echo "Swagger documentation generated in ./docs"
