# Benchmark Snapshot

This file records the current performance baseline for `feat-best`. It compares
the current structured fast path against the current regexp fallback path in the
same branch. It is not a master comparison; see the master-comparison documents
for historical branch-to-branch measurements.

## Environment

- Date: `2026-05-13 CST`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Go: `go1.26.1 linux/amd64`
- Branch: `feat-best`
- Commit: `bfbe10c`

## Command

Each `ns/op` value below is the median of 3 runs from:

```sh
go test -run '^$' -bench '^(BenchmarkDatakitFixtures|BenchmarkOfficialLogCases|BenchmarkCommonComposedPatterns)/' -benchmem -benchtime=200ms -count=3 .
```

All benchmarked cases in these suites are parity-checked against the regexp path:
the fast path must return the same extraction result, and Datakit pipeline
fixtures must also select the same matching pattern index before they are used
for timing.

The raw output for this run was captured locally in
`/tmp/grok-fastpath-bench-20260513.txt`, with parsed medians in
`/tmp/grok-fastpath-bench-medians-20260513.txt`.

## Datakit Fixtures

The Datakit fixture benchmark covers real grok-backed fixture cases from
`testdata/datakit_pipeline_cases.json`.

| Fixture | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| Apache access log | `133.7` | `3112` | `23.3x` |
| Apache error log | `607.7` | `5919` | `9.7x` |
| Consul log | `635.8` | `16319` | `25.7x` |
| Dameng log | `272.5` | `1417` | `5.2x` |
| Elasticsearch index slow log | `95.47` | `9445` | `98.9x` |
| Elasticsearch log | `369.5` | `9807` | `26.5x` |
| Elasticsearch search slow log | `98.35` | `15926` | `161.9x` |
| Jenkins log | `79.36` | `2985` | `37.6x` |
| Kafka log | `302.5` | `2786` | `9.2x` |
| Kingbase log | `306.9` | `1163` | `3.8x` |
| MySQL log | `70.29` | `1712` | `24.4x` |
| MySQL slow log | `941.2` | `3956` | `4.2x` |
| Nginx access log | `183.3` | `4247` | `23.2x` |
| Nginx error log 1 | `181.3` | `5227` | `28.8x` |
| Nginx error log 2 | `96.77` | `1311` | `13.5x` |
| PostgreSQL log | `816.9` | `10005` | `12.2x` |
| RabbitMQ log | `64.62` | `2362` | `36.6x` |
| Redis log | `76.81` | `1020` | `13.3x` |
| Solr log | `100.8` | `1613` | `16.0x` |
| SQLServer log | `65.52` | `1548` | `23.6x` |
| TDengine 200 log | `209.5` | `5309` | `25.3x` |
| TDengine 204 log | `238.1` | `6874` | `28.9x` |
| Tomcat Catalina log | `112.3` | `1835` | `16.3x` |
| Tomcat access log | `151.5` | `1885` | `12.4x` |

## Official Log Cases

These cases use public log-format examples from upstream project
documentation, where available.

| Case | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| Apache combined log | `554.8` | `71242` | `128.4x` |
| Elasticsearch search slow log | `98.13` | `7674` | `78.2x` |
| Kafka server log | `334.8` | `2533` | `7.6x` |
| Nginx combined access | `248.3` | `60618` | `244.1x` |
| Nginx error upstream | `213.2` | `8080` | `37.9x` |
| PostgreSQL duration statement | `175.9` | `1088` | `6.2x` |
| RabbitMQ default log | `59.80` | `1656` | `27.7x` |
| Redis server log | `234.5` | `957.2` | `4.1x` |
| Solr request log | `2754` | `2755` | `1.0x` |
| Tomcat access log | `144.7` | `1632` | `11.3x` |

## Common Composed Patterns

The common composed-pattern suite checks whether the structured fast path
generalizes outside Datakit collector fixtures.

| Pattern | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| `app_kv` | `205.3` | `1305` | `6.4x` |
| `bracket_chain_optional_node` | `289.4` | `4963` | `17.1x` |
| `custom_alias_nginx_error` | `216.4` | `22354` | `103.3x` |
| `custom_alias_postfix` | `89.29` | `9633` | `107.9x` |
| `gateway_access` | `141.2` | `24209` | `171.5x` |
| `go_gin_access` | `244.4` | `4374` | `17.9x` |
| `go_logfmt_service` | `326.6` | `2804` | `8.6x` |
| `go_worker_optional_trace` | `225.9` | `1255` | `5.6x` |
| `java_logback` | `158.3` | `1775` | `11.2x` |
| `java_spring_boot` | `257.3` | `1862` | `7.2x` |
| `k8s_controller_runtime` | `497.9` | `3526` | `7.1x` |
| `logrus_text` | `256.1` | `1498` | `5.8x` |
| `node_pino` | `191.8` | `1002` | `5.2x` |
| `python_gunicorn_access` | `309.3` | `39940` | `129.1x` |
| `python_uvicorn` | `252.9` | `1157` | `4.6x` |
| `syslog_program_pid` | `244.0` | `7007` | `28.7x` |
| `zap_console` | `211.9` | `1256` | `5.9x` |

## Notes

- `Solr request log` is effectively at parity in this run. Its pattern is
  complex enough that the structured path does not materially beat regexp on
  this sample.
- Datakit `Elasticsearch_search_slow_log` uses a different sample from the
  official-log case, so its regexp baseline is higher in the Datakit fixture
  table.
- This snapshot intentionally excludes matcher-set dispatch and buffer-reuse
  measurements. Those APIs have separate notes in the historical perf docs.
