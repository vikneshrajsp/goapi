# goapi

A Go API built with Chi that provides coin balance endpoints, plus production-oriented observability with Prometheus, Grafana, Jaeger, Loki, and Promtail.

## Features

- REST API for coin balance retrieval and updates
- Authentication middleware (username + auth token)
- Health endpoints (`/ping`, `/health`)
- Prometheus metrics endpoint (`/metrics`)
- OpenTelemetry tracing exported to Jaeger
- Structured JSON logging with Logrus
- Dockerized app with hardened runtime image
- Local observability stack via Docker Compose

## Prerequisites

- Go `1.25.x`
- Docker + Docker Compose

## Run Locally (without Docker)

```bash
make run
```

Server runs on: `http://localhost:8080`

## Run with Full Dependencies (Docker Compose)

```bash
docker compose up -d --build
```

## Service URLs

- API base URL: `http://localhost:8080`
- Ping: `http://localhost:8080/ping`
- Health: `http://localhost:8080/health`
- Metrics: `http://localhost:8080/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (user: `admin`, pass: `admin`)
- Jaeger: `http://localhost:16686`
- Loki (API): `http://localhost:3100`
- Grafana dashboards (folder): `goapi`

## API Endpoints

- `GET /account/coins?username=<user>`
- `PUT /account/coins?username=<user>`
- `GET /ping`
- `GET /health`
- `GET /metrics`

### Example Requests

Get coin balance:

```bash
curl -s -H "Authorization: 123AL100" \
  "http://localhost:8080/account/coins?username=alex"
```

Update coin balance:

```bash
curl -s -X PUT \
  -H "Authorization: 123AL100" \
  -H "Content-Type: application/json" \
  -d '{"balance":150}' \
  "http://localhost:8080/account/coins?username=alex"
```

## OpenAPI Documentation

- OpenAPI spec file: `openapi/openapi.yaml`

You can view it with Swagger Editor:
- [https://editor.swagger.io/](https://editor.swagger.io/) (paste the YAML content)

## Testing

Run all tests (unit + integration):

```bash
make test
```

Run only unit tests:

```bash
make test-unit
```

Run only integration tests:

```bash
make test-integration
```

Generate local coverage report:

```bash
go test -tags=integration -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## CI Gate

GitHub Actions workflow enforces:

- Tests must pass (unit + integration)
- Total code coverage must be `>= 80%`

Workflow file: `.github/workflows/ci.yml`

## Grafana Dashboards

Provisioned dashboards:

- `GoAPI Performance Overview`
- `GoAPI GC Deep Dive`
- `GoAPI SLO / Error Budget`

## Prometheus alerting (SLO multi-burn)

Rules live in `deploy/prometheus/rules/goapi_slo_alerts.yml` and load via `deploy/prometheus/prometheus.yml`.

- **Recording series**: `goapi:slo_error_ratio:1h`, `goapi:slo_burn_rate:1h`, etc.
- **Multi-window alert**: `GoAPI_ErrorBudget_MultiWindowFastBurn` fires when **both** `burn_rate_1h > 14.4` and `burn_rate_6h > 6` (99.9% availability target; aligns with ~2%/1h and ~5%/6h budget slices over a ~30-day window—see comments in the rules file).

View firing alerts in Prometheus: `http://localhost:9090/alerts`

## Learning docs (Prometheus, Loki, GC, profiling)

- `.docs/prometheus-querying-scenarios.md`
- `.docs/loki-querying-scenarios.md`
- `.docs/golang-gc-metrics.md`
- `.docs/golang-memory-profiling.md`

## Tracing Behavior

- Request traces are exported to Jaeger.
- `/metrics` is intentionally excluded from tracing noise.

## Sample Query Cookbook

### Prometheus queries

- Request rate by route:
  - `sum by (route) (rate(goapi_http_requests_total[5m]))`
- 5xx error ratio:
  - `sum(rate(goapi_http_requests_total{status=~"5.."}[5m])) / clamp_min(sum(rate(goapi_http_requests_total[5m])), 1)`
- P95 latency by route:
  - `histogram_quantile(0.95, sum by (le, route) (rate(goapi_http_request_duration_seconds_bucket[5m])))`
- Heap in use bytes:
  - `go_memstats_heap_inuse_bytes`
- GC cycles per second:
  - `rate(go_gc_cycles_automatic_gc_cycles_total[5m])`

### Loki queries (Grafana Explore)

- All app logs:
  - `{job="goapi"}`
- Error logs only:
  - `{job="goapi"} |= "\"level\":\"error\""`
- Logs for coin endpoints:
  - `{job="goapi"} |= "/account/coins"`
- Slow requests hints:
  - `{job="goapi"} |= "\"GET http://"`
