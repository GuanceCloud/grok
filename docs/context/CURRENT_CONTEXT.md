# Current Context

This is a local handoff note for ongoing performance work on
`github.com/GuanceCloud/grok`.

## Branch State

- Branch: `feat-best`
- Remote: `origin/feat-best`
- Compare worktree used for review: `/tmp/grok-master-compare` at `master`

## Current Goal

Keep the structured matcher wins, remove stale performance documentation, and
make the multi-pattern pipeline path cheaper for `pipeline-go`.

## Current Status

- `go test -run 'TestMatcherSet|TestDatakitFixturesMatchRegexp' ./...` passes.
- `BenchmarkDatakitFixtures` is currently `24/24` faster than regexp fallback.
- `ElasticSearch_log` now hits the structured path and is no longer the
  unresolved outlier described by the old docs.
- `MatcherSet` now exposes reusable-buffer APIs:
  - `MatchCount`
  - `RunFirstTo`
- Unanchored structured search with writable optional groups is guarded:
  complex cases are limited to a position-zero attempt, very small risky cases
  skip structured matching, and deterministic wrapped optionals stay linear.
- Terminal greedy captures followed only by their closing literal are treated as
  linear and avoid the recursive change-log path.
- Backtracking change logs are pooled, so complex structured patterns no longer
  allocate a rollback buffer on every match.
- Parser backtracking classification is cached on each structured step at
  compile time.
- Linear dissect runners execute directly in the outer runner now, avoiding the
  previous per-step callback layer.
- `MatcherSet` uses a `uint64` candidate bitset for small sets, removing the
  dispatch-side `[]bool` allocation from `RunFirstTo`.
- `ElasticSearch_search_slow_log` now hits its specialized runner; the matcher
  accepts the normalized `(?:query|fetch)` pattern form produced by
  denormalization.
- Anchored dissect runners have a terminal `GREEDYDATA` fast path for common
  fixed-prefix plus message-tail patterns.

## Key Benchmark Snapshot

Recent Datakit fixture numbers:

- `ElasticSearch_log`: `322.2ns/op`, `64 B/op`, `1 alloc/op`
- `ElasticSearch_search_slow_log`: `101.5-103.5ns/op`, `80 B/op`, `1 alloc/op`
- `Apache_error_log`: `726.9ns/op`, `64 B/op`, `1 alloc/op`
- `Consul_log`: `756.2ns/op`, `64 B/op`, `1 alloc/op`
- `PostgreSQL_log`: `864.0ns/op`, `128 B/op`, `1 alloc/op`
- `MySQL_slow_log`: `950.1ns/op`, `240 B/op`, `1 alloc/op`
- `Nginx_error_log2`: `67.4ns/op`, `48 B/op`, `1 alloc/op`

Common-pattern generality check:

- `11/11` measured common application patterns are faster on the fast path.
- `go_worker_optional_trace` is now around `175ns/op`, `80 B/op`,
  `1 alloc/op` after deterministic wrapped optionals were linearized.

Realistic business-log check:

- `business_order_logfmt`: `244.1-245.2ns/op`, `144 B/op`, `1 alloc/op`; `RunTo`: `210.5-211.7ns/op`, `0 alloc`
- `api_gateway_access`: `160.9-170.3ns/op`, `112 B/op`, `1 alloc/op`; `RunTo`: `134.0-142.4ns/op`, `0 alloc`
- `optional_trace_worker`: `187.8ns/op`, `80 B/op`, `1 alloc/op`
- `java_order_service`: `192.2-193.9ns/op`, `112 B/op`, `1 alloc/op`; `RunTo`: `162.8-163.9ns/op`, `0 alloc`
- `db_slow_query`: `121.4-121.9ns/op`, `64 B/op`, `1 alloc/op`; `RunTo`: `105.2-106.5ns/op`, `0 alloc`
- `message_queue_consumer`: `199.4-200.6ns/op`, `128 B/op`, `1 alloc/op`; `RunTo`: `174.8-175.2ns/op`, `0 alloc`
- `python_gunicorn_order_access`: `377.8ns/op`, `144 B/op`, `1 alloc/op`
- `k8s_controller_runtime`: `461.9ns/op`, `128 B/op`, `1 alloc/op`

Pipeline dispatch with reusable buffers:

- `apache`: `117.0-117.6ns/op`, `96 B/op`, `1 alloc/op` -> `97.6-105.3ns/op`, `0 B/op`, `0 allocs/op`
- `elasticsearch`: `672.2-673.1ns/op`, `224 B/op`, `3 allocs/op` -> `595.8-603.6ns/op`, `0 B/op`, `0 allocs/op`
- `kafka`: `477.8-478.8ns/op`, `304 B/op`, `4 allocs/op` -> `397.1-403.5ns/op`, `0 B/op`, `0 allocs/op`
- `mysql`: `76.4-78.9ns/op`, `64 B/op`, `1 alloc/op` -> `59.0-61.3ns/op`, `0 B/op`, `0 allocs/op`
- `nginx`: `182.2-184.8ns/op`, `144 B/op`, `1 alloc/op` -> `152.4-162.5ns/op`, `0 B/op`, `0 allocs/op`
- `tomcat`: `114.7-115.7ns/op`, `80 B/op`, `1 alloc/op` -> `94.5-95.0ns/op`, `0 B/op`, `0 allocs/op`

## Integration Guidance

For `pipeline-go`, compile related grok patterns into a `MatcherSet` and reuse a
per-worker buffer:

```go
buf := make([]string, 0, set.MatchCount())
id, values, err := set.RunFirstTo(line, true, buf)
```

This keeps existing matching order and semantic confirmation while removing the
result-slice allocation from most hot dispatch paths.

## Files Most Relevant To Continue

- `matcher_set.go`
- `datakit_pipeline_test.go`
- `structured_fastpath.go`
- `regex_prefilter.go`
- `grok.go`
- `docs/perf/BENCHMARKS.md`
- `docs/perf/PERFORMANCE_NOTES.md`
