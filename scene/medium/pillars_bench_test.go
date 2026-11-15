package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

var sinkF float64

func BenchmarkPillarsDensity(b *testing.B) {
	p := Pillars{Count: 3, Height: 7, Width: 10, Thick: 1.6, Seed: 9, Filaments: 0.5}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := float64(i % 97)
		sinkF += p.Density(maths.V(f/10-4.8, f/14, f/12-4))
	}
}

// BenchmarkPillarsDensityOld is the old Density, for the same minute as
// BenchmarkPillarsDensity.
func BenchmarkPillarsDensityOld(b *testing.B) {
	p := Pillars{Count: 3, Height: 7, Width: 10, Thick: 1.6, Seed: 9, Filaments: 0.5}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := float64(i % 97)
		sinkF += oldDensity(p, maths.V(f/10-4.8, f/14, f/12-4))
	}
}

func BenchmarkPillarsDensityPrepared(b *testing.B) {
	p := Pillars{Count: 3, Height: 7, Width: 10, Thick: 1.6, Seed: 9, Filaments: 0.5}.Prepared()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := float64(i % 97)
		sinkF += p.Density(maths.V(f/10-4.8, f/14, f/12-4))
	}
}
