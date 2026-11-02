package planet

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// A Zone is the planet between two latitudes, in degrees, and the
// column of air over it, top down.
type Zone struct {
	south, north float64
	column       []Layer
}

// NewZone is a zone, refused if its latitudes are out of order or off
// the planet, or its column's layers are out of order.
func NewZone(south, north float64, column []Layer) (Zone, error) {
	if !(south >= -90 && north <= 90 && north > south) {
		return Zone{}, fmt.Errorf("a zone from %g to %g degrees", south,
			north)
	}
	for i := 1; i < len(column); i++ {
		if column[i].top < column[i-1].bottom {
			return Zone{}, fmt.Errorf("layer %d starts at %g pascals, "+
				"above the last one's bottom at %g", i, column[i].top,
				column[i-1].bottom)
		}
	}
	return Zone{south: south, north: north, column: column}, nil
}

// A Body is a planet: its radius in metres, its gravity in metres a
// second squared, the albedo of the deck under its air, and its zones,
// which must cover it from pole to pole.
type Body struct {
	radius, gravity, deck float64
	zones                 []Zone
	rings                 []Ring
	storms                Storms
	churn                 []Layer
	churning              float64
	cap, clearing         []Layer
	flow                  Flow
	top, hottest          float64
	cooked                Haze
	cooking               float64
	airglow, airband      float64
	airfronts             float64
}

// Marked is the body with the columns its vortices show: a cap, the high
// haze an anticyclone lifts in its middle, white on jupiter; and a
// clearing, the deeper cloud a cyclone opens to, dark.
func (body Body) Marked(cap, clearing []Layer) (Body, error) {
	if len(cap) == 0 || len(clearing) == 0 {
		return body, fmt.Errorf("a cap of %d layers and a clearing of %d",
			len(cap), len(clearing))
	}
	body.cap, body.clearing = cap, clearing
	return body, nil
}

// Stormy is the body with storms tearing and mixing its bands.
func (body Body) Stormy(storms Storms) Body {
	body.storms = storms
	return body
}

// Churned is the body with a column of its own where its bands mix: the air the
// shear brings up and stirs, showing at a band's edge by a strength from zero
// to one, and fading into the bands either side.
func (body Body) Churned(column []Layer, strength float64) (Body, error) {
	if !(strength >= 0 && strength <= 1) || len(column) == 0 {
		return body, fmt.Errorf("a churn of strength %g and %d layers",
			strength, len(column))
	}
	body.churn, body.churning = column, strength
	return body, nil
}

// NewBody is a body, refused if its zones leave a latitude uncovered.
func NewBody(radius, gravity, deck float64, zones []Zone) (Body, error) {
	if !(radius > 0) || !(gravity > 0) || !(deck >= 0 && deck <= 1) {
		return Body{}, fmt.Errorf("a body of radius %g, gravity %g and "+
			"deck %g", radius, gravity, deck)
	}
	south := -90.0
	for _, zone := range zones {
		if zone.south != south {
			return Body{}, fmt.Errorf("a gap or overlap at %g degrees",
				south)
		}
		south = zone.north
	}
	if south != 90 {
		return Body{}, fmt.Errorf("zones stop at %g degrees", south)
	}
	return Body{radius: radius, gravity: gravity, deck: deck,
		zones: zones}, nil
}

// Radius is the body's radius in metres.
func (body Body) Radius() float64 { return body.radius }

// A Face is a body as an eye sees it under a star: what each point of
// it sends toward the eye, as light in libtheme's bands made into the
// tracer's three channels at an exposure.
type Face struct {
	pole, star, eye maths.Vec
	light           bands.Light
	norths          []float64
	rings           []Ring
	storms          Storms
	ways            frame
	whirls          []vortex
	flow            Flow
	top, hottest    float64
	churned         int
	churning        float64
	capped, cleared int
	stacks          [][][]optics
	ladders         [][][][]optics
	cooking         float64
	airglow         float64
	airband         float64
	fronts, calm    float64
	deck            float64
	exposure        float64
}

// NewFace is the body seen from an eye at a place in the body's own
// space, with its pole one way and the star another, the star's light
// on a surface square to it given band by band.
func NewFace(body Body, pole, star, eye maths.Vec, light bands.Light,
	exposure float64) *Face {
	columns := make([][]Layer, len(body.zones))
	norths := make([]float64, len(body.zones))
	for i, zone := range body.zones {
		norths[i], columns[i] = zone.north, zone.column
	}
	churned, capped, cleared := -1, -1, -1
	if body.churn != nil {
		churned = len(columns)
		columns = append(columns, body.churn)
	}
	if body.cap != nil {
		capped, cleared = len(columns), len(columns)+1
		columns = append(columns, body.cap, body.clearing)
	}
	stacks := make([][][]optics, len(columns))
	for i, column := range columns {
		stacks[i] = stacksOf(column, light, body.gravity)
	}
	var ladders [][][][]optics
	if body.cooking > 0 {
		for _, zone := range body.zones {
			var rungsOf [][][]optics
			for _, column := range body.ladder(zone.column) {
				rungsOf = append(rungsOf, stacksOf(column, light,
					body.gravity))
			}
			ladders = append(ladders, rungsOf)
		}
	}
	ways := frameOf(pole.Unit(), eye)
	face := &Face{pole: pole.Unit(), star: star.Unit(), eye: eye,
		storms: body.storms, ways: ways,
		whirls: body.storms.vortices(ways), flow: body.flow,
		top: body.top, hottest: body.hottest,
		churned: churned, churning: body.churning,
		capped: capped, cleared: cleared,
		light: light, norths: norths, rings: body.rings, stacks: stacks,
		ladders: ladders, cooking: body.cooking,
		airglow: body.airglow, airband: body.airband,
		fronts:   body.airfronts,
		deck:     body.deck,
		exposure: exposure}
	face.calm = face.calmness()
	return face
}

// stacksOf is a column's optics in each of the light's bands, its
// layers top down.
func stacksOf(column []Layer, light bands.Light,
	gravity float64) [][]optics {
	first, last := light.Span()
	var made [][]optics
	for band := first; band <= last; band++ {
		nm := centre(band)
		stack := make([]optics, len(column))
		for k, layer := range column {
			stack[k] = layer.optics(nm, gravity)
		}
		made = append(made, stack)
	}
	return made
}

// latitudeOf is the latitude, in degrees, of a way up from the middle,
// by a pole. two unit ways can dot to an ulp past one, which arcsine
// would make no number, so the dot is held to one at most.
func latitudeOf(up, pole maths.Vec) float64 {
	return math.Asin(math.Max(-1, math.Min(1, up.Dot(pole)))) /
		maths.Degree
}

// blend is the zone a latitude in degrees falls in, the zone across the
// nearer edge, and how much of that one's air shows: half at the edge,
// none an edge's width in from it.
func (face *Face) blend(latitude float64) (here, there int,
	share float64) {
	here = face.zoneOf(latitude)
	width := face.storms.edge
	if !(width > 0) {
		return here, here, 0
	}
	if here > 0 {
		if in := latitude - face.norths[here-1]; in < width {
			return here, here - 1, 0.5 * (1 - in/width)
		}
	}
	if here < len(face.norths)-1 {
		if in := face.norths[here] - latitude; in < width {
			return here, here + 1, 0.5 * (1 - in/width)
		}
	}
	return here, here, 0
}

// zoneOf is the zone a latitude in degrees falls in.
func (face *Face) zoneOf(latitude float64) int {
	for i, north := range face.norths {
		if latitude < north {
			return i
		}
	}
	return len(face.norths) - 1
}

// centre is a band's middle wavelength, in nanometres, by its log.
func centre(band bands.Band) float64 {
	lo, hi := band.Edges()
	return math.Sqrt(lo * hi)
}

// neighbour is the zone across the nearer edge from a latitude in
// degrees, or its own zone where it has but one.
func (face *Face) neighbour(latitude float64) int {
	here := face.zoneOf(latitude)
	below, above := here-1, here+1
	switch {
	case below < 0 && above >= len(face.norths):
		return here
	case below < 0:
		return above
	case above >= len(face.norths):
		return below
	}
	if latitude-face.norths[below] < face.norths[here]-latitude {
		return below
	}
	return above
}

// a mix is the air a point of the face shows: its own zone's column,
// then each blend in turn, so much of another column's light taken in.
type mix struct {
	here   int
	blends []blend
	rung   int
	toward float64
}

// a blend is so much of a column's light taken into a mix.
type blend struct {
	column int
	share  float64
}

// mixAt is the face's mix at a latitude in degrees, with a lift from
// minus one to one and the marks its vortices leave.
//
// at a band's edge the two zones blend and the churn shows; where the
// folded flow lifts, more of the churn rises into view, and where it
// sinks, more of the next zone's air is pulled across; a cyclone opens a
// clearing and an anticyclone lifts a cap, last, over all of it.
//
// so the folds and the vortices draw in every shade between five columns:
// modelled.
func (face *Face) mixAt(latitude, lift float64, marked marks) mix {
	here, there, share := face.blend(latitude)
	if share == 0 {
		there = face.neighbour(latitude)
	}
	made := mix{here: here}
	made.add(there, math.Min(1, share+0.5*math.Max(0, -lift)))
	if face.churned >= 0 {
		made.add(face.churned, math.Min(1,
			face.churning*(2*share+math.Max(0, lift))))
	}
	if face.capped >= 0 {
		made.add(face.cleared, marked.cleared)
		made.add(face.capped, marked.capped)
	}
	return made
}

// add takes so much of a column into the mix, if any.
func (made *mix) add(column int, share float64) {
	if share > 0 {
		made.blends = append(made.blends, blend{column, share})
	}
}

// At is the light a point of the body sends toward the eye, every layer
// of its storms drawn. the night side and the far side send none.
func (face *Face) At(p maths.Vec) maths.Light { return face.Within(p, 0) }

// Within is the light a point of the body sends toward the eye, for a
// pixel so many metres wide there: its storms drawn down to the finest
// the pixel tells apart, so far off they cost little and alias none.
func (face *Face) Within(p maths.Vec, footprint float64) maths.Light {
	up := p.Unit()
	toward := face.eye.Sub(p).Unit()
	star, eye := up.Dot(face.star), up.Dot(toward)
	if !(eye > 0) {
		return maths.Light{}
	}
	mixed, marked, came := face.airAt(up, footprint/p.Len())
	front := 0.0
	if face.airglow > 0 && math.Abs(came) < face.airband {
		front = face.front(up, footprint/p.Len())
	}
	reached := 0.0
	if star > 0 {
		reached = shadowed(face.rings, p, face.star, face.pole)
	}
	turn := -face.star.Dot(toward)
	first, _ := face.light.Span()
	lines := make(map[float64]float64, len(face.stacks[mixed.here]))
	for i := range face.stacks[mixed.here] {
		band := first + bands.Band(i)
		sent := 0.0
		if reached > 0 {
			sent = face.sent(mixed, i, star, eye, turn)
		}
		lo, hi := band.Edges()
		lines[centre(band)] = (face.light.Power(band)*reached*sent +
			face.glowed(marked.cleared, lo, hi) +
			face.glowedAt(came, eye, lo, hi)*front) * face.exposure
	}
	red, green, blue := srgb.Light(observer.Swatch(bands.Lines(lines)))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

// airAt is the mix of columns a way of the face shows, the marks its
// vortices leave and the latitude in degrees its air came from: read
// from the flow, where the body has one, or from its storms' folds and
// spots, drawn down to the finest a pixel there tells apart, in radians.
func (face *Face) airAt(up maths.Vec, finest float64) (mix, marks,
	float64) {
	var latitude, lift float64
	var marked marks
	if face.flow != nil {
		latitude, lift, marked = face.flowed(up)
	} else {
		latitude, lift, marked = face.storms.moved(up, face.ways,
			face.whirls, finest)
	}
	mixed := face.mixAt(latitude, lift, marked)
	mixed.rung, mixed.toward = face.cookedBy(marked.age)
	return mixed, marked, latitude
}

// sent is the light the mix sends back in one band, over the star's:
// its own column's, with each blend taken in turn.
func (face *Face) sent(mixed mix, i int, star, eye, turn float64) float64 {
	sent := face.from(mixed, mixed.here, i, star, eye, turn)
	for _, each := range mixed.blends {
		other := face.from(mixed, each.column, i, star, eye, turn)
		sent += each.share * (other - sent)
	}
	return sent
}

// from is the light one column sends back in one band, over the star's:
// a zone's as cooked as the mix's air, between the rungs either side,
// and any other column's as it is.
func (face *Face) from(mixed mix, column, i int, star, eye,
	turn float64) float64 {
	if face.ladders == nil || column >= len(face.ladders) {
		return reflected(face.stacks[column][i], face.deck, star, eye, turn)
	}
	rungs := face.ladders[column]
	below := reflected(rungs[mixed.rung][i], face.deck, star, eye, turn)
	if mixed.toward == 0 {
		return below
	}
	above := reflected(rungs[mixed.rung+1][i], face.deck, star, eye, turn)
	return below + mixed.toward*(above-below)
}
