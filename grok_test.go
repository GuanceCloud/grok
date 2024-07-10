package grok

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDenormalizeGlobalPatterns(t *testing.T) {
	if denormalized, errs := DenormalizePatternsFromMap(defalutPatterns); len(errs) != 0 {
		t.Error(errs)
	} else {
		if len(defalutPatterns) != len(denormalized) {
			t.Error("len(GlobalPatterns) != len(denormalized)")
		}
		for k := range denormalized {
			if _, ok := defalutPatterns[k]; !ok {
				t.Errorf("%s not exists", k)
			}
		}
	}
}

func BenchmarkFindStringSubmatch(b *testing.B) {
	re := regexp.MustCompile(`(\w+):(\w+):(\w+):(\w+):(\w+):(\w+)`)

	str := "hello:world:foo:hello:world:foo"
	b.Run("FindStringSubmatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			match := re.FindStringSubmatch(str)
			if len(match) != 7 {
				b.Fatal(match)
			}
		}
	})

	b.Run("FindStringSubmatchIndex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			match := re.FindStringSubmatchIndex(str)
			m := make([]string, 0, 6)
			for j := 1; j < 7; j++ {
				m = append(m, str[match[2*j]:match[2*j+1]])
			}
			_ = m
		}
	})

}

func TestParse(t *testing.T) {
	patternINT, err := DenormalizePattern(defalutPatterns["INT"])
	if err != nil {
		t.Error(err)
	}

	patterns := map[string]*GrokPattern{
		"INT": patternINT,
	}

	denormalized, errs := DenormalizePatternsFromMap(defalutPatterns, patterns)
	if len(errs) != 0 {
		t.Error(errs)
	}
	g, err := CompilePattern("%{DAY:day}", PatternStorage{denormalized})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.Run("Tue qds", true)
	if err != nil {
		t.Error(err)
	}

	if v, ok := g.GetValByName("day", ret); !ok || v != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", v)
	}
}

func TestParseFromPathPattern(t *testing.T) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		t.Error(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		t.Error(errs)
	}
	g, err := CompilePattern("%{DAY:day}", PatternStorage{de})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.Run("Tue qds", true)
	if err != nil {
		t.Error(err)
	}

	if v, ok := g.GetValByName("day", ret); !ok || v != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", v)
	}
}

func TestLoadPatternsFromPathErr(t *testing.T) {
	_, err := LoadPatternsFromPath("./Lorem ipsum Minim qui in.")
	if err == nil {
		t.Fatalf("AddPatternsFromPath should returns an error when path is invalid")
	}
}

func TestRunWithTypeInfo(t *testing.T) {
	tCase := []struct {
		data string
		ptn  string
		ret  []any
	}{
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:float}`,
			ret: []any{
				int64(1),
				true,
				float64(1.1),
			},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				int64(1),
				true,
				int64(0),
			},
		},
		{
			data: `1 ijk123abc
true 1.1`,
			ptn: `%{INT:A:int} %{WORD:S:string}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				int64(1),
				"ijk123abc",
				true,
				int64(0),
			},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				"1",
				true,
				int64(0),
			},
		},
	}

	for _, item := range tCase {
		t.Run(item.ptn, func(t *testing.T) {
			g, err := CompilePattern(item.ptn, PatternStorage{defalutDenormalizedPatterns})
			if err != nil {
				t.Fatal(err)
			}
			v, err := g.RunWithTypeInfo(item.data, true)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, item.ret, v)
		})
	}
}

func BenchmarkFromMap(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Error(err)
	}

	for n := 0; n < b.N; n++ {
		de, errs := DenormalizePatternsFromMap(pathPatterns)
		if len(errs) != 0 {
			b.Error(err)
			b.Error(de)
		}
	}
}
