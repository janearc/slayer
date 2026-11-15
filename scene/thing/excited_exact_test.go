package thing

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
)

// oldLines, oldEmits and oldExcitation are Excited's methods as they were
// written, the lines sorted out of the map at every call.
func oldLines(e Excited) []float64 {
	out := make([]float64, 0, len(e.Lines))
	for line := range e.Lines {
		out = append(out, line)
	}
	sort.Float64s(out)
	return out
}

func oldEmits(e Excited, band maths.Band) float64 {
	width := e.Width
	if width <= 0 {
		width = 1
	}
	sum := 0.0
	for _, line := range oldLines(e) {
		strength := e.Lines[line]
		d := (band.Nanometres - line) / width
		sum += strength * math.Exp(-d*d)
	}
	return sum
}

func oldExcitation(e Excited, band maths.Band) float64 {
	nearest, apart := 0.0, math.Inf(1)
	for _, line := range oldLines(e) {
		if d := math.Abs(band.Nanometres - line); d < apart {
			nearest, apart = line, d
		}
	}
	if x, is := e.Excitations[nearest]; is && x > 0 {
		return x
	}
	return 1
}

// randomExcited is a gas of up to a dozen lines, some with excitations of
// zero, below zero or none, and a width that is sometimes not above zero.
func randomExcited(r *rand.Rand) Excited {
	e := Excited{Lines: map[float64]float64{}, Excitations: map[float64]float64{},
		Width: r.Float64()*6 - 1, Passes: r.Float64()}
	for n := r.Intn(13); n > 0; n-- {
		nm := math.Round(r.Float64()*4000) / 4
		e.Lines[nm] = r.Float64() * 3
		switch r.Intn(4) {
		case 0:
			e.Excitations[nm] = r.Float64()*4 - 1
		case 1:
			e.Excitations[nm] = r.Float64() * 4
		}
	}
	if r.Intn(5) == 0 {
		e.Excitations[0] = 3 // the entry an empty gas is read by
	}
	return e
}

// a gas built by a literal, and the same one prepared, answer every band
// as the old methods did, bit for bit, over seeded random gases and bands.
func TestExcitedIsTheOldArithmeticBitForBit(t *testing.T) {
	r := rand.New(rand.NewSource(20260910))
	special := []float64{0, math.NaN(), math.Inf(1), math.Inf(-1), -1, 1e300}
	for n := 0; n < 3000; n++ {
		lit := randomExcited(r)
		prep := lit.Prepared()
		bandsAt := append([]float64{}, special...)
		for k := 0; k < 40; k++ {
			bandsAt = append(bandsAt, r.Float64()*1100)
		}
		for nm := range lit.Lines {
			bandsAt = append(bandsAt, nm, nm+0.125)
		}
		for _, nm := range bandsAt {
			band := maths.Band{Nanometres: nm}
			for name, e := range map[string]Excited{"literal": lit, "prepared": prep} {
				if got, want := e.Emits(band), oldEmits(lit, band); math.Float64bits(got) != math.Float64bits(want) {
					t.Fatalf("%s %+v Emits(%v) = %v, was %v", name, lit, nm, got, want)
				}
				if got, want := e.Excitation(band), oldExcitation(lit, band); math.Float64bits(got) != math.Float64bits(want) {
					t.Fatalf("%s %+v Excitation(%v) = %v, was %v", name, lit, nm, got, want)
				}
			}
		}
	}
}

// a prepared gas is read from many goroutines at once with no write.
func TestAPreparedExcitedMayBeSharedAcrossWorkers(t *testing.T) {
	e := benchExcited().Prepared()
	want := oldEmits(e, maths.Band{Nanometres: 600})
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 2000; n++ {
				if got := e.Emits(maths.Band{Nanometres: 600}); got != want {
					t.Errorf("got %v, want %v", got, want)
					return
				}
				e.Excitation(maths.Band{Nanometres: 600})
			}
		}()
	}
	wg.Wait()
}
