package grok

import (
	"strings"
	"testing"
)

type observationCatalogCase struct {
	name    string
	pattern string
}

func loadObservationCatalog(t testing.TB) ([]observationCatalogCase, PatternStorage) {
	t.Helper()

	patterns := CopyDefalutPatterns()
	patterns["log_date"] = `%{YEAR}-%{MONTHNUM}-%{MONTHDAY}%{SPACE}%{HOUR}:%{MINUTE}:%{SECOND}%{SPACE}(?:CST|UTC)`
	patterns["status"] = `(LOG|ERROR|FATAL|PANIC|WARNING|NOTICE|INFO)`
	patterns["session_id"] = `([.0-9a-z]*)`
	patterns["application_name"] = `(\[%{GREEDYDATA:application_name}?\])`
	patterns["remote_host"] = `(\[\[?%{HOST:remote_host}?\]?\])`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	return []observationCatalogCase{
		{name: "simple_structured", pattern: `%{INT:code:int} %{WORD:name}`},
		{name: "bounded_greedy_tail", pattern: `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`},
		{name: "postgres_complex", pattern: `%{log_date:time}%{SPACE}\[%{INT:process_id}\]%{SPACE}(%{WORD:db_name}?%{SPACE}%{application_name}%{SPACE}%{USER:user}?%{SPACE}%{remote_host}%{SPACE})?%{session_id:session_id}%{SPACE}(%{status:status}:)?`},
		{name: "raw_regex_prefilter", pattern: `^foo.*bar$`},
		{name: "httpd20_errorlog", pattern: `%{HTTPD20_ERRORLOG}`},
		{name: "apache_error_pid_only", pattern: `\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{INT:pid}\] `},
		{name: "leading_greedy_fallback", pattern: `%{GREEDYDATA:msg}`},
	}, PatternStorage{denorm}
}

func TestObservePatternDecision(t *testing.T) {
	catalog, storage := loadObservationCatalog(t)

	cases := []struct {
		name          string
		pattern       string
		wantCompiled  bool
		wantHeuristic bool
		wantVerdict   patternScoreVerdict
		wantDiverges  bool
	}{
		{
			name:          "simple structured align",
			pattern:       catalog[0].pattern,
			wantCompiled:  true,
			wantHeuristic: true,
			wantVerdict:   patternScoreFastPathPreferred,
			wantDiverges:  false,
		},
		{
			name:          "postgres complex heuristic now allows structured",
			pattern:       catalog[2].pattern,
			wantCompiled:  true,
			wantHeuristic: true,
			wantVerdict:   patternScorePrefilterOnly,
			wantDiverges:  true,
		},
		{
			name:          "leading greedy aligns on fallback",
			pattern:       catalog[6].pattern,
			wantCompiled:  true,
			wantHeuristic: false,
			wantVerdict:   patternScoreRegexpPreferred,
			wantDiverges:  false,
		},
		{
			name:          "raw regex prefilter only",
			pattern:       catalog[3].pattern,
			wantCompiled:  true,
			wantHeuristic: true,
			wantVerdict:   patternScorePrefilterOnly,
			wantDiverges:  true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			observation, err := observePatternDecision(tc.pattern, storage)
			if err != nil {
				t.Fatal(err)
			}
			if observation.StructuredCompiled != tc.wantCompiled {
				t.Fatalf("StructuredCompiled = %t, want %t", observation.StructuredCompiled, tc.wantCompiled)
			}
			if observation.StructuredHeuristicAllow != tc.wantHeuristic {
				t.Fatalf("StructuredHeuristicAllow = %t, want %t", observation.StructuredHeuristicAllow, tc.wantHeuristic)
			}
			if observation.Score.Verdict != tc.wantVerdict {
				t.Fatalf("Score.Verdict = %q, want %q", observation.Score.Verdict, tc.wantVerdict)
			}
			if observation.Diverges != tc.wantDiverges {
				t.Fatalf("Diverges = %t, want %t", observation.Diverges, tc.wantDiverges)
			}
			if observation.Diverges != (observation.StructuredHeuristicAllow != scorePrefersStructured(observation.Score)) {
				t.Fatal("Diverges should reflect score/runtime decision mismatch")
			}
		})
	}
}

func TestObservePatternDecisionCatalog(t *testing.T) {
	catalog, storage := loadObservationCatalog(t)

	var divergenceCount int
	var heuristicAllowCount int
	var heuristicFallbackCount int
	var scoreFastPathCount int
	var scorePrefilterCount int

	for _, item := range catalog {
		observation, err := observePatternDecision(item.pattern, storage)
		if err != nil {
			t.Fatalf("%s: %v", item.name, err)
		}
		if observation.Diverges {
			divergenceCount++
		}
		if observation.StructuredHeuristicAllow {
			heuristicAllowCount++
		} else {
			heuristicFallbackCount++
		}
		switch observation.Score.Verdict {
		case patternScoreFastPathPreferred:
			scoreFastPathCount++
		case patternScorePrefilterOnly:
			scorePrefilterCount++
		}
	}

	if divergenceCount < 1 {
		t.Fatal("expected at least one stable score/heuristic divergence in observation catalog")
	}
	if heuristicAllowCount < 1 || heuristicFallbackCount < 1 {
		t.Fatal("expected catalog to cover both heuristic-allow and heuristic-fallback cases")
	}
	if scoreFastPathCount < 1 || scorePrefilterCount < 1 {
		t.Fatal("expected catalog to cover both fast-path-preferred and prefilter-only score verdicts")
	}
}

func TestDumpPatternDecisionObservation(t *testing.T) {
	got, err := dumpPatternDecisionObservation(`%{INT:code:int} %{WORD:name}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	wantLines := []string{
		"PatternDecisionObservation",
		`score_verdict="fast-path preferred"`,
		`structured_compiled=true`,
		`heuristic_allows_fast_path=true`,
		`diverges=false`,
		`  parser_count=2`,
		`  optional_count=0`,
		"score_reasons",
	}

	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Fatalf("dumpPatternDecisionObservation missing %q\nfull output:\n%s", want, got)
		}
	}
}

func BenchmarkObservePatternDecisionCatalog(b *testing.B) {
	catalog, storage := loadObservationCatalog(b)

	for _, item := range catalog {
		item := item
		b.Run(item.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				observation, err := observePatternDecision(item.pattern, storage)
				if err != nil {
					b.Fatal(err)
				}
				if observation.Pattern == "" {
					b.Fatal("empty observation pattern")
				}
			}
		})
	}
}
