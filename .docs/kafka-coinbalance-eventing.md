# Kafka coin balance eventing (beginner guide)

This document explains how coin-balance change eventing works in this codebase after adding Kafka.

It is written as a practical walkthrough:

1. what changed,
2. how data flows,
3. how to run and validate locally,
4. how to test and troubleshoot.

---

## 1) What problem this solves

Before this change, updating `/account/coins` only updated PostgreSQL and returned an HTTP response.

Now, every successful balance update also emits a durable domain event:

- event type: `coinbalance_change`
- transport: Kafka topic `coinbalance_change`
- consumers:
  - `coinbalance-notifier` worker (logs and webhook notification),
  - `coinbalance-metrics` worker (exports Prometheus metrics).

This decouples write-path API behavior from async downstream processing.

---

## 2) High-level architecture

```mermaid
flowchart LR
  user[Client] --> api[goapi API]
  api --> db[(PostgreSQL)]
  api --> topic[(Kafka topic coinbalance_change)]

  notifier[coinbalance-notifier group] --> topic
  notifier --> db
  notifier --> webhook[Per-user Webhook URL]

  metrics[coinbalance-metrics group] --> topic
  metrics --> promMetrics[/metrics]
  prom[Prometheus] --> promMetrics
  grafana[Grafana] --> prom
```

Important Kafka behavior:

- Both consumers read **the same topic**,
- but they use **different consumer groups**,
- so each group gets all messages independently.

---

## 3) Core code changes

## 3.1 Database schema and repository

### New migration

- `internal/database/migrations/000003_user_webhooks.up.sql`
- `internal/database/migrations/000003_user_webhooks.down.sql`

Adds table:

- `user_webhooks(username PK -> users.username, webhook_url, created_at, updated_at)`

### Repository interface extensions

`internal/database/types.go` adds:

- `SetUserWebhookURL(ctx, username, url) error`
- `GetUserWebhookURL(ctx, username) (string, error)`

Implemented in:

- `internal/database/postgres.go` (real SQL),
- `internal/database/mock.go` (in-memory map for tests).

---

## 3.2 Producer in API path

Kafka producer implementation:

- `internal/messaging/kafka/producer.go`

Key production-minded defaults:

- required acks: all,
- retries and max attempts,
- bounded read/write timeouts,
- explicit topic, explicit broker list.

Update path integration:

- `internal/handlers/update_coin_balance.go`

Flow in handler:

1. read previous balance,
2. apply DB update,
3. build `CoinBalanceChanged` event (`event_id`, timestamp, delta),
4. publish with producer.

If publish fails, handler returns internal error (write+event consistency at request boundary).

---

## 3.3 New authenticated webhook endpoint

Route:

- `PUT /account/webhook?username=<user>`

Body:

```json
{ "webhook_url": "https://example.com/hook" }
```

Wiring:

- `internal/handlers/set_webhook.go` (URL validation + persistence),
- `internal/handlers/handler.go` route registration under existing `Authorize(repo)` middleware.

No environment variable is used for webhook URL; it is user-managed at runtime and stored in PostgreSQL.

---

## 3.4 Two asynchronous worker services

### 1) Webhook notifier worker

- entrypoint: `cmd/coinbalance-notifier/main.go`
- logic: `internal/workers/notifier/worker.go`
- consumer group: `coinbalance-notifier`

Behavior:

1. consume event,
2. load user webhook URL from Postgres,
3. POST event JSON to webhook,
4. log timestamp + username + previous/current/delta.

### 2) Metrics worker

- entrypoint: `cmd/coinbalance-metrics/main.go`
- logic: `internal/workers/eventmetrics/worker.go`
- consumer group: `coinbalance-metrics`

Behavior:

1. consume event,
2. update dedicated Prometheus metrics:
   - `goapi_coinbalance_events_total{username=...}`
   - `goapi_coinbalance_delta` histogram
   - `goapi_coinbalance_current{username=...}`

---

## 3.5 Health checks and readiness

### API health includes Kafka dependency

- `internal/handlers/health_handler.go`

`/health` now checks Kafka producer connectivity; if Kafka is down, API health returns `503`.

### Worker health endpoints

- notifier health on port `8091`,
- metrics health on port `8092`.

Each worker checks Kafka connectivity as part of readiness.

---

## 4) Docker compose wiring

`docker-compose.yml` now includes:

- Confluent Kafka broker (KRaft mode),
- one-time `kafka-init` service to create topic,
- `coinbalance-notifier` service,
- `coinbalance-metrics` service,
- existing `goapi` + observability stack.

Prometheus scrape config (`deploy/prometheus/prometheus.yml`) includes:

- `job: coinbalance-metrics` -> `coinbalance-metrics:8092/metrics`

Grafana dashboard (`deploy/grafana/dashboards/goapi-performance.json`) adds panels for:

- event throughput (`goapi_coinbalance_events_total`),
- event delta distribution (`goapi_coinbalance_delta`).

---

## 5) Event schema

Defined in:

- `internal/messaging/events/coinbalance_change.go`

Fields:

- `schema_version`
- `event_id`
- `event_type`
- `username`
- `previous_balance`
- `current_balance`
- `delta`
- `occurred_at`

This schema is stable, explicit, and easy to evolve with versioning.

---

## 6) How events flow (step-by-step)

1. Client calls:
   - `PUT /account/coins?username=alex`
2. API validates auth and input.
3. API updates `coin_balances` in Postgres.
4. API publishes `coinbalance_change` event to Kafka.
5. Kafka persists event.
6. Notifier group consumes the event:
   - fetches per-user webhook URL from Postgres,
   - calls webhook,
   - logs event fields and timestamp.
7. Metrics group consumes the same event:
   - updates Prometheus counters/histograms/gauges.
8. Prometheus scrapes metrics worker.
9. Grafana visualizes the eventing metrics.

---

## 7) Local run / smoke test

Bring up stack:

```bash
docker compose up -d --build
```

Register webhook for a user:

```bash
curl -X PUT 'http://localhost:8080/account/webhook?username=alex' \
  -H 'Authorization: 123AL100' \
  -H 'Content-Type: application/json' \
  -d '{"webhook_url":"https://example.com/webhook"}'
```

Trigger balance change:

```bash
curl -X PUT 'http://localhost:8080/account/coins?username=alex' \
  -H 'Authorization: 123AL100' \
  -H 'Content-Type: application/json' \
  -d '{"balance":322}'
```

Check metrics worker output:

```bash
curl http://localhost:8092/metrics | grep goapi_coinbalance_events_total
```

Check Prometheus query:

```bash
curl -G 'http://localhost:9090/api/v1/query' \
  --data-urlencode 'query=sum(goapi_coinbalance_events_total{job="coinbalance-metrics"})'
```

### Distributed traces (Jaeger)

The API injects W3C trace context (`traceparent`, etc.) into Kafka message headers when publishing. Each worker initializes OTLP export like the API, extracts that context when consuming, and records **consumer** spans. In Jaeger you should see **one trace** for a balance update that includes:

- `PUT /account/coins` (service **goapi**), Postgres query spans, and **`coinbalance_change publish`** (producer),
- **`coinbalance_change receive`** spans from **coinbalance-notifier** and **coinbalance-metrics** (two consumer groups → two sibling consume spans for the same message).

Open [http://localhost:16686](http://localhost:16686), pick service **goapi**, search recent traces, then open a trace after triggering `PUT /account/coins`. Filter by trace ID in logs if you correlate with Loki. If worker spans are missing, confirm workers have `OTEL_EXPORTER_OTLP_ENDPOINT` pointing at Jaeger (see `docker-compose.yml`) and that notifier/metrics containers were restarted after changes.

---

## 8) Test coverage added for this feature

### Unit tests

- webhook handler validation + persistence path:
  - `internal/handlers/set_webhook_test.go`
- update handler publishes event:
  - `internal/handlers/update_coin_balance_test.go`
- notifier worker behavior:
  - success, invalid payload, missing webhook, non-2xx webhook:
  - `internal/workers/notifier/worker_test.go`
- metrics worker behavior:
  - consume success, invalid payload, health success/failure:
  - `internal/workers/eventmetrics/worker_test.go`
- Kafka producer/consumer helper tests:
  - `internal/messaging/kafka/producer_consumer_test.go`

### Testcontainers integration

- Kafka producer->consumer end-to-end:
  - `internal/messaging/kafka/kafka_container_test.go`
- Existing Postgres integration tests remain in:
  - `internal/database/postgres_container_test.go`

### Combined coverage gate

`make coverage-check` runs with tags:

- `integration testcontainers`

and enforces minimum coverage threshold.

---

## 9) Operational notes and nuances

- Kafka topic creation is bootstrapped by `kafka-init`. If this service fails, API/workers wait via compose dependency.
- API `/health` now reflects Kafka availability; this is intentional for production readiness signaling.
- Notifier currently logs and returns error on webhook non-2xx; for stricter delivery guarantees you can add retry/DLQ policy later.
- Consumer groups are intentionally separate to avoid coupling metrics and side-effect notifications.
- Webhook URL ownership is per-user and persisted in Postgres, so restarts do not lose registration.

---

## 10) Beginner glossary

- **Topic**: append-only stream where events are written.
- **Producer**: app component that writes events to a topic.
- **Consumer group**: independent subscription identity; each group sees all topic events.
- **Offset**: position in topic consumed by a group.
- **At-least-once delivery**: consumer may process duplicates in some failure scenarios; design handlers to be idempotent where possible.
- **KRaft**: Kafka mode without ZooKeeper.

