# 🚀 go-events

> **Production-ready event tracking service with JWT/API key authentication, async event processing, and admin dashboard.**

Built with Go, PostgreSQL, NATS, Redis, and React-Admin — a scalable event ingestion platform for modern applications.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/NATS-2.10-27AAE1?logo=nats" alt="NATS">
  <img src="https://img.shields.io/badge/Redis-8-DC382D?logo=redis" alt="Redis">
  <img src="https://img.shields.io/badge/Gin-✓-00D2FF?logo=gin" alt="Gin">
  <img src="https://img.shields.io/badge/React--Admin-✓-6CA5DB?logo=react" alt="React-Admin">
  <img src="https://img.shields.io/badge/Docker-✓-2496ED?logo=docker" alt="Docker">
</p>

---

## ✨ Features

- **Event ingestion API** – Create, list, and retrieve events with pagination and filtering.
- **Dual authentication** – JWT tokens for admin access, API keys for programmatic event creation.
- **Async event processing** – Events are published to NATS and processed asynchronously by background workers (2 replicas).
- **Admin dashboard** – React-Admin powered UI for managing accounts, API keys, and browsing events.
- **Rate limiting** – Redis-backed global and per-endpoint rate limiting (configurable).
- **API key management** – Generate, activate/deactivate, and delete API keys. Keys are stored as SHA256 hashes for security.
- **Role-based access control** – Admin and user roles with granular endpoint permissions.
- **Metrics & monitoring** – Prometheus metrics exposed at `/metrics` + Grafana dashboards.
- **Docker-first** – Full stack orchestrated with Docker Compose (API, worker, PostgreSQL, Redis, NATS, Caddy, Prometheus, Grafana).

---

## 🧱 Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.24 |
| **API Framework** | Gin |
| **Database** | PostgreSQL 18 |
| **Cache** | Redis 8 |
| **Message Broker** | NATS 2.10 (JetStream) |
| **Admin Panel** | React-Admin (TypeScript, Vite) |
| **Auth** | JWT + API Key (SHA256 hashed) |
| **Rate Limiting** | Redis-backed token bucket |
| **Reverse Proxy** | Caddy (self-signed TLS) |
| **Metrics** | Prometheus + Grafana |
| **Containerization** | Docker & Docker Compose |

---

## 📸 Screenshots

| Admin | Grafana |
|:---------:|:--------:|
| ![Admin](screenshots/admin.png) | ![Grafana](screenshots/grafana.png) |

---

## 🚀 Quick Start

### Prerequisites

- [Go](https://go.dev/) ≥ 1.24 (for local development)
- [Docker](https://docker.com) and Docker Compose (recommended)
- [Node.js](https://nodejs.org/) ≥ 20 (for admin panel builds)
- [golang-migrate](https://github.com/golang-migrate/migrate) (optional, for manual migrations)

### 1. Clone & configure

```bash
git clone https://github.com/neo-vai/go-events.git
cd go-events
cp .env.example .env
# Edit .env with your own secrets (JWT_SECRET, DB_PASSWORD, etc.)
```

### 2. Environment variables

Edit `.env` with your own values:

```
# Database
DATABASE_URL="postgres://user:pass@db:5432/eventtracker"
DATABASE_URL_LOCALHOST="postgres://user:pass@localhost:5432/eventtracker"

# Redis
REDIS_URL="redis:6379"
REDIS_PASSWORD="redispass"

# Auth
JWT_SECRET="change-this-to-a-long-secret"    # min 32 characters

# NATS
BROKER_URL="nats://nats:4222"

# Rate limiting
RATE_LIMIT_GLOBAL="100/1m"    # 100 requests per minute
RATE_LIMIT_LOGIN="5/1m"       # 5 login attempts per minute
```

### 3. Run with Docker

```bash
make up
```

This command will:
- Build the admin panel (if not already built)
- Create Docker images for the API and worker
- Start the full stack (API, worker, PostgreSQL, Redis, NATS, Caddy, Prometheus, Grafana)

### 4. Generate Swagger documentation (optional)

API endpoints are annotated with Swagger comments. To generate and serve interactive API documentation:

```bash
# Install the swag CLI (one-time)
make swagger-install

# Generate Swagger docs from code annotations
make swagger
```

Then access the Swagger UI at `https://localhost/swagger/index.html`.

### 5. Access the services

| Service          | URL                                          | Credentials          |
|------------------|----------------------------------------------|----------------------|
| **API**          | `https://localhost`                          | —                    |
| **Swagger UI**   | `https://localhost/swagger/index.html`       | —                    |
| **Admin UI**     | `https://localhost/admin`                    | admin account        |
| **Grafana**      | `http://localhost:3001`                      | `admin` / `admin`    |
| **Prometheus**   | `http://localhost:9090`                      | —                    |

### 6. Admin account

#### Default seed account

The application creates a default admin account automatically on first startup based on the `.env` configuration:

```
# .env
SEED_ADMIN_EMAIL=admin@example.com
SEED_ADMIN_PASSWORD=Admin123
```

If the account already exists (e.g., after restart), the startup continues normally.  
To disable auto-creation by removing these variables from your `.env` file.

#### Login to the Admin UI

1. Open **Swagger UI** at `https://localhost/swagger/index.html` or use `curl`:

   ```bash
   curl -k -X POST https://localhost/api/v1/login \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@example.com","password":"Admin123"}'
   ```

2. Use the returned JWT token to authenticate in the **Admin UI** at `https://localhost/admin`, or pass it as `Authorization: Bearer <token>` in subsequent API requests.

> 💡 **Tip:** All API requests below (register, login, create events, etc.) can also be executed directly from the **Swagger UI** — open `https://localhost/swagger/index.html` in your browser and use the "Try it out" button on any endpoint.

---

## 🛠️ Makefile Commands

A `Makefile` is provided with common development and deployment tasks.

| Command                  | Description                                    |
|--------------------------|------------------------------------------------|
| `make admin-build`       | Build admin panel static files                 |
| `make build`             | Build Docker images                            |
| `make up`                | Build admin (if needed) and start all services |
| `make down`              | Stop all services                              |
| `make logs`              | Tail logs from all containers                  |
| `make dev`               | Run API locally (without Docker)               |
| `make test`              | Run all tests with a fresh test database       |
| `make migrate-install`   | Install golang-migrate CLI                     |
| `make migrate-up`        | Apply database migrations                      |
| `make migrate-down`      | Rollback migrations                            |
| `make swagger-install`   | Install swag CLI (if not already installed)          |
| `make swagger`           | Generate Swagger documentation from code annotations |

---

## 📡 REST API

**Base URL:** `/api/v1`  
**Authentication:** JWT (`Authorization: Bearer <token>`) or API key (`X-API-Key: <key>`)

### Public Endpoints

| Method | Path         | Description                    |
|--------|--------------|--------------------------------|
| POST   | `/accounts`  | Create a new account           |
| POST   | `/login`     | Authenticate and receive JWT   |

### Protected Endpoints (JWT or API Key)

| Method | Path                     | Description                                  |
|--------|--------------------------|----------------------------------------------|
| GET    | `/account`               | Get current account details                  |
| PATCH  | `/account`               | Update current account email                 |
| DELETE | `/account`               | Delete current account                       |
| POST   | `/account/keys`          | Generate a new API key (full key once)       |
| GET    | `/account/keys`          | List API keys (prefix only)                  |
| PATCH  | `/account/keys/{key_id}` | Activate/deactivate an API key               |
| DELETE | `/account/keys/{key_id}` | Delete an API key                            |
| POST   | `/events`                | Create an event (async via NATS)             |
| GET    | `/events`                | List events with pagination & filters        |
| GET    | `/events/{id}`           | Get a single event                           |

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

### 📮 Request / Response Examples

**Create account:**

```bash
POST /api/v1/accounts
{
  "email": "user@example.com",
  "password": "StrongPass123"
}
```

**Login:**

```bash
POST /api/v1/login
{
  "email": "user@example.com",
  "password": "StrongPass123"
}
```

Response:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "accountId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Create event (with API key):**

```bash
POST /api/v1/events
X-API-Key: your-api-key-here
{
  "username": "john_doe",
  "name": "user_login",
  "payload": "{\"ip\":\"192.168.1.1\"}"
}
```

Response: `202 Accepted` with `Location` header pointing to the eventual event resource.

**List events with pagination:**

```bash
GET /api/v1/events?_start=0&_end=20&_sort=createdAt&_order=DESC&filter={"username":"john_doe"}
```

---

## 🖥️ Admin Panel

The admin panel is built with **React-Admin** (TypeScript, Vite) and served by Caddy under `/admin`. Only accounts with role `admin` can log in.

- **Dashboard** – Key metrics: total accounts, events today, active API keys.
- **Accounts** – List, filter, create, edit, delete accounts. Change roles and active status.
- **API Keys** – View all API keys, activate/deactivate, and delete.
- **Events** – Browse all events across accounts, view payload details.

---

## 📊 Monitoring

### Prometheus Metrics

The API exposes metrics at `/metrics`. Example queries:

- `gin_requests_total` – total HTTP requests
- `gin_request_duration_seconds` – request latency
- `go_goroutines` – number of goroutines

### Grafana Setup

1. Open `http://localhost:3001` in your browser.
2. Log in with default credentials: `admin` / `admin` (you will be prompted to change the password).
3. Add a data source:
   - Click **Connections** → **Data sources** → **Add data source**.
   - Choose **Prometheus**.
   - Set URL to `http://prometheus:9090` (Docker internal network).
   - Click **Save & test**.
4. Import a dashboard:
   - Click **Dashboards** → **Import**.
   - You can use the official Go process dashboard (ID `6671`) or create your own panels.
   - Select the Prometheus data source and import.

---

## 🧪 Testing

The project includes unit and integration tests. Tests run against dedicated PostgreSQL, Redis, and NATS containers.

```bash
make test
```

This command:
- Starts separate PostgreSQL, Redis, and NATS containers for testing
- Applies migrations to the test database
- Runs `go test -p 1 ./... -v`
- Cleans up containers after tests finish

---

## 📁 Project Structure

```
├── admin/                    # React-Admin frontend
│   ├── src/
│   │   ├── dashboard/        # Dashboard widgets
│   │   ├── layout/           # Custom layout & app bar
│   │   ├── login/            # Custom login page
│   │   └── resources/        # Resource configurations (accounts, apiKeys, events)
│   ├── package.json
│   └── vite.config.ts
├── cmd/
│   ├── api/main.go           # API server entrypoint
│   └── worker/               # NATS consumer worker entrypoint
├── internal/
│   ├── broker/               # NATS client & publisher/subscriber
│   ├── config/               # Configuration loading & validation
│   ├── handler/              # HTTP handlers (account, admin, apikey, auth, event)
│   ├── middleware/           # Auth, logging, CORS, rate limiting (Redis)
│   ├── model/                # Domain models (account, apikey, event)
│   ├── pagination/           # React-Admin pagination parser
│   ├── repository/           # PostgreSQL repositories + Redis cache
│   ├── service/              # Business logic layer
│   ├── testutil/             # Integration test helpers
│   └── validator/            # Custom validation rules
├── migrations/               # Database migration files
├── docker-compose.yml        # Main Docker Compose configuration
├── docker-compose.test.yml   # Test environment configuration
├── Dockerfile                # API server image
├── Dockerfile.worker         # Worker image
├── Caddyfile                 # Caddy reverse proxy config
├── prometheus.yml            # Prometheus scraping configuration
├── Makefile                  # Development & deployment commands
└── .env.example              # Environment variables template
```

---

## 🔑 Key Highlights

- **Security-first** – API keys stored as SHA256 hashes, JWT with configurable expiration, rate limiting on auth endpoints.
- **Async by default** – Events are published to NATS JetStream and processed by background workers, ensuring API responsiveness.
- **Redis-backed rate limiting** – Configurable global and per-endpoint limits using a token bucket algorithm.
- **React-Admin integration** – Pagination and filtering follow React-Admin conventions (`_start`, `_end`, `_sort`, `_order`, `filter`).
- **Full-text search** – Events support filtering by username, event name, and time range.
- **Docker-first deployment** – Single `make up` command starts the entire stack with all dependencies.
- **Observable** – Prometheus metrics + Grafana dashboards for production monitoring.
- **Extensible** – Clean architecture with handler → service → repository layers, easy to add new features.

---

## 📄 License

This project is licensed under the MIT License.

---

<p align="center">
  Made with 💙 by <a href="https://github.com/neo-vai">NeoVai</a>
</p>