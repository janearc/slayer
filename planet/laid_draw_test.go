//go:build draw

package planet

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// tilting is defined as the exponent of the Henyey-Greenstein
// denominator: three halves.
const tilting = 1.5

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

// clamp is a number held between a low and a high.
func clamp(low, high, amount float64) float64 {
	return math.Max(low, math.Min(high, amount))
}

// way is a vector of the laid face's numbers, by its name.
func (paint *painter) way(name string) maths.Vec {
	return maths.V(paint.value(name+"_x"), paint.value(name+"_y"), paint.value(name+"_z"))
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

// count is one for a thing that holds, to add up.
func count(holds bool) int {
	if holds {
		return step
	}
	return nothing
}

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
