package grok

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegexpFilterEnabledForHighCapturePattern(t *testing.T) {
	g := compileRawRegexpForTest(t, `^prefix=(?P<a>[A-Z]+) code=(?P<b>[0-9]+) region=(?P<c>[a-z]+) trace=(?P<d>[a-z0-9-]+)$`)

	if g.filter == nil {
		t.Fatal("expected regexp filter to be enabled")
	}
	if g.filter.captureCount != 4 {
		t.Fatalf("filter capture count = %d, want 4", g.filter.captureCount)
	}
	if strings.Contains(g.filter.expr, "?P<") {
		t.Fatalf("filter expression still contains captures: %q", g.filter.expr)
	}
}

func TestRegexpFilterDisabledForLowCapturePattern(t *testing.T) {
	g := compileRawRegexpForTest(t, `^prefix=(?P<a>[A-Z]+) code=(?P<b>[0-9]+)$`)

	if g.filter != nil {
		t.Fatalf("expected regexp filter to stay disabled, got %q", g.filter.expr)
	}
}

func TestRegexpFilterPreservesFallbackSemantics(t *testing.T) {
	expr := `^prefix=(?P<a>[A-Z]+) code=(?P<b>[0-9]+) region=(?P<c>[a-z]+) trace=(?P<d>[a-z0-9-]+) status=(?P<e>ok|warn)$`

	withFilter := compileRawRegexpForTest(t, expr)
	if withFilter.filter == nil {
		t.Fatal("expected regexp filter to be enabled")
	}

	captureOnly := compileRawRegexpForTest(t, expr)
	captureOnly.filter = nil

	lines := []string{
		`prefix=API code=200 region=cn trace=req-42 status=ok`,
		`prefix=API code=200 region=cn trace=req-42 status=nope`,
		`prefix=API code=oops region=cn trace=req-42 status=ok`,
	}

	for _, line := range lines {
		filterRet, filterErr := withFilter.Run(line, true)
		captureRet, captureErr := captureOnly.Run(line, true)

		if !sameMatchError(filterErr, captureErr) {
			t.Fatalf("error mismatch for %q: filter=%v capture=%v", line, filterErr, captureErr)
		}
		if filterErr == nil && !reflect.DeepEqual(filterRet, captureRet) {
			t.Fatalf("result mismatch for %q: filter=%#v capture=%#v", line, filterRet, captureRet)
		}
	}
}

func BenchmarkRegexpFilterFallbackMismatch(b *testing.B) {
	expr := `^prefix=(?P<a>[A-Z]+) code=(?P<b>[0-9]+) region=(?P<c>[a-z]+) trace=(?P<d>[a-z0-9-]+) shard=(?P<e>[0-9]+) status=(?P<f>ok|warn)$`
	line := `prefix=API code=200 region=cn trace=req-42 shard=7 status=nope`

	withFilter := compileRawRegexpForTest(b, expr)
	if withFilter.filter == nil {
		b.Fatal("expected regexp filter to be enabled")
	}

	captureOnly := compileRawRegexpForTest(b, expr)
	captureOnly.filter = nil

	benchCases := []struct {
		name string
		re   *GrokRegexp
	}{
		{name: "filter+capture", re: withFilter},
		{name: "capture-only", re: captureOnly},
	}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := bc.re.Run(line, true)
				if !errors.Is(err, ErrMismatch) {
					b.Fatalf("expected mismatch, got %v", err)
				}
			}
		})
	}
}

func compileRawRegexpForTest(t testing.TB, expr string) *GrokRegexp {
	t.Helper()

	g, err := CompilePattern2(&GrokPattern{
		pattern:      expr,
		denormalized: expr,
		varbType:     map[string]string{},
	}, nil)
	if err != nil {
		t.Fatalf("compile pattern %q: %v", expr, err)
	}
	return g
}
