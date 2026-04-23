package grok

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatcherSetCandidateIDsPrefixPrunesByAnchoredPrefix(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "foo", Pattern: `^foo.*bar$`},
		testMatcherSetPattern{ID: "zip", Pattern: `^zip.*zap$`},
	)

	assert.Equal(t, []string{"foo"}, ms.CandidateIDs("fooxbar"))
	assert.Equal(t, []string{"zip"}, ms.CandidateIDs("zipxxzap"))
	assert.Empty(t, ms.CandidateIDs("xxfooxbar"))
}

func TestMatcherSetCandidateIDsExactLiteralPrunes(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "ab", Pattern: `^(?:foo|bar)$`},
		testMatcherSetPattern{ID: "bc", Pattern: `^(?:bar|baz)$`},
	)

	assert.Equal(t, []string{"ab", "bc"}, ms.CandidateIDs("bar"))
	assert.Empty(t, ms.CandidateIDs("qux"))
	assert.Empty(t, ms.CandidateIDs("xxbarxx"))
}

func TestMatcherSetCandidateIDsRequiredAtomsPrune(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "baz", Pattern: `^foo.*bar.*baz$`},
		testMatcherSetPattern{ID: "qux", Pattern: `^foo.*bar.*qux$`},
	)

	assert.Equal(t, []string{"baz"}, ms.CandidateIDs("fooxbarzzbaz"))
	assert.Empty(t, ms.CandidateIDs("foobaz"))
	assert.Equal(t, []string{"baz"}, ms.CandidateIDs("fooquxbarbaz"))
}

func TestMatcherSetRunFirstUsesInOrderConfirmation(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "miss", Pattern: `^foo\d+baz$`},
		testMatcherSetPattern{ID: "hit", Pattern: `^foo(?P<num>\d+)bar$`},
	)

	id, ret, err := ms.RunFirst("foo42bar", true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hit", id)
	assert.Equal(t, []string{"42"}, ret)

	_, _, err = ms.RunFirst("bar42foo", true)
	assert.ErrorIs(t, err, ErrMismatch)
}

func TestMatcherSetCandidateIDsHandlesOverlappingAtoms(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "short", Pattern: `^svc.*trace=req.*accepted$`},
		testMatcherSetPattern{ID: "long", Pattern: `^svc.*trace=req-42.*accepted$`},
	)

	assert.Equal(t, []string{"short", "long"}, ms.CandidateIDs("svc trace=req-42 accepted"))
	assert.Equal(t, []string{"short"}, ms.CandidateIDs("svc trace=req-43 accepted"))
	assert.Empty(t, ms.CandidateIDs("svc trace=req-42 rejected"))
}

func TestMatcherSetPrefersLinearForSmallWeakSet(t *testing.T) {
	ms := newTestMatcherSet(t,
		testMatcherSetPattern{ID: "a", Pattern: `%{WORD:x} %{GREEDYDATA:y}`},
		testMatcherSetPattern{ID: "b", Pattern: `%{INT:x} %{GREEDYDATA:y}`},
		testMatcherSetPattern{ID: "c", Pattern: `%{DATA:x} %{LOGLEVEL:y}`},
		testMatcherSetPattern{ID: "d", Pattern: `%{DATA:x} %{GREEDYDATA:y}`},
		testMatcherSetPattern{ID: "e", Pattern: `%{NOTSPACE:x} %{GREEDYDATA:y}`},
	)

	if !ms.preferLinear {
		t.Fatal("expected weak small set to prefer linear execution")
	}
}

func TestMatcherSetUsesDispatcherForLargeAnchoredSet(t *testing.T) {
	patterns := make([]testMatcherSetPattern, 0, 9)
	for i := 0; i < 9; i++ {
		patterns = append(patterns, testMatcherSetPattern{
			ID:      fmt.Sprintf("id-%d", i),
			Pattern: fmt.Sprintf(`^svc%02d %%{GREEDYDATA:msg}$`, i),
		})
	}

	ms := newTestMatcherSet(t, patterns...)
	if ms.preferLinear {
		t.Fatal("expected anchored set to keep dispatcher enabled")
	}
}

func TestMatcherSetPrefersLinearForLargeWeakBuckets(t *testing.T) {
	patterns := make([]testMatcherSetPattern, 0, 7)
	for i := 0; i < 7; i++ {
		patterns = append(patterns, testMatcherSetPattern{
			ID:      fmt.Sprintf("weak-%d", i),
			Pattern: `%{NOTSPACE:a} %{GREEDYDATA:b}`,
		})
	}

	ms := newTestMatcherSet(t, patterns...)
	if !ms.preferLinear {
		t.Fatal("expected low-quality index set to prefer linear execution")
	}
}

func TestMatcherSetPrefersLinearWhenBestBucketIsStillBroad(t *testing.T) {
	patterns := []testMatcherSetPattern{
		{ID: "word", Pattern: `^svc %{WORD:value} msg=%{GREEDYDATA:msg}$`},
		{ID: "int", Pattern: `^svc %{INT:value} msg=%{GREEDYDATA:msg}$`},
		{ID: "number", Pattern: `^svc %{NUMBER:value} msg=%{GREEDYDATA:msg}$`},
		{ID: "host", Pattern: `^svc %{HOSTNAME:value} msg=%{GREEDYDATA:msg}$`},
		{ID: "ns", Pattern: `^svc %{NOTSPACE:value} msg=%{GREEDYDATA:msg}$`},
		{ID: "data", Pattern: `^svc %{DATA:value} msg=%{GREEDYDATA:msg}$`},
	}

	ms := newTestMatcherSet(t, patterns...)
	if !ms.preferLinear {
		t.Fatal("expected broad buckets on a medium set to prefer linear execution")
	}
}

type testMatcherSetPattern struct {
	ID      string
	Pattern string
}

func newTestMatcherSet(t testing.TB, patterns ...testMatcherSetPattern) *MatcherSet {
	t.Helper()

	items := make([]MatcherSetPattern, 0, len(patterns))
	for _, pattern := range patterns {
		matcher, err := CompilePattern(pattern.Pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatalf("compile %q: %v", pattern.ID, err)
		}
		items = append(items, MatcherSetPattern{
			ID:      pattern.ID,
			Matcher: matcher,
		})
	}

	ms, err := NewMatcherSet(items)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

func BenchmarkMatcherSetRunFirst(b *testing.B) {
	patterns := []testMatcherSetPattern{
		{ID: "apache", Pattern: `^\[%{TIMESTAMP_ISO8601:time}\] %{LOGLEVEL:level} %{GREEDYDATA:msg}$`},
		{ID: "nginx", Pattern: `^%{IPORHOST:client} - - \[%{HTTPDATE:time}\] "%{WORD:method} %{DATA:path} HTTP/%{NUMBER:ver}" %{INT:status} %{INT:bytes}$`},
		{ID: "es", Pattern: `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[%{NOTSPACE:name}%{SPACE}\]%{SPACE}(\[%{HOSTNAME:nodeId}\])?.*`},
		{ID: "consul", Pattern: `^%{MONTH} %{MONTHDAY} %{TIME} %{HOSTNAME:host} consul\[%{POSINT:pid}\]: %{GREEDYDATA:msg}$`},
		{ID: "target", Pattern: `^foo%{SPACE}%{WORD:name}%{SPACE}bar%{SPACE}%{INT:id}$`},
	}

	ms := newTestMatcherSet(b, patterns...)
	matchers := ms.entries
	line := "foo alice bar 42"

	b.Run("matcher_set", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			id, ret, err := ms.RunFirst(line, true)
			if err != nil {
				b.Fatal(err)
			}
			if id != "target" || len(ret) != 2 {
				b.Fatalf("unexpected result id=%q ret=%v", id, ret)
			}
		}
	})

	b.Run("manual_loop", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			found := false
			for _, entry := range matchers {
				ret, err := entry.matcher.Run(line, true)
				if err == nil {
					if entry.id != "target" || len(ret) != 2 {
						b.Fatalf("unexpected result id=%q ret=%v", entry.id, ret)
					}
					found = true
					break
				}
				if err != ErrMismatch {
					b.Fatal(err)
				}
			}
			if !found {
				b.Fatal("expected match")
			}
		}
	})
}

func BenchmarkMatcherSetRunFirstLargeSet(b *testing.B) {
	patterns := make([]testMatcherSetPattern, 0, 65)
	for i := 0; i < 64; i++ {
		patterns = append(patterns, testMatcherSetPattern{
			ID:      fmt.Sprintf("decoy-%02d", i),
			Pattern: fmt.Sprintf(`^service=svc%02d level=%%{LOGLEVEL:level} msg=%%{GREEDYDATA:msg}$`, i),
		})
	}
	patterns = append(patterns, testMatcherSetPattern{
		ID:      "target",
		Pattern: `^service=checkout level=%{LOGLEVEL:level} trace=%{NOTSPACE:trace} msg=%{GREEDYDATA:msg}$`,
	})

	ms := newTestMatcherSet(b, patterns...)
	matchers := ms.entries
	line := "service=checkout level=INFO trace=req-42 msg=payment accepted"

	b.Run("matcher_set", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			id, ret, err := ms.RunFirst(line, true)
			if err != nil {
				b.Fatal(err)
			}
			if id != "target" || len(ret) != 3 {
				b.Fatalf("unexpected result id=%q ret=%v", id, ret)
			}
		}
	})

	b.Run("manual_loop", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			found := false
			for _, entry := range matchers {
				ret, err := entry.matcher.Run(line, true)
				if err == nil {
					if entry.id != "target" || len(ret) != 3 {
						b.Fatalf("unexpected result id=%q ret=%v", entry.id, ret)
					}
					found = true
					break
				}
				if err != ErrMismatch {
					b.Fatal(err)
				}
			}
			if !found {
				b.Fatal("expected match")
			}
		}
	})
}
