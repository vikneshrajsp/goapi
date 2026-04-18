# Loki querying (LogQL practice scenarios)

Loki indexes **labels**, not full text. Your Promtail pipeline labels Docker logs with `job="goapi"` when scraping the `goapi` container.

Open **Grafana → Explore → Loki**.

## Mental model

- **Label selectors** narrow which streams to scan: `{job="goapi"}`.
- **Line filters** search inside log lines: `|= "error"`, `!= "health"`.
- **JSON logs**: your app uses Logrus JSON; filter on substrings or parse JSON with `| json` when fields are stable.

## Scenario 1: everything from the API container

```logql
{job="goapi"}
```

Interpretation: full firehose; good for confirming ingestion works.

## Scenario 2: only lines mentioning errors

```logql
{job="goapi"} |= "error"
```

Interpretation: quick triage; tune string to match your JSON (`"level":"error"`).

## Scenario 3: exclude noisy health checks

```logql
{job="goapi"} != "/health" != "/ping"
```

Interpretation: focus on business routes.

## Scenario 4: isolate coin endpoints

```logql
{job="goapi"} |= "/account/coins"
```

Interpretation: correlate log lines with coin balance traffic.

## Scenario 5: combine label + substring + limit

```logql
{job="goapi"} |= "panic" | line_format "{{ __line__ }}"
```

Interpretation: chase crashes; pair with Grafana time range zoom.

## Scenario 6: approximate rate of error logs

```logql
sum(count_over_time({job="goapi"} |= `error` [1m]))
```

Interpretation: rough error-line frequency per minute (not identical to HTTP 5xx unless logs align).

## Practical tips

- Narrow time ranges first; full scans over long ranges get slow.
- Prefer consistent labels from Promtail (`job`, `container`) before heavy regex.
- Correlate with traces: pick `trace_id` from logs if you add trace IDs to structured logs later.
