package grok

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomScannerScanIntoMarksHits(t *testing.T) {
	scanner := newAtomScanner([]string{"foo", "bar", "baz", "trace="})
	hits := make([]bool, 4)

	scanner.ScanInto("level=info trace=req-42 foo message", hits)

	assert.Equal(t, []bool{true, false, false, true}, hits)
}

func TestAtomScannerScanHandlesRepeatedAtoms(t *testing.T) {
	scanner := newAtomScanner([]string{"foo", "oo", "bar"})
	result := scanner.Scan("xxfooyyfoo")

	assert.True(t, result.Has(0))
	assert.True(t, result.Has(1))
	assert.False(t, result.Has(2))
}

func TestAtomScannerScanHandlesSharedPrefixes(t *testing.T) {
	scanner := newAtomScanner([]string{"trace=", "trace=req", "trace=req-42", "req-42"})
	result := scanner.Scan("service trace=req-42 accepted")

	assert.True(t, result.Has(0))
	assert.True(t, result.Has(1))
	assert.True(t, result.Has(2))
	assert.True(t, result.Has(3))
}

func BenchmarkAtomScannerScan(b *testing.B) {
	atoms := []string{
		"service=checkout",
		"level=INFO",
		"trace=req-42",
		"msg=payment accepted",
		"client=",
		"status=",
		"node=",
		"[ERROR]",
	}
	line := "service=checkout level=INFO trace=req-42 msg=payment accepted"
	scanner := newAtomScanner(atoms)

	b.Run("scanner", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hits := make([]bool, len(atoms))
			scanner.ScanInto(line, hits)
		}
	})

	b.Run("contains_loop", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hits := make([]bool, len(atoms))
			for j, atom := range atoms {
				hits[j] = strings.Contains(line, atom)
			}
		}
	})
}

func BenchmarkAtomScannerSharedPrefixes(b *testing.B) {
	atoms := []string{
		"trace=",
		"trace=req",
		"trace=req-42",
		"trace=req-420",
		"trace=req-4200",
		"req-42",
		"req-420",
		"req-4200",
	}
	line := "service=checkout trace=req-42 msg=payment accepted"
	scanner := newAtomScanner(atoms)

	b.Run("scanner", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hits := make([]bool, len(atoms))
			scanner.ScanInto(line, hits)
		}
	})

	b.Run("contains_loop", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hits := make([]bool, len(atoms))
			for j, atom := range atoms {
				hits[j] = strings.Contains(line, atom)
			}
		}
	})
}
