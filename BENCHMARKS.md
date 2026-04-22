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
- Result: `19/24` cases show clear gains over `regexp`, `3/24` are effectively at parity, and `2/24` are slightly slower in the full aggregate run
- Biggest wins: Elasticsearch, Jenkins, RabbitMQ, Apache access, Nginx access, PostgreSQL, SQLServer
- Generic improvement in this round: nested optional/submatcher patterns that contain `GREEDYDATA` followed by repeated suffix literals now backtrack like `regexp`, which lets real user-composed layouts such as PostgreSQL stay on the fast path without capture drift

## Datakit Fixtures

| Fixture | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache access | 206.0 | 1750.0 | 8.5x | 288/208 | 2/2 |
| Apache error | 3485.0 | 3420.0 | 1.0x | 144/144 | 2/2 |
| Consul | 11743.0 | 10523.0 | 0.9x | 146/144 | 2/2 |
| Dameng | 126.6 | 721.6 | 5.7x | 48/112 | 1/2 |
| Elasticsearch log | 230.0 | 8028.0 | 34.9x | 256/160 | 2/2 |
| Elasticsearch index slow log | 316.7 | 6221.0 | 19.6x | 272/195 | 2/2 |
| Elasticsearch search slow log | 408.1 | 13782.0 | 33.8x | 272/211 | 2/2 |
| Jenkins | 138.9 | 1684.0 | 12.1x | 240/112 | 2/2 |
| Kafka | 160.3 | 1535.0 | 9.6x | 64/144 | 1/2 |
| Kingbase | 155.7 | 608.1 | 3.9x | 64/144 | 1/2 |
| MySQL | 100.3 | 864.7 | 8.6x | 64/144 | 1/2 |
| MySQL slow log | 2056.0 | 2051.0 | 1.0x | 658/658 | 2/2 |
| Nginx access | 331.8 | 2401.0 | 7.2x | 560/305 | 3/2 |
| Nginx error log 1 | 424.0 | 3049.0 | 7.2x | 592/354 | 3/2 |
| Nginx error log 2 | 135.4 | 632.8 | 4.7x | 48/112 | 1/2 |
| PostgreSQL | 615.3 | 5588.0 | 9.1x | 320/306 | 2/2 |
| RabbitMQ | 67.81 | 1212.0 | 17.9x | 48/112 | 1/2 |
| Redis | 163.3 | 543.7 | 3.3x | 80/176 | 1/2 |
| Solr | 162.1 | 800.7 | 4.9x | 64/144 | 1/2 |
| SQLServer | 82.97 | 775.5 | 9.3x | 48/112 | 1/2 |
| TDengine 200 | 2828.0 | 2790.0 | 1.0x | 144/144 | 2/2 |
| TDengine 204 | 3651.0 | 3662.0 | 1.0x | 144/144 | 2/2 |
| Tomcat access | 296.4 | 1004.0 | 3.4x | 560/305 | 3/2 |
| Tomcat catalina | 218.3 | 912.0 | 4.2x | 80/176 | 1/2 |

## UTF Micro Benchmark

This verifies that the fast path still behaves well with UTF content and Unicode trimming.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL with UTF message | 85.36 | 447.2 | 5.2x | 48/112 | 1/2 |

## Common Pattern Micro Benchmarks

These are small focused checks for common upstream grok layouts outside the Datakit fixture set.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured SYSLOG line | 152.3 | 4625.0 | 30.4x | 48/112 | 1/2 |

## Notes

- Near-parity or slightly slower cases today: `Apache error`, `Consul`, `MySQL slow log`, `TDengine 200`, and `TDengine 204`
- Regex-only paths now avoid extra unnamed capture groups for plain `%{PATTERN}` expansion, which lowers `B/op` substantially on many real patterns even when the fast path is disabled
- User-defined patterns that wrap the whole expression in a redundant anonymous capture, such as `(LOG|ERROR|...)` or `([.0-9a-z]*)`, now get normalized to non-capturing form before denormalization; this trims regex work without changing exposed named fields
- Structured literal parsing now understands repeated plain literals such as ` +` and ` *`, which lets common syslog-style upstream patterns reach the fast path without hard-coding product-specific matchers
- Backtracking fast paths now support `GREEDYDATA` with repeated suffix literals by trying greedier cut points first and backing off only when later steps fail; this is what moved the real PostgreSQL fixture from parity to about `9x` faster while keeping capture parity with `regexp`
- The fast path already removes one allocation for most hot log formats; Elasticsearch remains at `2 allocs/op` and is still about `20x-35x` faster than pure `regexp`
- Aggregate full-suite runs are still noisy on a couple of syslog/error layouts. `Consul` was slightly slower in the full run above, but remained near-parity in a dedicated longer rerun (`~9.2us` fast vs `~9.1us` regexp), so it should be treated as a fallback-quality shape rather than a guaranteed win
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of synthetic samples
