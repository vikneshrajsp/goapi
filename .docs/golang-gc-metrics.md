# Reading Go GC metrics (from Prometheus `/metrics`)

Go exposes runtime metrics via `prometheus/client_golang`, commonly prefixed `go_*` and `process_*`.

## Core GC signals

### `go_gc_duration_seconds`

Histogram of GC stop-the-world pause times.

Useful PromQL:

```promql
rate(go_gc_duration_seconds_sum[5m])
/
rate(go_gc_duration_seconds_count[5m])
```

Interpretation: average pause duration over the window (not the same as total GC cost).

### `go_gc_cycles_automatic_gc_cycles_total`

Counter of completed GC cycles.

```promql
rate(go_gc_cycles_automatic_gc_cycles_total[5m])
```

Interpretation: GC cycles per second; rises when heap churn increases.

### `go_memstats_next_gc_bytes`

Target heap size that triggers the next GC cycle (GOGC-driven).

```promql
go_memstats_next_gc_bytes
```

Interpretation: moves with allocation patterns; sudden jumps often correlate with traffic spikes.

### Heap usage

```promql
go_memstats_heap_inuse_bytes
go_memstats_heap_alloc_bytes
go_memstats_heap_sys_bytes
```

Interpretation:

- **inuse**: active heap objects.
- **alloc**: cumulative allocations (counter semantics differ; prefer docs for exact meaning in your Go version export).
- **sys**: memory obtained from OS for heap.

### Stack and GC metadata

```promql
go_memstats_stack_inuse_bytes
go_memstats_mspan_inuse_bytes
go_memstats_mcache_inuse_bytes
```

Interpretation: helps distinguish heap growth vs stack-heavy workloads.

## How to read changes over time

Compare **rates** vs **instant values**:

```promql
deriv(go_memstats_heap_inuse_bytes[10m])
```

Interpretation: slope of heap usage; pair with request-rate panels.

## Pitfalls

- **Low `GOGC`**: more frequent GC, lower latency variance but higher CPU.
- **High allocation churn**: GC time rises even if heap looks flat—pair GC panels with CPU (`process_cpu_seconds_total`).
- **Metrics are process-wide**: cannot attribute GC to a single HTTP route without custom labels.
