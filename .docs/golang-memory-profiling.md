# Profiling Go memory (practical workflow)

This guide focuses on **heap profiling** using `pprof`. Your API does not enable `pprof` endpoints yet; for local profiling you typically import:

```go
import _ "net/http/pprof"
```

…and serve the default mux or mount routes manually. **Do not expose `pprof` publicly** in production without authentication.

## 1) Capture a heap profile (local)

Run the binary with `pprof` enabled, then:

```bash
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/heap
```

Interpretation:

- **Top view**: hottest allocation sites.
- **Flame graph**: shows stack frames contributing allocations.
- **peek/list**: inspect specific functions.

## 2) Compare two heap snapshots (diff)

Take two profiles minutes apart during load:

```bash
curl -s http://localhost:8080/debug/pprof/heap > heap-a.pb.gz
# ... generate load ...
curl -s http://localhost:8080/debug/pprof/heap > heap-b.pb.gz

go tool pprof -base heap-a.pb.gz heap-b.pb.gz
```

Interpretation: highlights **growth** between snapshots (leaks often appear here).

## 3) Use `allocs` vs `heap`

- **`/debug/pprof/allocs`**: allocation counts since process start (where objects were allocated).
- **`/debug/pprof/heap`**: live heap objects (where memory remains).

```bash
go tool pprof http://localhost:8080/debug/pprof/allocs
```

Interpretation: `allocs` helps find hot allocation paths even if objects are short-lived.

## 4) CPU vs memory

High allocations often correlate with CPU spent in GC:

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

Interpretation: if CPU profile shows `runtime.gc` / `mallocgc` prominently, pair with heap profiles.

## 5) Production-safe patterns

- Gate `pprof` behind admin auth or bind to localhost-only.
- Prefer **continuous profiling** products (Datadog, Grafana Pyroscope, Google Cloud Profiler) for long-term trends.
- Combine with Prometheus:

```promql
rate(go_memstats_heap_inuse_bytes[5m])
```

Interpretation: rising heap with flat traffic suggests leaks or caches growing without bounds.

## Quick checklist

1. Confirm whether growth is **heap** or **RSS** (OS-level) using process metrics.
2. Take **two heap profiles** and diff.
3. Zoom into **allocation sites** and reduce struct copying, slice growth, or JSON churn.
4. Re-measure GC pause and allocation rates.
