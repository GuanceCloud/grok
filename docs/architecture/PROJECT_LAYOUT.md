# Project Layout

This repository stays as a single Go package on purpose. The current and
planned organization follows the execution pipeline rather than splitting into
many subpackages too early.

## Public Entry Points

- `CompilePattern` / `CompilePattern2`
- `GrokRegexp.Run`
- `GrokRegexp.RunTo`
- `GrokRegexp.RunWithTypeInfo`
- `GrokRegexp.RunWithTypeInfoTo`

For `pipeline-go`, the intended reusable-buffer typed entrypoint is:

- `GrokRegexp.RunWithTypeInfoTo`

This avoids per-call result allocation while preserving the current matching
and casting semantics.

## Current File Groups

### API and compile path

- [grok.go](../../grok.go): main runtime type, fallback execution, buffer pools
- [api_run.go](../../api_run.go): exported reusable-buffer entrypoints
- [pattern.go](../../pattern.go): pattern denormalization, regexp compilation cache

### Scan and filter path

- [atom_scanner.go](../../atom_scanner.go): root-package compatibility wrapper for the scan layer
- [internal/scan/scanner.go](../../internal/scan/scanner.go): shared atom scanner implementation
- [filter_ir.go](../../filter_ir.go): root-package compatibility wrapper for filter IR compilation
- [internal/filter/ir.go](../../internal/filter/ir.go): compiled filter IR implementation
- [regex_filter.go](../../regex_filter.go): root-package compatibility wrapper for regexp filter compilation
- [internal/filter/regex.go](../../internal/filter/regex.go): regexp filter implementation
- [regex_prefilter.go](../../regex_prefilter.go): single-pattern literal/atom prefilter
- [matcher_set.go](../../matcher_set.go): multi-pattern candidate dispatch
- [multi_pattern_filter.go](../../multi_pattern_filter.go): root-package compatibility wrapper for the multi-pattern backend
- [internal/backend/backend.go](../../internal/backend/backend.go): backend abstraction
- [internal/backend/hyperscan_disabled.go](../../internal/backend/hyperscan_disabled.go): default stub backend
- [internal/backend/hyperscan_enabled.go](../../internal/backend/hyperscan_enabled.go): build-tag placeholder for future backend
- [hyperscan_disabled.go](../../hyperscan_disabled.go): root-package disabled-build constant wrapper
- [hyperscan_enabled.go](../../hyperscan_enabled.go): root-package enabled-build constant wrapper

### Match path

- [structured_fastpath.go](../../structured_fastpath.go): structured matcher, parser logic, fast path compilation
- [anchored_dissect.go](../../anchored_dissect.go): root-package compatibility wrapper for the anchored runner
- [internal/match/anchored.go](../../internal/match/anchored.go): anchored linear/dissect-style runner implementation
- [internal/match/model.go](../../internal/match/model.go): shared match-layer model types and enums
- [internal/match/ir.go](../../internal/match/ir.go): shared match-layer IR helpers
- [matcher_helpers.go](../../matcher_helpers.go): token validators and parser helpers

### Validation and performance

- [grok_test.go](../../grok_test.go): unit tests and micro-benchmarks
- [mutation_parity_test.go](../../mutation_parity_test.go): fast-path vs regexp parity checks
- [fuzz_fastpath_test.go](../../fuzz_fastpath_test.go): fuzz parity targets
- [datakit_pipeline_test.go](../../datakit_pipeline_test.go): real pipeline fixtures and dispatch benchmarks
- [common_patterns_test.go](../../common_patterns_test.go): composed-pattern coverage
- [docs/perf/BENCHMARKS.md](../perf/BENCHMARKS.md): benchmark snapshots
- [docs/perf/PERFORMANCE_NOTES.md](../perf/PERFORMANCE_NOTES.md): optimization conclusions and roadmap

### Repository docs

- [README.md](../../README.md): package overview and public entrypoints
- [docs/context/CURRENT_CONTEXT.md](../context/CURRENT_CONTEXT.md): local handoff note for ongoing work

## Near-Term Direction

The intended execution layering is:

1. `scan`
2. `filter`
3. `match`
4. `capture`

That translates to the following roadmap:

1. Build a shared `atom scanner` and move `MatcherSet` off repeated `strings.Contains`.
2. Compile current prefilter logic into a dedicated `filter IR`.
3. Keep expanding `anchored dissect runner` for safe linear patterns.
4. Treat Hyperscan and future SIMD/JIT work as optional backends under the scan/filter layer.

## Directory Rules

- Keep the runtime code in the root package until there is a strong reason to split it into subpackages.
- When a runtime layer is stable enough to isolate, move the implementation under `internal/<layer>/` and keep a thin root-package compatibility wrapper if callers or tests still expect package-local names.
- Add new files by concern, not by collector or benchmark.
- Put Markdown docs under `docs/` instead of the repository root:
  - `docs/perf/` for benchmark/result notes
  - `docs/architecture/` for layout/design notes
  - `docs/context/` for local handoff notes
- Prefer file names that match the layer they belong to:
  - `api_*`
  - `*_filter`
  - `*_scanner`
  - `*_matcher`
  - `*_dissect`
  - `*_test`
