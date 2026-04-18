# Prometheus querying (practice scenarios)

Prometheus exposes a time-series database plus **PromQL**. Your app scrapes `/metrics`; custom HTTP metrics include `goapi_http_requests_*`. Go runtime metrics begin with `go_*`.

## Mental model

- **`rate()` / `irate()`**: convert counters into per-second rates. Use `rate()` over `irate()` unless you investigate spikes.
- **`histogram_quantile()`**: compute approximate percentiles from Prometheus histogram `_bucket` metrics.
- **`sum by (...)`**: aggregate while keeping grouping labels.

## Scenario 1: requests per second (overall)

Use your counter totals:

```promql
sum(rate(goapi_http_requests_total[5m]))
```

Interpretation: total HTTP throughput across all routes/methods/statuses.

## Scenario 2: requests per second by HTTP route label

Your middleware labels include `route`, `method`, `status`:

```promql
sum by (route) (rate(goapi_http_requests_total[5m]))
```

Interpretation: which Chi route patterns dominate traffic.

## Scenario 3: server error rate (ratio)

Treat `5xx` statuses as failures:

```promql
sum(rate(goapi_http_requests_total{status=~"5.."}[5m]))
/
clamp_min(sum(rate(goapi_http_requests_total[5m])), 1)
```

Interpretation: fraction of traffic returning server errors.

## Scenario 4: approximate P95 latency across all routes

Histogram:

```promql
histogram_quantile(
  0.95,
  sum by (le) (rate(goapi_http_request_duration_seconds_bucket[5m]))
)
```

Interpretation: “95% of observed request durations fall below this seconds value,” based on scraped buckets.

## Scenario 5: alert-style burn rate intuition

Your recording rules expose:

- `goapi:slo_error_ratio:1h`
- `goapi:slo_burn_rate:1h`

Compare thresholds used in alerting by graphing burn rate:

```promql
goapi:slo_burn_rate:1h
```

Interpretation: burn rate expresses “how fast you consume error budget versus a 99.9% allowance.” Values above `14.4` for 1h correlate with aggressive budget consumption (see deployed rule comments).

## Scenario 6: validate Prometheus self-scrape

Check Prometheus health targets indirectly:

```promql
up{job="prometheus"}
```

Interpretation: `1` means last scrape succeeded for that target.

## Tips

- Prefer consistent range windows (`5m`, `1h`) when comparing panels.
- Use `clamp_min(..., 1e-9)` when dividing by sums that might be zero in dev.
