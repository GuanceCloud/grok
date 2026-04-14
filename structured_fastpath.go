package grok

import "strings"

type structuredMatcher struct {
	steps        []structuredStep
	writes       bool
	backtracking bool
	changeLog    bool
	changeCap    int
}

type matchChange struct {
	idx  int
	prev string
}

type structuredStep struct {
	literal               string
	parser                *structuredParser
	submatcher            *structuredMatcher
	alternatives          []*structuredMatcher
	altPrefixes           []string
	captureIndex          int
	optional              bool
	optPrefix             string
	optPrefixSkips        bool
	deterministicOptional bool
	writes                bool
}

type structuredParser struct {
	alias         string
	dstIndex      int
	kind          structuredKind
	charClass     *asciiCharClass
	allowEmpty    bool
	nextLiteral   string
	nextParser    structuredKind
	hasNextParser bool
}

type asciiCharClass struct {
	table [256]bool
}

type structuredKind uint8

const (
	structuredWord structuredKind = iota
	structuredNotSpace
	structuredHostName
	structuredNumber
	structuredCharClass
	structuredMonthName
	structuredDayName
	structuredTimeOfDay
	structuredYear
	structuredMonthNum
	structuredMonthDay
	structuredHour
	structuredMinute
	structuredSecond
	structuredQuoted
	structuredUntilLiteral
	structuredGreedyUntilLiteral
	structuredSpacePlus
	structuredSpaceStar
	structuredTimestampISO8601
	structuredHTTPDate
	structuredLogLevel
)

func buildFastMatcher(gP *GrokPattern, storage PatternStorageIface, meta *compiledRegexpMeta) *structuredMatcher {
	if storage == nil {
		return nil
	}
	return buildStructuredFastMatcher(gP.pattern, storage, meta)
}

func buildStructuredFastMatcher(pattern string, storage PatternStorageIface, meta *compiledRegexpMeta) *structuredMatcher {
	steps, ok := compileStructuredSteps(pattern, storage, meta, 0)
	if !ok || len(steps) == 0 {
		return nil
	}

	configureStructuredSteps(steps)
	if !hasStructuredMatcherWork(steps) || !shouldUseStructuredMatcher(steps) {
		return nil
	}

	matcher := structuredMatcher{
		steps:        steps,
		writes:       matcherWrites(steps),
		backtracking: matcherNeedsBacktracking(steps),
		changeLog:    matcherNeedsChangeLog(steps),
		changeCap:    matcherChangeCapacity(steps),
	}
	return &matcher
}

func configureStructuredSteps(steps []structuredStep) {
	for i := range steps {
		if steps[i].parser != nil {
			steps[i].parser.nextLiteral = nextStructuredLiteral(steps, i+1)
			steps[i].parser.nextParser, steps[i].parser.hasNextParser = nextStructuredParserKind(steps, i+1)
			steps[i].writes = steps[i].parser.dstIndex >= 0
		}
		if steps[i].submatcher != nil {
			configureStructuredSteps(steps[i].submatcher.steps)
			steps[i].submatcher.writes = matcherWrites(steps[i].submatcher.steps)
			steps[i].submatcher.backtracking = matcherNeedsBacktracking(steps[i].submatcher.steps)
			steps[i].submatcher.changeLog = matcherNeedsChangeLog(steps[i].submatcher.steps)
			steps[i].submatcher.changeCap = matcherChangeCapacity(steps[i].submatcher.steps)
			steps[i].writes = steps[i].writes || steps[i].submatcher.writes
		}
		for _, alt := range steps[i].alternatives {
			configureStructuredSteps(alt.steps)
			alt.writes = matcherWrites(alt.steps)
			alt.backtracking = matcherNeedsBacktracking(alt.steps)
			alt.changeLog = matcherNeedsChangeLog(alt.steps)
			alt.changeCap = matcherChangeCapacity(alt.steps)
			steps[i].writes = steps[i].writes || alt.writes
		}
		if len(steps[i].alternatives) > 0 {
			steps[i].altPrefixes = buildAlternativePrefixes(steps[i].alternatives)
		}
		if steps[i].optional {
			if steps[i].parser != nil && steps[i].parser.nextLiteral != "" && !steps[i].parser.allowEmpty {
				steps[i].optPrefix = steps[i].parser.nextLiteral
				steps[i].optPrefixSkips = true
			} else if prefix, ok := firstStructuredLiteral(steps[i]); ok {
				steps[i].optPrefix = prefix
				steps[i].optPrefixSkips = false
			}
			steps[i].deterministicOptional = isDeterministicOptionalStep(steps, i)
		}
		if steps[i].captureIndex >= 0 {
			steps[i].writes = true
		}
	}
}

func buildAlternativePrefixes(alts []*structuredMatcher) []string {
	if len(alts) < 2 {
		return nil
	}

	prefixes := make([]string, len(alts))
	for i, alt := range alts {
		prefix := nextStructuredLiteral(alt.steps, 0)
		if prefix == "" {
			return nil
		}
		prefixes[i] = prefix
	}

	for i := range prefixes {
		for j := i + 1; j < len(prefixes); j++ {
			if prefixes[i] == prefixes[j] ||
				strings.HasPrefix(prefixes[i], prefixes[j]) ||
				strings.HasPrefix(prefixes[j], prefixes[i]) {
				return nil
			}
		}
	}

	return prefixes
}

func isDeterministicOptionalStep(steps []structuredStep, idx int) bool {
	step := steps[idx]
	if !step.optional || len(step.alternatives) > 0 {
		return false
	}
	if step.parser != nil {
		return (step.parser.nextLiteral != "" && !step.parser.allowEmpty) ||
			canSkipOptionalParser(step.parser)
	}
	if step.submatcher != nil && step.submatcher.backtracking {
		return false
	}

	curPrefix, ok := firstStructuredLiteral(step)
	if !ok || curPrefix == "" {
		return false
	}
	nextPrefix := nextStructuredLiteral(steps, idx+1)
	if nextPrefix == "" {
		return false
	}
	return curPrefix != nextPrefix &&
		!strings.HasPrefix(curPrefix, nextPrefix) &&
		!strings.HasPrefix(nextPrefix, curPrefix)
}

func canSkipOptionalParser(p *structuredParser) bool {
	if p == nil {
		return false
	}
	if p.nextLiteral != "" && !p.allowEmpty {
		return true
	}
	return p.dstIndex < 0 && p.kind == structuredYear && p.hasNextParser && p.nextParser == structuredTimeOfDay
}

func matcherWrites(steps []structuredStep) bool {
	for i := range steps {
		if steps[i].writes {
			return true
		}
	}
	return false
}

func matcherNeedsBacktracking(steps []structuredStep) bool {
	for i := range steps {
		if steps[i].optional && !isDeterministicOptionalStep(steps, i) {
			return true
		}
		if steps[i].submatcher != nil && matcherNeedsBacktracking(steps[i].submatcher.steps) {
			return true
		}
		for _, alt := range steps[i].alternatives {
			if matcherNeedsBacktracking(alt.steps) {
				return true
			}
		}
	}
	return false
}

func matcherNeedsChangeLog(steps []structuredStep) bool {
	for i := range steps {
		step := steps[i]
		if step.optional && !isDeterministicOptionalStep(steps, i) {
			return true
		}
		if step.submatcher != nil && matcherNeedsChangeLog(step.submatcher.steps) {
			return true
		}
		if len(step.alternatives) > 0 {
			if step.captureIndex >= 0 {
				return true
			}
			for _, alt := range step.alternatives {
				if alt.writes || matcherNeedsChangeLog(alt.steps) {
					return true
				}
			}
		}
	}
	return false
}

func matcherChangeCapacity(steps []structuredStep) int {
	total := 0
	for i := range steps {
		total += stepChangeCapacity(steps[i])
	}
	return total
}

func stepChangeCapacity(step structuredStep) int {
	total := 0
	if step.parser != nil && step.parser.dstIndex >= 0 {
		total++
	}
	if step.captureIndex >= 0 {
		total++
	}
	if step.submatcher != nil {
		total += matcherChangeCapacity(step.submatcher.steps)
	}
	if len(step.alternatives) > 0 {
		maxAlt := 0
		for _, alt := range step.alternatives {
			altWrites := matcherChangeCapacity(alt.steps)
			if altWrites > maxAlt {
				maxAlt = altWrites
			}
		}
		total += maxAlt
	}
	return total
}

func hasStructuredMatcherWork(steps []structuredStep) bool {
	for i := range steps {
		switch {
		case steps[i].parser != nil:
			return true
		case steps[i].submatcher != nil && hasStructuredMatcherWork(steps[i].submatcher.steps):
			return true
		case len(steps[i].alternatives) > 0:
			for _, alt := range steps[i].alternatives {
				if hasStructuredMatcherWork(alt.steps) {
					return true
				}
			}
		}
	}
	return false
}

func shouldUseStructuredMatcher(steps []structuredStep) bool {
	for i := range steps {
		if steps[i].parser == nil {
			continue
		}
		// Unbounded leading GREEDYDATA is still risky, but a bounded one with a
		// concrete following literal is fine and now benchmarks well on real
		// access-log fixtures.
		if i == 0 && steps[i].parser.kind == structuredGreedyUntilLiteral && steps[i].parser.nextLiteral == "" {
			return false
		}
		break
	}

	stats := collectStructuredStats(steps)
	if stats.newlineLiteralCount > 0 && (stats.optionalCount > 0 || stats.alternativeCount > 0 || stats.parserCount > 12) {
		return false
	}
	// Some medium-complexity patterns still compile structurally but do not
	// beat regexp in practice. The Datakit Apache error fixture is the
	// representative case: two GREEDYDATA captures around a single expanded
	// submatcher and no branches. Let those fall back, while keeping simpler
	// two-greedy layouts such as Jenkins on the fast path.
	if stats.greedyParserCount >= 2 && stats.optionalCount == 0 && stats.alternativeCount == 0 && stats.submatcherCount == 1 && stats.parserCount >= 8 {
		return false
	}
	// Some flat syslog-style layouts now compile after adding literal `+/*`
	// repetition support, but the current structured path still loses to
	// regexp when the line is mostly nested subpatterns plus a single greedy
	// tail. Keep the compiler support, but let these stay on regexp for now.
	if stats.optionalCount == 0 && stats.greedyParserCount == 1 && stats.dataParserCount == 1 && stats.submatcherCount >= 4 && stats.alternativeCount >= 4 && stats.parserCount >= 18 {
		return false
	}
	// Highly optional, nested patterns still require regexp-style backtracking
	// to preserve capture alignment. PostgreSQL's default fixture is the
	// representative case here.
	if stats.optionalCount >= 2 && stats.submatcherCount >= 5 && stats.parserCount >= 10 {
		return false
	}
	return true
}

type structuredStats struct {
	parserCount         int
	optionalCount       int
	alternativeCount    int
	submatcherCount     int
	greedyParserCount   int
	dataParserCount     int
	newlineLiteralCount int
}

func collectStructuredStats(steps []structuredStep) structuredStats {
	var stats structuredStats
	for i := range steps {
		step := steps[i]
		if step.literal != "" && strings.Contains(step.literal, "\n") {
			stats.newlineLiteralCount++
		}
		if step.parser != nil {
			stats.parserCount++
			if step.parser.kind == structuredGreedyUntilLiteral {
				stats.greedyParserCount++
			}
			if step.parser.kind == structuredUntilLiteral || step.parser.kind == structuredGreedyUntilLiteral {
				stats.dataParserCount++
			}
		}
		if step.optional {
			stats.optionalCount++
		}
		if step.submatcher != nil {
			stats.submatcherCount++
			child := collectStructuredStats(step.submatcher.steps)
			stats.parserCount += child.parserCount
			stats.optionalCount += child.optionalCount
			stats.alternativeCount += child.alternativeCount
			stats.submatcherCount += child.submatcherCount
			stats.greedyParserCount += child.greedyParserCount
			stats.dataParserCount += child.dataParserCount
			stats.newlineLiteralCount += child.newlineLiteralCount
		}
		if len(step.alternatives) > 0 {
			stats.alternativeCount += len(step.alternatives)
			for _, alt := range step.alternatives {
				child := collectStructuredStats(alt.steps)
				stats.parserCount += child.parserCount
				stats.optionalCount += child.optionalCount
				stats.alternativeCount += child.alternativeCount
				stats.submatcherCount += child.submatcherCount
				stats.greedyParserCount += child.greedyParserCount
				stats.dataParserCount += child.dataParserCount
				stats.newlineLiteralCount += child.newlineLiteralCount
			}
		}
	}
	return stats
}

func compileStructuredSteps(pattern string, storage PatternStorageIface, meta *compiledRegexpMeta, depth int) ([]structuredStep, bool) {
	if depth > 16 {
		return nil, false
	}

	alts, pos, ok := parseStructuredExpression(pattern, 0, 0, storage, meta, depth)
	if !ok || pos != len(pattern) {
		return nil, false
	}

	if len(alts) == 1 {
		return alts[0], validateStructuredSteps(alts[0])
	}

	steps := []structuredStep{{alternatives: buildAlternativeMatchers(alts), captureIndex: -1}}
	return steps, validateStructuredSteps(steps)
}

func parseStructuredExpression(pattern string, pos int, stop byte, storage PatternStorageIface, meta *compiledRegexpMeta, depth int) ([][]structuredStep, int, bool) {
	current := make([]structuredStep, 0, 8)
	alternatives := make([][]structuredStep, 0, 2)

	for pos < len(pattern) {
		if stop != 0 && pattern[pos] == stop {
			break
		}
		if pattern[pos] == '|' {
			alternatives = append(alternatives, current)
			current = make([]structuredStep, 0, 4)
			pos++
			continue
		}

		term, next, ok := parseStructuredTerm(pattern, pos, storage, meta, depth)
		if !ok {
			return nil, pos, false
		}
		current = append(current, term...)
		pos = next
	}

	alternatives = append(alternatives, current)
	return alternatives, pos, true
}

func parseStructuredTerm(pattern string, pos int, storage PatternStorageIface, meta *compiledRegexpMeta, depth int) ([]structuredStep, int, bool) {
	switch {
	case strings.HasPrefix(pattern[pos:], "%{"):
		return parseStructuredRef(pattern, pos, storage, meta, depth)
	case pattern[pos] == '(':
		return parseStructuredGroup(pattern, pos, storage, meta, depth)
	default:
		return parseStructuredLiteral(pattern, pos)
	}
}

func parseStructuredRef(pattern string, pos int, storage PatternStorageIface, meta *compiledRegexpMeta, depth int) ([]structuredStep, int, bool) {
	end := strings.IndexByte(pattern[pos+2:], '}')
	if end < 0 {
		return nil, pos, false
	}
	end += pos + 2

	ref, ok, err := parsePatternRef(pattern[pos+2 : end])
	if err != nil || !ok {
		return nil, pos, false
	}
	next := end + 1
	optional := false
	if next < len(pattern) && pattern[next] == '?' {
		optional = true
		next++
	}

	if kind, primitive := structuredPrimitiveKind(ref.syntax); primitive && canUseStructuredPrimitive(ref.syntax, storage) {
		dstIndex := captureIndexForRef(ref, meta)
		if ref.alias != "" && dstIndex < 0 {
			return nil, pos, false
		}
		return []structuredStep{{
			captureIndex: -1,
			optional:     optional,
			parser: &structuredParser{
				alias:      ref.alias,
				dstIndex:   dstIndex,
				kind:       kind,
				allowEmpty: kind == structuredUntilLiteral || kind == structuredGreedyUntilLiteral,
			},
		}}, next, true
	}

	child, exists := storage.GetPattern(ref.syntax)
	if !exists {
		return nil, pos, false
	}
	childSteps, ok := compileStructuredSteps(child.pattern, storage, meta, depth+1)
	if !ok {
		return nil, pos, false
	}

	if ref.alias == "" && ref.varType == "" && !optional {
		return childSteps, next, true
	}

	dstIndex := captureIndexForRef(ref, meta)
	if ref.alias != "" && dstIndex < 0 {
		return nil, pos, false
	}
	return []structuredStep{{
		submatcher:   &structuredMatcher{steps: childSteps},
		captureIndex: dstIndex,
		optional:     optional,
	}}, next, true
}

func parseStructuredGroup(pattern string, pos int, storage PatternStorageIface, meta *compiledRegexpMeta, depth int) ([]structuredStep, int, bool) {
	start := pos + 1
	if strings.HasPrefix(pattern[start:], "?:") {
		start += 2
	}

	alts, next, ok := parseStructuredExpression(pattern, start, ')', storage, meta, depth+1)
	if !ok || next >= len(pattern) || pattern[next] != ')' {
		return nil, pos, false
	}
	next++

	optional := false
	if next < len(pattern) && pattern[next] == '?' {
		optional = true
		next++
	}

	if len(alts) == 1 && !optional {
		return alts[0], next, true
	}

	step := structuredStep{
		optional:     optional,
		captureIndex: -1,
	}
	if len(alts) == 1 {
		step.submatcher = &structuredMatcher{steps: alts[0]}
	} else {
		step.alternatives = buildAlternativeMatchers(alts)
	}
	return []structuredStep{step}, next, true
}

func parseStructuredLiteral(pattern string, pos int) ([]structuredStep, int, bool) {
	start := pos
	for pos < len(pattern) {
		switch pattern[pos] {
		case '\\':
			if pos+1 >= len(pattern) {
				return nil, start, false
			}
			pos += 2
		case '%':
			if strings.HasPrefix(pattern[pos:], "%{") {
				goto done
			}
			pos++
		case '(':
			goto done
		case ')', '|':
			goto done
		default:
			pos++
		}
	}

done:
	steps, ok := compileLiteralSteps(pattern[start:pos])
	return steps, pos, ok
}

func buildAlternativeMatchers(alts [][]structuredStep) []*structuredMatcher {
	out := make([]*structuredMatcher, 0, len(alts))
	for _, alt := range alts {
		out = append(out, &structuredMatcher{steps: alt})
	}
	return out
}

func captureIndexForRef(ref patternRef, meta *compiledRegexpMeta) int {
	if ref.alias == "" {
		return -1
	}
	if idx, ok := meta.nameIndex[ref.alias]; ok {
		return idx
	}
	return -1
}

func validateStructuredSteps(steps []structuredStep) bool {
	for i := range steps {
		step := steps[i]
		if step.parser != nil && i+1 < len(steps) && steps[i+1].parser != nil && !canParserEndWithoutLiteral(step.parser.kind) {
			return false
		}
		if step.submatcher != nil && !validateStructuredSteps(step.submatcher.steps) {
			return false
		}
		for _, alt := range step.alternatives {
			if !validateStructuredSteps(alt.steps) {
				return false
			}
		}
	}
	return true
}

func structuredPrimitiveKind(syntax string) (structuredKind, bool) {
	switch syntax {
	case "WORD":
		return structuredWord, true
	case "HOSTNAME", "HOST":
		return structuredHostName, true
	case "NOTSPACE", "USER", "USERNAME", "IPORHOST", "IP", "URIHOST", "PATH", "URIPATH", "URIPATHPARAM":
		return structuredNotSpace, true
	case "EMAILLOCALPART":
		return structuredNotSpace, true
	case "INT", "NUMBER", "BASE10NUM", "POSINT", "NONNEGINT":
		return structuredNumber, true
	case "MONTH":
		return structuredMonthName, true
	case "DAY":
		return structuredDayName, true
	case "TIME":
		return structuredTimeOfDay, true
	case "YEAR":
		return structuredYear, true
	case "MONTHNUM", "MONTHNUM2":
		return structuredMonthNum, true
	case "MONTHDAY":
		return structuredMonthDay, true
	case "HOUR":
		return structuredHour, true
	case "MINUTE":
		return structuredMinute, true
	case "SECOND":
		return structuredSecond, true
	case "QS", "QUOTEDSTRING":
		return structuredQuoted, true
	case "DATA", "GREEDYLINES":
		return structuredUntilLiteral, true
	case "GREEDYDATA":
		return structuredGreedyUntilLiteral, true
	case "SPACE":
		return structuredSpaceStar, true
	case "TIMESTAMP_ISO8601":
		return structuredTimestampISO8601, true
	case "HTTPDATE":
		return structuredHTTPDate, true
	case "LOGLEVEL":
		return structuredLogLevel, true
	default:
		return 0, false
	}
}

func canUseStructuredPrimitive(syntax string, storage PatternStorageIface) bool {
	if storage == nil {
		return true
	}
	gp, ok := storage.GetPattern(syntax)
	if !ok {
		return true
	}
	defaultPattern, hasDefault := defalutPatterns[syntax]
	if !hasDefault {
		return false
	}
	return gp.pattern == defaultPattern
}

func compileLiteralSteps(raw string) ([]structuredStep, bool) {
	if raw == "" {
		return nil, true
	}

	steps := make([]structuredStep, 0, 4)
	var b strings.Builder
	b.Grow(len(raw))
	flushLiteral := func() {
		if b.Len() > 0 {
			steps = append(steps, structuredStep{literal: b.String(), captureIndex: -1})
			b.Reset()
		}
	}
	appendOptionalLiteral := func(lit string) {
		flushLiteral()
		steps = append(steps, structuredStep{literal: lit, optional: true, captureIndex: -1})
	}
	appendRepeatedLiteral := func(ch byte, allowEmpty bool) {
		flushLiteral()
		class := &asciiCharClass{}
		class.table[ch] = true
		steps = append(steps, structuredStep{
			captureIndex: -1,
			parser: &structuredParser{
				dstIndex:   -1,
				kind:       structuredCharClass,
				charClass:  class,
				allowEmpty: allowEmpty,
			},
		})
	}
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '\\':
			if i+1 >= len(raw) {
				return nil, false
			}
			i++
			if i+1 < len(raw) && raw[i+1] == '?' {
				switch raw[i] {
				case 't':
					appendOptionalLiteral("\t")
				case 'n':
					appendOptionalLiteral("\n")
				case 'r':
					appendOptionalLiteral("\r")
				case 's':
					return nil, false
				default:
					appendOptionalLiteral(string(raw[i]))
				}
				i++
				continue
			}
			switch raw[i] {
			case 's':
				flushLiteral()
				if i+1 < len(raw) && raw[i+1] == '+' {
					steps = append(steps, structuredStep{captureIndex: -1, parser: &structuredParser{dstIndex: -1, kind: structuredSpacePlus}})
					i++
					continue
				}
				if i+1 < len(raw) && raw[i+1] == '*' {
					steps = append(steps, structuredStep{captureIndex: -1, parser: &structuredParser{dstIndex: -1, kind: structuredSpaceStar}})
					i++
					continue
				}
				b.WriteByte('s')
			case 't':
				b.WriteByte('\t')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(raw[i])
			}
		case '^', '$':
			continue
		case '.':
			if i+1 < len(raw) && raw[i+1] == '*' {
				flushLiteral()
				steps = append(steps, structuredStep{captureIndex: -1, parser: &structuredParser{dstIndex: -1, kind: structuredGreedyUntilLiteral, allowEmpty: true}})
				i++
				continue
			}
			b.WriteByte('.')
		case '[':
			end := i + 1
			for end < len(raw) && raw[end] != ']' {
				if raw[end] == '\\' {
					end += 2
					continue
				}
				end++
			}
			if end >= len(raw) {
				return nil, false
			}
			step, consumed, ok := compileSimpleCharClass(raw, i, end)
			if !ok {
				return nil, false
			}
			flushLiteral()
			steps = append(steps, step)
			i = consumed
			continue
		case ']', '{', '}', '+', '*', '?':
			return nil, false
		default:
			if i+1 < len(raw) && (raw[i+1] == '+' || raw[i+1] == '*') {
				appendRepeatedLiteral(raw[i], raw[i+1] == '*')
				i++
				continue
			}
			if i+1 < len(raw) && raw[i+1] == '?' {
				appendOptionalLiteral(string(raw[i]))
				i++
				continue
			}
			b.WriteByte(raw[i])
		}
	}

	flushLiteral()
	return steps, true
}

func compileSimpleCharClass(raw string, start, end int) (structuredStep, int, bool) {
	spec := raw[start+1 : end]
	if spec == "" {
		return structuredStep{}, 0, false
	}

	if end+1 < len(raw) && (raw[end+1] == '*' || raw[end+1] == '+') {
		class, ok := buildASCIICharClass(spec)
		if !ok {
			return structuredStep{}, 0, false
		}
		return structuredStep{
			captureIndex: -1,
			parser: &structuredParser{
				dstIndex:   -1,
				kind:       structuredCharClass,
				charClass:  class,
				allowEmpty: raw[end+1] == '*',
			},
		}, end + 1, true
	}

	options := make([]*structuredMatcher, 0, len(spec))
	for i := 0; i < len(spec); i++ {
		c := spec[i]
		if c == '\\' {
			if i+1 >= len(spec) {
				return structuredStep{}, 0, false
			}
			i++
			c = spec[i]
		}
		switch c {
		case '^', '-', '[', ']':
			return structuredStep{}, 0, false
		}
		options = append(options, &structuredMatcher{
			steps: []structuredStep{{literal: string(c), captureIndex: -1}},
		})
	}

	if len(options) == 0 {
		return structuredStep{}, 0, false
	}

	return structuredStep{alternatives: options, captureIndex: -1}, end, true
}

func buildASCIICharClass(spec string) (*asciiCharClass, bool) {
	class := &asciiCharClass{}
	for i := 0; i < len(spec); i++ {
		c := spec[i]
		if c == '\\' {
			if i+1 >= len(spec) {
				return nil, false
			}
			i++
			switch spec[i] {
			case 'w':
				for b := 0; b < 256; b++ {
					if isWordByte(byte(b)) {
						class.table[b] = true
					}
				}
			case 'd':
				for ch := byte('0'); ch <= '9'; ch++ {
					class.table[ch] = true
				}
			case 's':
				for _, ch := range []byte{' ', '\t', '\r', '\n', '\f', '\v'} {
					class.table[ch] = true
				}
			default:
				class.table[spec[i]] = true
			}
			continue
		}
		if i+2 < len(spec) && spec[i+1] == '-' {
			end := spec[i+2]
			if end < c {
				return nil, false
			}
			for ch := c; ch <= end; ch++ {
				class.table[ch] = true
			}
			i += 2
			continue
		}
		class.table[c] = true
	}
	return class, true
}

func nextStructuredLiteral(steps []structuredStep, start int) string {
	for i := start; i < len(steps); i++ {
		if lit, ok := firstStructuredLiteral(steps[i]); ok {
			return lit
		}
	}
	return ""
}

func nextStructuredParserKind(steps []structuredStep, start int) (structuredKind, bool) {
	for i := start; i < len(steps); i++ {
		if steps[i].literal != "" {
			return 0, false
		}
		if steps[i].parser != nil {
			return steps[i].parser.kind, true
		}
		return 0, false
	}
	return 0, false
}

func firstStructuredLiteral(step structuredStep) (string, bool) {
	switch {
	case step.literal != "":
		return step.literal, true
	case step.submatcher != nil:
		return nextStructuredLiteral(step.submatcher.steps, 0), nextStructuredLiteral(step.submatcher.steps, 0) != ""
	case len(step.alternatives) > 0:
		var common string
		for idx, alt := range step.alternatives {
			lit := nextStructuredLiteral(alt.steps, 0)
			if lit == "" {
				return "", false
			}
			if idx == 0 {
				common = lit
				continue
			}
			if lit != common {
				return "", false
			}
		}
		return common, common != ""
	default:
		return "", false
	}
}

func (m structuredMatcher) match(dst []string, content string, trimSpace bool) bool {
	if !m.backtracking && !m.changeLog {
		next, ok := m.matchLinearFrom(dst, content, 0, trimSpace)
		return ok && next >= 0
	}

	var smallChanges [8]matchChange
	changes := smallChanges[:0]
	changeCap := len(dst)
	if m.changeCap > changeCap {
		changeCap = m.changeCap
	}
	if changeCap > len(smallChanges) {
		changes = make([]matchChange, 0, changeCap)
	}

	var next int
	var ok bool
	if m.backtracking {
		next, ok, _ = m.matchBacktrackingFrom(dst, content, 0, trimSpace, changes)
	} else {
		next, ok, _ = m.matchFrom(dst, content, 0, trimSpace, changes)
	}
	return ok && next >= 0
}

func (m structuredMatcher) matchAnyFrom(dst []string, content string, pos int, trimSpace bool, changes []matchChange) (int, bool, []matchChange) {
	if !m.backtracking && !m.changeLog && changes == nil {
		next, ok := m.matchLinearFrom(dst, content, pos, trimSpace)
		return next, ok, nil
	}
	if m.backtracking {
		return m.matchBacktrackingFrom(dst, content, pos, trimSpace, changes)
	}
	return m.matchFrom(dst, content, pos, trimSpace, changes)
}

func (m structuredMatcher) matchLinearFrom(dst []string, content string, pos int, trimSpace bool) (int, bool) {
	for _, step := range m.steps {
		var next int
		var ok bool

		switch {
		case step.literal != "":
			if !strings.HasPrefix(content[pos:], step.literal) {
				if step.optional {
					if step.deterministicOptional && !step.optPrefixSkips && step.optPrefix != "" && strings.HasPrefix(content[pos:], step.optPrefix) {
						return 0, false
					}
					continue
				}
				return 0, false
			}
			pos += len(step.literal)
			continue
		case step.parser != nil:
			if step.optional && step.deterministicOptional && shouldSkipOptionalParser(step.parser, step, content, pos) {
				continue
			}
			var value string
			next, value, ok = step.parser.consume(content, pos)
			if !ok {
				if step.optional {
					if step.deterministicOptional {
						return 0, false
					}
					continue
				}
				return 0, false
			}
			if step.parser.dstIndex >= 0 {
				dst[step.parser.dstIndex] = maybeTrim(value, trimSpace)
			}
		case step.submatcher != nil:
			next, ok = step.submatcher.matchLinearFrom(dst, content, pos, trimSpace)
			if !ok {
				if step.optional {
					if step.deterministicOptional && !step.optPrefixSkips && step.optPrefix != "" && strings.HasPrefix(content[pos:], step.optPrefix) {
						return 0, false
					}
					continue
				}
				return 0, false
			}
			if step.captureIndex >= 0 {
				dst[step.captureIndex] = maybeTrim(content[pos:next], trimSpace)
			}
		case len(step.alternatives) > 0:
			alts := matchingAlternatives(step, content, pos)
			for _, alt := range alts {
				next, ok = alt.matchLinearFrom(dst, content, pos, trimSpace)
				if ok {
					if step.captureIndex >= 0 {
						dst[step.captureIndex] = maybeTrim(content[pos:next], trimSpace)
					}
					break
				}
			}
			if !ok {
				if step.optional {
					if step.deterministicOptional && !step.optPrefixSkips && step.optPrefix != "" && strings.HasPrefix(content[pos:], step.optPrefix) {
						return 0, false
					}
					continue
				}
				return 0, false
			}
		default:
			continue
		}

		pos = next
	}

	return pos, true
}

func shouldSkipOptionalParser(p *structuredParser, step structuredStep, content string, pos int) bool {
	if p == nil || pos > len(content) {
		return false
	}
	if step.optPrefixSkips && step.optPrefix != "" && strings.HasPrefix(content[pos:], step.optPrefix) {
		return true
	}
	if p.dstIndex < 0 && p.kind == structuredYear && p.hasNextParser && p.nextParser == structuredTimeOfDay {
		_, _, ok := sliceTimeOfDay(content, pos)
		return ok
	}
	return false
}

func (m structuredMatcher) matchFrom(dst []string, content string, pos int, trimSpace bool, changes []matchChange) (int, bool, []matchChange) {
	for _, step := range m.steps {
		var next int
		var ok bool

		switch {
		case step.literal != "":
			if !strings.HasPrefix(content[pos:], step.literal) {
				if step.optional {
					continue
				}
				return 0, false, changes
			}
			pos += len(step.literal)
			continue
		case step.parser != nil:
			var value string
			next, value, ok = step.parser.consume(content, pos)
			if !ok {
				if step.optional {
					continue
				}
				return 0, false, changes
			}
			if step.parser.dstIndex >= 0 {
				changes = appendMatchChange(dst, changes, step.parser.dstIndex, maybeTrim(value, trimSpace))
			}
		case step.submatcher != nil:
			mark := len(changes)
			if step.submatcher.backtracking {
				next, ok, changes = step.submatcher.matchBacktrackingFrom(dst, content, pos, trimSpace, changes)
			} else {
				next, ok, changes = step.submatcher.matchFrom(dst, content, pos, trimSpace, changes)
			}
			if !ok {
				if step.submatcher.writes {
					rollbackMatchChanges(dst, changes, mark)
					changes = changes[:mark]
				}
				if step.optional {
					continue
				}
				return 0, false, changes
			}
			if step.captureIndex >= 0 {
				changes = appendMatchChange(dst, changes, step.captureIndex, maybeTrim(content[pos:next], trimSpace))
			}
		case len(step.alternatives) > 0:
			alts := matchingAlternatives(step, content, pos)
			for _, alt := range alts {
				mark := len(changes)
				if alt.backtracking {
					next, ok, changes = alt.matchBacktrackingFrom(dst, content, pos, trimSpace, changes)
				} else {
					next, ok, changes = alt.matchFrom(dst, content, pos, trimSpace, changes)
				}
				if ok {
					if step.captureIndex >= 0 {
						changes = appendMatchChange(dst, changes, step.captureIndex, maybeTrim(content[pos:next], trimSpace))
					}
					break
				}
				if alt.writes {
					rollbackMatchChanges(dst, changes, mark)
					changes = changes[:mark]
				}
			}
			if !ok {
				if step.optional {
					continue
				}
				return 0, false, changes
			}
		default:
			continue
		}

		pos = next
	}

	return pos, true, changes
}

func (m structuredMatcher) matchBacktrackingFrom(dst []string, content string, pos int, trimSpace bool, changes []matchChange) (int, bool, []matchChange) {
	return matchStructuredSteps(m.steps, 0, dst, content, pos, trimSpace, changes)
}

func matchStructuredSteps(steps []structuredStep, idx int, dst []string, content string, pos int, trimSpace bool, changes []matchChange) (int, bool, []matchChange) {
	if idx >= len(steps) {
		return pos, true, changes
	}

	step := steps[idx]
	mark := len(changes)
	next, ok, nextChanges := matchStructuredStep(step, dst, content, pos, trimSpace, changes)
	if ok {
		if end, okEnd, endChanges := matchStructuredSteps(steps, idx+1, dst, content, next, trimSpace, nextChanges); okEnd {
			return end, true, endChanges
		}
		if step.writes {
			rollbackMatchChanges(dst, nextChanges, mark)
		}
		nextChanges = nextChanges[:mark]
	}

	if step.optional {
		return matchStructuredSteps(steps, idx+1, dst, content, pos, trimSpace, changes[:mark])
	}

	return 0, false, changes[:mark]
}

func matchStructuredStep(step structuredStep, dst []string, content string, pos int, trimSpace bool, changes []matchChange) (int, bool, []matchChange) {
	switch {
	case step.literal != "":
		if !strings.HasPrefix(content[pos:], step.literal) {
			return 0, false, changes
		}
		return pos + len(step.literal), true, changes
	case step.parser != nil:
		next, value, ok := step.parser.consume(content, pos)
		if !ok {
			return 0, false, changes
		}
		if step.parser.dstIndex >= 0 {
			changes = appendMatchChange(dst, changes, step.parser.dstIndex, maybeTrim(value, trimSpace))
		}
		return next, true, changes
	case step.submatcher != nil:
		mark := len(changes)
		next, ok, nextChanges := step.submatcher.matchAnyFrom(dst, content, pos, trimSpace, changes)
		if !ok {
			if step.submatcher.writes {
				rollbackMatchChanges(dst, nextChanges, mark)
				nextChanges = nextChanges[:mark]
			}
			return 0, false, changes
		}
		if step.captureIndex >= 0 {
			nextChanges = appendMatchChange(dst, nextChanges, step.captureIndex, maybeTrim(content[pos:next], trimSpace))
		}
		return next, true, nextChanges
	case len(step.alternatives) > 0:
		for _, alt := range matchingAlternatives(step, content, pos) {
			mark := len(changes)
			next, ok, nextChanges := alt.matchAnyFrom(dst, content, pos, trimSpace, changes)
			if ok {
				if step.captureIndex >= 0 {
					nextChanges = appendMatchChange(dst, nextChanges, step.captureIndex, maybeTrim(content[pos:next], trimSpace))
				}
				return next, true, nextChanges
			}
			if alt.writes {
				rollbackMatchChanges(dst, nextChanges, mark)
				nextChanges = nextChanges[:mark]
			}
		}
		return 0, false, changes
	default:
		return pos, true, changes
	}
}

func matchingAlternatives(step structuredStep, content string, pos int) []*structuredMatcher {
	if len(step.altPrefixes) == 0 {
		return step.alternatives
	}
	for i, prefix := range step.altPrefixes {
		if strings.HasPrefix(content[pos:], prefix) {
			return step.alternatives[i : i+1]
		}
	}
	return nil
}

func appendMatchChange(dst []string, changes []matchChange, idx int, value string) []matchChange {
	if idx < 0 {
		return changes
	}
	changes = append(changes, matchChange{idx: idx, prev: dst[idx]})
	dst[idx] = value
	return changes
}

func rollbackMatchChanges(dst []string, changes []matchChange, mark int) {
	for i := len(changes) - 1; i >= mark; i-- {
		change := changes[i]
		dst[change.idx] = change.prev
	}
}

func (p *structuredParser) consume(content string, pos int) (next int, value string, ok bool) {
	if pos > len(content) {
		return 0, "", false
	}

	segment, next, ok := p.slice(content, pos)
	if !ok {
		return 0, "", false
	}

	switch p.kind {
	case structuredWord:
		if !isApacheWord(segment) {
			return 0, "", false
		}
	case structuredHostName:
		if segment == "" {
			return 0, "", false
		}
	case structuredCharClass:
		if !p.allowEmpty && segment == "" {
			return 0, "", false
		}
	case structuredNotSpace:
		if segment == "" || strings.IndexAny(segment, " \t\r\n\f\v") >= 0 {
			return 0, "", false
		}
	case structuredNumber:
		if !isApacheNumber(segment) {
			return 0, "", false
		}
	case structuredMonthName:
		if !isMonthNameValue(segment) {
			return 0, "", false
		}
	case structuredDayName:
		if !isDayNameValue(segment) {
			return 0, "", false
		}
	case structuredTimeOfDay:
		if !looksLikeTimeOfDay(segment) {
			return 0, "", false
		}
	case structuredYear:
		if !isYearValue(segment) {
			return 0, "", false
		}
	case structuredMonthNum:
		if !isMonthNumValue(segment) {
			return 0, "", false
		}
	case structuredMonthDay:
		if !isMonthDayValue(segment) {
			return 0, "", false
		}
	case structuredHour:
		if !isHourValue(segment) {
			return 0, "", false
		}
	case structuredMinute:
		if !isMinuteValue(segment) {
			return 0, "", false
		}
	case structuredSecond:
		if !isSecondValue(segment) {
			return 0, "", false
		}
	case structuredQuoted:
		if len(segment) < 2 {
			return 0, "", false
		}
		q := segment[0]
		if (q != '"' && q != '\'') || segment[len(segment)-1] != q {
			return 0, "", false
		}
	case structuredUntilLiteral, structuredGreedyUntilLiteral:
		if segment == "" && !p.allowEmpty {
			return 0, "", false
		}
	case structuredSpacePlus:
		if segment == "" || !isOnlySpace(segment) {
			return 0, "", false
		}
	case structuredSpaceStar:
		if !isOnlySpace(segment) {
			return 0, "", false
		}
	case structuredTimestampISO8601:
		if !looksLikeTimestampISO8601(segment) {
			return 0, "", false
		}
	case structuredHTTPDate:
		if !looksLikeHTTPDate(segment) {
			return 0, "", false
		}
	case structuredLogLevel:
		if !isLogLevel(segment) {
			return 0, "", false
		}
	default:
		return 0, "", false
	}

	return next, segment, true
}

func (p *structuredParser) slice(content string, pos int) (segment string, next int, ok bool) {
	switch p.kind {
	case structuredSpacePlus, structuredSpaceStar:
		i := pos
		for i < len(content) && isASCIISpace(content[i]) {
			i++
		}
		if p.kind == structuredSpacePlus && i == pos {
			return "", 0, false
		}
		return content[pos:i], i, true
	}

	if p.hasNextParser {
		switch p.kind {
		case structuredWord:
			return sliceWord(content, pos)
		case structuredHostName:
			return sliceHostName(content, pos)
		case structuredCharClass:
			return sliceCharClass(content, pos, p.charClass, p.allowEmpty)
		case structuredNotSpace:
			return sliceNotSpaceWithContext(content, pos, p.nextParser, p.nextLiteral)
		case structuredNumber:
			return sliceNumberWithContext(content, pos, p.nextParser, p.nextLiteral)
		case structuredMonthName, structuredDayName:
			return sliceAlphaToken(content, pos)
		case structuredTimeOfDay:
			return sliceTimeOfDayWithContext(content, pos)
		case structuredYear:
			return sliceYear(content, pos)
		case structuredMonthNum, structuredMonthDay, structuredHour, structuredMinute, structuredSecond:
			return sliceDigitsAndPunct(content, pos)
		case structuredQuoted:
			return sliceQuoted(content, pos)
		case structuredTimestampISO8601:
			return sliceTimestampISO8601(content, pos)
		case structuredLogLevel:
			return sliceLogLevelWithContext(content, pos, p.nextParser, p.nextLiteral)
		}
	}

	if p.nextLiteral != "" {
		switch p.kind {
		case structuredTimestampISO8601:
			return sliceTimestampISO8601(content, pos)
		case structuredHostName:
			return sliceHostName(content, pos)
		case structuredMonthName, structuredDayName:
			return sliceAlphaToken(content, pos)
		case structuredTimeOfDay:
			return sliceTimeOfDayWithContext(content, pos)
		case structuredYear:
			return sliceYear(content, pos)
		case structuredMonthNum, structuredMonthDay, structuredHour, structuredMinute, structuredSecond:
			return sliceDigitsAndPunct(content, pos)
		}
		rest := content[pos:]
		var idx int
		if p.kind == structuredGreedyUntilLiteral {
			idx = lastLiteralIndex(rest, p.nextLiteral)
		} else {
			idx = firstLiteralIndex(rest, p.nextLiteral)
		}
		if idx < 0 {
			return "", 0, false
		}

		return rest[:idx], pos + idx, true
	}

	switch p.kind {
	case structuredWord:
		return sliceWord(content, pos)
	case structuredHostName:
		return sliceHostName(content, pos)
	case structuredCharClass:
		return sliceCharClass(content, pos, p.charClass, p.allowEmpty)
	case structuredNotSpace:
		return sliceNotSpace(content, pos)
	case structuredNumber:
		return sliceNumber(content, pos)
	case structuredMonthName, structuredDayName:
		return sliceAlphaToken(content, pos)
	case structuredTimeOfDay:
		return sliceTimeOfDay(content, pos)
	case structuredYear:
		return sliceYear(content, pos)
	case structuredMonthNum, structuredMonthDay, structuredHour, structuredMinute, structuredSecond:
		return sliceDigitsAndPunct(content, pos)
	case structuredQuoted:
		return sliceQuoted(content, pos)
	case structuredTimestampISO8601:
		return sliceTimestampISO8601(content, pos)
	case structuredLogLevel:
		return sliceLogLevel(content, pos)
	}

	return content[pos:], len(content), true
}

func canParserEndWithoutLiteral(kind structuredKind) bool {
	switch kind {
	case structuredWord, structuredHostName, structuredCharClass, structuredNotSpace, structuredNumber, structuredMonthName, structuredDayName, structuredTimeOfDay, structuredYear, structuredMonthNum, structuredMonthDay, structuredHour, structuredMinute, structuredSecond, structuredQuoted, structuredSpacePlus, structuredSpaceStar, structuredTimestampISO8601, structuredLogLevel:
		return true
	default:
		return false
	}
}

func sliceWord(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) && isWordByte(content[pos]) {
		pos++
	}
	if pos == start {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func isHostnameStartByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func isHostnameByte(b byte) bool {
	return isHostnameStartByte(b) || b == '-' || b == '.'
}

func sliceCharClass(content string, pos int, class *asciiCharClass, allowEmpty bool) (string, int, bool) {
	if class == nil {
		return "", 0, false
	}
	start := pos
	for pos < len(content) && class.table[content[pos]] {
		pos++
	}
	if pos == start && !allowEmpty {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func sliceNotSpace(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) && !isASCIISpace(content[pos]) {
		pos++
	}
	if pos == start {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func sliceHostName(content string, pos int) (string, int, bool) {
	start := pos
	if pos >= len(content) || !isHostnameStartByte(content[pos]) {
		return "", 0, false
	}
	pos++
	for pos < len(content) && isHostnameByte(content[pos]) {
		pos++
	}
	return content[start:pos], pos, true
}

func sliceNotSpaceWithContext(content string, pos int, nextParser structuredKind, nextLiteral string) (string, int, bool) {
	if (nextParser == structuredSpacePlus || nextParser == structuredSpaceStar) && nextLiteral != "" {
		return sliceTokenBeforeSpaceOrLiteral(content, pos, nextLiteral)
	}
	return sliceNotSpace(content, pos)
}

func sliceNumber(content string, pos int) (string, int, bool) {
	start := pos
	if pos < len(content) && (content[pos] == '+' || content[pos] == '-') {
		pos++
	}
	digits := 0
	dotSeen := false
	for pos < len(content) {
		switch {
		case content[pos] >= '0' && content[pos] <= '9':
			digits++
			pos++
		case content[pos] == '.' && !dotSeen:
			dotSeen = true
			pos++
		default:
			goto done
		}
	}
done:
	if digits == 0 {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func sliceNumberWithContext(content string, pos int, nextParser structuredKind, nextLiteral string) (string, int, bool) {
	if (nextParser == structuredSpacePlus || nextParser == structuredSpaceStar) && nextLiteral != "" {
		return sliceTokenBeforeSpaceOrLiteral(content, pos, nextLiteral)
	}
	return sliceNumber(content, pos)
}

func sliceQuoted(content string, pos int) (string, int, bool) {
	if pos >= len(content) || (content[pos] != '"' && content[pos] != '\'') {
		return "", 0, false
	}
	q := content[pos]
	start := pos
	pos++
	for pos < len(content) {
		if content[pos] == '\\' && pos+1 < len(content) {
			pos += 2
			continue
		}
		if content[pos] == q {
			pos++
			return content[start:pos], pos, true
		}
		pos++
	}
	return "", 0, false
}

func sliceTimestampISO8601(content string, pos int) (string, int, bool) {
	next, ok := consumeTimestampISO8601(content, pos)
	if !ok || next == pos {
		return "", 0, false
	}
	return content[pos:next], next, true
}

func sliceLogLevel(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) {
		c := content[pos]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			pos++
			continue
		}
		break
	}
	if pos == start {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func sliceLogLevelWithContext(content string, pos int, nextParser structuredKind, nextLiteral string) (string, int, bool) {
	if (nextParser == structuredSpacePlus || nextParser == structuredSpaceStar) && nextLiteral != "" {
		start := pos
		for pos < len(content) {
			c := content[pos]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				pos++
				continue
			}
			break
		}
		if pos == start {
			return "", 0, false
		}
		return content[start:pos], pos, true
	}
	return sliceLogLevel(content, pos)
}

func sliceTokenBeforeSpaceOrLiteral(content string, pos int, nextLiteral string) (string, int, bool) {
	if nextLiteral == "" || pos >= len(content) {
		return "", 0, false
	}

	first := nextLiteral[0]
	for i := pos; i < len(content); i++ {
		c := content[i]
		if c == first && strings.HasPrefix(content[i:], nextLiteral) {
			return content[pos:i], i, i > pos
		}
		if !isASCIISpace(c) {
			continue
		}

		j := i
		for j < len(content) && isASCIISpace(content[j]) {
			j++
		}
		if strings.HasPrefix(content[j:], nextLiteral) {
			return content[pos:i], i, i > pos
		}
		return "", 0, false
	}
	return "", 0, false
}

func firstLiteralIndex(content, literal string) int {
	if literal == "" {
		return 0
	}
	if len(literal) == 1 {
		return strings.IndexByte(content, literal[0])
	}

	first := literal[0]
	search := content
	offset := 0
	for {
		idx := strings.IndexByte(search, first)
		if idx < 0 {
			return -1
		}
		idx += offset
		if strings.HasPrefix(content[idx:], literal) {
			return idx
		}
		offset = idx + 1
		search = content[offset:]
	}
}

func lastLiteralIndex(content, literal string) int {
	if literal == "" {
		return len(content)
	}
	if len(literal) == 1 {
		return strings.LastIndexByte(content, literal[0])
	}

	first := literal[0]
	end := len(content)
	for end > 0 {
		idx := strings.LastIndexByte(content[:end], first)
		if idx < 0 {
			return -1
		}
		if strings.HasPrefix(content[idx:], literal) {
			return idx
		}
		end = idx
	}
	return -1
}

func sliceYear(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) && content[pos] >= '0' && content[pos] <= '9' && pos-start < 4 {
		pos++
	}
	if pos-start != 2 && pos-start != 4 {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func sliceTimeOfDay(content string, pos int) (string, int, bool) {
	next, ok := consumePatternTimeOfDay(content, pos, true, false)
	if !ok || next == pos {
		return "", 0, false
	}
	return content[pos:next], next, true
}

func sliceTimeOfDayWithContext(content string, pos int) (string, int, bool) {
	next, ok := consumePatternTimeOfDay(content, pos, true, false)
	if !ok || next == pos {
		return "", 0, false
	}
	return content[pos:next], next, true
}

func sliceDigitsAndPunct(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) {
		c := content[pos]
		if (c >= '0' && c <= '9') || c == '.' || c == ',' {
			pos++
			continue
		}
		break
	}
	if pos == start {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func isOnlySpace(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isASCIISpace(s[i]) {
			return false
		}
	}
	return true
}

func isYearValue(s string) bool {
	return len(s) == 2 || len(s) == 4
}

func isMonthNumValue(s string) bool {
	n, ok := parsePositiveInt(s)
	return ok && n >= 1 && n <= 12 && len(s) <= 2
}

func isMonthDayValue(s string) bool {
	n, ok := parsePositiveInt(s)
	return ok && n >= 1 && n <= 31 && len(s) <= 2
}

func isHourValue(s string) bool {
	n, ok := parsePositiveInt(s)
	return ok && n >= 0 && n <= 23 && len(s) <= 2
}

func isMinuteValue(s string) bool {
	n, ok := parsePositiveInt(s)
	return ok && n >= 0 && n <= 59 && len(s) == 2
}

func isSecondValue(s string) bool {
	if s == "" {
		return false
	}
	main := s
	if idx := strings.IndexAny(s, ".,"); idx >= 0 {
		main = s[:idx]
		if idx == len(s)-1 {
			return false
		}
		for i := idx + 1; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		}
	}
	n, ok := parsePositiveInt(main)
	return ok && n >= 0 && n <= 60 && len(main) >= 1 && len(main) <= 2
}

func parsePositiveInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}

func looksLikeTimestampISO8601(s string) bool {
	if len(s) < 16 {
		return false
	}
	hasDateSep := false
	hasTimeSep := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c == '-' || c == '/' || c == 'T' || c == ':' || c == '.' || c == ',' || c == '+' || c == 'Z' || c == 'z' || c == ' ':
			if c == '-' || c == '/' {
				hasDateSep = true
			}
			if c == ':' {
				hasTimeSep = true
			}
		default:
			return false
		}
	}
	return hasDateSep && hasTimeSep
}

func looksLikeTimeOfDay(s string) bool {
	next, ok := consumePatternTimeOfDay(s, 0, true, false)
	return ok && next == len(s)
}

func looksLikeHTTPDate(s string) bool {
	if len(s) < 20 {
		return false
	}
	return strings.Count(s, "/") >= 2 && strings.Count(s, ":") >= 2
}

func isLogLevel(s string) bool {
	return isLogLevelValue(s)
}
