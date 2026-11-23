package skylight

import (
	"math"
	"math/rand/v2"
	"testing"
)

// The numbers of the tests of each photon's own draws.
const (
	someNumber   = 5     // picked: any photon's number
	fewDraws     = 3     // picked: a photon that drew a few numbers
	moreDraws    = 7     // picked: one that drew more
	nearerBand   = 1.1   // picked: a bluer band, its air a tenth thicker
	ratioPhotons = 20000 // picked: enough for a ratio in a second
)

// drawsAfter is the first number the photon after someNumber draws, from
// a walker that drew so many numbers for someNumber first.
func drawsAfter(drawn int) float64 {
	w := &walker{random: rand.NewPCG(seedOf, none), seed: seedOf}
	w.begin(someNumber)
	for range drawn {
		w.uniform()
	}
	w.begin(someNumber + one)
	return w.uniform()
}

// a photon draws the same numbers whatever the photon before it drew,
// so one walked at two wavelengths walks from the same numbers.
func TestAPhotonDrawsItsOwnNumbers(t *testing.T) {
	few, more := drawsAfter(fewDraws), drawsAfter(moreDraws)
	if few != more {
		t.Errorf("the photon after one that drew %d numbers drew %g, "+
			"after one that drew %d, %g", fewDraws, few, moreDraws, more)
	}
}

// the anti-test: another photon draws other numbers, so the check would
// see a walk that seeded every photon alike.
func TestAnotherPhotonDrawsOtherNumbers(t *testing.T) {
	w := &walker{random: rand.NewPCG(seedOf, none), seed: seedOf}
	w.begin(someNumber)
	first := w.uniform()
	w.begin(someNumber + one)
	if second := w.uniform(); first == second {
		t.Errorf("photons %d and %d both drew %g first", someNumber,
			someNumber+one, first)
	}
}

// thicker is the mixed column with every shell's scattering and
// extinction so many times as much: the same air at a bluer wavelength.
func thicker(times float64) Column {
	column := mixed()
	for index := range column.Shells {
		column.Shells[index].GasScattering *= times
		column.Shells[index].ParticleExtinction *= times
	}
	return column
}

// ratioSpread is how far the log of the ratio of two fields' radiance
// strays from its mean over the bins both hold light in: the noise a
// colour made of the two would show.
func ratioSpread(t *testing.T, first, second *Field) float64 {
	t.Helper()
	var logs []float64
	for bin, light := range first.radiance {
		if light > none && second.radiance[bin] > none {
			logs = append(logs, math.Log(light/second.radiance[bin]))
		}
	}
	if len(logs) == none {
		t.Fatal("no bin holds light in both fields")
	}
	mean := 0.0
	for _, each := range logs {
		mean += each / float64(len(logs))
	}
	spread := 0.0
	for _, each := range logs {
		spread += (each - mean) * (each - mean) / float64(len(logs))
	}
	return math.Sqrt(spread)
}

// walked is the field of a column walked from a seed on the ground grid.
func walked(t *testing.T, column Column, seed uint64) *Field {
	t.Helper()
	field, _, err := Solve(column, groundGrid(), ratioPhotons, seed,
		workersOf)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// two wavelengths walked from one seed make a ratio, a colour, with less
// noise than the same two walked from two seeds.
func TestOneSeedKeepsTheColourQuieter(t *testing.T) {
	near := walked(t, thicker(one), seedOf)
	togetherAir := thicker(nearerBand)
	together := ratioSpread(t, near, walked(t, togetherAir, seedOf))
	apartAir := thicker(nearerBand)
	apart := ratioSpread(t, near, walked(t, apartAir,
		seedOf+anotherSeedFrom))
	if !(together < apart) {
		t.Errorf("the ratio's spread from one seed is %g, from two %g",
			together, apart)
	}
}

// the anti-test: one air from two seeds has a ratio that strays, so the
// spread sees noise, and from one seed has none.
func TestTheSpreadSeesNoise(t *testing.T) {
	air := walked(t, thicker(one), seedOf)
	sameAir := thicker(one)
	same := ratioSpread(t, air, walked(t, sameAir, seedOf))
	if same != none {
		t.Errorf("one air from one seed strays by %g", same)
	}
	apartAir := thicker(one)
	apart := ratioSpread(t, air, walked(t, apartAir,
		seedOf+anotherSeedFrom))
	if !(apart > none) {
		t.Errorf("one air from two seeds strays by %g", apart)
	}
}
