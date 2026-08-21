package grok

import (
	"strings"
	"testing"
)

func TestCompilePatternScoreIRVerdict(t *testing.T) {
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

	cases := []struct {
		name    string
		pattern string
		want    patternScoreVerdict
	}{
		{
			name:    "simple structured",
			pattern: `%{INT:code:int} %{WORD:name}`,
			want:    patternScoreFastPathPreferred,
		},
		{
			name:    "bounded greedy tail",
			pattern: `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`,
			want:    patternScoreFastPathPreferred,
		},
		{
			name:    "raw regex with good prefilter",
			pattern: `^foo.*bar$`,
			want:    patternScorePrefilterOnly,
		},
		{
			name:    "greedy only regexp",
			pattern: `%{GREEDYDATA:a}(?:%{GREEDYDATA:b})?`,
			want:    patternScoreRegexpPreferred,
		},
		{
			name:    "postgres style complex optional",
			pattern: `%{log_date:time}%{SPACE}\[%{INT:process_id}\]%{SPACE}(%{WORD:db_name}?%{SPACE}%{application_name}%{SPACE}%{USER:user}?%{SPACE}%{remote_host}%{SPACE})?%{session_id:session_id}%{SPACE}(%{status:status}:)?`,
			want:    patternScorePrefilterOnly,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			score, err := compilePatternScoreIR(tc.pattern, PatternStorage{denorm})
			if err != nil {
				t.Fatal(err)
			}
			if score.Verdict != tc.want {
				t.Fatalf("compilePatternScoreIR(%q).Verdict = %q, want %q", tc.pattern, score.Verdict, tc.want)
			}
			if len(score.Reasons) == 0 {
				t.Fatalf("expected non-empty reasons for %q", tc.pattern)
			}
		})
	}
}

func TestDumpPatternScore(t *testing.T) {
	got, err := dumpPatternScore(`%{INT:code:int} %{WORD:name}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	wantLines := []string{
		"PatternScore",
		`verdict="fast-path preferred"`,
		`prefix=""`,
		`suffix=""`,
		`min_match_length=3`,
		`  deterministic_delimiter_count=1`,
		`  primitive_ref_count=2`,
		`  wide_ref_count=0`,
		"score favors structured execution but does not change runtime behavior",
	}

	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Fatalf("dumpPatternScore missing %q\nfull output:\n%s", want, got)
		}
	}
}
