# Event Tracking API

A production-ready event tracking service with JWT/API key authentication, asynchronous event processing via NATS, Redis-backed rate limiting, and an admin dashboard. Built with Go, PostgreSQL, React-Admin, and Docker.

## Quick Start

1. Clone and configure:
```
git clone https://github.com/neo-vai/go-events.git
cd go-events
cp .env.example .env
```
   Edit `.env` with your own secrets (JWT_SECRET, DB_PASSWORD, etc.).

2. Build and start all services:
```
make up
```
   This command will build the admin panel, create Docker images, and start the whole stack.

3. Access the services:
    - API: `https://localhost` (self‑signed certificate)
    - Admin UI: `https://localhost/admin`
    - Grafana: `http://localhost:3001` (login: `admin` / `admin`)
    - Prometheus: `http://localhost:9090`

## Services & Ports

| Service      | Description                                          | Host Port | Internal |
|--------------|------------------------------------------------------|-----------|----------|
| **go-events**| REST API server (Gin)                                | –         | 8080     |
| **worker**   | Background event processor (2 replicas)              | –         | –        |
| **db**       | PostgreSQL 18 – primary data store                   | 5432      | 5432     |
| **redis**    | Redis 8 – API key cache & rate limiting              | 6379      | 6379     |
| **nats**     | NATS 2.10 – message broker with JetStream support    | 4222      | 4222     |
| **nginx**    | Reverse proxy & static admin hosting                 | 80, 443   | –        |
| **prometheus** | Metrics collection                                 | 9090      | –        |
| **grafana**  | Metrics visualization                                | 3001      | 3000     |

## Using Grafana

1. Open `http://localhost:3001` in your browser.
2. Log in with the default credentials: `admin` / `admin` (you will be prompted to change the password).
3. Add a data source:
    - Click **Connections** → **Data sources** → **Add data source**.
    - Choose **Prometheus**.
    - Set URL to `http://prometheus:9090` (Docker internal network).
    - Click **Save & test**.
4. Import a dashboard:
    - Click **Dashboards** → **Import**.
    - You can use the official Go process dashboard (ID `6671`) or create your own panels.
    - Select the Prometheus data source and import.

The API exposes Prometheus metrics at `/metrics`. Example queries:
- `gin_requests_total` – total HTTP requests
- `gin_request_duration_seconds` – request latency
- `go_goroutines` – number of goroutines

## Admin Panel

The admin panel is built with React-Admin and is served by Nginx under `/admin`. Only accounts with role `admin` can log in.

- **Dashboard**: Shows key metrics (total accounts, events today, active API keys).
- **Accounts**: List, filter, create, edit, delete accounts. You can change role and active status.
- **API Keys**: View all API keys, activate/deactivate, delete.
- **Events**: Browse all events across accounts, view payload details.

## REST API

Base URL: `/api/v1`  
Authentication: JWT (`Authorization: Bearer <token>`) or API key (`X-API-Key: <key>`)

### Public Endpoints

| Method | Path           | Description                           |
|--------|----------------|---------------------------------------|
| POST   | `/accounts`    | Create a new account                  |
| POST   | `/login`       | Authenticate and receive JWT token    |

### Protected Endpoints (JWT or API Key)

| Method | Path                           | Description                                      |
|--------|--------------------------------|--------------------------------------------------|
| GET    | `/account`                     | Get current account details                      |
| PATCH  | `/account`                     | Update current account email                     |
| DELETE | `/account`                     | Delete current account                           |
| POST   | `/account/keys`                | Generate a new API key (full key returned once)  |
| GET    | `/account/keys`                | List API keys (prefix only)                      |
| PATCH  | `/account/keys/{key_id}`       | Activate/deactivate an API key                   |
| DELETE | `/account/keys/{key_id}`       | Delete an API key                                |
| POST   | `/events`                      | Create an event (async)                          |
| GET    | `/events`                      | List events with pagination & filters            |
| GET    | `/events/{id}`                 | Get a single event                               |

### Admin Endpoints (JWT + admin role only)

| Method | Path                       | Description                          |
|--------|----------------------------|--------------------------------------|
| GET    | `/admin/accounts`          | List all accounts                    |
| GET    | `/admin/accounts/{id}`     | Get account details                  |
| POST   | `/admin/accounts`          | Create account (can set role)        |
| PUT    | `/admin/accounts/{id}`     | Update account (role, active, email) |
| DELETE | `/admin/accounts/{id}`     | Delete account                       |
| GET    | `/admin/events`            | List all events across accounts      |
| GET    | `/admin/events/{id}`       | Get event details                    |
| GET    | `/admin/api-keys`          | List all API keys                    |
| PUT    | `/admin/api-keys/{id}`     | Update API key active status         |
| DELETE | `/admin/api-keys/{id}`     | Delete API key                       |
| GET    | `/admin/stats`             | System statistics                    |

### Request / Response Examples

Create account:
```
POST /api/v1/accounts
{
  "email": "user@example.com",
  "password": "StrongPass123"
}
```

Login:
```
POST /api/v1/login
{
  "email": "user@example.com",
  "password": "StrongPass123"
}
```
Response:
```
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "accountId": "550e8400-e29b-41d4-a716-446655440000"
}
```

Create event (with API key):
```
POST /api/v1/events
X-API-Key: your-api-key-here
{
  "username": "john_doe",
  "name": "user_login",
  "payload": "{\"ip\":\"192.168.1.1\"}"
}
```
Response: `202 Accepted` with `Location` header pointing to the eventual event resource.

List events with pagination:
```
GET /api/v1/events?_start=0&_end=20&_sort=createdAt&_order=DESC&filter={"username":"john_doe"}
```

## Environment Variables

Copy `.env.example` to `.env` and adjust the values. Important variables:

| Variable           | Description                                      | Example                       |
|--------------------|--------------------------------------------------|-------------------------------|
| `DATABASE_URL`     | PostgreSQL connection string                     | `postgres://user:pass@db:5432/eventtracker` |
| `REDIS_URL`        | Redis address                                    | `redis:6379`                  |
| `REDIS_PASSWORD`   | Redis password                                   | `redispass`                   |
| `JWT_SECRET`       | Secret for signing JWT (min 32 chars)            | `change-this-to-a-long-secret`|
| `BROKER_URL`       | NATS connection URL                              | `nats://nats:4222`            |
| `RATE_LIMIT_GLOBAL`| Global rate limit (requests/duration)            | `100/1m`                      |
| `RATE_LIMIT_LOGIN` | Login endpoint rate limit                        | `5/1m`                        |

## Makefile Commands

| Command               | Description                                    |
|-----------------------|------------------------------------------------|
| `make up`             | Build admin and start all services             |
| `make down`           | Stop all services                              |
| `make logs`           | Tail logs from all containers                  |
| `make test`           | Run all tests with a fresh test database       |
| `make migrate-up`     | Apply database migrations                      |
| `make migrate-down`   | Rollback migrations                            |
| `make swagger`        | Generate Swagger docs (requires swag)          |

## Testing

The project includes unit and integration tests.

```
make test
```
This command:
- Starts a separate PostgreSQL, Redis, and NATS containers for testing
- Applies migrations
- Runs `go test -p 1 ./... -v`
- Cleans up containers after tests finish

## Project Structure

```
├── admin/                # React-Admin frontend
├── cmd/
│   ├── api/              # Main API server entrypoint
│   └── worker/           # NATS consumer worker entrypoint
├── internal/
│   ├── broker/           # NATS client and publisher/subscriber
│   ├── config/           # Configuration loading and validation
│   ├── handler/          # HTTP handlers (account, apikey, event, admin, auth)
│   ├── middleware/       # Auth, logging, CORS, rate limiting
│   ├── model/            # Domain models
│   ├── pagination/       # React-Admin pagination parser
│   ├── repository/       # PostgreSQL repositories
│   ├── service/          # Business logic
│   └── validator/        # Custom validation rules
├── migrations/           # Database migration files
├── docker-compose.yml    # Main Docker Compose configuration
├── Dockerfile            # API server image
├── Dockerfile.worker     # Worker image
├── Makefile              # Development and deployment commands
└── README.md             # This file
```