package thing

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The outcomes of the draw that decides how a random line's excitation is
// given. Any other outcome leaves the line with none.
const (
	excitedExactMayBeBelowNothing = nought // defined: the first outcome
	excitedExactAboveNothing      = unity  // defined: the second outcome
)

// oldLines, oldEmits and oldExcitation are the reference for Excited's
// methods: the lines are sorted out of the map at every call. A prepared
// gas must answer as they do, bit for bit.
func oldLines(gas Excited) []float64 {
	lines := make([]float64, nought, len(gas.Lines))
	for line := range gas.Lines {
		lines = append(lines, line)
	}
	sort.Float64s(lines)
	return lines
}

// oldEmits is Excited.Emits over the lines sorted at every call.
func oldEmits(gas Excited, band maths.Band) float64 {
	width := gas.Width
	if width <= nought {
		width = unity
	}
	var sum float64
	for _, line := range oldLines(gas) {
		strength := gas.Lines[line]
		scaled := (band.Nanometres - line) / width
		sum += strength * math.Exp(-scaled*scaled)
	}
	return sum
}

// oldExcitation is Excited.Excitation over the lines sorted at every call.
func oldExcitation(gas Excited, band maths.Band) float64 {
	nearest, apart := float64(nought), math.Inf(unity)
	for _, line := range oldLines(gas) {
		if distance := math.Abs(band.Nanometres - line); distance < apart {
			nearest, apart = line, distance
		}
	}
	if x, is := gas.Excitations[nearest]; is && x > nought {
		return x
	}
	return unity
}

// excitedExactShape is how random gases are drawn: the ranges of the draws,
// each a number picked in numbers.yaml.
type excitedExactShape struct {
	widthScale, widthShift           float64
	mostLines                        int
	lineSpan, lineQuantum            float64
	strengthScale                    float64
	excitationKinds, emptyOdds       int
	excitationScale, excitationShift float64
	emptyGasExcitation               float64
}

// excitedExactShapeFor is the shape the tests draw their gases in.
func excitedExactShapeFor(t *testing.T) excitedExactShape {
	t.Helper()
	return excitedExactShape{
		widthScale:         picked(t, "excited_exact-width-scale"),
		widthShift:         picked(t, "excited_exact-width-shift"),
		mostLines:          pickedCount(t, "excited_exact-most-lines"),
		lineSpan:           picked(t, "excited_exact-line-span"),
		lineQuantum:        picked(t, "excited_exact-line-quantum"),
		strengthScale:      picked(t, "excited_exact-strength-scale"),
		excitationKinds:    pickedCount(t, "excited_exact-excitation-kinds"),
		emptyOdds:          pickedCount(t, "excited_exact-empty-odds"),
		excitationScale:    picked(t, "excited_exact-excitation-scale"),
		excitationShift:    picked(t, "excited_exact-excitation-shift"),
		emptyGasExcitation: picked(t, "excited_exact-empty-gas-excitation"),
	}
}

// randomExcited is a gas of up to a dozen lines, some with excitations of
// zero, below zero or none, and a width that is sometimes not above zero.
func randomExcited(source *rand.Rand, shape excitedExactShape) Excited {
	gas := Excited{Lines: map[float64]float64{},
		Excitations: map[float64]float64{},
		Width:       source.Float64()*shape.widthScale - shape.widthShift,
		Passes:      source.Float64()}
	for count := source.Intn(shape.mostLines); count > nought; count-- {
		excitedExactAddLine(source, shape, gas)
	}
	if source.Intn(shape.emptyOdds) == nought {
		// the entry an empty gas is read by
		gas.Excitations[nought] = shape.emptyGasExcitation
	}
	return gas
}

// excitedExactAddLine puts one random line, and perhaps its excitation, in
// the gas.
func excitedExactAddLine(source *rand.Rand, shape excitedExactShape,
	gas Excited) {
	steps := math.Round(source.Float64() * shape.lineSpan)
	nanometres := steps / shape.lineQuantum
	gas.Lines[nanometres] = source.Float64() * shape.strengthScale
	switch source.Intn(shape.excitationKinds) {
	case excitedExactMayBeBelowNothing:
		gas.Excitations[nanometres] =
			source.Float64()*shape.excitationScale - shape.excitationShift
	case excitedExactAboveNothing:
		gas.Excitations[nanometres] = source.Float64() * shape.excitationScale
	}
}

// excitedExactSpecialBands are the bands that are tried against every gas:
// nothing, not a number, both infinities, below nothing and past any line.
func excitedExactSpecialBands(t *testing.T) []float64 {
	t.Helper()
	return []float64{nought, math.NaN(), math.Inf(unity), math.Inf(-unity),
		picked(t, "excited_exact-below-band"),
		picked(t, "excited_exact-huge-band")}
}

// excitedExactBands are the special bands, random bands, and each line with
// a band on it and a band beside it.
func excitedExactBands(t *testing.T, source *rand.Rand,
	gas Excited) []float64 {
	t.Helper()
	bandsAt := append([]float64{}, excitedExactSpecialBands(t)...)
	each := pickedCount(t, "excited_exact-bands-each")
	span := picked(t, "excited_exact-band-span")
	for k := nought; k < each; k++ {
		bandsAt = append(bandsAt, source.Float64()*span)
	}
	offset := picked(t, "excited_exact-line-offset")
	for nanometres := range gas.Lines {
		bandsAt = append(bandsAt, nanometres, nanometres+offset)
	}
	return bandsAt
}

// excitedExactAgrees stops the test where a gas, literal or prepared, does
// not answer at a band as the reference does, bit for bit.
func excitedExactAgrees(t *testing.T, literal, prepared Excited,
	nanometres float64) {
	t.Helper()
	band := maths.NewBand(nanometres)
	for name, gas := range map[string]Excited{
		"literal": literal, "prepared": prepared} {
		got, want := gas.Emits(band), oldEmits(literal, band)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("%s %+v Emits(%v) = %v, want %v",
				name, literal, nanometres, got, want)
		}
		got, want = gas.Excitation(band), oldExcitation(literal, band)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("%s %+v Excitation(%v) = %v, want %v",
				name, literal, nanometres, got, want)
		}
	}
}

// a gas built by a literal, and the same one prepared, answer every band
// as the reference methods do, bit for bit, over seeded random gases and
// bands.
func TestExcitedIsTheOldArithmeticBitForBit(t *testing.T) {
	seed := int64(pickedCount(t, "excited_exact-seed"))
	source := rand.New(rand.NewSource(seed))
	shape := excitedExactShapeFor(t)
	gases := pickedCount(t, "excited_exact-gases")
	for round := nought; round < gases; round++ {
		literal := randomExcited(source, shape)
		prepared := literal.Prepared()
		for _, nanometres := range excitedExactBands(t, source, literal) {
			excitedExactAgrees(t, literal, prepared, nanometres)
		}
	}
}

// excitedExactRead reads a prepared gas many times at one band, and tells
// the group when it is done. It stops on the first answer that is not want.
func excitedExactRead(t *testing.T, group *sync.WaitGroup, gas Excited,
	band maths.Band, want float64, reads int) {
	defer group.Done()
	for read := nought; read < reads; read++ {
		if got := gas.Emits(band); got != want {
			t.Errorf("got %v, want %v", got, want)
			return
		}
		gas.Excitation(band)
	}
}

// a prepared gas is read from many goroutines at once with no write.
func TestAPreparedExcitedMayBeSharedAcrossWorkers(t *testing.T) {
	gas := benchExcited().Prepared()
	band := maths.NewBand(picked(t, "excited_exact-shared-band"))
	want := oldEmits(gas, band)
	workers := pickedCount(t, "excited_exact-workers")
	reads := pickedCount(t, "excited_exact-reads")
	var group sync.WaitGroup
	for worker := nought; worker < workers; worker++ {
		group.Add(unity)
		go excitedExactRead(t, &group, gas, band, want, reads)
	}
	group.Wait()
}
