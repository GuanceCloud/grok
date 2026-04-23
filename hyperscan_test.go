//go:build !hyperscan || !cgo

package grok

import "testing"

func TestDefaultMultiPatternFilterBackendUsesStub(t *testing.T) {
	if defaultMultiPatternFilterBackend == nil {
		t.Fatal("expected default multi-pattern backend")
	}
	if defaultMultiPatternFilterBackend.Name() != hyperscanDisabledBackendName {
		t.Fatalf("backend name = %q, want %q", defaultMultiPatternFilterBackend.Name(), hyperscanDisabledBackendName)
	}
	if defaultMultiPatternFilterBackend.Available() {
		t.Fatal("expected hyperscan backend to be unavailable by default")
	}

	filter, err := compileMultiPatternFilter([]string{"foo", "bar"})
	if err != nil {
		t.Fatalf("compile filter: %v", err)
	}
	if filter == nil {
		t.Fatal("expected stub multi-pattern filter")
	}
	if filter.Backend() != hyperscanDisabledBackendName {
		t.Fatalf("filter backend = %q, want %q", filter.Backend(), hyperscanDisabledBackendName)
	}
	if !filter.MatchString("anything at all") {
		t.Fatal("expected stub filter to allow fallback regexp matching")
	}
	if err := filter.Close(); err != nil {
		t.Fatalf("close filter: %v", err)
	}

	loaded, err := loadMultiPatternFilter([]byte("serialized-database"))
	if err != nil {
		t.Fatalf("load filter: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded stub multi-pattern filter")
	}
	if !loaded.MatchString("anything at all") {
		t.Fatal("expected loaded stub filter to allow fallback regexp matching")
	}
}

func TestCompilePatternAttachesDefaultMultiPatternFilter(t *testing.T) {
	g, err := CompilePattern(`%{WORD:name}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if g.multiFilter == nil {
		t.Fatal("expected compiled pattern to attach a multi-pattern filter")
	}
	if g.multiFilter.Backend() != hyperscanDisabledBackendName {
		t.Fatalf("compiled filter backend = %q, want %q", g.multiFilter.Backend(), hyperscanDisabledBackendName)
	}

	ret, err := g.Run("hello", true)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(ret) != 1 || ret[0] != "hello" {
		t.Fatalf("unexpected match result: %#v", ret)
	}
}
