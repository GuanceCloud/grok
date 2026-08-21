# Datakit Pipeline Fixtures

`datakit_pipeline_cases.json` is a snapshot of real collector pipeline fixtures scanned from the local Datakit source tree:

- source root: `/home/vircoys/go/src/gitlab.jiagouyun.com/cloudcare-tools/datakit/internal/plugins/inputs/`
- selection rule: collectors that define both `PipelineConfig()` and `LogExamples()`

Each entry contains:

- `collector`: collector/package name
- `package_dir`: Datakit package directory
- `pipeline_source`: file that defines `PipelineConfig()`
- `example_source`: file that defines `LogExamples()`
- `pipelines`: default pipeline script map returned by the collector
- `examples`: sample log map returned by the collector

Use these fixtures for regression tests and benchmarks against real-world Grok patterns instead of synthetic-only cases.
