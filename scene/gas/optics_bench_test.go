package gas

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

var sinkF float64

func BenchmarkGasOptics(b *testing.B) {
	g := Haze()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += g.Extinction(maths.Band{Nanometres: 400 + float64(i%300)})
	}
}

func BenchmarkGasOpticsAir(b *testing.B) {
	g := Air()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkF += g.Extinction(maths.Band{Nanometres: 400 + float64(i%300)})
	}
}

// BenchmarkGasOpticsOld is the old derivation, for the same minute as
// BenchmarkGasOptics.
func BenchmarkGasOpticsOld(b *testing.B) {
	g := Haze()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a, s := oldOptics(g, maths.Band{Nanometres: 400 + float64(i%300)})
		sinkF += a + s
	}
}
