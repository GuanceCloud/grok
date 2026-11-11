package grok

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const regexpExecutionPattern = `%{TIMESTAMP_ISO8601:time} %{LOGLEVEL:status} %{NUMBER:pid:int} \[%{NOTSPACE:thread}\] \[%{DATA:logger}\] (?:%{TRACE:trace_id}%{SPACE}%{SPAN:span_id}%{SPACE}) %{DATA:tags} `

func compileRegexpExecution(tb testing.TB) *GrokRegexp {
	tb.Helper()
	patterns := CopyDefalutPatterns()
	patterns["TRACE"] = `[a-zA-Z0-9]*`
	patterns["SPAN"] = `[a-zA-Z0-9]*`
	storage, invalid := DenormalizePatternsFromMap(patterns)
	if len(invalid) != 0 {
		tb.Fatal(invalid)
	}
	g, err := CompilePattern(regexpExecutionPattern, PatternStorage{storage})
	if err != nil {
		tb.Fatal(err)
	}
	return g
}

func regexpExecutionInputs() []struct{ name, content string } {
	header := "2026-11-11 12:00:00.123 INFO 123 [worker-1] [example.Logger] abc123 def456 tag=value "
	prefix := "unrelated prefix [worker] [logger] trace span tags \n"
	return []struct{ name, content string }{
		{"normal", header + "- operation complete"},
		{"stack_16k", header + "- operation failed\n" + strings.Repeat("\tat example.Service.call(Service.java:42)\n", 400)},
		{"prefix_16k", strings.Repeat(prefix, 340) + header + "- operation complete"},
		{"mismatch_16k", strings.Repeat(prefix, 340)},
	}
}

func TestRegexpExecutionAndLegacyFastPathAPI(t *testing.T) {
	g := compileRegexpExecution(t)
	if !g.FastPathDisabled() {
		t.Fatal("structured matching must always be disabled")
	}
	g.DisableFastPath()
	if !g.FastPathDisabled() {
		t.Fatal("legacy disable call changed the execution policy")
	}
	reference := rawRegexpCopy(g)
	buf := make([]string, g.MatchCount())
	typedBuf := make([]any, g.MatchCount())
	for _, input := range regexpExecutionInputs() {
		for _, trim := range []bool{false, true} {
			name := input.name + "/raw"
			if trim {
				name = input.name + "/trimmed"
			}
			t.Run(name, func(t *testing.T) {
				meta := RunMeta{Path: RunPathFastPath, FallbackReason: FallbackBudgetExceeded, WorkUnits: 999}
				want, wantErr := reference.Run(input.content, trim)
				got, err := g.RunToWithMeta(input.content, trim, buf, &meta)
				if !reflect.DeepEqual(got, want) || !errors.Is(err, wantErr) {
					t.Fatalf("string capture mismatch: %v, %v; want %v, %v", got, err, want, wantErr)
				}
				if err == nil && &got[0] != &buf[0] {
					t.Fatal("string result buffer was not reused")
				}
				if meta.Path != RunPathRegexp || meta.FallbackReason != FallbackNone || meta.WorkUnits != 0 || meta.PatternHash != g.PatternHash() {
					t.Fatalf("unexpected regexp metadata: %+v", meta)
				}
				typedWant, wantErr := reference.RunWithTypeInfo(input.content, trim)
				typed, err := g.RunWithTypeInfoToWithMeta(input.content, trim, typedBuf, &meta)
				if !reflect.DeepEqual(typed, typedWant) || !errors.Is(err, wantErr) {
					t.Fatalf("typed capture mismatch: %v, %v; want %v, %v", typed, err, typedWant, wantErr)
				}
				if err == nil && &typed[0] != &typedBuf[0] {
					t.Fatal("typed result buffer was not reused")
				}
				if meta.Path != RunPathRegexp || meta.FallbackReason != FallbackNone || meta.WorkUnits != 0 {
					t.Fatalf("unexpected typed metadata: %+v", meta)
				}
			})
		}
	}
}

func TestLegacyRunMetadataConstants(t *testing.T) {
	for _, tc := range []struct {
		path  RunPath
		value uint8
		name  string
	}{
		{RunPathRegexp, 1, "regexp"}, {RunPathFastPath, 2, "fast_path"}, {RunPathFallback, 3, "fallback"},
	} {
		if uint8(tc.path) != tc.value || tc.path.String() != tc.name {
			t.Fatalf("legacy path changed: %v", tc)
		}
	}
	for i, tc := range []struct {
		reason FallbackReason
		name   string
	}{
		{FallbackNone, "none"}, {FallbackCompileRisk, "compile_risk"}, {FallbackBudgetExceeded, "budget_exceeded"},
		{FallbackFastPathDisabled, "fast_path_disabled"}, {FallbackFastPathMismatch, "fast_path_mismatch"},
	} {
		if int(tc.reason) != i || tc.reason.String() != tc.name {
			t.Fatalf("legacy fallback changed: %v", tc)
		}
	}
}

func BenchmarkRegexpExecution(b *testing.B) {
	for _, input := range regexpExecutionInputs() {
		b.Run(input.name, func(b *testing.B) {
			g := compileRegexpExecution(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := g.Run(input.content, true); err != nil && !errors.Is(err, ErrMismatch) {
					b.Fatal(err)
				}
			}
		})
	}
}
