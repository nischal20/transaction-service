# Transactions Service

A transactions service for managing cardholder accounts and financial operations, built in Go.

## Requirements

- Go 1.23+
- PostgreSQL 16+ (or Docker & Docker Compose)

---

## Quick Start

### Option A — Docker (recommended, zero setup)

```bash
./run.sh docker
```

Starts the API and a Postgres container. No other setup needed.

- API: **http://localhost:8080**
- Swagger UI: **http://localhost:9001/swagger/index.html**

### Option B — Local (requires a running Postgres instance)

```bash
# Copy env template and fill in your Postgres credentials
cp .env.example .env

# Apply database migrations
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=$DB_SSLMODE" \
  -f migrations/001_init.sql \
  -f migrations/002_audit_logs.sql

./run.sh
```

---

## Router

The project uses [chi](https://github.com/go-chi/chi) as its HTTP router. chi is 100% compatible with the Go standard library — every handler is a plain `http.HandlerFunc` and every middleware is a plain `func(http.Handler) http.Handler`, so it is easy to use and can be swapped out without touching any handler or middleware code.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API listen port |
| `DB_HOST` | *(required)* | PostgreSQL host |
| `DB_PORT` | *(required)* | PostgreSQL port |
| `DB_USER` | *(required)* | PostgreSQL user |
| `DB_PASSWORD` | *(required)* | PostgreSQL password |
| `DB_NAME` | *(required)* | PostgreSQL database name |
| `DB_SSLMODE` | *(required)* | PostgreSQL SSL mode (`disable` for local dev) |
| `SWAGGER_PORT` | `9001` | Swagger UI port — set `""` to disable in production |

Copy `.env.example` to `.env` and fill in your values — `run.sh` loads it automatically on `./run.sh local`.

---

## API

Full API documentation is available via Swagger UI once the service is running:

**http://localhost:9001/swagger/index.html**

Every response includes an `X-Request-ID` header (UUID) for tracing requests across logs.

### Amount handling

Amounts are sent and received in **rupees** (decimal) but stored internally in **paise** (integer minor units, 1 rupee = 100 paise). The conversion happens at the HTTP boundary — all internal layers (service, repository, database) only deal with paise.

| Layer | Type | Example |
|-------|------|---------|
| API request / response | `float64` rupees | `123.45` |
| Service, repository, DB (`BIGINT`) | `int64` paise | `12345` |

The API always accepts a **positive amount**. The service applies the correct sign before storing based on operation type:

- **Purchase / Withdrawal** (op types 1, 2, 3) — stored as **negative** (e.g. `₹50.00` → `-5000` paise)
- **Credit Voucher** (op type 4) — stored as **positive** (e.g. `₹60.00` → `6000` paise)

This follows the spec: *"Transactions of type purchase and withdrawal are registered with negative amounts, while transactions of credit voucher are registered with positive value."*

---

### Idempotency

Every `POST /transactions` request **must** include an `X-Idempotency-Key` header — a client-generated unique string (e.g. an order ID or UUID) that identifies the intent of the request.

```
X-Idempotency-Key: order-abc-123
```

| Scenario | Behaviour |
|----------|-----------|
| First request with a key | Transaction is created, `201 Created` returned |
| Same key + same body | Cached response returned byte-for-byte, no duplicate created |
| Same key + **different** body | `422 Unprocessable Entity` — conflict detected |
| Header missing | `400 Bad Request` |

#### How it works — dedicated `idempotency_keys` table

Idempotency is stored in its own table, completely separate from `transactions`:

```sql
CREATE TABLE idempotency_keys (
    key            VARCHAR(255) PRIMARY KEY,
    request_hash   VARCHAR(64)  NOT NULL,   -- SHA-256 of the raw request body
    response_code  INT          NOT NULL,
    response_body  JSONB        NOT NULL,   -- the exact HTTP response that was returned
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

Advantages over a `UNIQUE` column on `transactions`:

- **Full HTTP response cached** — replays return the exact same bytes, not a re-serialised model
- **Conflict detection** — `request_hash` catches reuse of the same key with a different body
- **Separation of concerns** — `transactions` stays pure business data; idempotency is infrastructure
- **Independent TTL/cleanup** — expired keys can be purged without touching business rows
- **Works for any endpoint** — not coupled to the transaction domain

Handler flow on every `POST /transactions`:
1. Read `X-Idempotency-Key` — reject if missing
2. Hash the raw request body (SHA-256)
3. Lookup `idempotency_keys` — if found and hash matches, return the cached response
4. If found but hash differs — return `422`
5. If not found — process the transaction, save `{key, hash, 201, response_body}`, return response

**Example:**

```bash
# First call — creates the transaction
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: order-abc-123" \
  -d '{"account_id": 1, "operation_type_id": 4, "amount": 112.34}'

# Retry with the same key — returns the identical cached response, no new transaction
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: order-abc-123" \
  -d '{"account_id": 1, "operation_type_id": 4, "amount": 112.34}'
```

---

## Audit Logging

Every state-changing operation writes an entry to the `audit_logs` table (created by `migrations/002_audit_logs.sql`):

| Event | Resource | Trigger |
|-------|----------|---------|
| `account.created` | `account` | `POST /accounts` |
| `transaction.created` | `transaction` | `POST /transactions` |

Each entry captures the event type, affected resource, resource ID, and the `X-Request-ID` for cross-log tracing.

The business row insert and its audit log entry are written inside the **same `*sql.Tx`**. If either write fails the whole transaction rolls back — the audit trail is always consistent with the business tables.

---

## Project Structure

```
.
├── cmd/api/              # Entry point — wires config, repos, services, handlers
├── config/               # Config loaded from environment variables
├── database/             # PostgresConnector implementation
├── migrations/           # SQL migrations (001_init, 002_audit_logs)
├── docs/                 # Swagger spec (generated by swaggo/swag — do not edit)
├── internal/
│   ├── apperr/           # Typed error sentinels (ErrValidation, ErrNotFound)
│   ├── dto/              # Request / response shapes for HTTP boundary
│   ├── handler/          # Chi router, middleware, HTTP handlers
│   │   ├── account/      # POST /accounts, GET /accounts/{accountId}
│   │   └── transaction/  # POST /transactions
│   ├── idempotency/      # Idempotency interface + PostgreSQL implementation
│   ├── model/            # Domain structs and business constants
│   ├── repository/
│   │   ├── account/              # AccountRepository interface
│   │   ├── transaction/          # TransactionRepository interface
│   │   ├── postgres/account/     # PostgreSQL AccountRepository
│   │   └── postgres/transaction/ # PostgreSQL TransactionRepository
│   ├── service/
│   │   ├── account/      # Account business logic + AccountServicer interface
│   │   └── transaction/  # Transaction business logic + TransactionServicer interface
│   └── utils/            # Shared helpers (request ID context, JSON response writers)
├── .env.example          # Config template — copy to .env and fill in values
├── Dockerfile
├── docker-compose.yml
└── run.sh                # One-command launcher
```
