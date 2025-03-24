# REST API Design

## Base Path
- `/api/v1`

## Endpoints

### Accounts
- `POST /api/v1/accounts` → create new account
- `GET /api/v1/accounts/:id` → fetch account info
- `PUT /api/v1/accounts/:id` → update account
- `DELETE /api/v1/accounts/:id` → delete account

### API Keys
- `POST /api/v1/accounts/:account_id/keys` → generate key
- `GET /api/v1/accounts/:account_id/keys` → list keys
- `PATCH /api/v1/accounts/:account_id/keys/:key_id` → activate/deactivate
- `DELETE /api/v1/accounts/:account_id/keys/:key_id` → delete key

### Events
- `POST /api/v1/events` → create new event
- `GET /api/v1/events` → list events with query filters `account_id`, `user`, `api_key_id`
- `GET /api/v1/events/:id` → fetch specific event

## Swagger
- `/swagger/index.html`