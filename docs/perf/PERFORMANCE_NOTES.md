# Performance Notes

This document records the current performance conclusions, the next-stage optimization plan, and the external references used to shape that plan.

## Status

As of `2026-04-22`, the current implementation has two meaningful performance layers:

- a structured fast path for common grok fragments and fixture-heavy log formats
- a lightweight `regexp` prefilter that can reject obvious mismatches before running the fallback engine

The current benchmark headline is:

- `23/24` Datakit fixture cases are faster than the shipped `regexp` fallback path
- the remaining slower case is `ElasticSearch_log`
- the previously slow `Apache error`, `Consul`, and `MySQL slow log` fixtures are now all faster than fallback

Recent reruns after the latest fuzz fixes were roughly:

| Fixture | Fast ns/op | Regexp ns/op | Speedup |
| --- | ---: | ---: | ---: |
| Apache error | `~661` | `~3330` | `~5x` |
| Consul | `~780` | `~9840` | `~12.5x` |
| MySQL slow log | `~1520` | `~1970` | `~1.3x` |
| ElasticSearch log | `~8860` | `~8400` | `0.95x` |

The `regexp` prefilter benchmark is also materially positive on mismatch-heavy cases:

- literal-set mismatch with prefilter: `~8.8ns/op`
- literal-set mismatch without prefilter: `~26.8ns/op`

## Review Conclusions

After multiple review passes, parity tests, and longer fuzz campaigns, the conclusion is:

1. There is no obvious small local change left that is likely to produce another broad `2x` class win.
2. The current design already captures the most valuable near-term optimizations:
   - literal-aware prefiltering
   - structured parsing for stable grok fragments
   - selective backtracking only where regex semantics require it
3. Continuing to add more one-off fast-path heuristics is still possible, but the expected return has dropped. Most remaining gains are likely incremental, not architectural.
4. The next meaningful step is not "more matcher tricks"; it is better dispatch and better specialization at the pipeline level.

## What We Learned From Fuzzing

Fuzzing was not just a confidence pass. It found real semantic drift that the fixed fixture corpus did not catch.

The notable bug classes found in this round included:

- case-insensitive literal regexes incorrectly using a case-sensitive prefilter
- widened numeric semantics for `POSINT`, `NONNEGINT`, `INT`, and `NUMBER`
- loose `SECOND` and `TIME` parsing, especially around trailing digits
- missing `\b`-style hostname and month-name boundary checks
- over-permissive IPv6-like host parsing
- `NOTSPACE` followed by a literal needing regex-like greedy backtracking

That result changed the optimization bar for future work:

- correctness has to stay ahead of throughput
- new fast paths should be justified by fixture wins, not by elegance alone
- anything more ambitious than the current matcher layer should be designed with parity verification built in

## Next-Stage Plan

The next stage should prioritize architecture changes that can move many patterns at once.

### 1. Global atom/prefilter dispatcher

Build a compile-time index of stable literals or atoms across many patterns, then use that index to cheaply prune the candidate set before running full matching.

Why this is promising:

- it targets the common production case where many patterns exist but each log line only matches very few
- it follows the same general direction as `FilteredRE2` and `aho-corasick` style prefiltering
- it is more likely to produce step-function gains than adding one more specialized primitive

Suggested shape:

- extract a small set of required literals/atoms from each compiled pattern
- group patterns by cheap discriminators such as anchored prefix, literal set, or token family
- run a dispatcher before `Run`/`RunWithTypeInfo` to narrow candidates
- keep the final semantic engine unchanged: structured matcher first, fallback regexp second

Concrete draft:

- compile each pattern into a small `patternAtomProfile`:
  - `anchoredPrefix`
  - `exactLiterals`
  - `requiredAtoms`
  - `minWidth`
  - `hasStructuredFastPath`
- build one shared `atomDispatcher` for a pattern family:
  - `prefixBuckets map[string][]int`
  - `exactBuckets map[int]map[string][]int`
  - `atomPostings map[string][]int`
  - `fallback []int` for patterns with weak extraction
- dispatch in two stages:
  - stage 1: cheap rejects from prefix / exact-length buckets
  - stage 2: ordered-atom check to produce a small candidate list
- run the existing engines only for the remaining candidates:
  - `fastMatcher` first when present
  - current `regexp` path second

Guardrails for implementation:

- do not change capture semantics inside the dispatcher
- preserve current single-pattern APIs and add the dispatcher as an opt-in multi-pattern layer
- measure candidate-count reduction, not just wall-clock time
- fuzz against the current per-pattern execution path before enabling it by default

### 2. Dissect-style delimiter parser plus grok hybrid

Introduce a delimiter-oriented parser for stable log skeletons, then hand the irregular tail or subfields to grok only where regex power is actually needed.

Why this is promising:

- many current fixtures are closer to delimiter splitting than true regex matching
- stable access logs, syslog-like lines, and many database logs are natural fits
- this matches Elastic's documented recommendation to use `dissect` when structure is stable and combine it with `grok` for irregular sections

Suggested shape:

- add a compile-time detector for patterns that are mostly literal delimiters plus typed fields
- compile those into a dissect-like runner
- optionally allow a hybrid mode where a dissect segment feeds a smaller grok segment

### 3. Optional external high-throughput regex backend

An external engine such as Hyperscan is only worth considering as an optional filtering backend, not as the primary capture engine.

Why it is lower priority:

- the integration cost is high
- Go deployment becomes more complex
- the biggest benefit is usually high-throughput multi-pattern scanning, not rich capture extraction

If explored, it should be framed as:

- optional
- match-or-candidate filtering only
- never the sole correctness path

## Lower-Priority Ideas

These are still valid ideas, but they do not currently look like the best next investment:

- adding more one-off structured primitives without a broader dispatch layer
- replacing Go `regexp` with another RE2 wrapper and expecting large wins by itself
- pushing more aggressive heuristics into the matcher without parity evidence
- adding caches for matched input strings in the hot path

## Proposed Execution Order

1. Prototype a global atom extraction and candidate dispatcher on top of the existing compiler output.
2. Add a small dissect-style prototype for one or two fixture families with the most stable delimiters.
3. Re-run the full Datakit benchmark and long fuzz parity suite before broadening either mechanism.
4. Only evaluate external engines after the in-process dispatcher and dissect experiments have real numbers.

## References

The following references were used to ground the current plan:

- RE2 README: linear-time guarantees, tradeoffs, and the point that RE2 is not intended to be the fastest engine in every case  
  <https://github.com/google/re2>
- RE2 filtered matching API, showing the general shape of multi-regex prefiltering  
  <https://docs.rs/re2/latest/re2/filtered/struct.FilteredRE2.html>
- Aho-Corasick documentation, including notes on internal prefilters and when they help  
  <https://docs.rs/aho-corasick/latest/aho_corasick/>
- Andrew Gallant's regex internals write-up, especially literal extraction and prefilter strategy  
  <https://burntsushi.net/regex-internals/>
- Elastic documentation for `dissect`, including the recommendation that it is faster than grok on stable formats  
  <https://www.elastic.co/docs/solutions/observability/streams/management/extract/dissect>
- Elastic integration guidance stating that `dissect` is usually `2-4x` faster than `grok` for suitable patterns  
  <https://www.elastic.co/guide/en/integrations-developer/current/tips-for-building.html>
- Elastic's `DISSECT` plus `GROK` hybrid guidance  
  <https://www.elastic.co/guide/en/elasticsearch/reference/8.19/esql-process-data-with-dissect-and-grok.html>
- Hyperscan compilation documentation for multi-pattern high-throughput scanning  
  <https://intel.github.io/hyperscan/dev-reference/compilation.html>

## Notes

- `BENCHMARKS.md` remains the detailed result snapshot for the current codebase.
- This document is intentionally about direction and architecture, not exhaustive benchmark tables.
- If future changes materially alter the headline result, update this file and `BENCHMARKS.md` together.
