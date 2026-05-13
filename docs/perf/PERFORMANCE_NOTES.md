# Performance Notes

This document records the current conclusions for the performance work on
`github.com/GuanceCloud/grok`. See [BENCHMARKS.md](./BENCHMARKS.md) for the
latest measured numbers.

## Current Status

As of `2026-04-24`, the implementation has three useful performance layers:

- structured fast paths for common grok fragments and stable log formats
- compile-time linearization for terminal greedy captures and deterministic
  wrapped optional fields
- pooled backtracking change logs, avoiding a per-match heap allocation on
  complex structured patterns
- compile-time parser backtracking flags, avoiding repeated per-step
  backtracking classification while matching
- a lightweight regexp prefilter for obvious fallback mismatches
- `MatcherSet` dispatch for pipeline-style multi-pattern matching

The current Datakit fixture headline is:

- `24/24` fixture cases are faster than regexp fallback
- `ElasticSearch_log` is no longer a slow-path outlier
- the main remaining production concern is caller-side allocation and dispatch,
  not single-pattern regexp fallback

## API Guidance

For single compiled patterns, callers that run in a tight loop should prefer the
existing reusable-buffer APIs:

- `GrokRegexp.RunTo`
- `GrokRegexp.RunWithTypeInfoTo`
- `GrokRegexp.MatchCount`

For Datakit / `pipeline-go` style pipelines with multiple grok patterns, callers
should prefer:

- `NewMatcherSet`
- `MatcherSet.MatchCount`
- `MatcherSet.RunFirstTo`

Example:

```go
buf := make([]string, 0, set.MatchCount())
id, values, err := set.RunFirstTo(line, true, buf)
```

This avoids the per-line result slice allocation for most measured pipeline
cases and keeps the semantic confirmation path unchanged.

## Review Conclusions

1. The broad single-pattern wins are already captured by structured runners and
   compile-time linearization rules.
2. Adding more one-off runners should now require a concrete fixture or
   production profile win.
3. Unanchored patterns with writable optional groups are not universally good
   structured-search candidates. Complex cases now get a position-zero fast
   attempt before regexp fallback; very small risky cases skip structured
   matching unless they can be proven linear.
4. The next high-value area is reducing dispatch and allocation overhead for
   multi-pattern pipelines.
5. Correctness remains the gating condition: structured fast paths must keep
   parity with regexp fallback.

## What Fuzzing Found

Fuzzing found real semantic drift that fixed fixtures did not catch. Important
bug classes included:

- case-insensitive literal regexes incorrectly using a case-sensitive prefilter
- widened numeric semantics for `POSINT`, `NONNEGINT`, `INT`, and `NUMBER`
- loose `SECOND` and `TIME` parsing around trailing digits
- missing hostname and month-name boundary checks
- over-permissive IPv6-like host parsing
- `NOTSPACE` followed by a literal needing regex-like greedy backtracking
- `GREEDYDATA` / `DATA` structured segments incorrectly accepting `\n`
  when regexp `.` would not
- month-name helpers accepting non-default aliases such as `Ma`, `Mr`, and
  `Ot` that the shipped `MONTH` pattern rejects
- the Tomcat Catalina runner accepting `MONTHDAY=0` even though the default
  `MONTHDAY` pattern starts at `1`

Future fast paths should include parity tests or fuzz targets before being
enabled broadly.

## Next Work

The useful follow-up work is now narrower:

1. Integrate `MatcherSet.RunFirstTo` in `pipeline-go` and measure there.
2. Remove the remaining dispatcher allocation for large indexed matcher sets.
3. Continue narrowing master gaps for simple mostly-linear business logs with
   generic compile-time rules, not pattern-name-specific runners.
4. Use pprof from real pipeline workloads before adding more specialized
   runners.
5. Keep optional external regex engines framed as filtering backends only, not
   capture engines.

## References

- RE2 README: <https://github.com/google/re2>
- RE2 filtered matching API: <https://docs.rs/re2/latest/re2/filtered/struct.FilteredRE2.html>
- Aho-Corasick documentation: <https://docs.rs/aho-corasick/latest/aho_corasick/>
- Regex internals and prefilter strategy: <https://burntsushi.net/regex-internals/>
- Elastic dissect guidance: <https://www.elastic.co/docs/solutions/observability/streams/management/extract/dissect>
- Elastic integration tips: <https://www.elastic.co/guide/en/integrations-developer/current/tips-for-building.html>
- Hyperscan compilation documentation: <https://intel.github.io/hyperscan/dev-reference/compilation.html>
