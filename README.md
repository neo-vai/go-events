# Event Tracking API

This service provides a REST API for tracking events. It allows you to create accounts, manage API keys, and send events with metadata. Events can be queried using various filters.

## Core Concepts

The service manages three main entities:

- **Account** – represents a user or system that owns events and API keys. Each account has a unique ID, name, email, login, and a hashed password.
- **API Key** – a secret token associated with an account. API keys can be activated or deactivated and are used for authentication instead of JWT.
- **Event** – a record of something that happened. Each event belongs to an account, has a name, a username, an optional API key ID, and a JSON payload.

## Authentication

The API supports two authentication methods:

- **JWT** – obtained via the login endpoint using account credentials. Sent in the `Authorization: Bearer <token>` header.
- **API Key** – sent in the `X-API-Key` header. The key must be active and belong to the account.

Protected endpoints automatically accept either method. The authentication middleware tries JWT first, then falls back to API key validation.

## How It Works Internally

The application follows a layered architecture:

- **Handlers** (in `internal/handler`) parse HTTP requests, validate input, and call the appropriate service.
- **Services** (in `internal/service`) contain business logic. They coordinate repositories and apply rules (e.g., password hashing, key generation).
- **Repositories** (in `internal/repository`) abstract database operations. The only implementation uses PostgreSQL with pgx.
- **Models** (in `internal/model`) define the core data structures.

Middleware adds cross-cutting concerns:

- Request ID generation and propagation
- Structured JSON logging (method, path, status, latency)
- CORS headers
- Per-IP rate limiting
- Authentication via JWT or API key

## Database Schema

Three tables are used:

- `accounts` – stores account details, including a hashed password.
- `api_keys` – stores API keys linked to an account. Deleting an account cascades to its keys.
- `events` – stores event records. The `api_key_id` column is nullable (set to NULL if the key is deleted).

Indexes exist on `events.account_id`, `events.username`, `events.name`, and `events.created_at` for faster queries.

## Event Filtering Logic

The `ListEvents` handler supports filtering by `account_id`, `user` (username), and `api_key_id`. The service layer combines these filters:

- If multiple filters are provided, the results are intersected in memory.
- Because the repository only supports a subset of combined filters, the service fetches the broadest set and then narrows it down.

## Swagger Documentation

The API is documented with Swagger 2.0. The spec is generated from code annotations in the handlers. The UI is served at `/swagger/index.html`.

## Testing Approach

- Unit tests use mocks generated with `testify/mock`.
- Integration tests use a real PostgreSQL database (via a test container). The `testutil` package sets up a connection pool and truncates tables between tests to ensure isolation.

## Project Layout

- `cmd/api/main.go` – entry point, sets up the database pool, wires dependencies, and starts the HTTP server.
- `internal/handler/` – HTTP layer (account, api key, auth, event). Also contains DTOs for requests/responses.
- `internal/middleware/` – reusable Gin middleware.
- `internal/service/` – business logic.
- `internal/repository/` – database implementations, each with an interface.
- `internal/model/` – plain structs representing database rows.
- `migrations/` – SQL files for creating and dropping the schema.
- `docs/` – auto-generated Swagger files.