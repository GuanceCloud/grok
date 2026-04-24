# Production Fixture Master Comparison

This compares the current `feat-best` worktree against `/tmp/grok-master-compare`
at `master` using real Datakit collector fixture examples from
`testdata/datakit_pipeline_cases.json`.

The Datakit fixture benchmark is the closest in-repo proxy for production data:
each case comes from a collector pipeline and its example log line.

## Commands

```sh
go test -run '^$' -bench '^BenchmarkDatakitFixtures$' -benchmem -benchtime=500ms -count=3 .
```

Raw outputs from this run:

- current: `/tmp/grok-current-datakit.bench`
- master: `/tmp/grok-master-datakit.bench`

`Nginx_error_log2` was re-run after enabling the bare nginx error runner:

```sh
go test -run '^$' -bench '^BenchmarkDatakitFixtures/nginx/nginx/nginx/Nginx_error_log2' -benchmem -benchtime=500ms -count=5 .
```

Additional targeted re-runs after the generic linear-runner and matcher-set
bitset optimizations used:

```sh
go test -run '^$' -bench 'BenchmarkDatakitFixtures/.*/.*/.*/.*/fast$' -benchmem -benchtime=1s -count=2
go test -run '^$' -bench '^BenchmarkDatakitPipelineDispatch$' -benchmem -benchtime=1s -count=2
```

## Result Summary

- The original full comparison had `21/24` Datakit fixture cases faster than
  `master`; targeted re-runs after the follow-up fixes now put all known
  production fixtures ahead of `master`.
- Targeted re-runs after the latest generic work moved the previous near-parity
  cases (`Apache_access_log`, `RabbitMQ_log`) slightly ahead of `master`.
- `ElasticSearch_search_slow_log` now uses the intended specialized runner
  after matching the normalized `(?:...)` pattern form.
- Most complex backtracking cases now allocate once instead of twice because
  the structured change log is pooled.
- `MatcherSet.RunFirstTo` now uses a small-set bitset candidate path, so
  Datakit pipeline dispatch with a reusable result buffer reaches `0 allocs/op`
  for the measured small pipelines.

## Latest Incremental Measurements

These are current `feat-best` targeted re-runs after the generic linear runner
and matcher-set bitset changes.

| Case | current ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `apache/apache/apache/Apache_access_log` | `124.1-124.9` | `96` | `1` |
| `elasticsearch/.../ElasticSearch_search_slow_log` | `101.5-103.5` | `80` | `1` |
| `nginx/nginx/nginx/Nginx_access_log` | `168.9-169.1` | `144` | `1` |
| `rabbitmq/rabbitmq/rabbitmq/RabbitMQ_log` | `67.6-67.8` | `48` | `1` |
| `tomcat/tomcat/tomcat/Tomcat_access_log` | `123.3-125.3` | `144` | `1` |
| `tdengine/.../tdengine_log_200` | `216.8-221.2` | `64` | `1` |

Pipeline dispatch with reusable buffers:

| Pipeline | matcher_set_reuse ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `apache` | `97.6-105.3` | `0` | `0` |
| `elasticsearch` | `595.8-603.6` | `0` | `0` |
| `nginx` | `152.4-162.5` | `0` | `0` |
| `postgresql` | `792.3-829.8` | `0` | `0` |
| `tomcat` | `94.5-95.0` | `0` | `0` |

## Full Comparison

`current/master` below is an ns/op ratio. Values below `1.00x` are faster than
master.

| Fixture | current ns/op | master ns/op | current/master | current B/alloc | master B/alloc |
| --- | ---: | ---: | ---: | ---: | ---: |
| `apache/apache/apache/Apache_access_log` | `129.3` | `135.4` | `0.95x` | `96 B / 1` | `96 B / 1` |
| `apache/apache/apache/Apache_error_log` | `728.4` | `3575.7` | `0.20x` | `64 B / 1` | `160 B / 2` |
| `consul/consul/consul/Consul_log` | `729.7` | `15035.0` | `0.05x` | `64 B / 1` | `179 B / 2` |
| `dameng/dameng/dameng/dameng_log` | `62.3` | `343.4` | `0.18x` | `48 B / 1` | `48 B / 1` |
| `elasticsearch/elasticsearch/elasticsearch/ElasticSearch_index_slow_log` | `98.7` | `354.0` | `0.28x` | `80 B / 1` | `272 B / 2` |
| `elasticsearch/elasticsearch/elasticsearch/ElasticSearch_log` | `290.1` | `379.7` | `0.76x` | `64 B / 1` | `256 B / 2` |
| `elasticsearch/elasticsearch/elasticsearch/ElasticSearch_search_slow_log` | `102.5` | `451.2` | `0.23x` | `80 B / 1` | `272 B / 2` |
| `jenkins/jenkins/jenkins/Jenkins_log` | `82.5` | `89.5` | `0.92x` | `48 B / 1` | `48 B / 1` |
| `kafka/kafka/kafka/Kafka_log` | `88.9` | `163.6` | `0.54x` | `64 B / 1` | `64 B / 1` |
| `kingbase/kingbase/kingbase/Kingbase_log` | `70.1` | `424.2` | `0.17x` | `64 B / 1` | `64 B / 1` |
| `mysql/mysql/mysql/MySQL_log` | `76.2` | `88.2` | `0.86x` | `64 B / 1` | `64 B / 1` |
| `mysql/mysql/mysql/MySQL_slow_log` | `950.1` | `2119.3` | `0.45x` | `240 B / 1` | `658 B / 2` |
| `nginx/nginx/nginx/Nginx_access_log` | `173.5` | `203.3` | `0.85x` | `144 B / 1` | `144 B / 1` |
| `nginx/nginx/nginx/Nginx_error_log1` | `129.1` | `263.8` | `0.49x` | `160 B / 1` | `160 B / 1` |
| `nginx/nginx/nginx/Nginx_error_log2` | `67.4` | `137.5` | `0.49x` | `48 B / 1` | `48 B / 1` |
| `postgresql/postgresql/postgresql/PostgreSQL_log` | `864.0` | `5757.7` | `0.15x` | `128 B / 1` | `384 B / 2` |
| `rabbitmq/rabbitmq/rabbitmq/RabbitMQ_log` | `66.7` | `69.0` | `0.97x` | `48 B / 1` | `48 B / 1` |
| `redis/redis/redis/Redis_log` | `80.9` | `169.7` | `0.48x` | `80 B / 1` | `80 B / 1` |
| `solr/solr/solr/Solr_log` | `103.4` | `159.8` | `0.65x` | `64 B / 1` | `64 B / 1` |
| `sqlserver/sqlserver/sqlserver/SQLServer_log` | `67.9` | `81.4` | `0.83x` | `48 B / 1` | `48 B / 1` |
| `tdengine/tdengine/tdengine/tdengine_log_200` | `200.2` | `2861.7` | `0.07x` | `64 B / 1` | `144 B / 2` |
| `tdengine/tdengine/tdengine/tdengine_log_204` | `213.6` | `3737.0` | `0.06x` | `64 B / 1` | `144 B / 2` |
| `tomcat/tomcat/tomcat/Tomcat_Catalina_log` | `115.2` | `218.2` | `0.53x` | `80 B / 1` | `80 B / 1` |
| `tomcat/tomcat/tomcat/Tomcat_access_log` | `127.8` | `176.6` | `0.72x` | `144 B / 1` | `144 B / 1` |

## Remaining Optimization Targets

1. Short mostly-linear custom business logs can still be faster on master. A
   dedicated simple-linear opcode runner would be more disruptive but likely
   better than adding more pattern-specific runners.
2. `pipeline-go` should use `MatcherSet.RunFirstTo` with a per-worker reusable
   buffer to remove caller-side result allocation in multi-pattern pipelines.
