# Master Comparison

This file compares `feat-best` against `/tmp/grok-master-compare` at `master`
using the same benchmark sources.

## Commands

```sh
go test -run '^$' -bench '^BenchmarkRealisticPatterns$' -benchmem -benchtime=500ms -count=1
go test -run '^$' -bench '^BenchmarkDatakitFixtures/(apache|consul|elasticsearch|nginx|postgresql|tdengine)' -benchmem -benchtime=150ms -count=1
```

For the master run, `realistic_patterns_test.go` was copied into the master
worktree so both branches used the same business-log examples.

## Realistic Business Logs

These examples are intentionally not collector-specific. They cover common
business/service logs: order APIs, payment workers, gateway access logs, Java
services, Python access logs, Kubernetes controllers, syslog, DB slow query
lines, optional trace fields, and message queue consumers.

| Case | feat-best fast | master fast | Result |
| --- | ---: | ---: | --- |
| business order logfmt | `290.6ns/op` | `202.8ns/op` | master faster |
| business payment worker | `291.2ns/op` | `243.0ns/op` | master faster |
| API gateway access | `194.2ns/op` | `147.7ns/op` | master faster |
| Java order service | `243.9ns/op` | `173.7ns/op` | master faster |
| Python gunicorn order access | `377.8ns/op` | `176.3ns/op` | master faster |
| Kubernetes controller runtime | `461.9ns/op` | `1920ns/op` | feat-best faster |
| Syslog auth failure | `243.5ns/op` | `4759ns/op` | feat-best faster |
| DB slow query | `184.1ns/op` | `84.69ns/op` | master faster |
| Optional trace worker | `187.8ns/op` | `131.0ns/op` | master faster |
| Message queue consumer | `231.9ns/op` | `173.3ns/op` | master faster |

Takeaway: the current branch is not uniformly faster than master for short
mostly-linear business logs, but the gap is now much narrower after the linear
commit optimizations for terminal greedy captures and deterministic optional
wrapped parsers. It still wins strongly on harder parser shapes.

## Datakit Real Fixture Subset

These are real examples from `testdata/datakit_pipeline_cases.json`.

| Fixture | feat-best fast | master fast | Result |
| --- | ---: | ---: | --- |
| Apache access | `127.3ns/op` | `151.6ns/op` | feat-best faster |
| Apache error | `781.4ns/op` | `3824ns/op` | feat-best faster |
| Consul | `802.6ns/op` | `12816ns/op` | feat-best faster |
| Elasticsearch index slow log | `102.8ns/op` | `636.2ns/op` | feat-best faster |
| Elasticsearch log | `322.2ns/op` | `293.7ns/op` | master faster |
| Elasticsearch search slow log | `101.5-103.5ns/op` | `454.3ns/op` | feat-best faster |
| Nginx access | `167.1ns/op` | `232.6ns/op` | feat-best faster |
| Nginx error log 1 | `141.4ns/op` | `321.3ns/op` | feat-best faster |
| Nginx error log 2 | `67.4ns/op` | `137.5ns/op` | feat-best faster |
| PostgreSQL | `973.6ns/op` | `6329ns/op` | feat-best faster |
| TDengine 200 | `261.8ns/op` | `3092ns/op` | feat-best faster |
| TDengine 204 | `275.6ns/op` | `4207ns/op` | feat-best faster |

Takeaway: the current branch is much better on the Datakit shapes that used to
fall back to regexp or expensive generic matching. The Elasticsearch search-slow
case now hits its specialized runner after matching the normalized `(?:...)`
pattern form.

## Current Judgment

The optimization is strong for Datakit-like collector logs, especially complex
bracketed, syslog, database, and slow-log shapes. It is not yet general enough
to claim a blanket improvement across arbitrary business logs.

Before merging broadly, the next target should be the remaining simple
mostly-linear service logs where master remains faster:

- logfmt-style business events
- simple gateway access lines
- short message queue consumer logs

These cases need either more generic linear commit rules or a better
compile-time choice to keep master-like execution for simple patterns while
retaining the new Datakit wins.
