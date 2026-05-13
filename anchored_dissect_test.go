package grok

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnchoredDissectRunnerAttached(t *testing.T) {
	current, err := CompilePattern(`^foo %{WORD:name} bar %{INT:id}$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected structured fast matcher")
	}
	if current.fastMatcher.anchoredRunner == nil {
		t.Fatal("expected anchored dissect runner")
	}

	regexpOnly, err := CompilePattern(`^foo %{WORD:name} bar %{INT:id}$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	line := "foo alice bar 42"
	fastRet, fastErr := current.Run(line, true)
	regexpRet, regexpErr := regexpOnly.Run(line, true)
	assert.Equal(t, regexpErr, fastErr)
	assert.Equal(t, regexpRet, fastRet)
}

func TestAnchoredDissectRunnerRejectsBacktrackingPatterns(t *testing.T) {
	current, err := CompilePattern(`^foo %{GREEDYDATA:name} bar %{INT:id} baz$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected structured fast matcher")
	}
	if current.fastMatcher.anchoredRunner != nil {
		t.Fatal("expected backtracking pattern to skip anchored dissect runner")
	}
}

func BenchmarkRunAnchoredDissectRunner(b *testing.B) {
	g, err := CompilePattern(`^foo %{WORD:name} bar %{INT:id}$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	if g.fastMatcher == nil || g.fastMatcher.anchoredRunner == nil {
		b.Fatal("expected anchored dissect runner")
	}

	regexpOnly, err := CompilePattern(`^foo %{WORD:name} bar %{INT:id}$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	line := "foo alice bar 42"

	b.Run("fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := g.Run(line, true); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("regexp", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := regexpOnly.Run(line, true); err != nil {
				b.Fatal(err)
			}
		}
	})
}
