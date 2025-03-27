# Event Tracking API

This service provides a REST API for tracking events. It allows you to create accounts, manage API keys, and send events with metadata. Events can be queried using various filters.

## Authentication

The API supports two authentication methods:

- **JWT** – obtained via the login endpoint using account credentials. Sent in the `Authorization: Bearer <token>` header.
- **API Key** – sent in the `X-API-Key` header. The key must be active and belong to the account.

Protected endpoints automatically accept either method. The authentication middleware tries JWT first, then falls back to API key validation.

## API Endpoints

All endpoints are prefixed with `/api/v1`. Admin endpoints are under `/api/v1/admin`.

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/accounts` | Create a new account. |
| POST | `/login` | Authenticate and receive a JWT token. |

### Protected Endpoints (require authentication)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/accounts/:id` | Get account details (owner only). |
| PUT | `/accounts/:id` | Update account (owner only). |
| DELETE | `/accounts/:id` | Delete account (owner only). |
| POST | `/accounts/:id/keys` | Generate a new API key for the account. The full key is returned **only once**. |
| GET | `/accounts/:id/keys` | List all API keys for the account (only key prefixes are shown). |
| PATCH | `/accounts/:id/keys/:key_id` | Activate or deactivate an API key. |
| DELETE | `/accounts/:id/keys/:key_id` | Delete an API key. |
| POST | `/events` | Create a new event. If authenticated with an API key, the key ID is automatically recorded. |
| GET | `/events` | List events belonging to the authenticated account with optional pagination and filters. |
| GET | `/events/:id` | Get a single event by ID (must belong to the authenticated account). |

### Admin Endpoints (require admin role)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/accounts` | List all accounts (paginated, supports filtering). |
| GET | `/admin/accounts/:id` | Get account details. |
| POST | `/admin/accounts` | Create a new account (admin can set role). |
| PUT | `/admin/accounts/:id` | Update account (including role and active status). |
| DELETE | `/admin/accounts/:id` | Delete account. |
| GET | `/admin/events` | List all events across all accounts. |
| GET | `/admin/events/:id` | Get a single event. |
| GET | `/admin/api-keys` | List all API keys. |
| PUT | `/admin/api-keys/:id` | Update API key active status. |
| DELETE | `/admin/api-keys/:id` | Delete API key. |
| GET | `/admin/stats` | Get system statistics. |
