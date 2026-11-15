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
	half    = 0.5 // defined: half of a whole
	step    = 1   // defined: the step from a place in a list to the next
	forward = 1   // defined: the place after the first, where a loop over the rest begins
	home    = 0   // defined: the first of the mixed columns, the zone a point is in
	// rayleighScale is defined as the Rayleigh phase function's scale: three
	// quarters of one plus the cosine of the turn, squared.
	rayleighScale = 0.75
	// sphere is defined as the solid angle of a whole sphere: four pi
	// steradians.
	sphere = 4 * math.Pi
	// tilting is defined as the exponent of the Henyey-Greenstein
	// denominator: three halves.
	tilting = 1.5
)

// The places of a cell's numbers among the laid flow's cells, each
// defined by the order Fields gives them in and FlowWidth counts.
const (
	originField = 0 // defined: the first of a cell's numbers, where the air came from
	spinField   = 1 // defined: the second, its spin
	ageField    = 2 // defined: the third, its age
)

// The places of a layer's numbers in its row of the laid leaves, each
// defined by the order LeafWidth counts them in.
const (
	leafTau      = 0 // defined: the layer's depth
	leafAlbedo   = 1 // defined: the share it scatters
	leafGas      = 2 // defined: its gas's scattering
	leafHaze     = 3 // defined: its haze's scattering
	leafTogether = 4 // defined: the sum of the two
	leafSlot     = 5 // defined: its lobe's place
)

// The places of a band's numbers in its row of the laid shades, each
// defined by the order ShadeWidth counts them in.
const (
	shadeStar  = 0 // defined: the band's starlight
	shadeRed   = 1 // defined: the red a unit of its light is
	shadeGreen = 2 // defined: the green a unit of its light is
	shadeBlue  = 3 // defined: the blue a unit of its light is
	shadeGlow  = 4 // defined: its airglow
	shadeHeat  = 5 // defined: its heat's scale
	shadeFall  = 6 // defined: the temperature its heat falls by e at
)

// The places of a lobe's numbers in its row of the laid lobes, and of a
// stack's in its row of the laid starts, each defined by the order
// LobeWidth and StartWidth count them in.
const (
	lobeRest   = 0 // defined: a lobe's rest
	lobeNear   = 1 // defined: its near
	lobeTwice  = 2 // defined: its twice
	startFirst = 0 // defined: a stack's first leaf
	startCount = 1 // defined: how many leaves it has
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

// clamp is a number held between a low and a high.
func clamp(low, high, amount float64) float64 {
	return math.Max(low, math.Min(high, amount))
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

// way is a vector of the laid face's numbers, by its name.
func (paint *painter) way(name string) maths.Vec {
	return maths.V(paint.value(name+"_x"), paint.value(name+"_y"), paint.value(name+"_z"))
}

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

// latitudeFrom is the latitude of a direction, in degrees, from the pole.
func (paint *painter) latitudeFrom(up maths.Vec) float64 {
	sine := up.Dot(paint.way("pole"))
	return math.Asin(clamp(-whole, whole, sine)) / maths.Degree
}

// longitudeFrom is the longitude of a direction, in radians, east of the
// front, from minus pi to pi.
func (paint *painter) longitudeFrom(up maths.Vec) float64 {
	east, front := paint.way("east"), paint.way("front")
	return math.Atan2(up.Dot(east), up.Dot(front))
}

// flowed is Face.flowed.
func (paint *painter) flowed(up maths.Vec) (latitude, lift float64,
	made marks) {
	latitude = paint.latitudeFrom(up)
	longitude := paint.longitudeFrom(up)
	if longitude < nothing {
		longitude += circle
	}
	origin, spin, age := paint.read(latitude*maths.Degree, longitude)
	fading, walled := paint.value("fading"), paint.value("walled")
	gone := clamp(nothing, whole, (math.Abs(latitude)-fading)/
		(walled-fading))
	origin += gone * (latitude*maths.Degree - origin)
	spin *= whole - gone
	made.age = age
	cyclonic := spin
	if latitude < nothing {
		cyclonic = -spin
	}
	marked, marking := paint.value("marked"), paint.value("marking")
	lift = clamp(-whole, whole, -cyclonic*paint.value("lifting"))
	made.capped = clamp(nothing, whole, (-cyclonic-marked)/marking)
	made.cleared = clamp(nothing, whole, (cyclonic-marked)/marking)
	return origin / maths.Degree, lift, made
}

// zone is Face.zoneOf.
func (paint *painter) zone(latitude float64) int {
	norths := paint.laid.List("norths")
	for i, north := range norths {
		if latitude < north {
			return i
		}
	}
	return len(norths) - step
}

// mixing is the columns a point is mixed from and their shares, the
// first whole.
type mixing struct {
	columns []int
	shares  []float64
}

// add puts a column in the mix if it has a share.
func (mix *mixing) add(column int, share float64) {
	if positive(share) {
		mix.columns = append(mix.columns, column)
		mix.shares = append(mix.shares, share)
	}
}

// edgeShare is how much of a neighbouring zone shows by a zone's edge,
// given how far in from the edge a latitude is and how wide the edge is.
func edgeShare(in, width float64) float64 { return half * (whole - in/width) }

// mixed is Face.mixAt as columns and shares, the first whole.
func (paint *painter) mixed(latitude, lift float64, made marks) ([]int,
	[]float64) {
	norths, width := paint.laid.List("norths"), paint.value("edge")
	here := paint.zone(latitude)
	there := here
	var share float64
	if positive(width) {
		if in := latitude - at(norths, here-step); here > nothing && in < width {
			there, share = here-step, edgeShare(in, width)
		} else if in := norths[here] - latitude; here < len(norths)-step &&
			in < width {
			there, share = here+step, edgeShare(in, width)
		}
	}
	if share == nothing {
		there = paint.neighbour(latitude, here)
	}
	mix := &mixing{columns: []int{here}, shares: []float64{whole}}
	pulled := share + half*math.Max(nothing, -lift)
	mix.add(there, math.Min(whole, pulled))
	if churned := int(paint.value("churned")); churned >= nothing {
		pull := double*share + math.Max(nothing, lift)
		churning := paint.value("churning")
		mix.add(churned, math.Min(whole, churning*pull))
	}
	if capped := int(paint.value("capped")); capped >= nothing {
		cleared := int(paint.value("cleared"))
		mix.add(cleared, made.cleared)
		mix.add(capped, made.capped)
	}
	return mix.columns, mix.shares
}

// at is a list's number at a place, or zero outside it.
func at(list []float64, place int) float64 {
	if place < nothing || place >= len(list) {
		return nothing
	}
	return list[place]
}

// neighbour is Face.neighbour.
func (paint *painter) neighbour(latitude float64, here int) int {
	norths := paint.laid.List("norths")
	below, above := here-step, here+step
	switch {
	case below < nothing && above >= len(norths):
		return here
	case below < nothing:
		return above
	case above >= len(norths):
		return below
	}
	if latitude-norths[below] < norths[here]-latitude {
		return below
	}
	return above
}

// origin is where the air at a latitude and longitude, in degrees, came
// from, as the painter reads the flow.
func (paint *painter) origin(latitude, longitude float64) float64 {
	pole, east, front := paint.way("pole"), paint.way("east"),
		paint.way("front")
	high, round := latitude*maths.Degree, longitude*maths.Degree
	along := front.Scale(math.Cos(round))
	side := east.Scale(math.Sin(round))
	level := along.Add(side)
	across := level.Scale(math.Cos(high))
	lifted := pole.Scale(math.Sin(high))
	up := across.Add(lifted)
	came, _, _ := paint.flowed(up)
	return came
}

// steep is Face.steepness.
func (paint *painter) steep(up maths.Vec, finest float64) float64 {
	latitude := paint.latitudeFrom(up)
	longitude := paint.longitudeFrom(up) / maths.Degree
	reach := math.Max(paint.value("front_step"), finest/maths.Degree)
	wide := reach / math.Max(math.Cos(latitude*maths.Degree), paint.floor)
	northward := (paint.origin(latitude+reach, longitude) -
		paint.origin(latitude-reach, longitude)) / (double * reach)
	eastward := (paint.origin(latitude, longitude+wide) -
		paint.origin(latitude, longitude-wide)) / (double * reach)
	return math.Hypot(northward, eastward)
}

// leafRow is one layer's numbers among the laid leaves.
type leafRow struct{ tau, albedo, gas, haze, together, slot float64 }

// leafAt is the layer at a place among the laid leaves.
func leafAt(leaves []float64, place int) leafRow {
	row := leaves[place*LeafWidth : (place+step)*LeafWidth]
	return leafRow{tau: row[leafTau], albedo: row[leafAlbedo],
		gas: row[leafGas], haze: row[leafHaze],
		together: row[leafTogether], slot: row[leafSlot]}
}

// shone is shone for one stack of the laid leaves.
func (paint *painter) shone(stack int, tilts []float64, star, eye,
	turn float64) float64 {
	starts, leaves := paint.laid.List("starts"), paint.laid.List("leaves")
	first := int(starts[stack*StartWidth+startFirst])
	count := int(starts[stack*StartWidth+startCount])
	slant := whole/star + whole/eye
	var above, light float64
	for k := first; k < first+count; k++ {
		layer := leafAt(leaves, k)
		if positive(layer.tau) {
			var phase float64
			if layer.together != nothing {
				phase = (layer.gas*rayleighScale*(whole+turn*turn) +
					layer.haze*tilts[int(layer.slot)]) / layer.together
			}
			light += layer.albedo * phase / sphere * star / (star + eye) *
				math.Exp(-above*slant) * -math.Expm1(-layer.tau*slant)
		}
		above += layer.tau
	}
	return light + paint.value("deck")*star/math.Pi*math.Exp(-above*slant)
}

// from is Face.from: a column's light in a band, a zone's by its rung.
func (paint *painter) from(column, band, rung int, toward float64,
	tilts []float64, star, eye, turn float64) float64 {
	bands, columns := int(paint.value("bands")), int(paint.value("columns"))
	if column >= int(paint.value("ladders")) {
		return paint.shone(column*bands+band, tilts, star, eye, turn)
	}
	stack := (columns+column*int(paint.value("rungs"))+rung)*bands + band
	below := paint.shone(stack, tilts, star, eye, turn)
	if toward == nothing {
		return below
	}
	above := paint.shone(stack+bands, tilts, star, eye, turn)
	return below + toward*(above-below)
}

// tiltsAt is each lobe's tilt toward a turn.
func (paint *painter) tiltsAt(turn float64) []float64 {
	var tilts []float64
	lobes := paint.laid.List("lobes")
	for k := nothing; k < len(lobes); k += LobeWidth {
		spread := lobes[k+lobeNear] - lobes[k+lobeTwice]*turn
		tilts = append(tilts, lobes[k+lobeRest]/math.Pow(spread, tilting))
	}
	return tilts
}

// rungOf is the rung a laid zone's air has cooked to, and how far along
// toward the next it is.
func (paint *painter) rungOf(age float64) (int, float64) {
	cooking := paint.value("cooking")
	if cooking == nothing {
		return nothing, nothing
	}
	rungs := paint.value("rungs")
	way := (whole - math.Exp(-math.Max(nothing, age)/cooking)) * (rungs - whole)
	floor := int(math.Floor(way))
	rung := min(floor, int(rungs)-double)
	return rung, way - float64(rung)
}

// frontOf is how steep the air's fronts are at a direction, as a share of
// what is calm, held to the most a front shows; none where there is no
// airglow.
func (paint *painter) frontOf(up maths.Vec, finest float64,
	glows bool) float64 {
	if !glows {
		return nothing
	}
	steep := paint.steep(up, finest) / paint.value("calm")
	return math.Min(paint.value("most_front"), math.Max(nothing, steep-whole))
}

// within is Face.Within from the laid face alone, the rings' shadow
// given.
func (paint *painter) within(point maths.Vec, footprint,
	shadow float64) maths.Light {
	up, sun := point.Unit(), paint.way("star")
	toward := paint.way("eye").Sub(point).Unit()
	star, eye, turn := up.Dot(sun), up.Dot(toward), -sun.Dot(toward)
	if !positive(eye) {
		return maths.Grey(nothing)
	}
	came, lift, made := paint.flowed(up)
	columns, shares := paint.mixed(came, lift, made)
	rung, along := paint.rungOf(made.age)
	glows := positive(paint.value("glows")) && math.Abs(came) < paint.value("airband")
	front := paint.frontOf(up, footprint/point.Len(), glows)
	var reached float64
	if positive(star) {
		reached = shadow
	}
	tilts := paint.tiltsAt(turn)
	var red, green, blue float64
	shades := paint.laid.List("shades")
	for band := range int(paint.value("bands")) {
		shade := shades[band*ShadeWidth : (band+step)*ShadeWidth]
		var sent float64
		if positive(reached) {
			sent = paint.from(columns[home], band, rung, along, tilts, star,
				eye, turn)
			for k := forward; k < len(columns); k++ {
				other := paint.from(columns[k], band, rung, along, tilts, star,
					eye, turn)
				sent += shares[k] * (other - sent)
			}
		}
		power := shade[shadeStar] * reached * sent
		if hottest := paint.value("hottest"); made.cleared != nothing && hottest != nothing {
			kelvin := paint.value("top") + (hottest-paint.value("top"))*made.cleared
			if positive(kelvin) {
				power += made.cleared * shade[shadeHeat] /
					math.Expm1(shade[shadeFall]/kelvin)
			}
		}
		if glows {
			power += shade[shadeGlow] / math.Max(eye, paint.value("grazing")) * front
		}
		red, green, blue = red+power*shade[shadeRed],
			green+power*shade[shadeGreen], blue+power*shade[shadeBlue]
	}
	return maths.RGB(math.Max(nothing, red), math.Max(nothing, green),
		math.Max(nothing, blue))
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

// count is one for a thing that holds, to add up.
func count(holds bool) int {
	if holds {
		return step
	}
	return nothing
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
