package thing

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// benchExcited is a star-lit gas of four lines, as the pillars glow.
func benchExcited() Excited {
	return Excited{
		Lines:       map[float64]float64{656.3: 1, 500.7: 0.5, 672.4: 0.3, 486.1: 0.2},
		Excitations: map[float64]float64{656.3: 2, 500.7: 4},
		Width:       2, Passes: 0.3,
	}
}

var sinkF float64

func BenchmarkExcitedEmits(b *testing.B) {
	e := benchExcited()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += e.Emits(maths.Band{Nanometres: 500 + float64(i%200)})
	}
}

func BenchmarkExcitedExcitation(b *testing.B) {
	e := benchExcited()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += e.Excitation(maths.Band{Nanometres: 500 + float64(i%200)})
	}
}

func BenchmarkExcitedEmitsPrepared(b *testing.B) {
	e := benchExcited().Prepared()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += e.Emits(maths.Band{Nanometres: 500 + float64(i%200)})
	}
}

func BenchmarkExcitedExcitationPrepared(b *testing.B) {
	e := benchExcited().Prepared()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += e.Excitation(maths.Band{Nanometres: 500 + float64(i%200)})
	}
}

// BenchmarkExcitedEmitsOld is the old Emits, for the same minute.
func BenchmarkExcitedEmitsOld(b *testing.B) {
	e := benchExcited()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += oldEmits(e, maths.Band{Nanometres: 500 + float64(i%200)})
	}
}
