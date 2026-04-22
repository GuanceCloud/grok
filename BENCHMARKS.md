# Benchmark Comparison Against Regexp

These numbers compare the current structured fast path with the same patterns forced down the pure `regexp` path. Semantic parity is checked by `TestDatakitFixturesMatchRegexp`.

## Environment

- Date: `2026-04-22`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Command:
  - `go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^BenchmarkRunStructuredLogLevelUTF(|RegexpPath)$' -benchmem -benchtime=300ms`

## Summary

- Scope: `24` real grok-backed fixture cases from `testdata/datakit_pipeline_cases.json`
- Result: `20/24` cases show clear gains over `regexp`, `3/24` are effectively at parity, and only `1/24` is slightly slower in the full aggregate run
- Biggest wins: Elasticsearch, Jenkins, RabbitMQ, Apache access, Nginx access, PostgreSQL, SQLServer
- Generic improvement in this round: nested optional/submatcher patterns that contain `GREEDYDATA` followed by repeated suffix literals now backtrack like `regexp`, which lets real user-composed layouts such as PostgreSQL stay on the fast path without capture drift

## Datakit Fixtures

| Fixture | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache access | 200.7 | 1752.0 | 8.7x | 288/208 | 2/2 |
| Apache error | 3505.0 | 3392.0 | 1.0x | 144/144 | 2/2 |
| Consul | 9177.0 | 9308.0 | 1.0x | 146/144 | 2/2 |
| Dameng | 122.4 | 723.6 | 5.9x | 48/112 | 1/2 |
| Elasticsearch log | 236.1 | 8001.0 | 33.9x | 256/146 | 2/2 |
| Elasticsearch index slow log | 319.8 | 6336.0 | 19.8x | 272/177 | 2/2 |
| Elasticsearch search slow log | 399.2 | 13104.0 | 32.8x | 272/179 | 2/2 |
| Jenkins | 131.2 | 1685.0 | 12.8x | 240/112 | 2/2 |
| Kafka | 154.9 | 1512.0 | 9.8x | 64/144 | 1/2 |
| Kingbase | 153.2 | 606.5 | 4.0x | 64/144 | 1/2 |
| MySQL | 84.93 | 880.8 | 10.4x | 64/144 | 1/2 |
| MySQL slow log | 1985.0 | 2032.0 | 1.0x | 498/497 | 2/2 |
| Nginx access | 325.7 | 2378.0 | 7.3x | 560/305 | 3/2 |
| Nginx error log 1 | 421.3 | 2784.0 | 6.6x | 592/337 | 3/2 |
| Nginx error log 2 | 133.6 | 633.1 | 4.7x | 48/112 | 1/2 |
| PostgreSQL | 588.5 | 5595.0 | 9.5x | 320/272 | 2/2 |
| RabbitMQ | 68.07 | 1212.0 | 17.8x | 48/112 | 1/2 |
| Redis | 170.7 | 543.7 | 3.2x | 80/176 | 1/2 |
| Solr | 170.0 | 795.3 | 4.7x | 64/144 | 1/2 |
| SQLServer | 79.88 | 833.6 | 10.4x | 48/112 | 1/2 |
| TDengine 200 | 2834.0 | 2932.0 | 1.0x | 144/144 | 2/2 |
| TDengine 204 | 3640.0 | 3636.0 | 1.0x | 144/144 | 2/2 |
| Tomcat access | 300.2 | 1008.0 | 3.4x | 560/305 | 3/2 |
| Tomcat catalina | 212.4 | 901.7 | 4.2x | 80/176 | 1/2 |

## UTF Micro Benchmark

This verifies that the fast path still behaves well with UTF content and Unicode trimming.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL with UTF message | 83.97 | 453.5 | 5.4x | 48/112 | 1/2 |

## Common Pattern Micro Benchmarks

These are small focused checks for common upstream grok layouts outside the Datakit fixture set.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured SYSLOG line | 148.0 | 4743.0 | 32.0x | 48/114 | 1/2 |

## Notes

- Near-parity or slightly slower cases today: `Apache error`, `MySQL slow log`, `TDengine 200`, and `TDengine 204`
- Regex-only paths now avoid extra unnamed capture groups for plain `%{PATTERN}` expansion, which lowers `B/op` substantially on many real patterns even when the fast path is disabled
- Regex-only paths now also normalize nested anonymous grouping such as `(foo)(bar)` or `(\[%{GREEDYDATA}\])` into non-capturing form when no numeric backreferences are present. This trims submatch bookkeeping for user-defined composed patterns without changing named captures.
- User-defined patterns that wrap the whole expression in a redundant anonymous capture, such as `(LOG|ERROR|...)` or `([.0-9a-z]*)`, now get normalized to non-capturing form before denormalization; this trims regex work without changing exposed named fields
- Structured literal parsing now understands repeated plain literals such as ` +` and ` *`, which lets common syslog-style upstream patterns reach the fast path without hard-coding product-specific matchers
- Backtracking fast paths now support `GREEDYDATA` with repeated suffix literals by trying greedier cut points first and backing off only when later steps fail; this is what moved the real PostgreSQL fixture from parity to about `9x` faster while keeping capture parity with `regexp`
- The fast path already removes one allocation for most hot log formats; Elasticsearch remains at `2 allocs/op` and is still about `20x-35x` faster than pure `regexp`
- Aggregate full-suite runs are still noisy on a couple of syslog/error layouts, but the current run no longer shows a material regression on `Consul`; that fixture is now essentially parity with a slight edge to the current path
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of synthetic samples
