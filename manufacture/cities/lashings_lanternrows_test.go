package cities

import (
	"math"
	"slices"
	"testing"
)

// The lantern rows of the culture the tests check against: 8 m between
// rows, all at 9.2 m, lanterns 1.5 m apart and one bulb between each pair.
const (
	rowPitch   = 8.0
	rowHeight  = 9.2
	lanternGap = 1.5
)

// rowsHung is the lantern rows over the test street, or the test's end.
func rowsHung(t *testing.T, density float64) *Lashings {
	t.Helper()
	lashings, err := NewLashings(LashingsAsk{Density: density,
		Style: styleLanternRows}, facingBlocks(streetGap), loneSeed)
	if err != nil {
		t.Fatal(err)
	}
	return lashings
}

// alongOf are the distances along the street of the cables of a hanging,
// in order, for the test street, which runs along z.
func alongOf(lashings *Lashings) []float64 {
	var found []float64
	for _, cable := range solidsNamed(lashings.Design(), "lashing cable") {
		found = append(found, cable.Form.(lashingCable).Points[0].Z)
	}
	slices.Sort(found)
	return found
}

func TestRowsAreOneEveryPitchAndAllAtOneHeight(t *testing.T) {
	lashings := rowsHung(t, 1)
	along := alongOf(lashings)
	if len(along) < 2 {
		t.Fatalf("%d rows to look at", len(along))
	}
	for index := 1; index < len(along); index++ {
		if gap := along[index] - along[index-1]; math.Abs(gap-rowPitch) >
			closeEnough {
			t.Errorf("rows %d and %d are %v m apart, not %v", index-1, index,
				gap, rowPitch)
		}
	}
	for _, cable := range solidsNamed(lashings.Design(), "lashing cable") {
		points := cable.Form.(lashingCable).Points
		// a height is the roof's less a margin and each is rounded, so
		// the ends are the height to within a few units of the last place
		ends := []float64{points[0].Y, points[len(points)-1].Y}
		if math.Abs(ends[0]-rowHeight) > closeEnough ||
			math.Abs(ends[1]-rowHeight) > closeEnough {
			t.Errorf("a row is anchored at %v, not %v", ends, rowHeight)
		}
	}
}

func TestEachRowSagsSlightlyAndNoMoreThanItsSag(t *testing.T) {
	for _, cable := range solidsNamed(rowsHung(t, 1).Design(),
		"lashing cable") {
		lowest := math.Inf(1)
		for _, point := range cable.Form.(lashingCable).Points {
			lowest = math.Min(lowest, point.Y)
		}
		if drop := rowHeight - lowest; !(drop > 0 && drop < 0.5) {
			t.Errorf("a row sags %v m", drop)
		}
	}
}

func TestEveryLanternOfRowsIsRedAndOneSize(t *testing.T) {
	lashings := rowsHung(t, 1)
	lanterns := solidsNamed(lashings.Design(), "lantern")
	if len(lanterns) == 0 {
		t.Fatal("no lanterns")
	}
	for _, lantern := range lanterns {
		if lantern.Material.Name != lanterns[0].Material.Name ||
			lantern.Form.(ribbedLantern) != lanterns[0].Form.(ribbedLantern) {
			t.Errorf("a lantern is %v in %s, not %v in %s", lantern.Form,
				lantern.Material.Name, lanterns[0].Form,
				lanterns[0].Material.Name)
		}
	}
}

func TestOneBulbHangsBetweenEachPairOfLanternsOfARow(t *testing.T) {
	lashings := rowsHung(t, 1)
	design := lashings.Design()
	bulbs := len(solidsNamed(design, "bulb"))
	rows := len(alongOf(lashings))
	if want := lashings.Lanterns - rows; bulbs != want {
		t.Errorf("%d bulbs for %d lanterns in %d rows, not %d", bulbs,
			lashings.Lanterns, rows, want)
	}
}

func TestLanternsOfARowAreEvenlySpacedAlongIt(t *testing.T) {
	var across []float64
	row := math.NaN()
	for _, placed := range rowsHung(t, 1).Design().Elements {
		solid, is := placed.Element.(solid)
		if !is || solid.Name != "lantern" {
			continue
		}
		if math.IsNaN(row) {
			row = placed.Place.Origin.Z
		}
		if math.Abs(placed.Place.Origin.Z-row) < closeEnough {
			across = append(across, placed.Place.Origin.X)
		}
	}
	slices.Sort(across)
	for index := 2; index < len(across); index++ {
		if first, next := across[index-1]-across[index-2],
			across[index]-across[index-1]; math.Abs(first-next) > 0.05 {
			t.Errorf("lanterns %d and %d are %v and %v m apart", index-1,
				index, first, next)
		}
	}
}

func TestRowsDoNotReadDensityAndAFestivalIsTheDefault(t *testing.T) {
	if few, many := rowsHung(t, 0), rowsHung(t, 1); few.Cables != many.Cables ||
		few.Lanterns != many.Lanterns {
		t.Errorf("density changed rows: %d cables and %d lanterns, then %d "+
			"and %d", few.Cables, few.Lanterns, many.Cables, many.Lanterns)
	}
	plain := hungOver(t, facingBlocks(streetGap), 1, loneSeed)
	named, err := NewLashings(LashingsAsk{Density: 1, Style: styleFestival},
		facingBlocks(streetGap), loneSeed)
	if err != nil {
		t.Fatal(err)
	}
	if plain.Cables != named.Cables || plain.Lanterns != named.Lanterns {
		t.Errorf("a festival named is not the default: %d and %d, then %d "+
			"and %d", plain.Cables, plain.Lanterns, named.Cables,
			named.Lanterns)
	}
}

// Anti: a style that is none of the styles is refused.
func TestAStyleThatIsNoStyleIsRefused(t *testing.T) {
	_, err := NewLashings(LashingsAsk{Density: 1, Style: "bunting"},
		facingBlocks(streetGap), loneSeed)
	if err == nil {
		t.Error("a style of bunting was taken")
	}
}

func TestARibbedLanternIsSquatRibbedAndClosed(t *testing.T) {
	const ribs = 5
	mesh, err := ribbedLantern{Radius: 0.38, Stretch: 0.8, Ribs: ribs,
		Depth: 0.05}.Mesh(Detail("quick"))
	if err != nil {
		t.Fatal(err)
	}
	if !closedMesh(mesh) {
		t.Error("the lantern has a hole in it, or a face turned in")
	}
	wide, high := 0.0, 0.0
	for _, face := range mesh.Triangles {
		for _, corner := range face {
			wide = math.Max(wide, math.Hypot(corner.X, corner.Z))
			high = math.Max(high, math.Abs(corner.Y))
		}
	}
	// The widest ring is the ridge nearest the equator: the equator when
	// it is a ridge, and the ring before it when it is a groove, as it is
	// with an odd number of ribs, whose ridge is at a latitude short of it.
	rings := ribs * ringsPerRib
	widest := rings / 2
	if widest%ringsPerRib != 0 {
		widest--
	}
	reach := 0.38 * math.Sin(math.Pi*float64(widest)/float64(rings))
	if !(wide > high) || math.Abs(wide-reach) > closeEnough ||
		math.Abs(high-0.38*0.8) > closeEnough {
		t.Errorf("a lantern %v wide and %v high is not squat", wide, high)
	}
}

// Anti: a ribbed lantern with no rib, no depth in range, or no size is
// refused.
func TestARibbedLanternThatCannotBeMadeIsRefused(t *testing.T) {
	for _, lantern := range []ribbedLantern{
		{Radius: 0.3, Stretch: 0.8, Ribs: 0, Depth: 0.05},
		{Radius: 0.3, Stretch: 0.8, Ribs: 5, Depth: 1},
		{Radius: 0.3, Stretch: 0.8, Ribs: 5, Depth: -0.1},
		{Radius: 0, Stretch: 0.8, Ribs: 5, Depth: 0.05},
		{Radius: 0.3, Stretch: 0, Ribs: 5, Depth: 0.05}} {
		if _, err := lantern.Mesh(Detail("quick")); err == nil {
			t.Errorf("the lantern %v was made", lantern)
		}
	}
}

func TestEachLanternOfRowsHoldsALampThatStartsPastItsPaper(t *testing.T) {
	lashings := rowsHung(t, 1)
	lanterns := solidsNamed(lashings.Design(), "lantern")
	for _, lantern := range lanterns {
		if lantern.Lamp == nil || !(lantern.Lamp.Beyond >
			lantern.Form.(ribbedLantern).Radius) {
			t.Errorf("a lantern holds the lamp %v, which starts inside its "+
				"paper", lantern.Lamp)
		}
	}
}
