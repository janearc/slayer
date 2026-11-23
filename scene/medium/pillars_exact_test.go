package medium

import (
	"math"
	"math/rand"
	"sync"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// pillarsExactCasesFile is the table of pillars the exactness tests try:
// ordinary ones, counts of zero and below, widths and heights of zero and
// below, and fields that are no number.
const pillarsExactCasesFile = "testdata/pillars-exact-cases.csv"

// pillarsExactFilamented is the name of the case that has filaments and
// ordinary columns, in the table.
const pillarsExactFilamented = "pillars-exact-filamented"

// pillarsExactPlacement is the numbers the longhand columns are placed by.
type pillarsExactPlacement struct {
	baseDivisor, heightLeast, heightRange float64
	seedX, seedZ, seedHeight              uint64
}

// pillarsExactShape is the numbers the longhand density is shaped by: the
// wall, the columns and the filaments.
type pillarsExactShape struct {
	wall, wallRag, wallScale, wallEdge, taper float64
	noiseScale, build, edge                   float64
	streakScale, streakStretch                float64
	seedNoise, seedStreak                     uint64
	wallOctaves, noiseOctaves, streakOctaves  int
}

// pillarsExactNumber is a number the longhand arithmetic works with, by its
// name in numbers.yaml. The longhand functions are called from benchmarks,
// which have no test to stop, so a name the file does not hold panics.
func pillarsExactNumber(name string) float64 {
	chosen, err := chosenNumbers()
	if err != nil {
		panic(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}

// pillarsExactCount is pillarsExactNumber as a whole number.
func pillarsExactCount(name string) int {
	return int(pillarsExactNumber(name))
}

// pillarsExactPlaced is the placement numbers, read once.
var pillarsExactPlaced = sync.OnceValue(loadPillarsExactPlacement)

// pillarsExactShaped is the shape numbers, read once.
var pillarsExactShaped = sync.OnceValue(loadPillarsExactShape)

// loadPillarsExactPlacement reads the placement numbers.
func loadPillarsExactPlacement() pillarsExactPlacement {
	return pillarsExactPlacement{
		baseDivisor: pillarsExactNumber("pillars-exact-base-divisor"),
		heightLeast: pillarsExactNumber("pillars-exact-height-least"),
		heightRange: pillarsExactNumber("pillars-exact-height-range"),
		seedX:       uint64(pillarsExactCount("pillars-exact-seed-x")),
		seedZ:       uint64(pillarsExactCount("pillars-exact-seed-z")),
		seedHeight:  uint64(pillarsExactCount("pillars-exact-seed-height")),
	}
}

// loadPillarsExactShape reads the shape numbers.
func loadPillarsExactShape() pillarsExactShape {
	return pillarsExactShape{
		wall:          pillarsExactNumber("pillars-exact-wall"),
		wallRag:       pillarsExactNumber("pillars-exact-wall-rag"),
		wallScale:     pillarsExactNumber("pillars-exact-wall-scale"),
		wallEdge:      pillarsExactNumber("pillars-exact-wall-edge"),
		taper:         pillarsExactNumber("pillars-exact-taper"),
		noiseScale:    pillarsExactNumber("pillars-exact-noise-scale"),
		build:         pillarsExactNumber("pillars-exact-build"),
		edge:          pillarsExactNumber("pillars-exact-edge"),
		streakScale:   pillarsExactNumber("pillars-exact-streak-scale"),
		streakStretch: pillarsExactNumber("pillars-exact-streak-stretch"),
		seedNoise:     uint64(pillarsExactCount("pillars-exact-seed-noise")),
		seedStreak:    uint64(pillarsExactCount("pillars-exact-seed-streak")),
		wallOctaves:   pillarsExactCount("pillars-exact-wall-octaves"),
		noiseOctaves:  pillarsExactCount("pillars-exact-noise-octaves"),
		streakOctaves: pillarsExactCount("pillars-exact-streak-octaves"),
	}
}

// longhandColumns is Pillars.columns written out a second way.
func longhandColumns(pillars Pillars) []column {
	if pillars.Count <= nought {
		return nil
	}
	placed := pillarsExactPlaced()
	half := pillars.Width / double
	base := pillars.Width / (float64(pillars.Count) * placed.baseDivisor)
	out := make([]column, nought, pillars.Count)
	for i := range pillars.Count {
		k := int64(i)
		out = append(out, column{
			x: -half + pillars.Width*(float64(i)+halfway)/
				float64(pillars.Count) +
				(noise.Hash(pillars.Seed+placed.seedX, k)-halfway)*base,
			z: (noise.Hash(pillars.Seed+placed.seedZ, k) - halfway) *
				double * base,
			height: pillars.Height *
				(placed.heightLeast + placed.heightRange*
					noise.Hash(pillars.Seed+placed.seedHeight, k)),
			base: base,
		})
	}
	return out
}

// longhandWall is the share of the density the wall along the bottom gives at
// a point, or the fog where it gives less.
func longhandWall(pillars Pillars, at maths.Vec) float64 {
	shape := pillarsExactShaped()
	most := fog
	ragged := shape.wallRag * pillars.Height * noise.Fractal(pillars.Seed,
		at.X/pillars.Height*shape.wallScale, nought,
		at.Z/pillars.Height*shape.wallScale, shape.wallOctaves)
	if top := shape.wall*pillars.Height + ragged; at.Y < top {
		edge := maths.Smooth(top-shape.wallEdge*pillars.Height, top, at.Y)
		most = math.Max(most, unity-edge)
	}
	return most
}

// longhandColumn is the density of the column at an index of the pillars at a
// point, with its filaments. The point is not above the column's tip.
func longhandColumn(pillars Pillars, index int, body column,
	at maths.Vec) float64 {
	shape := pillarsExactShaped()
	radius := body.base * (unity - shape.taper*at.Y/body.height)
	if from := body.height - radius; at.Y > from {
		fraction := (at.Y - from) / radius
		radius *= math.Sqrt(math.Max(nought, unity-fraction*fraction))
	}
	rho := math.Hypot(at.X-body.x, at.Z-body.z)
	scale := shape.noiseScale / body.base
	seed := pillars.Seed + shape.seedNoise + uint64(index)
	x, y, z := at.X*scale, at.Y*scale, at.Z*scale
	grain := noise.Fractal(seed, x, y, z, shape.noiseOctaves)
	rho += (grain - halfway) * shape.build * body.base
	got := unity - maths.Smooth(radius-shape.edge*body.base, radius, rho)
	if pillars.Filaments > nought && got > nought {
		streakScale := shape.streakScale / body.base
		streak := noise.Fractal(seed+shape.seedStreak, at.X*streakScale,
			at.Y*streakScale/shape.streakStretch, at.Z*streakScale,
			shape.streakOctaves)
		got *= unity - pillars.Filaments*math.Min(unity,
			math.Max(nought, halfway-streak))
	}
	return got
}

// longhandDensity is Pillars.Density written out a second way, the columns
// placed afresh by the call.
func longhandDensity(pillars Pillars, at maths.Vec) float64 {
	if pillars.Height <= nought || pillars.Width <= nought ||
		pillars.Count <= nought {
		return nought
	}
	half := pillars.Width / double
	if at.Y < nought || at.Y > pillars.Height || math.Abs(at.X) > half ||
		math.Abs(at.Z) > half {
		return nought
	}
	most := longhandWall(pillars, at)
	for index, body := range longhandColumns(pillars) {
		if at.Y > body.height {
			continue
		}
		most = math.Max(most, longhandColumn(pillars, index, body, at))
	}
	return pillars.Thick * most
}

// pillarsExactSame is whether two numbers are the same bits.
func pillarsExactSame(first, second float64) bool {
	return math.Float64bits(first) == math.Float64bits(second)
}

// pillarsExactCase is the pillars a row of the table of cases gives.
func pillarsExactCase(t *testing.T, row map[string]string) Pillars {
	t.Helper()
	return Pillars{Count: cellCount(t, row, "count"),
		Height:    cell(t, row, "height"),
		Width:     cell(t, row, "width"),
		Thick:     cell(t, row, "thick"),
		Seed:      uint64(cellCount(t, row, "seed")),
		Filaments: cell(t, row, "filaments")}
}

// pillarsExactCases is every pillars of the table of cases, in order.
func pillarsExactCases(t *testing.T) []Pillars {
	t.Helper()
	var cases []Pillars
	for _, row := range readSheet(t, pillarsExactCasesFile) {
		cases = append(cases, pillarsExactCase(t, row))
	}
	return cases
}

// pillarsExactNamed is the pillars of the case of a name in the table.
func pillarsExactNamed(t *testing.T, name string) Pillars {
	t.Helper()
	rows := readSheet(t, pillarsExactCasesFile)
	return pillarsExactCase(t, rows.row(t, pillarsExactCasesFile, name))
}

// pillarsExactRun is how a probe draws its points: how many, how far past the
// box of the pillars they may stand across and up.
type pillarsExactRun struct {
	points         int
	across, upward float64
}

// pillarsExactProbe draws points at random from a seed and stops the test at
// the first where the pillars' density is not the longhand arithmetic's, bit
// for bit. It is run as a goroutine and never stops the test itself.
func pillarsExactProbe(t *testing.T, group *sync.WaitGroup, name string,
	literal, pillars Pillars, run pillarsExactRun, seed int64) {
	defer group.Done()
	random := rand.New(rand.NewSource(seed))
	for range run.points {
		at := maths.V((random.Float64()-halfway)*run.across*literal.Width,
			random.Float64()*run.upward*literal.Height,
			(random.Float64()-halfway)*run.across*literal.Width)
		got, want := pillars.Density(at), longhandDensity(literal, at)
		if !pillarsExactSame(got, want) {
			t.Errorf("%s %+v at %v: %v, want %v", name, literal, at, got, want)
			return
		}
	}
}

// pillarsExactTips stops the test where the tips of the pillars are not the
// tops of the longhand columns, bit for bit.
func pillarsExactTips(t *testing.T, name string, literal, pillars Pillars) {
	t.Helper()
	tips, want := pillars.Tips(), longhandColumns(literal)
	if len(tips) != len(want) {
		t.Fatalf("%s %+v: %d tips, %d columns", name, literal, len(tips),
			len(want))
	}
	for i, body := range want {
		if !pillarsExactSame(tips[i].X, body.x) ||
			!pillarsExactSame(tips[i].Y, body.height) ||
			!pillarsExactSame(tips[i].Z, body.z) {
			t.Errorf("%s %+v tip %d: %v", name, literal, i, tips[i])
		}
	}
}

// the density and the tips are the longhand arithmetic's bit for bit, for a
// literal and for a prepared value, over several pillars including counts of
// zero, one, nine (past the stack buffer) and below, widths and heights of
// either zero and below, and a field that is no number, at seeded random
// points, from many goroutines at once.
func TestPillarsAreTheLonghandArithmeticBitForBit(t *testing.T) {
	sourceSeed := int64(pickedCount(t, "pillars-exact-source-seed"))
	source := rand.New(rand.NewSource(sourceSeed))
	run := pillarsExactRun{points: pickedCount(t, "pillars-exact-points"),
		across: picked(t, "pillars-exact-across"),
		upward: picked(t, "pillars-exact-upward")}
	goroutines := pickedCount(t, "pillars-exact-goroutines")
	var group sync.WaitGroup
	for _, literal := range pillarsExactCases(t) {
		for name, pillars := range map[string]Pillars{"literal": literal,
			"prepared": literal.Prepared()} {
			for range goroutines {
				group.Add(unity)
				go pillarsExactProbe(t, &group, name, literal, pillars, run,
					source.Int63())
			}
			pillarsExactTips(t, name, literal, pillars)
		}
	}
	group.Wait()
}

// a prepared value's columns do not change when Thick or Filaments do, and
// its density is the longhand arithmetic's with the changed values; it does
// not see a later change to Count.
func TestAPreparedPillarsDoesNotDependOnThickOrFilaments(t *testing.T) {
	standing := pillarsExactNamed(t, pillarsExactFilamented)
	prepared := standing.Prepared()
	want := append([]column(nil), prepared.held...)
	at := maths.V(picked(t, "pillars-exact-probe-x"),
		picked(t, "pillars-exact-probe-y"), picked(t, "pillars-exact-probe-z"))
	thicks := []float64{nought, picked(t, "pillars-exact-thick-small"),
		picked(t, "pillars-exact-thick-large")}
	filaments := []float64{nought, picked(t, "pillars-exact-filaments-some"),
		unity}
	for _, thick := range thicks {
		for _, filament := range filaments {
			changed := prepared
			changed.Thick, changed.Filaments = thick, filament
			for i, body := range want {
				if changed.held[i] != body {
					t.Fatalf("column %d moved with Thick %v Filaments %v", i,
						thick, filament)
				}
			}
			literal := standing
			literal.Thick, literal.Filaments = thick, filament
			got := changed.Density(at)
			wantDensity := longhandDensity(literal, at)
			if !pillarsExactSame(got, wantDensity) {
				t.Errorf("Thick %v Filaments %v: %v, want %v", thick,
					filament, got, wantDensity)
			}
		}
	}
	later := prepared
	later.Count = unity
	if got := len(later.Tips()); got != standing.Count {
		t.Errorf("a prepared value saw a later Count: %d tips", got)
	}
}
