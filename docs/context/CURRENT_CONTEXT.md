# Current Context

This file is a local handoff note for ongoing performance work on `github.com/GuanceCloud/grok`.

## Branch State

- Branch: `feat-best`
- Latest pushed commit: `8f79eb0` (`improve structured matcher dispatch`)
- Remote: `origin/feat-best`

## Current Goal

The recent work focused on making the structured matcher more like a compile-time linear/dissect-style runner for stable log layouts, while keeping regex parity.

## What Was Implemented

### Regexp-side prefiltering

- Added `regex_prefilter.go`
- `regexp` fallback now supports:
  - anchored prefix rejection
  - literal-set rejection
  - required literal sequence rejection

This is wired through:

- `pattern.go`
- `grok.go`

### Structured matcher improvements

- Added required literal extraction for structured matchers
- Added wrapped-parser compaction for safe `literal + parser + literal` segments
- Re-exposed wrapped parser boundaries back into IR/slicing so earlier steps can still use them as cut points
- Fixed token slicing for:
  - `SPACE* + literal`
  - wrapped-next-step boundaries
  - `NOTSPACE`/numeric token cases where the old fast path silently failed and fell back to `regexp`

### Important effect

Two previously problematic cases now actually hit the structured fast matcher instead of failing and falling through to regex:

- `ElasticSearch_log`
- `Tomcat_Catalina_log`

## Validation Status

Latest verified state before writing this file:

- `go test ./...` passes
- Datakit full benchmark is back to `24/24` faster than `regexp`

## Key Benchmark Snapshot

Recent benchmark results:

- `ElasticSearch_log`
  - fast: about `631ns/op`, `64 B/op`, `1 alloc/op`
  - regexp: about `8010ns/op`, `146 B/op`, `2 allocs/op`

- `Tomcat_Catalina_log`
  - fast: about `278ns/op`, `80 B/op`, `1 alloc/op`
  - regexp: about `924ns/op`, `176 B/op`, `2 allocs/op`

- `Apache_error_log`
  - fast: about `563ns/op`
  - regexp: about `3407ns/op`

- `Consul_log`
  - fast: about `690ns/op`
  - regexp: about `9188ns/op`

- `MySQL_slow_log`
  - fast: about `1395ns/op`
  - regexp: about `2019ns/op`

## Important Tests Added/Used

Useful regression tests that reflect the current direction:

- `TestStructuredElasticSearchDefaultFixtureUsesFastMatcher`
- `TestStructuredTomcatCatalinaFixtureUsesFastMatcher`
- `TestStructuredWrappedParserStepMatchesRegexp`
- existing Datakit fixture parity tests
- existing fuzz targets in `fuzz_fastpath_test.go`

## Current Design Direction

The current code is moving toward:

1. stronger compile-time literal/atom prefiltering
2. more linear structured execution for anchored stable patterns
3. dissect-style compaction only when regex parity remains safe

The current approach is still built on top of `structuredStep`; there is not yet a separate standalone `anchored dissect runner`.

## If Continuing From Here

Recommended next steps:

1. Refresh `docs/perf/BENCHMARKS.md` with the latest post-fix numbers.
2. Re-run longer fuzz/parity passes after the latest wrapped-parser and slicing changes.
3. If more speed is needed, consider a dedicated `anchored linear runner` instead of continuing to increase `structuredStep` complexity.

## Files Most Relevant To Continue

- `structured_fastpath.go`
- `regex_prefilter.go`
- `grok.go`
- `pattern.go`
- `grok_test.go`
- `docs/perf/BENCHMARKS.md`
- `docs/perf/PERFORMANCE_NOTES.md`
