package planet

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
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
	if !(south >= -poleLatitude && north <= poleLatitude && north > south) {
		return Zone{}, fmt.Errorf("a zone from %g to %g degrees", south,
			north)
	}
	for i := adjacent; i < len(column); i++ {
		if column[i].top < column[i-adjacent].bottom {
			return Zone{}, fmt.Errorf("layer %d starts at %g pascals, "+
				"above the last one's bottom at %g", i, column[i].top,
				column[i-adjacent].bottom)
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
// haze an anticyclone lifts in its middle, white on a gas giant; and a
// clearing, the deeper cloud a cyclone opens to, dark.
func (body Body) Marked(cap, clearing []Layer) (Body, error) {
	if len(cap) == nothing || len(clearing) == nothing {
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
	if !(strength >= nothing && strength <= whole) ||
		len(column) == nothing {
		return body, fmt.Errorf("a churn of strength %g and %d layers",
			strength, len(column))
	}
	body.churn, body.churning = column, strength
	return body, nil
}

// NewBody is a body, refused if its zones leave a latitude uncovered.
func NewBody(radius, gravity, deck float64, zones []Zone) (Body, error) {
	if !(radius > nothing) || !(gravity > nothing) ||
		!(deck >= nothing && deck <= whole) {
		return Body{}, fmt.Errorf("a body of radius %g, gravity %g and "+
			"deck %g", radius, gravity, deck)
	}
	south := -poleLatitude
	for _, zone := range zones {
		if zone.south != south {
			return Body{}, fmt.Errorf("a gap or overlap at %g degrees",
				south)
		}
		south = zone.north
	}
	if south != poleLatitude {
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
	stacks          [][][]leaf
	ladders         [][][][]leaf
	cooking         float64
	airglow         float64
	airband         float64
	fronts, calm    float64
	deck            float64
	exposure        float64
	shades          []shade
	lobes           []lobe
	unit            float64
	quick           bool
}

// NewFace is the body seen from an eye at a place in the body's own
// space, with its pole one way and the star another, the star's light
// on a surface square to it given band by band.
func NewFace(body Body, pole, star, eye maths.Vec, light bands.Light,
	exposure float64) *Face {
	return newFace(body, pole, eye, star, eye, light, exposure)
}

// NewFaceFixed makes a face with a world-space prime meridian independent
// of the eye. The caller carries prime with the body as it turns. Lighting
// still uses the actual eye; this only fixes the flow's longitude frame.
// A zero or pole-parallel prime uses the same upright fallback as frameOf.
func NewFaceFixed(body Body, pole, prime, star, eye maths.Vec,
	light bands.Light, exposure float64) *Face {
	return newFace(body, pole, prime, star, eye, light, exposure)
}

func newFace(body Body, pole, prime, star, eye maths.Vec, light bands.Light,
	exposure float64) *Face {
	columns := make([][]Layer, len(body.zones))
	norths := make([]float64, len(body.zones))
	for i, zone := range body.zones {
		norths[i], columns[i] = zone.north, zone.column
	}
	churned, capped, cleared := absent, absent, absent
	if body.churn != nil {
		churned = len(columns)
		columns = append(columns, body.churn)
	}
	if body.cap != nil {
		capped, cleared = len(columns), len(columns)+adjacent
		columns = append(columns, body.cap, body.clearing)
	}
	stacks := make([][][]leaf, len(columns))
	for i, column := range columns {
		stacks[i] = stacksOf(column, light, body.gravity)
	}
	ladders := laddersOf(body, light)
	ways := frameOf(pole.Unit(), prime)
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
	face.shades = shadesOf(light)
	face.lobes = face.seat()
	face.unit = whiteUnit()
	face.quick = agreeing(light, face.shades, face.unit)
	return face
}

// laddersOf is each zone's ladder of columns as leaves in each of the light's
// bands, for a body that cooks, and none for one that does not.
func laddersOf(body Body, light bands.Light) [][][][]leaf {
	var ladders [][][][]leaf
	if body.cooking > nothing {
		for _, zone := range body.zones {
			var rungsOf [][][]leaf
			for _, column := range body.ladder(zone.column) {
				rungsOf = append(rungsOf, stacksOf(column, light,
					body.gravity))
			}
			ladders = append(ladders, rungsOf)
		}
	}
	return ladders
}

// stacksOf is a column's leaves in each of the light's bands, its
// layers top down.
func stacksOf(column []Layer, light bands.Light,
	gravity float64) [][]leaf {
	first, last := light.Span()
	var made [][]leaf
	for band := first; band <= last; band++ {
		nm := centre(band)
		stack := make([]leaf, len(column))
		for k, layer := range column {
			stack[k] = layer.optics(nm, gravity).leaf()
		}
		made = append(made, stack)
	}
	return made
}

// latitudeOf is the latitude, in degrees, of a way up from the middle,
// by a pole. two unit ways can dot to an ulp past one, which arcsine
// would make no number, so the dot is held to one at most.
func latitudeOf(up, pole maths.Vec) float64 {
	sine := up.Dot(pole)
	held := math.Max(-whole, math.Min(whole, sine))
	return math.Asin(held) / maths.Degree
}

// blend is the zone a latitude in degrees falls in, the zone across the
// nearer edge, and how much of that one's air shows: half at the edge,
// none an edge's width in from it.
func (face *Face) blend(latitude float64) (here, there int,
	share float64) {
	here = face.zoneOf(latitude)
	width := face.storms.edge
	if !(width > nothing) {
		return here, here, nothing
	}
	if here > nothing {
		if in := latitude - face.norths[here-adjacent]; in < width {
			return here, here - adjacent, half * (whole - in/width)
		}
	}
	if here < len(face.norths)-adjacent {
		if in := face.norths[here] - latitude; in < width {
			return here, here + adjacent, half * (whole - in/width)
		}
	}
	return here, here, nothing
}

// zoneOf is the zone a latitude in degrees falls in.
func (face *Face) zoneOf(latitude float64) int {
	for i, north := range face.norths {
		if latitude < north {
			return i
		}
	}
	return len(face.norths) - adjacent
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
	below, above := here-adjacent, here+adjacent
	switch {
	case below < nothing && above >= len(face.norths):
		return here
	case below < nothing:
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
	held   [mostBlends]blend
	taken  int
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
	if share == nothing {
		there = face.neighbour(latitude)
	}
	made := mix{here: here}
	sunk := half * math.Max(nothing, -lift)
	made.add(there, math.Min(whole, share+sunk))
	if face.churned >= nothing {
		raised := double*share + math.Max(nothing, lift)
		made.add(face.churned, math.Min(whole, face.churning*raised))
	}
	if face.capped >= nothing {
		made.add(face.cleared, marked.cleared)
		made.add(face.capped, marked.capped)
	}
	return made
}

// mostBlends is the most columns a mix takes in beside its own: its
// neighbour, the churn, a clearing and a cap. defined as the number of
// calls of add that mixAt makes.
const mostBlends = 4

// blends are the columns the mix took in, in the order it took them.
func (made *mix) blends() []blend { return made.held[:made.taken] }

// add takes so much of a column into the mix, if any. the mix holds its
// blends itself, so a point of the face makes no list.
func (made *mix) add(column int, share float64) {
	if share > nothing {
		made.held[made.taken] = blend{column, share}
		made.taken++
	}
}

// At is the light a point of the body sends toward the eye, every layer
// of its storms drawn. the night side and the far side send none.
func (face *Face) At(point maths.Vec) maths.Light {
	return face.Within(point, nothing)
}

// Within is the light a point of the body sends toward the eye, for a
// pixel so many metres wide there: its storms drawn down to the finest
// the pixel tells apart, so far off they cost little and alias none.
func (face *Face) Within(point maths.Vec, footprint float64) maths.Light {
	up := point.Unit()
	toward := face.eye.Sub(point).Unit()
	star, eye := up.Dot(face.star), up.Dot(toward)
	if !(eye > nothing) {
		return noLight
	}
	mixed, marked, came := face.airAt(up, footprint/point.Len())
	var front float64
	if face.airglow > nothing && math.Abs(came) < face.airband {
		front = face.front(up, footprint/point.Len())
	}
	var reached float64
	if star > nothing {
		reached = shadowed(face.rings, point, face.star, face.pole)
	}
	at := sight{mixed: mixed, marked: marked, star: star, eye: eye,
		turn: -face.star.Dot(toward), reached: reached, came: came,
		front: front}
	red, green, blue := srgb.Light(face.colour(&at))
	return maths.RGB(math.Max(nothing, red), math.Max(nothing, green),
		math.Max(nothing, blue))
}

// a sight is what Within works out of a point before it draws its bands: the
// air's mix and marks, the cosines of the star's and eye's heights and of
// the turn, how much starlight reaches it, the latitude its air came from,
// how strongly its airglow shows, and each of the face's lobes' phase at its
// turn.
type sight struct {
	mixed                mix
	marked               marks
	star, eye, turn      float64
	reached, came, front float64
	tilts                []float64
}

// colour is the swatch of a sight: its light in every band, through the
// observer. a face the observer's sums are worked out for does them itself;
// any other asks the observer.
func (face *Face) colour(at *sight) swatch.Swatch {
	held := linePool.Get().(*lineBox)
	defer linePool.Put(held)
	powers := held.powers[:nothing]
	if at.reached > nothing {
		held.tilts = face.tilted(held.tilts, at.turn)
		at.tilts = held.tilts
	}
	for i := range face.shades {
		powers = append(powers, face.bandLight(at, i))
	}
	held.powers = powers
	if face.quick {
		return colourOf(face.shades, face.unit, powers)
	}
	lines := held.byCentre
	clear(lines)
	for i, each := range face.shades {
		lines[each.centre] = powers[i]
	}
	return observer.Swatch(bands.Lines(lines))
}

// bandLight is the light a sight sends the eye in one band: the star's,
// what the mix sends back of it, and the glows.
func (face *Face) bandLight(at *sight, i int) float64 {
	shade := &face.shades[i]
	var sent float64
	if at.reached > nothing {
		sent = face.sent(at, i)
	}
	return (shade.power*at.reached*sent +
		face.glowedBy(at.marked.cleared, shade.warm) +
		face.glowedAt(at.came, at.eye, shade.held)*at.front) * face.exposure
}

// linePool holds the scratch Within draws a pixel's bands in, so each pixel
// does not make it: a slice of the bands' light, and a map of it for the
// observer. bands.Lines reads a map and keeps none of it.
var linePool = sync.Pool{New: newLineBox}

// newLineBox is a box for the pool.
func newLineBox() any {
	return &lineBox{byCentre: make(map[float64]float64, lineRoom)}
}

// lineRoom is how many bands the pool's maps start with room for.
var lineRoom = pickCount("line-room")

// lineBox is scratch in the pool, boxed so that putting it back does not
// allocate: the bands' light by the middle of each band, and as a list, and
// the phase of each lobe.
type lineBox struct {
	byCentre map[float64]float64
	powers   []float64
	tilts    []float64
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
func (face *Face) sent(at *sight, i int) float64 {
	sent := face.from(at, at.mixed.here, i)
	for _, each := range at.mixed.blends() {
		other := face.from(at, each.column, i)
		sent += each.share * (other - sent)
	}
	return sent
}

// from is the light one column sends back in one band, over the star's:
// a zone's as cooked as the mix's air, between the rungs either side,
// and any other column's as it is.
func (face *Face) from(at *sight, column, i int) float64 {
	if face.ladders == nil || column >= len(face.ladders) {
		return shone(face.stacks[column][i], at.tilts, face.deck, at.star,
			at.eye, at.turn)
	}
	cooked := face.ladders[column]
	below := shone(cooked[at.mixed.rung][i], at.tilts, face.deck, at.star,
		at.eye, at.turn)
	if at.mixed.toward == nothing {
		return below
	}
	above := shone(cooked[at.mixed.rung+adjacent][i], at.tilts, face.deck,
		at.star, at.eye, at.turn)
	return below + at.mixed.toward*(above-below)
}

// seat gives each leaf the place of its lobe among the face's lobes, one
// for each lean its layers have, and returns the lobes.
func (face *Face) seat() []lobe {
	var made []lobe
	places := map[lobe]int{}
	for _, column := range face.stacks {
		made = seatStacks(column, made, places)
	}
	for _, zone := range face.ladders {
		for _, rung := range zone {
			made = seatStacks(rung, made, places)
		}
	}
	return made
}

// seatStacks seats the leaves of a column's stacks in the places of the lobes
// so far, adding the lobes it has not seen.
func seatStacks(stacks [][]leaf, made []lobe, places map[lobe]int) []lobe {
	for _, stack := range stacks {
		for k := range stack {
			place, seen := places[stack[k].lobe]
			if !seen {
				place = len(made)
				places[stack[k].lobe] = place
				made = append(made, stack[k].lobe)
			}
			stack[k].slot = place
		}
	}
	return made
}

// tilted is the henyey phase of each of the face's lobes at a turn, in
// place of the slice given, which it makes room in as it needs.
func (face *Face) tilted(into []float64, turn float64) []float64 {
	into = into[:nothing]
	for _, shape := range face.lobes {
		into = append(into, shape.at(turn))
	}
	return into
}
