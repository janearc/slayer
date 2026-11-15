//go:build draw

package planet

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// aGrid is a flow on a grid, read between cells as the weather reads its
// own: a stand-in with fields from a hash and a slow wave, so every term of
// a face has something to draw.
type aGrid struct {
	across, rows      int
	limit             float64
	origin, spin, age []float64
}

// newGrid is a grid of weather with fields from a hash and a slow wave.
func newGrid(t *testing.T) *aGrid {
	t.Helper()
	grid := &aGrid{across: pickedInt(t, "grid-across"),
		rows:  pickedInt(t, "grid-rows"),
		limit: picked(t, "grid-limit") * maths.Degree}
	along, down := picked(t, "grid-wave-along"), picked(t, "grid-wave-down")
	swing, spinSwing := picked(t, "grid-origin-swing"), picked(t, "grid-spin-swing")
	noisy, ageSpan := picked(t, "grid-spin-noise"), picked(t, "grid-age-span")
	spinSeed, ageSeed := pickedSeed(t, "grid-spin-seed"), pickedSeed(t, "grid-age-seed")
	floor := picked(t, "grid-age-floor")
	for row := range grid.rows {
		latitude := -grid.limit + (float64(row)+half)*grid.rise()
		for i := range grid.across {
			at := int64(row*grid.across + i)
			wave := math.Sin(float64(i)/along) * math.Cos(float64(row)/down)
			grid.origin = append(grid.origin, latitude+swing*wave)
			grid.spin = append(grid.spin,
				spinSwing*wave+noisy*(noise.Hash(spinSeed, at)-half))
			grid.age = append(grid.age, ageSpan*noise.Hash(ageSeed, at)-floor)
		}
	}
	return grid
}

// laidSurvey is how many points of a laid face a test counts lit, in the
// airglow's band, cleared, and by the poles.
type laidSurvey struct{ lit, glowing, cleared, polar int }

// a face with a grid's weather lays out whole: painted from its numbers
// and lists alone it is Within, at points all over it, wide pixels and
// fine, every term of the giant drawn. the stand-in grid has strong spin
// everywhere, for the caps and clearings; the real weather is read by its
// own At, which the painter's reader has to match to the poles.
func TestALaidFaceIsTheFace(t *testing.T) {
	for name, flow := range map[string]GridFlow{
		"a stand-in grid": newGrid(t), "the weather": realWeather(t)} {
		body := stormyBody(t, false).Flowing(flow)
		pole, star, eye := look(t, "laid pole"), look(t, "laid star"),
			look(t, "laid eye")
		face := NewFace(body, pole, star, eye, wide(), picked(t, "laid-exposure"))
		laid, ok := face.Laid()
		if !ok {
			t.Fatalf("%s: the face was not laid", name)
		}
		got := laidAgrees(t, name, face,
			&painter{laid: laid, floor: picked(t, "painter-floor")}, body.Radius())
		if got.lit < pickedInt(t, "laid-least-lit") ||
			got.glowing < pickedInt(t, "laid-least-glowing") ||
			got.polar < pickedInt(t, "laid-least-polar") {
			t.Fatalf("%s: %d points lit, %d in the airglow's band, %d by "+
				"the poles: too few to say the layout is whole", name, got.lit,
				got.glowing, got.polar)
		}
		_, standing := flow.(*aGrid)
		if standing && got.cleared < pickedInt(t, "laid-least-cleared") {
			t.Fatalf("%s: %d points cleared", name, got.cleared)
		}
	}
}

// laidAgrees fails unless the painter makes Within at the sampled points
// of a face, and counts those lit, in the airglow's band, cleared, and
// past the latitude where a grid fades to its edge.
func laidAgrees(t *testing.T, name string, face *Face, paint *painter,
	radius float64) laidSurvey {
	t.Helper()
	var survey laidSurvey
	tolerance, brightFloor := picked(t, "laid-close"), picked(t, "laid-bright-floor")
	span, polarFrom := picked(t, "laid-footprint"), picked(t, "laid-polar-from")
	latSeed, lonSeed := pickedSeed(t, "laid-lat-seed"), pickedSeed(t, "laid-lon-seed")
	footSeed, every := pickedSeed(t, "laid-foot-seed"), pickedCount(t, "laid-wide-every")
	for k := range pickedCount(t, "laid-points") {
		point := pointOf(radius, latSeed, lonSeed, k)
		footprint := span * noise.Hash(footSeed, k) * float64(k%every)
		want := face.Within(point, footprint)
		got := paint.within(point, footprint, whole)
		most := math.Max(want.R, math.Max(want.G, want.B))
		apart := lightApart(got, want)
		if !(apart <= tolerance*math.Max(most, brightFloor)) {
			t.Fatalf("%s: at point %d the painter makes %v, Within %v", name,
				k, got, want)
		}
		up := point.Unit()
		latitude := math.Abs(latitudeOf(up, face.pole))
		_, _, made := face.flowed(up)
		survey.lit += count(positive(most))
		survey.cleared += count(positive(made.cleared))
		survey.glowing += count(latitude < face.airband)
		survey.polar += count(latitude > polarFrom && positive(most))
	}
	return survey
}

// lightApart is the largest difference of any of two lights' three.
func lightApart(got, want maths.Light) float64 {
	red, green := math.Abs(got.R-want.R), math.Abs(got.G-want.G)
	blue := math.Abs(got.B - want.B)
	return math.Max(red, math.Max(green, blue))
}

// rise is how far apart the grid's rows are, in radians.
func (grid *aGrid) rise() float64 { return double * grid.limit / float64(grid.rows) }

// Cells is the grid's size.
func (grid *aGrid) Cells() (int, int) { return grid.across, grid.rows }

// Wall is the latitude the grid's last rows stand inside of.
func (grid *aGrid) Wall() float64 { return grid.limit }

// Fields are the grid's three numbers a cell.
func (grid *aGrid) Fields() (origin, spin, age []float64) {
	return grid.origin, grid.spin, grid.age
}

// At is the grid read through the reader the laid face's painter uses.
func (grid *aGrid) At(latitude, longitude float64) (origin, spin,
	age float64) {
	laid := &Laid{numbers: map[string]float64{},
		lists: map[string][]float64{}}
	layFlow(laid, grid)
	return (&painter{laid: laid}).read(latitude, longitude)
}
