# Benchmark Snapshot

This file records the current performance baseline for `feat-best`. The old
`2026-04-22` numbers are no longer representative: `ElasticSearch_log` now uses
the structured fast path and is no longer a regexp-side outlier.

## Environment

- Date: `2026-04-24`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Branch: `feat-best`
- Compare worktree: `/tmp/grok-master-compare` at `master`

## Commands

```sh
go test -run 'TestMatcherSet|TestDatakitFixturesMatchRegexp' ./...
go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=300ms
go test -run '^$' -bench '^BenchmarkDatakitPipelineDispatch$' -benchmem -benchtime=200ms
go test -run '^$' -bench '^BenchmarkCommonComposedPatterns/(go_|java_|python_|node_|zap_|logrus_|k8s_)' -benchmem -benchtime=300ms
go test -run '^$' -bench '^BenchmarkMatcherSetRunFirst' -benchmem -benchtime=300ms
```

## Datakit Fixture Summary

The Datakit fixture benchmark covers `24` real grok-backed fixture cases from
`testdata/datakit_pipeline_cases.json`.

- Result: `24/24` fixture cases are faster than the current regexp fallback path.
- The regexp path in this benchmark still includes the lightweight regexp
  prefilter shipped in this branch.
- The current bottleneck has moved from single-pattern matching to pipeline
  dispatch and caller-side allocation behavior.

Representative results:

| Fixture | Fast ns/op | Regexp ns/op | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: |
| Apache access | `115.0` | `3092` | `96/208` | `1/2` |
| Apache error | `726.9` | `6058` | `64/145` | `1/2` |
| Consul | `756.2` | `16066` | `64/148` | `1/2` |
| ElasticSearch index slow log | `102.8` | `9457` | `80/178` | `1/2` |
| ElasticSearch log | `322.2` | `10315` | `64/154` | `1/2` |
| ElasticSearch search slow log | `101.5-103.5` | `17085` | `80/180` | `1/2` |
| MySQL slow log | `1063` | `4102` | `624/499` | `2/2` |
| Nginx access | `168.5` | `4241` | `144/304` | `1/2` |
| Nginx error log 1 | `136.1` | `5237` | `160/336` | `1/2` |
| Nginx error log 2 | `67.4` | `1296` | `48/112` | `1/2` |
| PostgreSQL | `887.7` | `10027` | `128/273` | `1/2` |
| RabbitMQ | `67.12` | `2393` | `48/112` | `1/2` |
| SQLServer | `65.93` | `1535` | `48/112` | `1/2` |
| TDengine 200 | `234.4` | `5442` | `64/144` | `1/2` |
| Tomcat catalina | `112.5` | `1829` | `80/177` | `1/2` |

## Master Comparison Notes

Against `/tmp/grok-master-compare` at `master`, this branch is materially faster
on the cases that were previously mostly regexp-bound:

| Fixture | master fast ns/op | feat-best fast ns/op |
| --- | ---: | ---: |
| Apache error | `3824` | `726.9` |
| Consul | `12816` | `756.2` |
| PostgreSQL | `6329` | `887.7` |
| TDengine 200 | `3092` | `234.4` |
| TDengine 204 | `4207` | `253.5` |
| ElasticSearch search slow log | `451.2` | `101.5-103.5` |

The previous Elasticsearch search-slow regression was caused by the specialized
runner comparing against the pre-normalized group syntax. The compiler stores
the normalized `(?:...)` pattern form, so the runner did not activate until that
match was fixed.

## Pipeline Dispatch

`MatcherSet.RunFirstTo` lets pipeline-style callers reuse one result buffer
across match attempts. This is the preferred API for `pipeline-go` integration:

```go
buf := make([]string, 0, set.MatchCount())
id, values, err := set.RunFirstTo(line, true, buf)
```

Small matcher sets now use a `uint64` candidate bitset on the dispatch path, so
`RunFirstTo` avoids both the result-slice allocation and the previous `[]bool`
candidate-set allocation for the measured Datakit pipelines.

Representative Datakit dispatch results:

| Pipeline | matcher_set ns/op | matcher_set_reuse ns/op | Alloc reduction |
| --- | ---: | ---: | ---: |
| apache | `117.0-117.6` | `97.6-105.3` | `96 B, 1 alloc -> 0` |
| elasticsearch | `672.2-673.1` | `595.8-603.6` | `224 B, 3 allocs -> 0` |
| kafka | `477.8-478.8` | `397.1-403.5` | `304 B, 4 allocs -> 0` |
| mysql | `76.4-78.9` | `59.0-61.3` | `64 B, 1 alloc -> 0` |
| nginx | `182.2-184.8` | `152.4-162.5` | `144 B, 1 alloc -> 0` |
| postgresql | `855.7-883.6` | `792.3-829.8` | `128 B, 1 alloc -> 0` |
| tdengine | `216.8-221.2` | `197.8-198.9` | `64 B, 1 alloc -> 0` |
| tomcat | `114.7-115.7` | `94.5-95.0` | `80 B, 1 alloc -> 0` |

Synthetic matcher-set benchmarks show the same behavior:

| Benchmark | matcher_set | matcher_set_reuse | manual_loop |
| --- | ---: | ---: | ---: |
| `BenchmarkMatcherSetRunFirst` | `168.3ns/op, 32 B, 1 alloc` | `151.9ns/op, 0 B, 0 allocs` | `247.4ns/op, 304 B, 5 allocs` |
| `BenchmarkMatcherSetRunFirstLargeSet` | `932.6ns/op, 128 B, 2 allocs` | `954.9ns/op, 80 B, 1 alloc` | `2600ns/op, 2096 B, 65 allocs` |

The large-set reuse case can still allocate when the set exceeds the small-set
bitset path or needs atom hit state beyond the inline bit mask.

## Single Pattern Reuse

Single compiled patterns can use `RunTo` with a caller-owned result buffer. This
does not change `Run` semantics, but it removes the result-slice allocation from
hot loops and gives short mostly-linear business logs a more representative
pipeline-style measurement.

```go
buf := make([]string, 0, g.MatchCount())
values, err := g.RunTo(line, true, buf)
```

Representative realistic-log results:

| Pattern | `Run` ns/op | `RunTo` ns/op | Alloc reduction |
| --- | ---: | ---: | ---: |
| business order logfmt | `244.1-245.2` | `210.5-211.7` | `144 B, 1 alloc -> 0` |
| API gateway access | `160.9-170.3` | `134.0-142.4` | `112 B, 1 alloc -> 0` |
| Java order service | `192.2-193.9` | `162.8-163.9` | `112 B, 1 alloc -> 0` |
| DB slow query | `121.4-121.9` | `105.2-106.5` | `64 B, 1 alloc -> 0` |
| message queue consumer | `199.4-200.6` | `174.8-175.2` | `128 B, 1 alloc -> 0` |

## Common Pattern Coverage

The common composed-pattern suite is useful for checking whether fixture-driven
optimizations generalize outside Datakit.

Representative current results:

| Pattern | Fast ns/op | Regexp ns/op | Notes |
| --- | ---: | ---: | --- |
| `go_logfmt_service` | `314.6` | `2690` | fast path wins |
| `go_gin_access` | `260.4` | `4242` | fast path wins |
| `go_worker_optional_trace` | `179.6` | `1284` | fast path wins |
| `java_logback` | `203.7` | `1739` | fast path wins |
| `java_spring_boot` | `227.5` | `1788` | fast path wins |
| `python_uvicorn` | `277.7` | `1187` | fast path wins |
| `python_gunicorn_access` | `371.1` | `39665` | fast path wins |
| `node_pino` | `155.4` | `994.1` | fast path wins |
| `zap_console` | `176.6` | `1259` | fast path wins |
| `logrus_text` | `274.9` | `1486` | fast path wins |
| `k8s_controller_runtime` | `459.4` | `3543` | fast path wins |

The remaining non-general case is writable optional groups in unanchored search
patterns that cannot be proven linear. The runtime limits complex cases to a
position-zero structured attempt and disables very small risky matchers, while
deterministic wrapped optionals now stay on the fast path.
