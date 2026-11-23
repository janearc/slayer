package arid

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// grid is a ground of heights held in rows, for the tests.
type grid struct {
	rows [][]float64
	step float64
}

// At is the height at a column and a row.
func (ground grid) At(across, down int) float64 { return ground.rows[down][across] }

// Size is the grid's columns, rows and step.
func (ground grid) Size() (int, int, float64) {
	return len(ground.rows[0]), len(ground.rows), ground.step
}

// shaped is a grid of columns and rows a step apart, each height from a
// column and a row.
func shaped(columns, rows int, step float64,
	height func(column, row int) float64) grid {
	made := grid{step: step}
	for row := range rows {
		line := make([]float64, columns)
		for column := range line {
			line[column] = height(column, row)
		}
		made.rows = append(made.rows, line)
	}
	return made
}

// basinHeight is a 7 by 7 ground: 5 m deep sea at the edge, a ring of land
// 2 m up inside it, and a hollow 1 m below the sea's level inside that.
func basinHeight(column, row int) float64 {
	switch max(abs(column-3), abs(row-3)) {
	case 3:
		return -5
	case 2:
		return 2
	}
	return -1
}

// abs is an integer's size.
func abs(value int) int { return max(value, -value) }

// rampHeight is a beach rising half a metre a sample eastward, from 10 m
// deep at the west edge to 10 m up at the east.
func rampHeight(column, _ int) float64 { return -10 + 0.5*float64(column) }

// diagonalHeight is a 5 by 4 ground 1 m up but for two pairs of samples
// 1 m down, each pair diagonal: one pair along the diagonal the ground's
// triangles draw, from the edge at 0, 0 to 1, 1, and one along the other,
// from the edge at 4, 0 to 3, 1.
func diagonalHeight(column, row int) float64 {
	switch [2]int{column, row} {
	case [2]int{0, 0}, [2]int{1, 1}, [2]int{4, 0}, [2]int{3, 1}:
		return -1
	}
	return 1
}

// planeHeight is ground rising 1.5 m a sample east and falling 0.4 m a
// sample south, from 6 m deep at the first sample.
func planeHeight(column, row int) float64 {
	return -6 + 1.5*float64(column) - 0.4*float64(row)
}

// lowLandHeight is land a centimetre over the sea's level, but for a
// first column 20 m deep.
func lowLandHeight(column, _ int) float64 {
	if column == 0 {
		return -20
	}
	return 0.01
}

// deepHeight is ground a metre under the sea's level.
func deepHeight(int, int) float64 { return -1 }

// missingHeight is ground that is not a number.
func missingHeight(int, int) float64 { return math.NaN() }

// straightDown is a ray falling from 10 m over a place.
func straightDown(east, south float64) maths.Ray {
	return maths.Ray{At: maths.V(east, 10, south), Dir: maths.V(0, -1, 0)}
}

// raisedSea is a sea whose surface stands 1 m over its mean everywhere,
// and reaches 1 m: its share over a coast is its height.
func raisedSea(t *testing.T) *Sea {
	t.Helper()
	field := &oceantex.Field{N: 2, Extent: 8, Height: []float64{1, 1, 1, 1},
		SlopeX: make([]float64, 4), SlopeY: make([]float64, 4)}
	sea, err := seaOf([]oceantex.WaveCascade{{Field: field, Reach: 1}})
	if err != nil {
		t.Fatal(err)
	}
	return sea
}

// ashore is a sea over a ground whose first sample is at the sea's middle
// and whose heights are counted from its mean.
func ashore(t *testing.T, sea *Sea, ground Ground) (*Sea, Shoreline) {
	t.Helper()
	shored, counted, err := sea.Ashore(ground, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return shored, counted
}

func TestCoastCutsOffAHollowAndKeepsTheEdgeJoined(t *testing.T) {
	shored, _ := ashore(t, raisedSea(t), shaped(7, 7, 10, basinHeight))
	shore := shored.coast
	for at, height := range shore.height {
		column, row := at%7, at/7
		hollow := max(abs(column-3), abs(row-3)) <= 1
		if shore.cut[at] != hollow {
			t.Errorf("sample %d, %d at %g m: cut %t, want %t", column, row,
				height, shore.cut[at], hollow)
		}
	}
}

func TestCoastJoinsOnlyAlongTheDrawnDiagonal(t *testing.T) {
	shored, _ := ashore(t, raisedSea(t), shaped(5, 4, 10, diagonalHeight))
	if shored.coast.cut[1*5+1] {
		t.Error("a sample joined by the drawn diagonal is cut off")
	}
	if !shored.coast.cut[1*5+3] {
		t.Error("a sample joined only by the undrawn diagonal is reached")
	}
}

func TestShoredSeaIsDryOverAHollow(t *testing.T) {
	open := fixtureSea(t)
	shored, _ := ashore(t, open, shaped(7, 7, 10, basinHeight))
	if _, hit := open.Hit(straightDown(30, 30), 0, 100); !hit {
		t.Fatal("the open sea does not stand over the hollow")
	}
	if along, hit := shored.Hit(straightDown(30, 30), 0, 100); hit {
		t.Fatalf("water met %g m down over the hollow", along)
	}
	if _, hit := shored.Hit(straightDown(0, 0), 0, 100); !hit {
		t.Fatal("no water over the sea at the edge")
	}
}

func TestShoredSeaStandsNoTallerThanTheDepthCarries(t *testing.T) {
	open := fixtureSea(t)
	ramp := shaped(41, 3, 5, rampHeight)
	shored, _ := ashore(t, open, ramp)
	var broke bool
	for index := range 400 {
		east, south := float64(index)*0.5+0.13, 3.7
		depth := -shored.coast.ground(shored.coast.locate(east, south))
		most := oceantex.BreakingRatio / sides * math.Max(depth, 0)
		if height := shored.height(east, south); math.Abs(height) > most+1e-12 {
			t.Fatalf("at %g m east, %g m of water holds a surface %g m "+
				"from its mean, over %g", east, depth, height, most)
		}
		broke = broke || math.Abs(open.height(east, south)) > most
	}
	if !broke {
		t.Fatal("the open sea stood within the bound too; the test shows nothing")
	}
}

func TestShoredSeaRisesNoFasterThanItsBound(t *testing.T) {
	shored, _ := ashore(t, fixtureSea(t), shaped(30, 30, 4, planeHeight))
	const apart = 0.3
	for index := range 4000 {
		turn := float64(index) * 2.399
		east := -20 + math.Mod(float64(index)*7.31, 160)
		south := -20 + math.Mod(float64(index)*3.17, 160)
		there := shored.height(east+apart*math.Cos(turn),
			south+apart*math.Sin(turn))
		change := math.Abs(there - shored.height(east, south))
		if change > shored.steepest*apart*(1+1e-9) {
			t.Fatalf("at %g, %g the surface rose %g in %g m, over the "+
				"bound %g a metre", east, south, change, apart, shored.steepest)
		}
	}
}

// slopedRise is how fast the sloped sea's surface rises, in metres a metre:
// a metre of height over the four metres between its samples.
const slopedRise = 0.25

// slopedSea is a sea of one band that rises slopedRise a metre in one
// direction and nowhere faster, with that as its steepest rise exactly. Its
// bound is the open sea's own and not the loose sum the bands of a wind sea
// give, so the coast's term is most of what a bound over it holds.
func slopedSea(t *testing.T) *Sea {
	t.Helper()
	field := &oceantex.Field{N: 2, Extent: 8, Height: []float64{0, 1, 0, 1},
		SlopeX: make([]float64, 4), SlopeY: make([]float64, 4)}
	sea, err := seaOf([]oceantex.WaveCascade{{Field: field, Reach: 1}})
	if err != nil {
		t.Fatal(err)
	}
	sea.steepest = slopedRise
	return sea
}

// southRampHeight is a beach falling half a metre a sample southward, from
// 10 m up at the first row.
func southRampHeight(_, row int) float64 { return 10 - 0.5*float64(row) }

// The bound over a sea whose own is exact holds, and holds only with the
// ground's term: a bound without it is crossed.
func TestShoredSeaRisesNoFasterThanItsBoundOverAnExactSea(t *testing.T) {
	open := slopedSea(t)
	shored, _ := ashore(t, open, shaped(30, 30, 4, planeHeight))
	if shored.steepest <= open.steepest {
		t.Fatalf("the bound %g is no wider than the open sea's %g",
			shored.steepest, open.steepest)
	}
	const apart = 0.3
	var crossed bool
	for index := range 4000 {
		turn := float64(index) * 2.399
		east := -20 + math.Mod(float64(index)*7.31, 160)
		south := -20 + math.Mod(float64(index)*3.17, 160)
		there := shored.height(east+apart*math.Cos(turn),
			south+apart*math.Sin(turn))
		change := math.Abs(there - shored.height(east, south))
		if change > shored.steepest*apart*(1+1e-9) {
			t.Fatalf("at %g, %g the surface rose %g in %g m, over the "+
				"bound %g a metre", east, south, change, apart, shored.steepest)
		}
		crossed = crossed || change > open.steepest*apart
	}
	if !crossed {
		t.Fatal("no pair rose past the open sea's bound; the test shows " +
			"nothing of the ground's term")
	}
}

func TestShoredSeaHitFindsWhatSteppingFinds(t *testing.T) {
	shored, _ := ashore(t, fixtureSea(t), shaped(41, 41, 5, rampHeight))
	for index := range 12 {
		angle := float64(index)
		ray := maths.Ray{At: maths.V(10, 1.5, 100),
			Dir: maths.V(math.Cos(angle), -0.02, math.Sin(angle))}
		wanted, found := brute(shored, ray, 300, 0.01)
		along, hit := shored.Hit(ray, 0, 300)
		if hit != found {
			t.Fatalf("ray %d: hit %t, stepping found %t", index, hit, found)
		}
		if hit && along > wanted+0.01 {
			t.Fatalf("ray %d: met at %g, after stepping did at %g", index,
				along, wanted)
		}
	}
}

func TestShoredNormalLeansWithTheShare(t *testing.T) {
	shored, _ := ashore(t, raisedSea(t), shaped(41, 3, 5, rampHeight))
	const reach = 1e-4
	for index := range 20 {
		east, south := 92.3+float64(index)*0.37, 2.1
		normal := shored.Normal(maths.V(east, 0, south))
		rise := (shored.height(east+reach, south) -
			shored.height(east-reach, south)) / (2 * reach)
		if math.Abs(normal.X/normal.Y+rise) > 1e-6 {
			t.Fatalf("at %g m east the normal %v does not lean with the "+
				"surface's rise %g", east, normal, rise)
		}
	}
}

// Over a beach that falls toward the south the surface's lean north and
// south follows the share's, the term of Normal that a ramp by column
// leaves at nought.
func TestShoredNormalLeansSouthWithTheShare(t *testing.T) {
	shored, _ := ashore(t, raisedSea(t), shaped(3, 41, 5, southRampHeight))
	const reach = 1e-4
	var leaned bool
	for index := range 40 {
		east, south := 2.3+float64(index%7), 102.1+float64(index)*0.37
		normal := shored.Normal(maths.V(east, 0, south))
		fall := (shored.height(east, south+reach) -
			shored.height(east, south-reach)) / (2 * reach)
		if math.Abs(normal.Z/normal.Y+fall) > 1e-6 {
			t.Fatalf("at %g m south the normal %v does not lean with the "+
				"surface's fall %g", south, normal, fall)
		}
		leaned = leaned || math.Abs(fall) > 1e-3
	}
	if !leaned {
		t.Fatal("the surface did not lean south anywhere; the test shows " +
			"nothing")
	}
}

func TestShorelineCountsWhatTheOpenSeaCovered(t *testing.T) {
	_, counted := ashore(t, raisedSea(t), shaped(7, 7, 10, basinHeight))
	want := Shoreline{Samples: 49, Land: 16, Lapped: 0, Hollow: 9,
		Filled: 9, Wet: 0}
	counted.Whole, counted.Open, counted.Shored = 0, 0, 0
	if counted != want {
		t.Fatalf("counted %+v, want %+v", counted, want)
	}
}

func TestShorelineCountsLowLandTheWavesLapped(t *testing.T) {
	_, counted := ashore(t, fixtureSea(t), shaped(30, 30, 2, lowLandHeight))
	if counted.Land != 29*30 || counted.Lapped == 0 || counted.Wet != 0 {
		t.Fatalf("land %d, lapped %d, wet %d: want %d, some, none",
			counted.Land, counted.Lapped, counted.Wet, 29*30)
	}
}

func TestShoredSeaOffTheGridTakesTheEdge(t *testing.T) {
	shored, _ := ashore(t, raisedSea(t), shaped(41, 3, 5, rampHeight))
	if height := shored.height(-500, 1); height != 1 {
		t.Errorf("beyond the deep edge the surface is %g, want the whole 1",
			height)
	}
	if height := shored.height(900, 1); height != 0 {
		t.Errorf("beyond the land edge the surface is %g, want 0", height)
	}
	if !shored.coast.wet(900, 1) {
		t.Error("beyond the grid is dry")
	}
}

func TestOpenSeaIsItsWaves(t *testing.T) {
	sea := fixtureSea(t)
	for index := range 50 {
		east, south := float64(index)*13.1, float64(index)*-7.7
		if sea.height(east, south) != sea.waves(east, south) {
			t.Fatalf("at %g, %g a sea with no coast is not its waves", east,
				south)
		}
	}
}

func TestAshoreRefusesWhatItCannotRead(t *testing.T) {
	sea := raisedSea(t)
	good := shaped(3, 3, 1, deepHeight)
	for name, row := range map[string]struct {
		ground Ground
		level  float64
	}{
		"no ground":     {nil, 0},
		"one column":    {shaped(1, 3, 1, rampHeight), 0},
		"no step":       {shaped(3, 3, 0, rampHeight), 0},
		"not a number":  {shaped(3, 3, 1, missingHeight), 0},
		"endless level": {good, math.Inf(1)},
	} {
		if _, _, err := sea.Ashore(row.ground, 0, 0, row.level); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
	var none *Sea
	if _, _, err := none.Ashore(good, 0, 0, 0); err == nil {
		t.Error("no sea met the ground")
	}
}

func TestShorelineSaysItsCounts(t *testing.T) {
	said := Shoreline{Samples: 49, Land: 16, Lapped: 3, Hollow: 9,
		Filled: 9, Whole: 2.56, Open: 1.5, Shored: 2.1}.String()
	for _, part := range []string{"49 heights", "16 are above", "over 3",
		"filled 9"} {
		if !strings.Contains(said, part) {
			t.Errorf("%q does not say %q", said, part)
		}
	}
	if strings.Contains(said, "either now") {
		t.Errorf("%q says a count the code fixes in advance", said)
	}
}
