package grok

import "testing"

func TestDumpGrokIR(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "simple refs",
			pattern: `%{INT:code:int} %{WORD:name}`,
			want: "GrokIR\n" +
				"Sequence\n" +
				"  Ref syntax=\"INT\" alias=\"code\" type=\"int\"\n" +
				"  Literal \" \"\n" +
				"  Ref syntax=\"WORD\" alias=\"name\" type=\"\"\n",
		},
		{
			name:    "optional capture alternate",
			pattern: `(%{WORD:db_name}|%{NUMBER:id})?`,
			want: "GrokIR\n" +
				"Repeat min=0 max=1 greedy=true\n" +
				"  Group kind=\"capture\"\n" +
				"    Alternate\n" +
				"      Ref syntax=\"WORD\" alias=\"db_name\" type=\"\"\n" +
				"      Ref syntax=\"NUMBER\" alias=\"id\" type=\"\"\n",
		},
		{
			name:    "char class separators",
			pattern: `%{YEAR}[./]%{MONTHNUM} %{TIME}`,
			want: "GrokIR\n" +
				"Sequence\n" +
				"  Ref syntax=\"YEAR\" alias=\"\" type=\"\"\n" +
				"  RawRegex \"[./]\"\n" +
				"  Ref syntax=\"MONTHNUM\" alias=\"\" type=\"\"\n" +
				"  Literal \" \"\n" +
				"  Ref syntax=\"TIME\" alias=\"\" type=\"\"\n",
		},
		{
			name:    "special group and nongreedy repeat",
			pattern: `(?s)%{DATA:msg}.*?`,
			want: "GrokIR\n" +
				"Sequence\n" +
				"  RawRegex \"(?s)\"\n" +
				"  Ref syntax=\"DATA\" alias=\"msg\" type=\"\"\n" +
				"  Repeat min=0 max=inf greedy=false\n" +
				"    RawRegex \".\"\n",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := dumpGrokIR(tc.pattern)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("dumpGrokIR(%q)\n got:\n%s\nwant:\n%s", tc.pattern, got, tc.want)
			}
		})
	}
}

func TestDumpMatchIR(t *testing.T) {
	pattern := `(%{WORD:db_name}|%{NUMBER:id})?`
	got, err := dumpMatchIR(pattern)
	if err != nil {
		t.Fatal(err)
	}

	want := "MatchIR\n" +
		"Repeat min=0 max=1 greedy=true\n" +
		"  Alternate\n" +
		"    Ref syntax=\"WORD\" alias=\"db_name\" type=\"\"\n" +
		"    Ref syntax=\"NUMBER\" alias=\"id\" type=\"\"\n"

	if got != want {
		t.Fatalf("dumpMatchIR(%q)\n got:\n%s\nwant:\n%s", pattern, got, want)
	}
}

func TestCompileGrokIRRepresentativePatterns(t *testing.T) {
	patterns := []string{
		`^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[i.s.s.(query|fetch)%{SPACE}\] (\[%{HOSTNAME:nodeId}\] )?\[%{NOTSPACE:index}\]\[%{INT}\] took\[.*\], took_millis\[%{INT:duration}\].*`,
		`\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{INT:pid}\] `,
		`%{log_date:time}%{SPACE}\[%{INT:process_id}\]%{SPACE}(%{WORD:db_name}?%{SPACE}%{application_name}%{SPACE}%{USER:user}?%{SPACE}%{remote_host}%{SPACE})?%{session_id:session_id}%{SPACE}(%{status:status}:)?`,
	}

	for _, pattern := range patterns {
		if _, err := compileGrokIR(pattern); err != nil {
			t.Fatalf("compileGrokIR(%q): %v", pattern, err)
		}
		if _, err := compileMatchIR(pattern); err != nil {
			t.Fatalf("compileMatchIR(%q): %v", pattern, err)
		}
	}
}

func TestCompilePatternPrefilterIR(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["date2"] = `%{YEAR}[./]%{MONTHNUM}[./]%{MONTHDAY} %{TIME}`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	cases := []struct {
		name          string
		pattern       string
		wantPrefix    string
		wantSuffix    string
		wantMinLength int
	}{
		{
			name:          "literal prefix and refs",
			pattern:       `\[%{HTTPDATE:time}\] %{WORD:method}`,
			wantPrefix:    "[",
			wantSuffix:    "",
			wantMinLength: 20,
		},
		{
			name:          "leading ref no literal prefix",
			pattern:       `%{INT:code:int} %{WORD:name}`,
			wantPrefix:    "",
			wantSuffix:    "",
			wantMinLength: 3,
		},
		{
			name:          "custom nested ref min length",
			pattern:       `%{date2:time}`,
			wantPrefix:    "",
			wantSuffix:    "",
			wantMinLength: 13,
		},
		{
			name:          "optional block drops guaranteed prefix",
			pattern:       `(%{WORD:name}:)?%{INT:id}`,
			wantPrefix:    "",
			wantSuffix:    "",
			wantMinLength: 1,
		},
		{
			name:          "single byte suffix survives filtering",
			pattern:       `\[%{HTTPDATE:time}\]$`,
			wantPrefix:    "[",
			wantSuffix:    "]",
			wantMinLength: 18,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			prefix, suffix, _, minLength := compilePatternPrefilterIR(tc.pattern, PatternStorage{denorm})
			if prefix != tc.wantPrefix || suffix != tc.wantSuffix || minLength != tc.wantMinLength {
				t.Fatalf("compilePatternPrefilterIR(%q) = (%q, %q, %d), want (%q, %q, %d)", tc.pattern, prefix, suffix, minLength, tc.wantPrefix, tc.wantSuffix, tc.wantMinLength)
			}
		})
	}
}

func BenchmarkCompilePatternIR(b *testing.B) {
	patterns := CopyDefalutPatterns()
	patterns["log_date"] = `%{YEAR}-%{MONTHNUM}-%{MONTHDAY}%{SPACE}%{HOUR}:%{MINUTE}:%{SECOND}%{SPACE}(?:CST|UTC)`
	patterns["status"] = `(LOG|ERROR|FATAL|PANIC|WARNING|NOTICE|INFO)`
	patterns["session_id"] = `([.0-9a-z]*)`
	patterns["application_name"] = `(\[%{GREEDYDATA:application_name}?\])`
	patterns["remote_host"] = `(\[\[?%{HOST:remote_host}?\]?\])`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	cases := []struct {
		name    string
		pattern string
	}{
		{
			name:    "simple",
			pattern: `%{INT:code:int} %{WORD:name}`,
		},
		{
			name:    "httpd20_errorlog",
			pattern: `%{HTTPD20_ERRORLOG}`,
		},
		{
			name:    "postgres_complex",
			pattern: `%{log_date:time}%{SPACE}\[%{INT:process_id}\]%{SPACE}(%{WORD:db_name}?%{SPACE}%{application_name}%{SPACE}%{USER:user}?%{SPACE}%{remote_host}%{SPACE})?%{session_id:session_id}%{SPACE}(%{status:status}:)?`,
		},
	}

	for _, tc := range cases {
		tc := tc
		b.Run(tc.name+"/grok_ir", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				compiled, err := compileGrokIR(tc.pattern)
				if err != nil {
					b.Fatal(err)
				}
				if compiled == nil || compiled.root == nil {
					b.Fatal("nil grok ir")
				}
			}
		})

		b.Run(tc.name+"/match_ir", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				compiled, err := compileMatchIR(tc.pattern)
				if err != nil {
					b.Fatal(err)
				}
				if compiled == nil || compiled.root == nil {
					b.Fatal("nil match ir")
				}
			}
		})

		b.Run(tc.name+"/prefilter_ir", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				prefix, suffix, _, minLength := compilePatternPrefilterIR(tc.pattern, PatternStorage{denorm})
				if minLength < 0 {
					b.Fatal("invalid min length")
				}
				_ = prefix
				_ = suffix
			}
		})

		b.Run(tc.name+"/score_ir", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				score, err := compilePatternScoreIR(tc.pattern, PatternStorage{denorm})
				if err != nil {
					b.Fatal(err)
				}
				if score.Verdict == "" {
					b.Fatal("empty verdict")
				}
			}
		})
	}
}
