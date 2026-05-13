# Benchmark Snapshot

This file records the current performance baseline for `feat-best`. It compares
the current structured fast path against the current regexp fallback path in the
same branch, using the external-facing `RunWithTypeInfo` API. The benchmarked
Grok patterns do not include explicit `:int`, `:float`, or `:bool` type
annotations.

This is not a master comparison; see the master-comparison documents for
historical branch-to-branch measurements.

## Environment

- Date: `2026-05-13 CST`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Go: `go1.26.1 linux/amd64`
- Branch: `feat-best`
- Commit: `bfbe10c` plus current worktree changes

## Command

Each `ns/op` value below is the median of 3 runs from:

```sh
go test -run '^$' -bench '^(BenchmarkDatakitFixturesWithTypeInfo|BenchmarkOfficialLogCasesWithTypeInfo|BenchmarkCommonComposedPatternsWithTypeInfo)/' -benchmem -benchtime=200ms -count=3 .
```

All benchmarked cases in these suites are parity-checked against the regexp path:
the fast path must return the same typed extraction result, and Datakit pipeline
fixtures must also select the same matching pattern index before they are used
for timing.

The raw output for this run was captured locally in
`/tmp/grok-fastpath-withtype-notype-bench-20260513.txt`, with parsed medians in
`/tmp/grok-fastpath-withtype-notype-medians-20260513.txt`.

## Datakit Fixtures

The Datakit fixture benchmark covers real grok-backed fixture cases from
`testdata/datakit_pipeline_cases.json`.

| Fixture | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| Apache access log | `241.4` | `3263` | `13.5x` |
| Apache error log | `733.1` | `6059` | `8.3x` |
| Consul log | `707.3` | `15846` | `22.4x` |
| Dameng log | `296.8` | `1487` | `5.0x` |
| Elasticsearch index slow log | `172.1` | `9645` | `56.0x` |
| Elasticsearch log | `374.8` | `9770` | `26.1x` |
| Elasticsearch search slow log | `176.9` | `16351` | `92.4x` |
| Jenkins log | `123.3` | `3059` | `24.8x` |
| Kafka log | `377.3` | `2785` | `7.4x` |
| Kingbase log | `347.6` | `1233` | `3.5x` |
| MySQL log | `128.0` | `1771` | `13.8x` |
| MySQL slow log | `1180` | `4248` | `3.6x` |
| Nginx access log | `342.3` | `4434` | `13.0x` |
| Nginx error log 1 | `320.3` | `5518` | `17.2x` |
| Nginx error log 2 | `145.4` | `1345` | `9.3x` |
| PostgreSQL log | `1053` | `10179` | `9.7x` |
| RabbitMQ log | `108.7` | `2432` | `22.4x` |
| Redis log | `144.7` | `1246` | `8.6x` |
| Solr log | `157.2` | `1644` | `10.5x` |
| SQLServer log | `109.8` | `1607` | `14.6x` |
| TDengine 200 log | `281.2` | `5342` | `19.0x` |
| TDengine 204 log | `291.0` | `6959` | `23.9x` |
| Tomcat Catalina log | `181.6` | `1887` | `10.4x` |
| Tomcat access log | `302.4` | `2048` | `6.8x` |

## Official Log Cases

These cases use public log-format examples from upstream project
documentation, where available.

| Case | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| Apache combined log | `747.0` | `70315` | `94.1x` |
| Elasticsearch search slow log | `171.4` | `7623` | `44.5x` |
| Kafka server log | `405.2` | `2537` | `6.3x` |
| Nginx combined access | `433.1` | `60465` | `139.6x` |
| Nginx error upstream | `362.0` | `8343` | `23.0x` |
| PostgreSQL duration statement | `301.6` | `1208` | `4.0x` |
| RabbitMQ default log | `103.5` | `1682` | `16.3x` |
| Redis server log | `321.5` | `998.9` | `3.1x` |
| Solr request log | `2942` | `2941` | `1.0x` |
| Tomcat access log | `293.1` | `1754` | `6.0x` |

## Common Composed Patterns

The common composed-pattern suite checks whether the structured fast path
generalizes outside Datakit collector fixtures.

| Pattern | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| `app_kv` | `228.9` | `1434` | `6.3x` |
| `bracket_chain_optional_node` | `370.3` | `4963` | `13.4x` |
| `custom_alias_nginx_error` | `365.7` | `21844` | `59.7x` |
| `custom_alias_postfix` | `175.9` | `9533` | `54.2x` |
| `gateway_access` | `252.8` | `24615` | `97.4x` |
| `go_gin_access` | `366.7` | `4466` | `12.2x` |
| `go_logfmt_service` | `322.2` | `2831` | `8.8x` |
| `go_worker_optional_trace` | `236.7` | `1371` | `5.8x` |
| `java_logback` | `229.8` | `1884` | `8.2x` |
| `java_spring_boot` | `286.9` | `1921` | `6.7x` |
| `k8s_controller_runtime` | `518.1` | `3533` | `6.8x` |
| `logrus_text` | `260.2` | `1571` | `6.0x` |
| `node_pino` | `226.5` | `1100` | `4.9x` |
| `python_gunicorn_access` | `490.6` | `39891` | `81.3x` |
| `python_uvicorn` | `282.4` | `1239` | `4.4x` |
| `syslog_program_pid` | `320.2` | `7017` | `21.9x` |
| `zap_console` | `202.6` | `1339` | `6.6x` |

## Notes

- `Solr request log` is effectively at parity in this run. Its pattern is
  complex enough that the structured path does not materially beat regexp on
  this sample.
- Datakit `Elasticsearch_search_slow_log` uses a different sample from the
  official-log case, so its regexp baseline is higher in the Datakit fixture
  table.
- A typed parity issue for unmatched optional captures was fixed during this
  measurement pass: typed fast path now initializes unmatched captures to the
  same values that the regexp path produces for empty captures.
