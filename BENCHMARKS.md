# Benchmark Comparison Against Regexp

See [PERFORMANCE_NOTES.md](./PERFORMANCE_NOTES.md) for the current high-level conclusions, next-stage plan, and external references behind the optimization direction.

These numbers compare the current structured fast path with the same patterns forced down the `regexp` path.

Note: the `regexp` path in these benchmarks is the current shipped fallback path, so it still includes the lightweight `regexp` prefilter added in this round. For fuzz parity we additionally compare against raw `regexp` with the prefilter disabled.

## Environment

- Date: `2026-04-22`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Verification:
  - `go test ./...`
  - `go test -run '^(TestDatakitFixturesMatchRegexp|TestDatakitFixturesMutationParity|TestCommonComposedPatternsMatchRegexp|TestCommonComposedPatternsMutationParity|TestTypedPatternsMutationParity|TestRunWithTypeInfoStructuredFastPathMatchesRegexp)$' -count=1 ./...`
- Benchmarks:
  - `go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=200ms`
  - `go test -run '^$' -bench '^BenchmarkRunStructured(ShortMismatch(|RegexpPath)|LogLevelUTF(|RegexpPath)|SyslogLine(|RegexpPath))$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^(BenchmarkRunWithTypeInfo(|RegexpPath|To|WithPoolParallel|WithPoolHelperParallel|StructuredCommon(|RegexpPath)|PipelineAccess(|RegexpPath)))$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^BenchmarkCommonComposedPatterns/(go_|java_|python_|node_|zap_|logrus_|k8s_)' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^BenchmarkRunRegexpPrefilterLiteralSetMismatch$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^(BenchmarkRunCommonApacheLog(|RegexpPath|To|ToParallel|Parallel|WithPoolParallel|WithPoolHelperParallel)|BenchmarkRunStructured(Composite(|RegexpPath)|SQLServer(|RegexpPath)|NginxAccess(|RegexpPath)|LogLevel(|RegexpPath))|BenchmarkBuildCaptureMapCapacityStrategy|BenchmarkAssembleCaptureMapCapacityStrategy|BenchmarkFindStringSubmatch|BenchmarkFromMap|BenchmarkCompilePatternCommonApacheLog(|Parallel))$' -benchmem -benchtime=300ms`

## Summary

- Scope: `24` real grok-backed Datakit fixture cases from `testdata/datakit_pipeline_cases.json`
- Result: `23/24` fixture cases are faster than the current `regexp` fallback path
- Remaining slower case: `ElasticSearch_log`
- Biggest wins in this run: `ElasticSearch_search_slow_log`, `Consul`, `ElasticSearch_index_slow_log`, `TDengine 204`, `TDengine 200`, `Python gunicorn access`
- This round also added a `regexp` prefilter plus additional structured-path correctness fixes found by fuzzing

## Fuzz Campaign

The following fuzz targets were added and run for `15s` each:

- `FuzzStructuredApacheErrorParity`: passed after about `114,857` execs
- `FuzzStructuredConsulFixtureParity`: initially found `3` real structured overmatch bugs, then passed after about `669,205` execs
- `FuzzStructuredMySQLSlowFixtureParity`: initially found `1` real structured overmatch bug, then passed after about `1,038,533` execs
- `FuzzRegexpPrefilterLiteralSetParity`: passed after about `3,916,430` execs

Concrete bugs found and fixed by fuzzing:

- case-insensitive `(?i)` literal regexes must not build a case-sensitive prefilter
- `POSINT`, `NONNEGINT`, `INT`, and `NUMBER` cannot share one widened numeric matcher
- `SECOND` and `TIME` must require two-digit seconds to match upstream grok semantics
- `HOSTNAME` and the hostname branch of `IPORHOST` need `\b`-style boundary checks
- `NUMBER` / `BASE10NUM` must reject trailing bare decimal points such as `0.`

## Datakit Fixtures

| Fixture | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache access | 313.7 | 1713.0 | 5.5x | 288/208 | 2/2 |
| Apache error | 563.9 | 3374.0 | 6.0x | 256/144 | 2/2 |
| Consul | 433.4 | 9014.0 | 20.8x | 272/147 | 3/2 |
| Dameng | 230.0 | 726.1 | 3.2x | 240/112 | 2/2 |
| Elasticsearch index slow log | 490.0 | 6037.0 | 12.3x | 272/178 | 2/2 |
| Elasticsearch log | 10793.0 | 7800.0 | 0.7x | 406/147 | 4/2 |
| Elasticsearch search slow log | 669.6 | 13847.0 | 20.7x | 272/180 | 2/2 |
| Jenkins | 216.9 | 1699.0 | 7.8x | 240/112 | 2/2 |
| Kafka | 194.2 | 1525.0 | 7.9x | 64/144 | 1/2 |
| Kingbase | 262.8 | 597.8 | 2.3x | 256/144 | 2/2 |
| MySQL | 172.6 | 892.8 | 5.2x | 64/144 | 1/2 |
| MySQL slow log | 1194.0 | 1989.0 | 1.7x | 816/498 | 3/2 |
| Nginx access | 448.9 | 2365.0 | 5.3x | 560/304 | 3/2 |
| Nginx error log 1 | 559.8 | 2820.0 | 5.0x | 592/338 | 3/2 |
| Nginx error log 2 | 221.3 | 639.8 | 2.9x | 48/112 | 1/2 |
| PostgreSQL | 900.5 | 5594.0 | 6.2x | 320/273 | 2/2 |
| RabbitMQ | 146.6 | 1231.0 | 8.4x | 48/112 | 1/2 |
| Redis | 197.5 | 539.3 | 2.7x | 80/176 | 1/2 |
| Solr | 233.1 | 996.9 | 4.3x | 64/145 | 1/2 |
| SQLServer | 164.3 | 785.9 | 4.8x | 48/112 | 1/2 |
| TDengine 200 | 227.3 | 2850.0 | 12.5x | 256/144 | 2/2 |
| TDengine 204 | 248.9 | 3671.0 | 14.7x | 256/144 | 2/2 |
| Tomcat access | 414.1 | 1001.0 | 2.4x | 560/305 | 3/2 |
| Tomcat catalina | 254.9 | 912.7 | 3.6x | 80/176 | 1/2 |

`ElasticSearch_log` remains the only slower Datakit fixture. A targeted rerun put it around `8.4us/op` fast vs `8.0us/op` regexp, so it is still near parity but on the wrong side.

## Structured Micro Benchmarks

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL UTF | 170.8 | 471.3 | 2.8x | 48/112 | 1/2 |
| Structured SYSLOG line | 200.5 | 7973.0 | 39.8x | 64/112 | 2/2 |
| Structured composite | 191.5 | 794.7 | 4.2x | 128/273 | 1/2 |
| Structured SQLServer | 167.7 | 500.2 | 3.0x | 48/112 | 1/2 |
| Structured Nginx access | 426.2 | 965.5 | 2.3x | 560/305 | 3/2 |
| Structured LOGLEVEL | 165.4 | 459.2 | 2.8x | 48/112 | 1/2 |

Short mismatch is now a useful reminder that the new regexp prefilter helps the fallback path too:

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured short mismatch | 33.42 | 4.696 | 0.1x | 48/0 | 1/0 |

## Typed Extraction Benchmarks

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| RunWithTypeInfo simple typed line | 104.2 | 198.1 | 1.9x | 56/121 | 2/3 |
| RunWithTypeInfo structured typed line | 394.5 | 680.3 | 1.7x | 384/257 | 6/6 |
| RunWithTypeInfo pipeline-go style access line | 552.1 | 24577.0 | 44.5x | 568/431 | 11/10 |
| RunWithTypeInfoTo | 89.35 | n/a | n/a | 8/n/a | 1/n/a |
| RunWithTypeInfo pooled parallel | 17.96 | n/a | n/a | 8/n/a | 1/n/a |
| RunWithTypeInfo pooled helper parallel | 17.92 | n/a | n/a | 8/n/a | 1/n/a |

## Common Application Composed Patterns

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Go logfmt service line | 312.0 | 1360.0 | 4.4x | 272/176 | 2/2 |
| Go Gin access line | 266.1 | 2315.0 | 8.7x | 160/208 | 2/2 |
| Go worker line with optional trace | 228.7 | 621.8 | 2.7x | 80/176 | 1/2 |
| Java logback line | 209.7 | 869.9 | 4.1x | 80/176 | 1/2 |
| Java Spring Boot line | 240.6 | 889.5 | 3.7x | 96/208 | 1/2 |
| Python uvicorn line | 162.2 | 575.1 | 3.5x | 64/144 | 1/2 |
| Python gunicorn access line | 447.6 | 24108.0 | 53.9x | 624/311 | 4/2 |
| Node pino line | 258.0 | 504.7 | 2.0x | 272/176 | 2/2 |
| Zap console line | 183.8 | 629.1 | 3.4x | 64/144 | 1/2 |
| Logrus text line | 260.1 | 778.6 | 3.0x | 256/144 | 2/2 |
| Kubernetes controller-runtime line | 508.6 | 1759.0 | 3.5x | 320/273 | 2/2 |

## Regexp Prefilter Benchmark

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Literal set mismatch with prefilter | 8.732 | 25.73 | 2.9x | 0/0 | 0/0 |

## Capture Map Benchmarks

| Benchmark | Variant | ns/op | B/op | Allocs/op |
| --- | --- | ---: | ---: | ---: |
| BuildCaptureMap COMMONAPACHELOG | no_prealloc | 30498 | 1139 | 6 |
| BuildCaptureMap COMMONAPACHELOG | num_subexp | 22154 | 847 | 5 |
| BuildCaptureMap COMMONAPACHELOG | named_fields | 22391 | 840 | 5 |
| BuildCaptureMap ElasticSearchDefault | no_prealloc | 12242 | 418 | 3 |
| BuildCaptureMap ElasticSearchDefault | num_subexp | 12621 | 425 | 3 |
| BuildCaptureMap ElasticSearchDefault | named_fields | 12319 | 419 | 3 |
| AssembleCaptureMap COMMONAPACHELOG | no_prealloc | 294.1 | 952 | 5 |
| AssembleCaptureMap COMMONAPACHELOG | num_subexp | 188.1 | 664 | 4 |
| AssembleCaptureMap COMMONAPACHELOG | named_fields | 197.4 | 664 | 4 |
| AssembleCaptureMap ElasticSearchDefault | no_prealloc | 82.13 | 336 | 2 |
| AssembleCaptureMap ElasticSearchDefault | num_subexp | 83.69 | 336 | 2 |
| AssembleCaptureMap ElasticSearchDefault | named_fields | 85.82 | 336 | 2 |

## Additional Benchmarks

| Benchmark | Result |
| --- | --- |
| `FindStringSubmatch` | `311.3ns/op`, `224 B/op`, `2 allocs/op` |
| `FindStringSubmatchIndex` | `286.4ns/op`, `112 B/op`, `1 allocs/op` |
| `FromMap` | `359859ns/op`, `590395 B/op`, `3665 allocs/op` |
| `CompilePatternCommonApacheLog` | `8954ns/op`, `54696 B/op`, `117 allocs/op` |
| `CompilePatternCommonApacheLogParallel` | `9516ns/op`, `54700 B/op`, `117 allocs/op` |
| `RunCommonApacheLog` | `542.4ns/op`, `656 B/op`, `4 allocs/op` |
| `RunCommonApacheLogRegexpPath` | `21560ns/op`, `342 B/op`, `2 allocs/op` |
| `RunCommonApacheLogParallel` | `164.4ns/op`, `656 B/op`, `4 allocs/op` |
| `RunCommonApacheLogTo` | `530.3ns/op`, `496 B/op`, `3 allocs/op` |
| `RunCommonApacheLogToParallel` | `139.3ns/op`, `496 B/op`, `3 allocs/op` |
| `RunCommonApacheLogWithPoolParallel` | `147.7ns/op`, `496 B/op`, `3 allocs/op` |
| `RunCommonApacheLogWithPoolHelperParallel` | `146.6ns/op`, `496 B/op`, `3 allocs/op` |

## Notes

- The Apache error, Consul, and MySQL slow-log regressions from the previous report are gone in this run.
- The main unresolved outlier is still `ElasticSearch_log`.
- The new regexp prefilter helps fallback mismatches materially, which is why some mismatch-only benchmarks now favor the regexp path.
- Fuzzing was useful here: it found real semantic drift that the fixed fixture corpus did not catch.
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of only synthetic samples.
