package grok

import "testing"

func TestMatcherSetFilterAcceptsRequiredAndAnyAtoms(t *testing.T) {
	postings := []matcherSetAtomPosting{
		{key: "foo"},
		{key: "bar"},
		{key: "baz"},
		{key: "qux"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "foo",
		literalSet:     []string{"bar", "baz"},
		required:       []string{"qux"},
	}, atomIDs)

	ctx := matcherSetEvalContext{
		Content:  "fooxbarqux",
		AtomHits: []bool{true, true, false, true},
	}
	if !filter.Accepts(ctx) {
		t.Fatal("expected filter to accept prefix + any literal + required atoms")
	}

	ctx = matcherSetEvalContext{
		Content:  "fooxbar",
		AtomHits: []bool{true, true, false, false},
	}
	if filter.Accepts(ctx) {
		t.Fatal("expected filter to reject when required atom is missing")
	}
}

func TestMatcherSetFilterAcceptsExactLiteralSet(t *testing.T) {
	filter := compileMatcherSetFilter(&regexpPrefilter{
		literalExact: true,
		exactByLen: map[int][]string{
			3: {"foo", "bar"},
		},
	}, nil)

	if !filter.Accepts(matcherSetEvalContext{Content: "foo"}) {
		t.Fatal("expected exact filter to accept literal match")
	}
	if filter.Accepts(matcherSetEvalContext{Content: "foox"}) {
		t.Fatal("expected exact filter to reject non-exact match")
	}
}

func TestMatcherSetFilterProgramMatchesStructChecks(t *testing.T) {
	postings := []matcherSetAtomPosting{
		{key: "foo"},
		{key: "bar"},
		{key: "baz"},
		{key: "qux"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "foo",
		literalPrefix:  "bar",
		literalSet:     []string{"bar", "baz"},
		required:       []string{"bar", "qux"},
	}, atomIDs)

	cases := []matcherSetEvalContext{
		{Content: "foobarqux", AtomHits: []bool{true, true, false, true}},
		{Content: "foobazqux", AtomHits: []bool{true, false, true, true}},
		{Content: "fooquxbar", AtomHits: []bool{true, true, false, true}},
		{Content: "barfooqux", AtomHits: []bool{true, true, false, true}},
	}

	for _, tc := range cases {
		got := filter.Accepts(tc)
		prog := runMatcherSetFilterProgram(filter, tc)
		if got != prog {
			t.Fatalf("program mismatch for %q: Accepts=%v program=%v", tc.Content, got, prog)
		}
	}
}

func TestMatcherSetFilterProgramJITDisabledByDefault(t *testing.T) {
	filter := compileMatcherSetFilter(&regexpPrefilter{
		literalExact: true,
		exactByLen: map[int][]string{
			3: {"foo", "bar"},
		},
	}, nil)

	if matcherSetFilterJITEnabled(filter) {
		t.Fatal("expected filter program JIT to be disabled by default")
	}
}

func TestMatcherSetFilterProgramJITCandidateForSupportedOpcodes(t *testing.T) {
	filter := compileMatcherSetFilter(&regexpPrefilter{
		literalExact: true,
		exactByLen: map[int][]string{
			3: {"foo", "bar"},
		},
	}, nil)

	info := matcherSetFilterJITInfo(filter)
	if info.Arch == "" || info.Backend == "" {
		t.Fatalf("expected non-empty JIT info, got %+v", info)
	}
	if len(info.SupportedOpcodes) == 0 {
		if matcherSetFilterJITCandidate(filter) {
			t.Fatal("expected unsupported architectures to reject JIT candidates")
		}
		return
	}
	if !matcherSetFilterJITCandidate(filter) {
		t.Fatal("expected exact literal set filter to be a JIT candidate on supported architectures")
	}
	if kind := matcherSetFilterProgramRunnerKind(filter); kind != "specialized" {
		t.Fatalf("expected specialized runner for supported exact filter, got %q", kind)
	}
	if info.Enabled {
		t.Fatal("expected stub JIT backend to remain disabled")
	}
}

func TestMatcherSetFilterProgramJITCandidateRejectsUnsupportedRequiredOrder(t *testing.T) {
	postings := []matcherSetAtomPosting{
		{key: "foo"},
		{key: "bar"},
		{key: "baz"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "foo",
		required:       []string{"foo", "bar", "baz"},
	}, atomIDs)

	if matcherSetFilterJITCandidate(filter) {
		t.Fatal("expected required-order filter to stay off the JIT candidate path")
	}
	if kind := matcherSetFilterProgramRunnerKind(filter); kind != "interpreter" {
		t.Fatalf("expected interpreter runner for unsupported required-order filter, got %q", kind)
	}
}

func BenchmarkMatcherSetFilterExecution(b *testing.B) {
	postings := []matcherSetAtomPosting{
		{key: "prefix=api"},
		{key: "region=cn"},
		{key: "status=ok"},
		{key: "trace=req-42"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "prefix=api",
		literalPrefix:  "region=cn",
		literalSet:     []string{"region=cn", "status=ok"},
		required:       []string{"trace=req-42"},
	}, atomIDs)
	ctx := matcherSetEvalContext{
		Content:  "prefix=api region=cn status=ok trace=req-42",
		AtomHits: []bool{true, true, true, true},
	}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionExactSet(b *testing.B) {
	filter := compileMatcherSetFilter(&regexpPrefilter{
		literalExact: true,
		exactByLen: map[int][]string{
			3: {"foo", "bar", "baz"},
		},
	}, nil)
	ctx := matcherSetEvalContext{Content: "bar"}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionAtomsOnly(b *testing.B) {
	postings := []matcherSetAtomPosting{
		{key: "prefix=api"},
		{key: "region=cn"},
		{key: "status=ok"},
		{key: "trace=req-42"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "prefix=api",
		literalPrefix:  "region=cn",
		literalSet:     []string{"region=cn", "status=ok"},
	}, atomIDs)
	ctx := matcherSetEvalContext{
		Content: "prefix=api region=cn status=ok",
		UseBits: true,
		AtomBits: (uint64(1) << 0) |
			(uint64(1) << 1) |
			(uint64(1) << 2),
	}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionExactPlusAtom(b *testing.B) {
	postings := []matcherSetAtomPosting{
		{key: "foo"},
		{key: "bar"},
	}
	atomIDs := matcherSetAtomIndex(postings)

	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "foo",
		literalExact:   true,
		exactByLen: map[int][]string{
			3: {"foo"},
		},
		literalPrefix: "foo",
		required:      nil,
	}, atomIDs)
	ctx := matcherSetEvalContext{
		Content:  "foo",
		UseBits:  true,
		AtomBits: uint64(1) << 0,
	}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionPrefixOnly(b *testing.B) {
	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: "prefix=api",
	}, nil)
	ctx := matcherSetEvalContext{Content: "prefix=api region=cn"}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionExactSetLong(b *testing.B) {
	const lit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	filter := compileMatcherSetFilter(&regexpPrefilter{
		literalExact: true,
		exactByLen: map[int][]string{
			len(lit): {lit},
		},
	}, nil)
	ctx := matcherSetEvalContext{Content: lit}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}

func BenchmarkMatcherSetFilterExecutionPrefixOnlyLong(b *testing.B) {
	const prefix = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	filter := compileMatcherSetFilter(&regexpPrefilter{
		anchoredPrefix: prefix,
	}, nil)
	ctx := matcherSetEvalContext{Content: prefix + " trailing payload"}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterStruct(filter, ctx) {
				b.Fatal("expected struct filter to accept")
			}
		}
	})

	b.Run("program", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !runMatcherSetFilterProgram(filter, ctx) {
				b.Fatal("expected program filter to accept")
			}
		}
	})
}
