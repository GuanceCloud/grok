package grok

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type datakitCompiledPattern struct {
	pattern    string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
}

type datakitCompiledPipeline struct {
	patterns   []datakitCompiledPattern
	currentSet *MatcherSet
	regexpSet  *MatcherSet
}

type datakitMatchedFixture struct {
	name       string
	collector  string
	pattern    string
	line       string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
}

func TestDatakitFixturesMatchRegexp(t *testing.T) {
	fixtures, unmatched := loadMatchedDatakitFixtures(t)
	if len(fixtures) == 0 {
		t.Fatal("expected matched datakit grok fixtures")
	}
	if len(unmatched) > 0 {
		t.Logf("unmatched fixture examples: %d", len(unmatched))
		for _, item := range unmatched {
			t.Log(item)
		}
	}
}

func BenchmarkDatakitFixtures(b *testing.B) {
	fixtures, _ := loadMatchedDatakitFixtures(b)
	if len(fixtures) == 0 {
		b.Fatal("expected matched datakit grok fixtures")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		b.Run(fixture.name+"/fast", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.current.Run(fixture.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})

		b.Run(fixture.name+"/regexp", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.regexpOnly.Run(fixture.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})
	}
}

func loadMatchedDatakitFixtures(t testing.TB) ([]datakitMatchedFixture, []string) {
	t.Helper()

	cases := loadDatakitFixtureCases(t)
	fixtures := make([]datakitMatchedFixture, 0, len(cases))
	unmatched := make([]string, 0)

	for _, c := range cases {
		for pipelineName, script := range c.Pipelines {
			compiled, err := compileDatakitPipeline(script)
			if err != nil {
				t.Fatalf("%s/%s compile pipeline: %v", c.Collector, pipelineName, err)
			}
			if len(compiled.patterns) == 0 {
				continue
			}

			for exampleGroup, examples := range c.Examples {
				for exampleName, line := range examples {
					name := sanitizeFixtureName(c.Collector + "/" + pipelineName + "/" + exampleGroup + "/" + exampleName)
					match, ok, err := matchDatakitExample(name, c.Collector, line, compiled)
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
					if !ok {
						unmatched = append(unmatched, name)
						continue
					}
					fixtures = append(fixtures, match)
				}
			}
		}
	}

	return fixtures, unmatched
}

func compileDatakitPipeline(script string) (*datakitCompiledPipeline, error) {
	patterns := CopyDefalutPatterns()
	lines := strings.Split(script, "\n")
	out := make([]datakitCompiledPattern, 0, 4)

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "add_pattern("):
			args, err := extractPipelineQuotedArgs(line)
			if err != nil {
				return nil, err
			}
			if len(args) >= 2 {
				patterns[args[0]] = args[1]
			}
		case strings.HasPrefix(line, "grok("):
			args, err := extractPipelineQuotedArgs(line)
			if err != nil {
				return nil, err
			}
			if len(args) == 0 {
				continue
			}
			current, regexpOnly, err := compileDatakitPattern(args[0], patterns)
			if err != nil {
				return nil, err
			}
			out = append(out, datakitCompiledPattern{
				pattern:    args[0],
				current:    current,
				regexpOnly: regexpOnly,
			})
		}
	}

	currentSet, err := buildDatakitMatcherSet(out, false)
	if err != nil {
		return nil, err
	}
	regexpSet, err := buildDatakitMatcherSet(out, true)
	if err != nil {
		return nil, err
	}

	return &datakitCompiledPipeline{
		patterns:   out,
		currentSet: currentSet,
		regexpSet:  regexpSet,
	}, nil
}

func compileDatakitPattern(pattern string, patterns map[string]string) (*GrokRegexp, *GrokRegexp, error) {
	snapshot := make(map[string]string, len(patterns))
	for k, v := range patterns {
		snapshot[k] = v
	}

	denorm, errs := DenormalizePatternsFromMap(snapshot)
	if len(errs) != 0 {
		return nil, nil, fmt.Errorf("denormalize %q: %v", pattern, errs)
	}

	storage := PatternStorage{denorm}
	current, err := CompilePattern(pattern, storage)
	if err != nil {
		return nil, nil, err
	}
	regexpOnly, err := CompilePattern(pattern, storage)
	if err != nil {
		return nil, nil, err
	}
	regexpOnly.fastMatcher = nil
	return current, regexpOnly, nil
}

func matchDatakitExample(name string, collector string, line string, pipeline *datakitCompiledPipeline) (datakitMatchedFixture, bool, error) {
	currentIdx, currentRet, currentG, err := firstMatchingPattern(line, pipeline, false)
	if err != nil {
		return datakitMatchedFixture{}, false, err
	}
	regexpIdx, regexpRet, regexpG, err := firstMatchingPattern(line, pipeline, true)
	if err != nil {
		return datakitMatchedFixture{}, false, err
	}

	if currentIdx != regexpIdx {
		return datakitMatchedFixture{}, false, fmt.Errorf("match order diverged: fast=%d regexp=%d", currentIdx, regexpIdx)
	}
	if currentIdx < 0 {
		return datakitMatchedFixture{}, false, nil
	}
	if !reflect.DeepEqual(currentRet, regexpRet) {
		return datakitMatchedFixture{}, false, fmt.Errorf("match result diverged on pattern %q", pipeline.patterns[currentIdx].pattern)
	}

	return datakitMatchedFixture{
		name:       name,
		collector:  collector,
		pattern:    pipeline.patterns[currentIdx].pattern,
		line:       line,
		current:    currentG,
		regexpOnly: regexpG,
	}, true, nil
}

func firstMatchingPattern(line string, pipeline *datakitCompiledPipeline, regexpOnly bool) (int, []string, *GrokRegexp, error) {
	if pipeline == nil {
		return -1, nil, nil, nil
	}

	set := pipeline.currentSet
	if regexpOnly {
		set = pipeline.regexpSet
	}
	if set == nil {
		return -1, nil, nil, nil
	}

	idx, ret, err := set.runFirstIndex(line, true)
	if err == ErrMismatch {
		return -1, nil, nil, nil
	}
	if err != nil {
		return -1, nil, nil, err
	}
	pattern := pipeline.patterns[idx]
	g := pattern.current
	if regexpOnly {
		g = pattern.regexpOnly
	}
	return idx, ret, g, nil
}

func buildDatakitMatcherSet(patterns []datakitCompiledPattern, regexpOnly bool) (*MatcherSet, error) {
	items := make([]MatcherSetPattern, 0, len(patterns))
	for idx, pattern := range patterns {
		g := pattern.current
		if regexpOnly {
			g = pattern.regexpOnly
		}
		items = append(items, MatcherSetPattern{
			ID:      strconv.Itoa(idx),
			Matcher: g,
		})
	}
	return NewMatcherSet(items)
}

func BenchmarkDatakitPipelineDispatch(b *testing.B) {
	cases := loadDatakitFixtureCases(b)
	for _, c := range cases {
		for pipelineName, script := range c.Pipelines {
			compiled, err := compileDatakitPipeline(script)
			if err != nil {
				b.Fatalf("%s/%s compile pipeline: %v", c.Collector, pipelineName, err)
			}
			if len(compiled.patterns) < 2 {
				continue
			}

			for _, examples := range c.Examples {
				for _, line := range examples {
					name := sanitizeFixtureName(c.Collector + "/" + pipelineName)
					b.Run(name, func(b *testing.B) {
						b.Run("matcher_set", func(b *testing.B) {
							b.ReportAllocs()
							for i := 0; i < b.N; i++ {
								_, _, err := compiled.currentSet.runFirstIndex(line, true)
								if err != nil {
									b.Fatal(err)
								}
							}
						})
						b.Run("matcher_set_reuse", func(b *testing.B) {
							buf := make([]string, 0, compiled.currentSet.MatchCount())
							b.ReportAllocs()
							for i := 0; i < b.N; i++ {
								_, _, err := compiled.currentSet.runFirstIndexTo(line, true, buf)
								if err != nil {
									b.Fatal(err)
								}
							}
						})
						b.Run("manual_loop", func(b *testing.B) {
							b.ReportAllocs()
							for i := 0; i < b.N; i++ {
								found := false
								for _, pattern := range compiled.patterns {
									ret, err := pattern.current.Run(line, true)
									if err == nil {
										if len(ret) == 0 {
											b.Fatal("empty result")
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
					})
					goto nextPipeline
				}
			}
		nextPipeline:
		}
	}
}

func extractPipelineQuotedArgs(line string) ([]string, error) {
	args := make([]string, 0, 2)
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"', '\'':
			quote := line[i]
			j := i + 1
			for j < len(line) {
				if line[j] == '\\' {
					j += 2
					continue
				}
				if line[j] == quote {
					break
				}
				j++
			}
			if j >= len(line) {
				return nil, fmt.Errorf("unterminated quoted arg: %s", line)
			}
			raw := line[i : j+1]
			value, err := unquotePipelineArg(raw)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
			i = j
		}
	}
	return args, nil
}

func unquotePipelineArg(raw string) (string, error) {
	if len(raw) < 2 {
		return "", fmt.Errorf("invalid quoted arg: %q", raw)
	}
	if raw[0] == '"' {
		return strconv.Unquote(raw)
	}

	var b strings.Builder
	b.Grow(len(raw) - 2)
	for i := 1; i < len(raw)-1; i++ {
		if raw[i] == '\\' && i+1 < len(raw)-1 {
			i++
		}
		b.WriteByte(raw[i])
	}
	return b.String(), nil
}

func sanitizeFixtureName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "\t", "_", "\n", "_")
	return replacer.Replace(name)
}
