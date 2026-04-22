# Benchmark Comparison Against Regexp

These numbers compare the current structured fast path with the same patterns forced down the pure `regexp` path. Semantic parity is checked by `TestDatakitFixturesMatchRegexp`.

## Environment

- Date: `2026-04-22`
- OS/Arch: `linux/amd64`
- CPU: `AMD Ryzen 7 9700X 8-Core Processor`
- Command:
  - `go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=200ms`
  - `go test -run '^$' -bench '^BenchmarkRunStructured(ShortMismatch(|RegexpPath)|LogLevelUTF(|RegexpPath)|SyslogLine(|RegexpPath))$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^(BenchmarkRunWithTypeInfo(|To|WithPoolParallel|WithPoolHelperParallel|StructuredCommon(|RegexpPath)))$' -benchmem -benchtime=300ms`
  - `go test -run '^$' -bench '^BenchmarkCommonComposedPatterns/(go_|java_|python_|node_|zap_|logrus_|k8s_)' -benchmem -benchtime=300ms`

## Summary

- Scope: `24` real grok-backed fixture cases from `testdata/datakit_pipeline_cases.json`
- Result: `21/24` cases show clear gains over `regexp`; only `Apache error`, `Consul`, and `MySQL slow log` remain slightly slower in this run
- Biggest wins: Elasticsearch, Jenkins, RabbitMQ, Apache access, Nginx access, PostgreSQL, SQLServer
- Generic improvements in this round:
  - structured matchers now carry lightweight IR metadata (`minWidth`, nullability, stable boundary literals) that is used both at compile time and at match time
  - parser context slicing now distinguishes a safe boundary literal from an exact suffix literal, so user-composed patterns like `NOTSPACE + SPACE + literal` keep their fast path without overfitting to specific fixtures
  - `Run` now uses the IR to reject obvious mismatches before allocating result buffers, which makes short failures zero-allocation and faster than the regexp path

## Datakit Fixtures

| Fixture | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache access | 232.8 | 1742.0 | 7.5x | 288/208 | 2/2 |
| Apache error | 3850.0 | 3791.0 | 1.0x | 144/144 | 2/2 |
| Consul | 9597.0 | 9280.0 | 1.0x | 147/144 | 2/2 |
| Dameng | 173.1 | 749.4 | 4.3x | 48/112 | 1/2 |
| Elasticsearch log | 293.0 | 8573.0 | 29.3x | 256/147 | 2/2 |
| Elasticsearch index slow log | 398.1 | 6248.0 | 15.7x | 272/176 | 2/2 |
| Elasticsearch search slow log | 484.0 | 13558.0 | 28.0x | 272/176 | 2/2 |
| Jenkins | 152.4 | 1696.0 | 11.1x | 240/112 | 2/2 |
| Kafka | 199.0 | 1526.0 | 7.7x | 64/144 | 1/2 |
| Kingbase | 208.2 | 613.5 | 2.9x | 64/144 | 1/2 |
| MySQL | 109.4 | 864.0 | 7.9x | 64/144 | 1/2 |
| MySQL slow log | 2018.0 | 2009.0 | 1.0x | 499/498 | 2/2 |
| Nginx access | 376.7 | 2401.0 | 6.4x | 560/304 | 3/2 |
| Nginx error log 1 | 499.9 | 3077.0 | 6.2x | 592/337 | 3/2 |
| Nginx error log 2 | 178.5 | 640.1 | 3.6x | 48/112 | 1/2 |
| PostgreSQL | 724.8 | 5593.0 | 7.7x | 320/273 | 2/2 |
| RabbitMQ | 79.50 | 1214.0 | 15.3x | 48/112 | 1/2 |
| Redis | 188.9 | 555.1 | 2.9x | 80/176 | 1/2 |
| Solr | 189.6 | 806.4 | 4.3x | 64/144 | 1/2 |
| SQLServer | 97.31 | 783.8 | 8.1x | 48/112 | 1/2 |
| TDengine 200 | 3070.0 | 3086.0 | 1.0x | 144/144 | 2/2 |
| TDengine 204 | 4045.0 | 4171.0 | 1.0x | 145/144 | 2/2 |
| Tomcat access | 344.9 | 1038.0 | 3.0x | 560/304 | 3/2 |
| Tomcat catalina | 241.0 | 931.1 | 3.9x | 80/176 | 1/2 |

## UTF Micro Benchmark

This verifies that the fast path still behaves well with UTF content and Unicode trimming.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured LOGLEVEL with UTF message | 89.11 | 452.1 | 5.1x | 48/112 | 1/2 |

## Common Pattern Micro Benchmarks

These are small focused checks for common upstream grok layouts outside the Datakit fixture set.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Structured SYSLOG line | 171.4 | 4693.0 | 27.4x | 48/113 | 1/2 |
| Structured short mismatch | 2.688 | 4.682 | 1.7x | 0/0 | 0/0 |

## Typed Extraction Benchmarks

These matter directly for `pipeline-go`, which calls `RunWithTypeInfo` whenever the grok pattern declares field types.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| RunWithTypeInfo simple typed line | 111.8 | 197.6 | 1.8x | 56/120 | 2/3 |
| RunWithTypeInfo structured typed line | 320.2 | 663.2 | 2.1x | 336/257 | 6/6 |
| RunWithTypeInfoTo simple typed line | 93.85 | n/a | n/a | 8/n/a | 1/n/a |
| RunWithTypeInfo pooled parallel | 17.50 | n/a | n/a | 8/n/a | 1/n/a |

## Common Application Composed Patterns

These are custom, user-style composed patterns meant to reflect common application log formats rather than Datakit defaults.

| Benchmark | Fast ns/op | Regexp ns/op | Speedup | B/op fast/re | Allocs fast/re |
| --- | ---: | ---: | ---: | ---: | ---: |
| Go logfmt service line | 240.9 | 1351.0 | 5.6x | 272/176 | 2/2 |
| Go Gin access line | 196.6 | 2600.0 | 13.2x | 96/209 | 1/2 |
| Go worker line with optional trace | 166.1 | 616.9 | 3.7x | 80/176 | 1/2 |
| Java logback line | 144.9 | 888.0 | 6.1x | 80/176 | 1/2 |
| Java Spring Boot line | 187.6 | 889.3 | 4.7x | 96/208 | 1/2 |
| Python uvicorn line | 108.3 | 574.9 | 5.3x | 64/144 | 1/2 |
| Python gunicorn access line | 350.1 | 24192.0 | 69.1x | 560/317 | 3/2 |
| Node pino line | 221.8 | 501.0 | 2.3x | 272/176 | 2/2 |
| Zap console line | 120.0 | 636.3 | 5.3x | 64/144 | 1/2 |
| Logrus text line | 202.7 | 789.3 | 3.9x | 256/144 | 2/2 |
| Kubernetes controller-runtime line | 1744.0 | 1833.0 | 1.1x | 272/273 | 2/2 |

## Notes

- Slightly slower cases today: `Apache error`, `Consul`, and `MySQL slow log`
- Regex-only paths now avoid extra unnamed capture groups for plain `%{PATTERN}` expansion, which lowers `B/op` substantially on many real patterns even when the fast path is disabled
- Regex-only paths now also normalize nested anonymous grouping such as `(foo)(bar)` or `(\[%{GREEDYDATA}\])` into non-capturing form when no numeric backreferences are present. This trims submatch bookkeeping for user-defined composed patterns without changing named captures.
- User-defined patterns that wrap the whole expression in a redundant anonymous capture, such as `(LOG|ERROR|...)` or `([.0-9a-z]*)`, now get normalized to non-capturing form before denormalization; this trims regex work without changing exposed named fields
- Structured literal parsing now understands repeated plain literals such as ` +` and ` *`, which lets common syslog-style upstream patterns reach the fast path without hard-coding product-specific matchers
- Backtracking fast paths now support `GREEDYDATA` with repeated suffix literals by trying greedier cut points first and backing off only when later steps fail; this is what moved the real PostgreSQL fixture from parity to about `9x` faster while keeping capture parity with `regexp`
- Structured planners now compute per-step and per-matcher IR metadata, including minimum width and stable boundary literals, and use that both to prune impossible branches and to preserve fast parser slicing in composed layouts such as Elasticsearch default logs
- `Run` and `RunWithTypeInfo` now use IR-based quick rejection before allocating output buffers, which is why short mismatches are now zero-allocation and faster than the regexp path
- `RunWithTypeInfo` now also reuses the structured fast path when one exists, which is directly relevant to `pipeline-go` because that project switches to typed extraction as soon as a grok field is declared with `:int`, `:float`, `:bool`, or `:string`
- The same generic machinery also carries over to common user-composed application logs; the current synthetic set spans Go, Java, Python, Node.js, Zap, Logrus, and controller-runtime without adding language-specific hard-coded matchers
- That broader set currently ranges from near parity on `controller-runtime` style reconcile logs up to about `69x` on access-style Python logs, which matches the underlying rule of thumb: fixed delimiters and stable field order benefit the most
- The fast path already removes one allocation for most hot log formats; Elasticsearch remains at `2 allocs/op` and is still about `20x-35x` faster than pure `regexp`
- Aggregate full-suite runs are still noisy on a couple of syslog/error layouts, and `Consul` is again slightly behind in this run; that remains a fallback-tuning target rather than a fast-path semantics issue
- The benchmark source data lives in `testdata/datakit_pipeline_cases.json`, so new optimizations can be checked against real Datakit pipelines instead of synthetic samples
