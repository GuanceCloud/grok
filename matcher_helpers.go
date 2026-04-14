package grok

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var monthNameValues = [...]string{
	"jan", "january", "januar",
	"feb", "february", "februar",
	"mr", "mar", "mär", "mrch", "march", "mrz", "märz",
	"apr", "april",
	"ma", "may", "mai",
	"jun", "june", "juni",
	"jul", "july",
	"aug", "august",
	"sep", "september",
	"ot", "oct", "okt", "october",
	"nov", "november",
	"dec", "dez", "december", "dezember",
}

var dayNameValues = [...]string{
	"mon", "monday",
	"tue", "tuesday",
	"wed", "wednesday",
	"thu", "thursday",
	"fri", "friday",
	"sat", "saturday",
	"sun", "sunday",
}

var logLevelValues = [...]string{
	"alert", "trace", "debug", "notice", "info", "warn", "warning",
	"war", "err", "er", "error", "crit", "cri", "critical", "fatal", "severe", "emerg", "emergency",
}

func isApacheWord(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if isWordByte(s[i]) {
			continue
		}
		return false
	}
	return true
}

func isApacheNumber(s string) bool {
	if len(s) == 0 {
		return false
	}

	i := 0
	if s[0] == '+' || s[0] == '-' {
		i++
		if i == len(s) {
			return false
		}
	}

	digits := 0
	dotSeen := false
	for ; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digits++
		case s[i] == '.' && !dotSeen:
			dotSeen = true
		default:
			return false
		}
	}

	return digits > 0
}

func maybeTrim(s string, trim bool) string {
	if !trim {
		return s
	}
	return trimMatch(s)
}

func consumeTimestampISO8601(s string, start int) (int, bool) {
	i := start
	if !consumeNDigits(s, &i, 4) || i >= len(s) || s[i] != '-' {
		return 0, false
	}
	i++
	if !consumeOneOrTwoDigits(s, &i) || i >= len(s) || s[i] != '-' {
		return 0, false
	}
	i++
	if !consumeOneOrTwoDigits(s, &i) || i >= len(s) || (s[i] != 'T' && s[i] != ' ') {
		return 0, false
	}
	i++
	if !consumeOneOrTwoDigits(s, &i) {
		return 0, false
	}
	if i < len(s) && s[i] == ':' {
		i++
	}
	if !consumeNDigits(s, &i, 2) {
		return 0, false
	}
	if i < len(s) && s[i] == ':' {
		i++
		if !consumeOneOrTwoDigits(s, &i) {
			return 0, false
		}
		if i < len(s) && (s[i] == '.' || s[i] == ',') {
			i++
			if !consumeAtLeastOneDigit(s, &i) {
				return 0, false
			}
		}
	}

	if i < len(s) {
		switch s[i] {
		case 'Z', 'z':
			i++
		case '+', '-':
			i++
			if !consumeOneOrTwoDigits(s, &i) {
				return 0, false
			}
			if i < len(s) && s[i] == ':' {
				i++
			}
			_ = consumeNDigits(s, &i, 2)
		}
	}

	return i, true
}

func consumeTimeOfDay(s string, start int) (int, bool) {
	i := start
	if !consumeOneOrTwoDigits(s, &i) || i >= len(s) || s[i] != ':' {
		return 0, false
	}
	i++
	if !consumeNDigits(s, &i, 2) {
		return 0, false
	}
	if i < len(s) && s[i] == ':' {
		i++
		if !consumeOneOrTwoDigits(s, &i) {
			return 0, false
		}
		if i < len(s) && (s[i] == '.' || s[i] == ',') {
			i++
			if !consumeAtLeastOneDigit(s, &i) {
				return 0, false
			}
		}
	}
	return i, true
}

func consumePatternTimeOfDay(s string, start int, allowLeadingNonDigit bool, allowTrailingNonDigit bool) (int, bool) {
	i := start
	if allowLeadingNonDigit && i < len(s) && !isASCIIDigit(s[i]) {
		i++
	}
	next, ok := consumeTimeOfDay(s, i)
	if !ok {
		return 0, false
	}
	i = next
	if allowTrailingNonDigit && i < len(s) && !isASCIIDigit(s[i]) {
		i++
	}
	return i, true
}

func consumeNDigits(s string, i *int, n int) bool {
	if *i+n > len(s) {
		return false
	}
	for j := 0; j < n; j++ {
		if s[*i+j] < '0' || s[*i+j] > '9' {
			return false
		}
	}
	*i += n
	return true
}

func consumeOneOrTwoDigits(s string, i *int) bool {
	if *i >= len(s) || s[*i] < '0' || s[*i] > '9' {
		return false
	}
	*i = *i + 1
	if *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
		*i = *i + 1
	}
	return true
}

func consumeAtLeastOneDigit(s string, i *int) bool {
	start := *i
	for *i < len(s) && s[*i] >= '0' && s[*i] <= '9' {
		*i = *i + 1
	}
	return *i > start
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func sliceAlphaToken(content string, pos int) (string, int, bool) {
	start := pos
	for pos < len(content) {
		r, size := utf8.DecodeRuneInString(content[pos:])
		if r == utf8.RuneError && size == 1 {
			break
		}
		if !unicode.IsLetter(r) {
			break
		}
		pos += size
	}
	if pos == start {
		return "", 0, false
	}
	return content[start:pos], pos, true
}

func isMonthNameValue(s string) bool {
	for _, candidate := range monthNameValues {
		if equalFoldLiteral(s, candidate) {
			return true
		}
	}
	return false
}

func isDayNameValue(s string) bool {
	for _, candidate := range dayNameValues {
		if equalFoldLiteral(s, candidate) {
			return true
		}
	}
	return false
}

func isLogLevelValue(s string) bool {
	for _, candidate := range logLevelValues {
		if equalFoldLiteral(s, candidate) {
			return true
		}
	}
	return false
}

func equalFoldLiteral(s, literal string) bool {
	if len(s) != len(literal) {
		return strings.EqualFold(s, literal)
	}
	for i := 0; i < len(s); i++ {
		a := s[i]
		b := literal[i]
		if a == b {
			continue
		}
		if a >= utf8.RuneSelf || b >= utf8.RuneSelf {
			return strings.EqualFold(s, literal)
		}
		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}
		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}
