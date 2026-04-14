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
| Apache access | 148.2 | 2166.0 | 14.6x | 96/385 | 1/2 |
| Apache error | 3811.0 | 3751.0 | 1.0x | 288/289 | 2/2 |
| Consul | 15982.0 | 16003.0 | 1.0x | 645/645 | 2/2 |
| Dameng | 145.6 | 859.4 | 5.9x | 48/272 | 1/2 |
| Elasticsearch log | 267.8 | 11118.0 | 41.5x | 256/387 | 2/2 |
| Elasticsearch index slow log | 371.8 | 8315.0 | 22.4x | 272/404 | 2/2 |
| Elasticsearch search slow log | 455.2 | 16666.0 | 36.6x | 272/437 | 2/2 |
| Jenkins | 97.7 | 1906.0 | 19.5x | 48/272 | 1/2 |
| Kafka | 173.3 | 1730.0 | 10.0x | 64/257 | 1/2 |
| Kingbase | 186.2 | 764.7 | 4.1x | 64/353 | 1/2 |
| MySQL | 97.6 | 992.8 | 10.2x | 64/304 | 1/2 |
| MySQL slow log | 2417.0 | 2731.0 | 1.1x | 947/947 | 2/2 |
| Nginx access | 221.1 | 2846.0 | 12.9x | 144/465 | 1/2 |
| Nginx error log 1 | 304.2 | 3099.0 | 10.2x | 160/512 | 1/2 |
| Nginx error log 2 | 157.3 | 804.1 | 5.1x | 48/240 | 1/2 |
| PostgreSQL | 6654.0 | 6792.0 | 1.0x | 706/706 | 2/2 |
| RabbitMQ | 75.0 | 1358.0 | 18.1x | 48/112 | 1/2 |
| Redis | 191.6 | 629.6 | 3.3x | 80/305 | 1/2 |
| Solr | 167.0 | 1123.0 | 6.7x | 64/417 | 1/2 |
| SQLServer | 89.0 | 935.5 | 10.5x | 48/272 | 1/2 |
| TDengine 200 | 3708.0 | 3883.0 | 1.0x | 193/192 | 2/2 |
| TDengine 204 | 4819.0 | 5315.0 | 1.1x | 192/193 | 2/2 |
| Tomcat access | 190.3 | 1208.0 | 6.3x | 144/465 | 1/2 |
| Tomcat catalina | 228.4 | 1133.0 | 5.0x | 80/305 | 1/2 |

## UTF Micro Benchmark

This verifies that the fast path still behaves well with UTF content and Unicode trimming.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL with UTF message | 79.75 | 586.3 | 7.4x | 48/272 | 1/2 |

## Notes

- Near-parity cases today: `Apache error`, `Consul`, `PostgreSQL`, `TDengine 200`, `TDengine 204`, and `MySQL slow log`
- The fast path already removes one allocation for most hot log formats; Elasticsearch remains at `2 allocs/op` but is still `22x-42x` faster than pure `regexp`
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of synthetic samples
