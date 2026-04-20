# PostgreSQL integration: design, wiring, tests, and observability

This document explains how PostgreSQL was added to goapi, how the code is structured, how a request flows from HTTP to the database, which tests protect the behavior, and how the database shows up in **traces (Jaeger)**, **metrics (Prometheus + Grafana)**, and **logs (Loki / optional file logging)**.

---

## 1. Goals and high-level design

- **Pluggable storage**: the app can use **PostgreSQL** (production default) or an in-memory **mock** (fast local/CI unit tests) without changing handler code.
- **One interface**: all HTTP and middleware code depend on `database.Repository` only; the concrete driver is chosen at process start.
- **Schema as code**: SQL migrations live in the repo, are **embedded** in the binary, and are applied automatically when the process starts with the Postgres driver.
- **Resilient startup**: the app **waits** until Postgres accepts connections, then **migrates**, then opens a **pgx connection pool** with conservative pool settings and **per-query timeouts**.
- **Observable dependency**: DB calls contribute **OpenTelemetry spans** under the inbound HTTP trace; Postgres **runtime metrics** come from **postgres_exporter** scraped by Prometheus and charted in Grafana.

---

## 2. Architecture overview

```mermaid
flowchart LR
  subgraph process["goapi process"]
    Main["cmd/api main"]
    Repo["database.Repository"]
    Router["server.NewRouter"]
    Handlers["handlers + middleware"]
    OTelHTTP["observability.Middleware"]
    PGSpan["postgres spans\notel Tracer goapi/postgres"]
    Pool["pgxpool.Pool"]
    Main --> Repo
    Main --> Router
    Router --> OTelHTTP
    Router --> Handlers
    Handlers --> Repo
    Repo --> Pool
    Repo --> PGSpan
  end

  subgraph external["Infrastructure"]
    PG[(PostgreSQL)]
    Jaeger["Jaeger OTLP"]
    Prom["Prometheus"]
    Exp["postgres_exporter"]
    Graf["Grafana"]
  end

  Pool --> PG
  OTelHTTP --> Jaeger
  PGSpan --> Jaeger
  Prom --> Exp
  Prom --> process
  Graf --> Prom
  Exp --> PG
```

**Important separation**:

- **Application traces** (Jaeger): spans created **inside goapi** for HTTP (`goapi/http`) and SQL (`goapi/postgres`). These prove “which queries ran” for a given request.
- **Postgres server metrics** (Grafana): scraped from **postgres_exporter**, not from inside the Go process. They answer “how busy is Postgres,” not “which SQL text ran.”

---

## 3. Configuration: environment variables

| Variable | Role |
|----------|------|
| **`GOAPI_DB_DRIVER`** | `postgres` or `mock`. **If unset, it defaults to `postgres`** (`ResolveDriver()` in `internal/database/config.go`). Comparison is **case-insensitive** after trim. |
| **`DATABASE_URL`** | Preferred single DSN for Postgres (used as-is when non-empty). Example: `postgres://goapi:goapi@postgres:5432/goapi?sslmode=disable`. |
| **`POSTGRES_*`** | If `DATABASE_URL` is empty, `DSNFromEnv()` builds a URL from `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_SSLMODE` (with defaults for local dev). Passwords are **URL-encoded** when embedded in the DSN. |

Docker Compose sets `GOAPI_DB_DRIVER=postgres` and `DATABASE_URL` pointing at the **`postgres` service name** on the Docker network (`postgres:5432`), not `localhost`, so the API container resolves the DB by DNS.

**Nuances**:

- Defaulting to **postgres** when `GOAPI_DB_DRIVER` is empty avoids accidentally running production config against mock in a forgotten env.
- Unit/integration handler tests typically call `t.Setenv("GOAPI_DB_DRIVER", database.DriverMock)` so no real DB is required.

---

## 4. Code layout (`internal/database`)

| File | Responsibility |
|------|----------------|
| `types.go` | `Repository` interface, `LoginDetails`, `CoinDetails`, `ErrUserNotFound`. |
| `config.go` | `ResolveDriver()`, `DSNFromEnv()`. |
| `new.go` | `database.New(ctx)` → mock or `newPostgresRepo`. |
| `mock.go` | In-memory maps mirroring seeded users/coins for tests. |
| `postgres.go` | `postgresRepo`: pgx pool, OTel spans per query, timeouts, `Close()`. |
| `migrate.go` | `waitForPostgresSQL`, `migrateUp` using **embedded** SQL + **golang-migrate** + `database/sql` with **`pgx` stdlib driver**. |
| `migrations/*.sql` | Versioned `up`/`down` migrations; initial schema + seed data. |

Migrations are loaded with:

```go
//go:embed migrations/*.sql
var embeddedMigrations embed.FS
```

So the **Docker image does not need separate SQL files on disk**; migrations ship inside the binary (good for minimal images like distroless).

---

## 5. Schema and migrations

**`000001_initial.up.sql`**: creates `users` (username PK, auth_token) and `coin_balances` (username FK to users, balance ≥ 0).

**`000002_seed_demo_users.up.sql`**: inserts demo users **alex / kevin / max** with tokens and balances aligned with the former mock data.

**Flow on Postgres startup** (`newPostgresRepo`):

1. **`waitForPostgresSQL`** (up to **60s** parent timeout): poll every **200ms** with `sql.Open("pgx", dsn)` + `PingContext` (2s per attempt) until success or context cancelled. This avoids racing Testcontainers/Docker Compose where the TCP port is open before Postgres accepts connections.
2. **`migrateUp`**: runs `m.Up()`; treats **`migrate.ErrNoChange`** as success so restarts are idempotent.
3. **`pgxpool.ParseConfig` + `NewWithConfig` + `Ping`**: verify pool connectivity.

---

## 6. Repository behavior: Postgres vs mock

Both implement `Repository`:

- **`GetLoginDetails` / `GetCoinDetails`**: map `pgx.ErrNoRows` to **`ErrUserNotFound`** (handlers map that to **HTTP 400** in several paths—business “not found,” not an internal 500).
- **`UpdateCoinDetails`**: rejects negative balance before hitting SQL; unknown user → `ErrUserNotFound`.
- **`postgresRepo.Close()`**: closes the pool; **`main`** uses a small **type assertion** helper so only Postgres implements optional `Close()` without leaking that into the interface.

**Per-query resilience** (`postgres.go`):

- Each exported method wraps the context in **`context.WithTimeout(..., dbOpTimeout)`** (8s) so stuck queries do not hang forever.

---

## 7. Application wiring (dependency injection)

### 7.1 `cmd/api/main.go`

1. Configure logging (`ResolveLogLevel`, optional file output).
2. **`repo, err := database.New(ctx)`** — single shared instance for the whole process.
3. **`defer closeRepository(repo)`** — closes pgx pool on shutdown when using Postgres.
4. **`observability.InitTracer`** — OTLP exporter (Jaeger in Compose).
5. **`server.NewRouter(repo)`** — Chi router gets the same `Repository` everywhere.

No per-request `database.New()`; that keeps pool usage and tracing context correct.

### 7.2 `internal/server/router.go`

- Chi middleware: strip slashes, logger, recoverer, then **`observability.Middleware`** (HTTP tracing + Prometheus histogram/counters).
- **`handlers.NewHandler(r, repo)`** registers routes.
- **`/metrics`** is exposed for Prometheus (not traced as a normal API route where configured to skip).

### 7.3 `internal/handlers/handler.go`

- **`/account/*`** routes use **`middleware.Authorize(repo)`** — the middleware calls **`repo.GetLoginDetails`** to validate the token for the `username` query param.

So **auth and coins both flow through the same repository** passed from `main`.

---

## 8. End-to-end request flow (conceptual)

```mermaid
sequenceDiagram
  participant C as Client
  participant Chi as Chi + OTel HTTP
  participant MW as Authorize(repo)
  participant H as getCoinBalance(repo)
  participant R as Repository
  participant PG as PostgreSQL

  C->>Chi: HTTP GET /account/coins
  Chi->>Chi: HTTP server span (goapi/http)
  Chi->>MW: next
  MW->>R: GetLoginDetails(ctx, user)
  R->>PG: SELECT users ...
  PG-->>R: row / no row
  MW->>H: next
  H->>R: GetCoinDetails(ctx, user)
  R->>PG: SELECT coin_balances ...
  Note over R,PG: Client span (goapi/postgres) child of HTTP span
  PG-->>R: row
  H-->>C: JSON 200
```

**Context propagation**: handlers and middleware use **`r.Context()`**. That context carries the trace from **`observability.Middleware`**; **`postgresRepo`** starts child spans from that same context, so Jaeger shows **HTTP → SQL** as one trace.

---

## 9. OpenTelemetry: Postgres as a traced dependency

Implementation lives in **`internal/database/postgres.go`** (not a separate driver wrapper library):

- Tracer name: **`goapi/postgres`** (`otel.Tracer("goapi/postgres")`).
- Each query uses **`SpanKindClient`** so the span represents an **outbound** client call.
- Semantic attributes include:
  - **`db.system`** = postgresql (`semconv.DBSystemPostgreSQL`)
  - **`db.statement`** = parameterized SQL text (no user values in the string—still treat as sensitive in policies)
  - **`peer.service`** = `postgresql` (helps dependency graphs)
  - **`server.address` / `server.port`** — derived from the pool DSN host/port (defaults `localhost` / `5432` if missing)

**`recordSpanErr`**: does **not** mark the span as failed for **`ErrUserNotFound`** or **`pgx.ErrNoRows`** (expected business/empty paths); other errors are recorded on the span.

**HTTP layer** (`internal/observability/telemetry.go`): records route pattern, method, status; marks span error for **5xx** responses. **Metrics** (`goapi_http_requests_total`, `goapi_http_request_duration_seconds`) are independent of the DB but correlate in dashboards by time.

---

## 10. Metrics: app vs Postgres exporter

### 10.1 Inside the Go process

- Prometheus scrapes **`goapi:8080/metrics`** (`job="goapi"`).
- HTTP metrics are labeled by **route pattern**, **method**, **status** (from the status recorder).

### 10.2 Postgres as infrastructure

- **`postgres_exporter`** connects to Postgres and exposes **`pg_*`** metrics.
- Prometheus scrapes **`postgres_exporter:9187`** with **`job="postgres"`** (see `deploy/prometheus/prometheus.yml`).
- Grafana dashboards (e.g. **GoAPI Performance Overview**) query metrics like **`pg_stat_database_*`** with **`job="postgres"`** and **`datname="goapi"`**.

**Nuance**: Grafana panels for Postgres **do not** use application trace data; they use **exporter** metrics. If Prometheus has not picked up the `postgres` scrape job (e.g. config not reloaded), those panels show **no data** until Prometheus is restarted or **`/-/reload`** is called (Compose enables **`--web.enable-lifecycle`**).

---

## 11. Docker Compose wiring (relevant parts)

- **`postgres`**: Postgres 16, volume, **healthcheck** (`pg_isready`).
- **`goapi`**: `depends_on` **postgres** with **`condition: service_healthy`**; env **`DATABASE_URL`** uses hostname **`postgres`**.
- **`postgres_exporter`**: reads **`DATA_SOURCE_NAME`** pointing at the same DB; Prometheus **`depends_on`** exporter for startup ordering.
- Host port **`5433:5432`** avoids clashing with a Postgres already bound to **5432** on the machine.

---

## 12. Testing strategy

Tests are layered so you can run fast checks without Docker and full checks with Docker.

### 12.1 Unit tests (default `go test ./...`)

- **`GOAPI_DB_DRIVER=mock`** via test helpers — **no Postgres**.
- Cover **`database`**: config, `New()`, mock repository behavior, small pure helpers (`recordSpanErr`, cancelled `waitForPostgresSQL`).
- Cover **handlers**, **middleware**, **server**, **api** error helpers, **observability** helpers.

### 12.2 Integration tests (`-tags=integration`)

- Build tag on tests such as **`handlers_integration_test.go`** that spin up the **mock** repo and real HTTP server paths.
- Exercise full handler + Chi + authorization behavior without a database container.

### 12.3 Postgres integration (`-tags=testcontainers`)

- **`internal/database/postgres_container_test.go`** (`//go:build testcontainers`): starts a **real Postgres** via Testcontainers, sets **`DATABASE_URL`**, calls **`database.New`**, runs through login/coins/update, unknown users, negative balance, **`Setup`**, **`Close`**.
- Requires **Docker** on the machine (and in CI if you run these in GitHub Actions).

### 12.4 Coverage gate (`Makefile`)

- **`make coverage`** / **`make coverage-check`**: runs **`integration` + `testcontainers`** against **`./api ./internal/...`** ( **`cmd/api` omitted** due to a Go coverage toolchain quirk when covering `./...` ).
- Enforces **minimum 85%** total statement coverage as configured in the Makefile/CI.

---

## 13. CI alignment

The GitHub Actions workflow runs:

1. **`go test ./...`** — quick, no Docker DB.
2. **`go test -tags='integration testcontainers' ... ./api ./internal/...`** with **coverage** — full stack including Postgres-in-container tests where Docker is available.

---

## 14. Operational checklist

| Task | Suggestion |
|------|------------|
| Run API locally against Docker Postgres | `docker compose up`, `DATABASE_URL`/`GOAPI_DB_DRIVER` as in Compose or override for host port **5433**. |
| Run without Postgres | `GOAPI_DB_DRIVER=mock` (handlers/tests). |
| Apply schema changes | Add **`00000n_*.up.sql`** / **`down.sql`** under `internal/database/migrations/`; redeploy — migrate runs on startup. |
| Reload Prometheus after editing `prometheus.yml` | **`curl -X POST localhost:9090/-/reload`** (with lifecycle enabled) or recreate the container. |
| View traces | Jaeger UI → service **`goapi`** → find trace → expand **`goapi/postgres`** child spans. |
| View DB metrics | Grafana → Prometheus datasource → panels filtered by **`job="postgres"`**. |

---

## 15. Summary

PostgreSQL was integrated by introducing a **`database.Repository`** abstraction, a **Postgres implementation** using **pgxpool** + **embedded golang-migrate migrations**, and **environment-driven selection** between Postgres and mock. **`main`** constructs the repository **once** and passes it into **`server.NewRouter`**, which flows into **handlers** and **authorization middleware**. **OpenTelemetry** spans for SQL use **client** semantics and **semantic conventions** so Jaeger shows Postgres as a dependency under each HTTP request; **Prometheus/Grafana** monitor Postgres separately via **postgres_exporter**. **Tests** combine **mock-based** speed with **Testcontainers-based** fidelity, and coverage is aggregated with **`integration` + `testcontainers`** tags for a realistic total.
