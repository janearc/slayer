package planet

import (
	_ "embed"
	"math"
	"sync"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/weather"
)

// What the laid tests count in, each defined.
const (
	step    = 1 // defined: the step from a place in a list to the next
	forward = 1 // defined: the place after the first, where a loop over the rest begins
	home    = 0 // defined: the first of the mixed columns, the zone a point is in
)

// The places of a cell's numbers among the laid flow's cells, each
// defined by the order Fields gives them in and FlowWidth counts.
const (
	originField = 0 // defined: the first of a cell's numbers, where the air came from
	spinField   = 1 // defined: the second, its spin
	ageField    = 2 // defined: the third, its age
)

// laidNumbersRaw is every number the laid and chart tests choose, with
// what it is and who chose it.
//
//go:embed testdata/laid-numbers.yaml
var laidNumbersRaw []byte

// laidVantagesRaw is the poles, stars and eyes the laid and chart tests
// look from.
//
//go:embed testdata/laid-vantages.yaml
var laidVantagesRaw []byte

// vantagePlace is one place a test looks from, by its name: a pole, star or
// eye, as the three parts of a vector.
type vantagePlace struct {
	Name                 string
	Right, Height, Depth float64
}

// vantageFile is the shape of laid-vantages.yaml.
type vantageFile struct{ Vantages []vantagePlace }

// readNumbers is laid-numbers.yaml, read.
func readNumbers() (numbers.Numbers, error) {
	return numbers.Read("laid-numbers.yaml", laidNumbersRaw)
}

// readVantages is laid-vantages.yaml, read.
func readVantages() (vantageFile, error) {
	var file vantageFile
	err := yamlfile.Read("laid-vantages.yaml", laidVantagesRaw, &file)
	return file, err
}

// chosenNumbers is laid-numbers.yaml, read once.
var chosenNumbers = sync.OnceValues(readNumbers)

// chosenVantages is laid-vantages.yaml, read once.
var chosenVantages = sync.OnceValues(readVantages)

// picked is a number someone chose, by its name in laid-numbers.yaml.
func picked(test testing.TB, name string) float64 {
	test.Helper()
	chosen, err := chosenNumbers()
	if err != nil {
		test.Fatal(err)
	}
	number, err := chosen.Get(name)
	if err != nil {
		test.Fatal(err)
	}
	return number.Value
}

// pickedInt is a number someone chose that counts whole things.
func pickedInt(test testing.TB, name string) int {
	test.Helper()
	return int(picked(test, name))
}

// pickedSeed is a number someone chose that seeds a hash.
func pickedSeed(test testing.TB, name string) uint64 {
	test.Helper()
	return uint64(picked(test, name))
}

// pickedCount is a number someone chose that counts draws, the end of a
// loop over a hash's keys.
func pickedCount(test testing.TB, name string) int64 {
	test.Helper()
	return int64(picked(test, name))
}

// look is where a test looks from, by its name in laid-vantages.yaml.
func look(test testing.TB, name string) maths.Vec {
	test.Helper()
	file, err := chosenVantages()
	if err != nil {
		test.Fatal(err)
	}
	for _, each := range file.Vantages {
		if each.Name == name {
			return maths.V(each.Right, each.Height, each.Depth)
		}
	}
	test.Fatalf("laid-vantages.yaml: no place called %q", name)
	return maths.V(nothing, nothing, nothing)
}

// a painter paints a laid face from its numbers and lists alone, in
// float64: what a renderer that cannot call Within has to do, written out
// once so the layout is proven whole before any other renderer copies it.
type painter struct {
	laid  *Laid
	floor float64
}

// value is one of the laid face's numbers.
func (paint *painter) value(name string) float64 { return paint.laid.Number(name) }

// flowCell is the flow's cells read at a cell, wrapped around the planet.
type flowCell struct {
	cells  []float64
	across int
}

// at is one field of the cell at a column and a row.
func (grid flowCell) at(column, row, field int) float64 {
	column = ((column % grid.across) + grid.across) % grid.across
	return grid.cells[(row*grid.across+column)*FlowWidth+field]
}

// between is one field read between the cells about a place: east of
// the column by so much, north of the row by so much, and the upper row.
func (grid flowCell) between(column, row, upper, field int, east,
	north float64) float64 {
	south := grid.at(column, row, field)*(whole-east) +
		grid.at(column+step, row, field)*east
	above := grid.at(column, upper, field)*(whole-east) +
		grid.at(column+step, upper, field)*east
	return south*(whole-north) + above*north
}

// read is the flow between cells: Weather.At.
func (paint *painter) read(latitude, longitude float64) (origin, spin,
	age float64) {
	across, rows := int(paint.value("across")), int(paint.value("rows"))
	limit, cells := paint.value("wall"), paint.laid.List("flow")
	rise, run := double*limit/float64(rows), circle/float64(across)
	last := float64(rows - step)
	down := math.Max(nothing, math.Min(last, (latitude+limit)/rise-half))
	round := longitude / run
	row, column := int(math.Floor(down)), int(math.Floor(round))
	north, east := down-float64(row), round-float64(column)
	upper := min(row+step, rows-step)
	grid := flowCell{cells: cells, across: across}
	origin = grid.between(column, row, upper, originField, east, north)
	spin = grid.between(column, row, upper, spinField, east, north)
	age = math.Max(nothing, grid.between(column, row, upper, ageField, east, north))
	edge := limit - rise/double
	if past := math.Abs(latitude) - edge; positive(past) {
		toOrigin, toAge := paint.value("south_origin"), paint.value("south_age")
		if positive(latitude) {
			toOrigin, toAge = paint.value("north_origin"), paint.value("north_age")
		}
		gone := math.Min(whole, past/(rightAngle-edge))
		origin += gone * (toOrigin - origin)
		spin -= gone * spin
		age += gone * (toAge - age)
	}
	return origin, spin, age
}

// realWeather is the weather a fluid made, run long enough to have spun
// up storms and aged its air.
func realWeather(t *testing.T) *weather.Weather {
	t.Helper()
	limit := picked(t, "weather-limit") * maths.Degree
	grid, err := weather.NewGrid(pickedInt(t, "weather-across"),
		pickedInt(t, "weather-rows"), limit, picked(t, "weather-radius"))
	if err != nil {
		t.Fatal(err)
	}
	stirring, err := weather.NewStirring(picked(t, "stirring-rate"),
		picked(t, "stirring-radius"), picked(t, "stirring-strength"))
	if err != nil {
		t.Fatal(err)
	}
	made, err := weather.New(grid, picked(t, "weather-day"),
		picked(t, "weather-drag"), picked(t, "weather-relax"), stirring,
		pickedSeed(t, "weather-seed"))
	if err != nil {
		t.Fatal(err)
	}
	made.Run(picked(t, "weather-run"))
	return made
}

// pointOf is the k-th point of a body of a radius, by a hash of two
// seeds: its latitude and its longitude.
func pointOf(radius float64, latSeed, lonSeed uint64, k int64) maths.Vec {
	high := math.Asin(double*noise.Hash(latSeed, k) - whole)
	round := circle * noise.Hash(lonSeed, k)
	return maths.V(math.Cos(high)*math.Cos(round), math.Sin(high),
		math.Cos(high)*math.Sin(round)).Scale(radius)
}

// a face whose weather is not on a grid, or which has none, is not laid:
// it is Within's alone to paint.
func TestOnlyAGriddedFaceIsLaid(t *testing.T) {
	pole, star, eye := look(t, "ungridded pole"), look(t, "ungridded star"),
		look(t, "ungridded eye")
	for name, body := range map[string]Body{
		"a face with storms and no flow": stormyBody(t, false),
		"a face with a flow off a grid":  stormyBody(t, true),
	} {
		face := NewFace(body, pole, star, eye, wide(), picked(t, "ungridded-exposure"))
		if _, ok := face.Laid(); ok {
			t.Errorf("%s was laid", name)
		}
	}
}

// the laid grid read between cells is the weather's own At everywhere,
// past its walls to the poles as well, where it fades to its edge rows'
// means: a third of the points asked are past the walls.
func TestTheLaidGridReadsAsTheWeather(t *testing.T) {
	sky := realWeather(t)
	laid := &Laid{numbers: map[string]float64{},
		lists: map[string][]float64{}}
	layFlow(laid, sky)
	paint := &painter{laid: laid, floor: picked(t, "painter-floor")}
	tolerance := picked(t, "read-close")
	latSeed, lonSeed := pickedSeed(t, "read-lat-seed"), pickedSeed(t, "read-lon-seed")
	wallSeed, wallEvery := pickedSeed(t, "read-wall-seed"), pickedCount(t, "read-wall-every")
	wallFrom, wallSpan := picked(t, "read-wall-from"), picked(t, "read-wall-span")
	for k := range pickedCount(t, "read-points") {
		latitude := (noise.Hash(latSeed, k) - half) * math.Pi
		if k%wallEvery == nothing {
			latitude = math.Copysign(wallFrom+wallSpan*noise.Hash(wallSeed, k),
				latitude) * maths.Degree
		}
		longitude := circle * noise.Hash(lonSeed, k)
		origin, spin, age := sky.At(latitude, longitude)
		gotOrigin, gotSpin, gotAge := paint.read(latitude, longitude)
		for _, field := range []struct {
			name      string
			got, want float64
		}{{"origin", gotOrigin, origin}, {"spin", gotSpin, spin},
			{"age", gotAge, age}} {
			if !(math.Abs(field.got-field.want) <=
				tolerance*math.Max(whole, math.Abs(field.want))) {
				t.Fatalf("at %g, %g field %s reads %g, the weather's %g",
					latitude, longitude, field.name, field.got, field.want)
			}
		}
	}
}
