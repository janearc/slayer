package planet

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// stillFlow is a flow of no weather: its air came from where it is, with no
// spin and no age.
type stillFlow struct{}

// At is the flow's origin, spin and age at a place: its own latitude, and
// none of either of the others.
func (stillFlow) At(latitude, _ float64) (origin, spin, age float64) {
	return latitude, nothing, nothing
}

// gridStub is a grid flow that says it has across by rows cells and holds
// length numbers in each of its fields.
type gridStub struct{ across, rows, length int }

// At is the flow's origin, spin and age at a place: none.
func (gridStub) At(_, _ float64) (origin, spin, age float64) {
	return nothing, nothing, nothing
}

// Cells is the grid's size, as the stub says it.
func (grid gridStub) Cells() (across, rows int) { return grid.across, grid.rows }

// Wall is the latitude the grid's last rows stand inside of: none.
func (gridStub) Wall() float64 { return nothing }

// Fields are the grid's three numbers a cell, each as long as the stub says.
func (grid gridStub) Fields() (origin, spin, age []float64) {
	return make([]float64, grid.length), make([]float64, grid.length),
		make([]float64, grid.length)
}

// degenerateRing is the ring the degenerate-input tests make.
func degenerateRing(t *testing.T) Ring {
	t.Helper()
	ring, err := NewRing(picked(t, "degenerate-ring-inner"),
		picked(t, "degenerate-ring-outer"), picked(t, "degenerate-ring-depth"),
		picked(t, "degenerate-ring-albedo"), picked(t, "degenerate-ring-lean"))
	if err != nil {
		t.Fatal(err)
	}
	return ring
}

// methaneRowText is a row of the methane table: a wavelength twice, a
// coefficient as written, and the wavelength again for the albedo.
func methaneRowText(vacuum float64, coefficient string) string {
	return fmt.Sprintf("%g %g %s %g\n", vacuum, vacuum, coefficient, vacuum)
}

// absorbingOf is the absorption MethaneAbsorbing makes of a table of the one
// row, or why it refuses it.
func absorbingOf(vacuum float64,
	coefficient string) (func(nm float64) float64, error) {
	row := methaneRowText(vacuum, coefficient)
	return MethaneAbsorbing(strings.NewReader(row))
}

// A methane row whose coefficient is not a number, is endless or is below
// zero is refused, in words that name its wavelength.
func TestAMethaneRowThatMakesNoSenseIsRefusedByName(t *testing.T) {
	vacuum := picked(t, "methane-test-vacuum")
	for _, bad := range []string{"NaN", "+Inf", "-Inf",
		fmt.Sprint(picked(t, "methane-test-negative"))} {
		_, err := absorbingOf(vacuum, bad)
		if err == nil {
			t.Fatalf("a coefficient of %s was taken", bad)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("%g", vacuum)) {
			t.Fatalf("the refusal %q does not name the row's wavelength",
				err)
		}
	}
}

// A methane row whose coefficient is zero or more is taken, and the
// absorption read back is the coefficient over the kilometre-amagat.
func TestAMethaneRowOfSenseIsTaken(t *testing.T) {
	vacuum := picked(t, "methane-test-vacuum")
	coefficient := picked(t, "methane-test-coefficient")
	absorbing, err := absorbingOf(vacuum, fmt.Sprint(coefficient))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := absorbing(vacuum), coefficient/kilometreAmagat; got != want {
		t.Fatalf("absorption %g, want %g", got, want)
	}
	none, err := absorbingOf(vacuum, fmt.Sprint(nothing))
	if err != nil {
		t.Fatal(err)
	}
	if got := none(vacuum); got != nothing {
		t.Fatalf("a row of no coefficient absorbs %g", got)
	}
}

// A gas with no cross-section is refused when it is made, not when light
// first meets it.
func TestAGasWithNoCrossSectionIsRefused(t *testing.T) {
	molar := picked(t, "degenerate-molar")
	if _, err := NewGas("bare", nil, molar); err == nil {
		t.Fatal("a gas with no cross-section was made")
	}
	if _, err := NewGas("flat", flatSection, molar); err != nil {
		t.Fatal(err)
	}
}

// flatSection is a cross-section of no size at every wavelength.
func flatSection(float64) float64 { return nothing }

// Phosphors that together hold more than the whole of a ring's mass are
// refused, and ones that hold exactly the whole give back what their shares
// of their yields sum to.
func TestPhosphorsOfMoreThanTheWholeAreRefused(t *testing.T) {
	ring := degenerateRing(t)
	large := picked(t, "glow-share-large")
	if _, err := ring.Glowing(map[string]float64{"zns-ag": large,
		"csi-tl": large}); err == nil {
		t.Fatal("phosphors by more than the whole of the mass were taken")
	}
	share := picked(t, "glow-share-half")
	glowing, err := ring.Glowing(map[string]float64{"zns-ag": share,
		"csi-tl": share})
	if err != nil {
		t.Fatal(err)
	}
	want := share*phosphors["zns-ag"].yield + share*phosphors["csi-tl"].yield
	if glowing.glow.yield != want {
		t.Fatalf("yield %g, want %g", glowing.glow.yield, want)
	}
}

// A ring is given no more moonlets than it may hold, and no size, slope or
// mass that is endless; the most it may hold, and a few, are taken.
func TestMoonletsBeyondTheMostOrEndlessAreRefused(t *testing.T) {
	ring := degenerateRing(t)
	seed := pickedSeed(t, "degenerate-seed")
	smallest := picked(t, "degenerate-moonlet-smallest")
	largest := picked(t, "degenerate-moonlet-largest")
	slope := picked(t, "degenerate-moonlet-slope")
	opens := picked(t, "degenerate-moonlet-opens")
	mass := picked(t, "degenerate-planet-mass")
	forever := math.Inf(whole)
	if _, err := ring.Cleared(seed, mostMoonlets+adjacent, smallest, largest,
		slope, opens, mass); err == nil {
		t.Fatal("more moonlets than a ring may hold were taken")
	}
	if _, err := ring.Cleared(seed, pickedInt(t, "degenerate-moonlet-count"),
		smallest, forever, slope, opens, mass); err == nil {
		t.Fatal("moonlets of an endless size were taken")
	}
	if _, err := ring.Cleared(seed, pickedInt(t, "degenerate-moonlet-count"),
		smallest, largest, forever, opens, mass); err == nil {
		t.Fatal("moonlets of an endless slope were taken")
	}
	if _, err := ring.Cleared(seed, pickedInt(t, "degenerate-moonlet-count"),
		smallest, largest, slope, opens, forever); err == nil {
		t.Fatal("moonlets about an endless mass were taken")
	}
	for _, count := range []int{mostMoonlets,
		pickedInt(t, "degenerate-moonlet-count")} {
		made, err := ring.Cleared(seed, count, smallest, largest, slope,
			opens, mass)
		if err != nil {
			t.Fatalf("%d moonlets: %v", count, err)
		}
		if len(made.moonlets) != count {
			t.Fatalf("%d moonlets made %d", count, len(made.moonlets))
		}
	}
}

// Grains of an endless size, slope or shear are refused; grains of a fair
// size are taken.
func TestGrainsOfEndlessSizeAreRefused(t *testing.T) {
	ring := degenerateRing(t)
	seed := pickedSeed(t, "degenerate-seed")
	smallest := picked(t, "degenerate-grain-smallest")
	largest := picked(t, "degenerate-grain-largest")
	slope := picked(t, "degenerate-grain-slope")
	shear := picked(t, "degenerate-grain-shear")
	clumping := picked(t, "degenerate-grain-clumping")
	forever := math.Inf(whole)
	if _, err := ring.Grained("smooth", seed, smallest, forever, slope, shear,
		clumping); err == nil {
		t.Fatal("grains of an endless size were taken")
	}
	if _, err := ring.Grained("smooth", seed, smallest, largest, forever,
		shear, clumping); err == nil {
		t.Fatal("grains of an endless slope were taken")
	}
	if _, err := ring.Grained("smooth", seed, smallest, largest, slope,
		forever, clumping); err == nil {
		t.Fatal("grains of an endless shear were taken")
	}
	if _, err := ring.Grained("smooth", seed, smallest, largest, slope, shear,
		clumping); err != nil {
		t.Fatal(err)
	}
}

// Air with an age that is not a number, or is below zero, is fresh; air of
// an endless age is cooked to the last rung; air between is between rungs.
func TestAirOfNoAgeIsFreshAndOfEndlessAgeIsFull(t *testing.T) {
	face := &Face{cooking: picked(t, "degenerate-cooking")}
	for _, age := range []float64{math.NaN(), -picked(t, "degenerate-age"),
		nothing} {
		if rung, toward := face.cookedBy(age); rung != nothing ||
			toward != nothing {
			t.Fatalf("air of age %g is at rung %d, %g toward the next", age,
				rung, toward)
		}
	}
	rung, toward := face.cookedBy(math.Inf(whole))
	if rung != rungs-double || toward != whole {
		t.Fatalf("endless air is at rung %d, %g toward the next", rung,
			toward)
	}
	rung, toward = face.cookedBy(picked(t, "degenerate-age"))
	if rung < nothing || rung > rungs-double || !(toward >= nothing &&
		toward < whole) {
		t.Fatalf("air cooked part way is at rung %d, %g toward the next",
			rung, toward)
	}
}

// A chart is not made of a cell that is not a number, is endless or is finer
// than its table holds; it is made of the finest the table holds and of a
// fair cell.
func TestAChartOfNoGridIsNotMade(t *testing.T) {
	face := &Face{flow: stillFlow{}}
	radius := picked(t, "degenerate-chart-radius")
	for name, cell := range map[string]float64{
		"not a number": math.NaN(), "endless": math.Inf(whole),
		"too fine": circle / float64(mostChartAcross+double*adjacent)} {
		if chart, ok := NewChart(face, radius, cell); ok || chart != nil {
			t.Fatalf("a chart of a cell that is %s was made", name)
		}
	}
	fine := circle / float64(mostChartAcross/double)
	for _, cell := range []float64{fine, picked(t, "degenerate-chart-cell")} {
		if chart, ok := NewChart(face, radius, cell); !ok || chart == nil {
			t.Fatalf("no chart of a cell of %g", cell)
		}
	}
}

// A chart asked for the light at the middle of the body asks the face, which
// has none there, and paints no tile; asked at a point of the surface it
// paints one.
func TestAChartAskedAtTheMiddleAsksTheFace(t *testing.T) {
	face, radius := chartFace(t)
	chart, ok := NewChart(face, radius, picked(t, "degenerate-chart-cell"))
	if !ok {
		t.Fatal("no chart")
	}
	middle := maths.V(nothing, nothing, nothing)
	if got := chart.Within(middle, nothing); got != noLight {
		t.Fatalf("light %v at the middle of the body", got)
	}
	if painted, _ := chart.tilesPainted(); painted != nothing {
		t.Fatalf("the middle of the body painted %d tiles", painted)
	}
	point := face.ways.toward(picked(t, "chart-paint-down"),
		picked(t, "chart-paint-along")).Scale(radius)
	chart.Within(point, nothing)
	if painted, _ := chart.tilesPainted(); painted != single {
		t.Fatalf("a point of the surface painted %d tiles", painted)
	}
}

// A grid whose fields are not the size its cells say, or which has no cells,
// cannot be laid out; one that holds what it says, even of one row, can.
func TestAGridThatHoldsLessThanItSaysIsNotLaid(t *testing.T) {
	cells := pickedInt(t, "degenerate-grid-cells")
	fair := gridStub{across: cells, rows: cells, length: cells * cells}
	if !gridHolds(fair) {
		t.Fatal("a grid of the size it says was refused")
	}
	short := fair
	short.length--
	if gridHolds(short) {
		t.Fatal("a grid with a field short of a number was laid")
	}
	if gridHolds(gridStub{across: cells, rows: nothing, length: nothing}) {
		t.Fatal("a grid of no rows was laid")
	}
	if gridHolds(gridStub{across: nothing, rows: cells, length: nothing}) {
		t.Fatal("a grid of no cells round was laid")
	}
	oneRow := gridStub{across: cells, rows: adjacent, length: cells}
	if !gridHolds(oneRow) {
		t.Fatal("a grid of one row was refused")
	}
}

// stormsOf is the storms the degenerate-input tests make, with so many
// eddies.
func stormsOf(t *testing.T, eddies float64) (Storms, error) {
	t.Helper()
	return NewStorms(pickedSeed(t, "degenerate-seed"),
		picked(t, "degenerate-storm-turbulence"), eddies,
		picked(t, "degenerate-storm-stretch"),
		picked(t, "degenerate-storm-edge"),
		pickedInt(t, "degenerate-storm-folds"), nil)
}

// Storms of endless eddies, stretch or edge are refused; storms of a fair
// number are made.
func TestStormsOfEndlessSizeAreRefused(t *testing.T) {
	seed := pickedSeed(t, "degenerate-seed")
	turbulence := picked(t, "degenerate-storm-turbulence")
	eddies := picked(t, "degenerate-storm-eddies")
	stretch := picked(t, "degenerate-storm-stretch")
	edge := picked(t, "degenerate-storm-edge")
	folds := pickedInt(t, "degenerate-storm-folds")
	forever := math.Inf(whole)
	if _, err := NewStorms(seed, turbulence, forever, stretch, edge, folds,
		nil); err == nil {
		t.Fatal("storms of endless eddies were made")
	}
	if _, err := NewStorms(seed, turbulence, eddies, forever, edge, folds,
		nil); err == nil {
		t.Fatal("storms of an endless stretch were made")
	}
	if _, err := NewStorms(seed, turbulence, eddies, stretch, forever, folds,
		nil); err == nil {
		t.Fatal("storms of an endless edge were made")
	}
	if _, err := stormsOf(t, eddies); err != nil {
		t.Fatal(err)
	}
}

// A spot at an endless longitude, or turning an endless way, is refused; a
// spot of fair numbers is made.
func TestASpotAtAnEndlessPlaceOrTurnIsRefused(t *testing.T) {
	latitude := picked(t, "degenerate-spot-latitude")
	longitude := picked(t, "degenerate-spot-longitude")
	radius := picked(t, "degenerate-spot-radius")
	turn := picked(t, "degenerate-spot-turn")
	forever := math.Inf(whole)
	if _, err := NewSpot(latitude, forever, radius, turn); err == nil {
		t.Fatal("a spot at an endless longitude was made")
	}
	if _, err := NewSpot(latitude, longitude, radius, -forever); err == nil {
		t.Fatal("a spot turning an endless way was made")
	}
	if _, err := NewSpot(latitude, longitude, radius, turn); err != nil {
		t.Fatal(err)
	}
}

// A swarm whose radii go by an endless slope is refused; a swarm of a fair
// slope holds the spots it was asked for.
func TestASwarmOfEndlessSlopeIsRefused(t *testing.T) {
	storms, err := stormsOf(t, picked(t, "degenerate-storm-eddies"))
	if err != nil {
		t.Fatal(err)
	}
	seed := pickedSeed(t, "degenerate-seed")
	count := pickedInt(t, "degenerate-swarm-count")
	smallest := picked(t, "degenerate-swarm-smallest")
	largest := picked(t, "degenerate-swarm-largest")
	forever := math.Inf(whole)
	if _, err := storms.Swarmed(seed, count, smallest, largest,
		forever); err == nil {
		t.Fatal("a swarm of an endless slope was made")
	}
	slope := picked(t, "degenerate-swarm-slope")
	made, err := storms.Swarmed(seed, count, smallest, largest, slope)
	if err != nil {
		t.Fatal(err)
	}
	if len(made.spots) != count {
		t.Fatalf("a swarm of %d spots holds %d", count, len(made.spots))
	}
}
