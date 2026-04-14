# Benchmark Comparison Against Regexp

These numbers compare the current structured fast path with the same patterns forced down the pure `regexp` path. Semantic parity is checked by `TestDatakitFixturesMatchRegexp`.

## Environment

- Date: `2026-04-14`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Command:
  - `go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^BenchmarkRunStructuredLogLevelUTF(|RegexpPath)$' -benchmem -benchtime=300ms`

## Summary

- Scope: `24` real grok-backed fixture cases from `testdata/datakit_pipeline_cases.json`
- Result: `18/24` cases show clear gains over `regexp`, `6/24` are effectively at parity, and none are materially slower in this run
- Biggest wins: Elasticsearch, Jenkins, RabbitMQ, Apache access, Nginx access, SQLServer

## Datakit Fixtures

| Fixture | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache access | 151.0 | 2019.0 | 13.4x | 96/224 | 1/2 |
| Apache error | 3912.0 | 4083.0 | 1.0x | 160/161 | 2/2 |
| Consul | 10384.0 | 11093.0 | 1.1x | 176/179 | 2/2 |
| Dameng | 144.3 | 836.9 | 5.8x | 48/144 | 1/2 |
| Elasticsearch log | 267.1 | 9180.0 | 34.4x | 256/176 | 2/2 |
| Elasticsearch index slow log | 379.0 | 7323.0 | 19.3x | 272/213 | 2/2 |
| Elasticsearch search slow log | 494.7 | 16834.0 | 34.0x | 272/230 | 2/2 |
| Jenkins | 100.1 | 1920.0 | 19.2x | 48/128 | 1/2 |
| Kafka | 175.2 | 1635.0 | 9.3x | 64/144 | 1/2 |
| Kingbase | 186.1 | 718.0 | 3.9x | 64/176 | 1/2 |
| MySQL | 98.2 | 941.6 | 9.6x | 64/160 | 1/2 |
| MySQL slow log | 2326.0 | 2427.0 | 1.0x | 659/656 | 2/2 |
| Nginx access | 247.8 | 2538.0 | 10.2x | 144/320 | 1/2 |
| Nginx error log 1 | 291.6 | 3330.0 | 11.4x | 160/369 | 1/2 |
| Nginx error log 2 | 150.1 | 767.7 | 5.1x | 48/128 | 1/2 |
| PostgreSQL | 6354.0 | 6465.0 | 1.0x | 385/384 | 2/2 |
| RabbitMQ | 83.6 | 1378.0 | 16.5x | 48/112 | 1/2 |
| Redis | 189.2 | 649.9 | 3.4x | 80/192 | 1/2 |
| Solr | 199.3 | 849.4 | 4.3x | 64/160 | 1/2 |
| SQLServer | 88.0 | 894.2 | 10.2x | 48/128 | 1/2 |
| TDengine 200 | 3381.0 | 3361.0 | 1.0x | 144/144 | 2/2 |
| TDengine 204 | 4356.0 | 3978.0 | 0.9x | 145/144 | 2/2 |
| Tomcat access | 204.6 | 1191.0 | 5.8x | 144/320 | 1/2 |
| Tomcat catalina | 250.9 | 1034.0 | 4.1x | 80/192 | 1/2 |

## UTF Micro Benchmark

This verifies that the fast path still behaves well with UTF content and Unicode trimming.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL with UTF message | 79.75 | 586.3 | 7.4x | 48/272 | 1/2 |

## Notes

- Near-parity cases today: `Apache error`, `Consul`, `PostgreSQL`, `MySQL slow log`, `TDengine 200`, and `TDengine 204`
- Regex-only paths now avoid extra unnamed capture groups for plain `%{PATTERN}` expansion, which lowers `B/op` substantially on many real patterns even when the fast path is disabled
- The fast path already removes one allocation for most hot log formats; Elasticsearch remains at `2 allocs/op` but is still `19x-34x` faster than pure `regexp`
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of synthetic samples
