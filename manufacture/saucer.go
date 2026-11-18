package manufacture

import (
	_ "embed"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
)

// The MCRN Larson is a flying saucer patterned on the plain saucers Gary
// Larson drew in The Far Side: a flat lens of titanium with a
// clear glass dome on top, a ring of portholes on its upper slope, and a
// disc set into its belly. The name is The Expanse's way of naming Mars's
// navy. Its proportions are picked by eye in saucer-choices.yaml; none is
// engineering.

//go:embed saucer-choices.yaml
var saucerChoices []byte

var saucerNumbers, saucerNumbersError = numbers.Read(
	"saucer-choices.yaml", saucerChoices)

// saucerValue reads the MCRN Larson's authored proportions.
func saucerValue(name string) float64 {
	if saucerNumbersError != nil {
		panic(saucerNumbersError)
	}
	value, err := saucerNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	noted(value)
	return value.Value
}

// The saucer's slots beyond body, its hull.
const (
	saucerDome     slot = leaf + spaceOne
	saucerBelly         = saucerDome + spaceOne
	saucerPorthole      = saucerBelly + spaceOne
)

// bothSides is a porthole's length, sunk as far into the slope as it
// stands proud of it.
const bothSides = 2

// saucer is the MCRN Larson, in flight or landed. It has one design, so
// seeds change nothing; landed, the same saucer has its gear down and
// its stairs out (saucer_landed.go).
type saucer struct {
	landed bool
}

// saucerRows supplies the one saucer there is, in flight and landed.
func saucerRows() map[string]maker {
	return map[string]maker{"mcrn-larson": saucer{},
		"mcrn-larson-landed": saucer{landed: true}}
}

// saucerSize is the saucer's proportions in metres, and its portholes.
//
// base is how high the flat of the underside stands above the saucer's
// own origin. In flight it is nought: the origin is the belly. Landed it
// is the height the gear holds the hull at, so the origin is the ground
// under the feet, which is where a scene stands a thing.
type saucerSize struct {
	radius, belly, bowl, band, rise float64
	dome, wall, seat, disc, drop    float64
	porthole, depth, up             float64
	levels, portholes               int
	base                            float64
}

// raised is a height above the flat of the underside, as a height above
// the saucer's origin.
func (size saucerSize) raised(above float64) float64 { return size.base + above }

// saucerShape turns the authored shares into metres.
func saucerShape() saucerSize {
	diameter := saucerValue("diameter")
	radius := diameter * spaceHalf
	return saucerSize{radius: radius,
		belly:     radius * saucerValue("belly"),
		bowl:      diameter * saucerValue("bowl"),
		band:      diameter * saucerValue("band"),
		rise:      diameter * saucerValue("rise"),
		dome:      radius * saucerValue("dome"),
		wall:      diameter * saucerValue("dome-wall"),
		seat:      diameter * saucerValue("dome-seat"),
		disc:      radius * saucerValue("disc"),
		drop:      diameter * saucerValue("disc-drop"),
		porthole:  diameter * saucerValue("porthole"),
		depth:     diameter * saucerValue("porthole-depth"),
		up:        saucerValue("porthole-up"),
		levels:    int(saucerValue("dome-levels")),
		portholes: int(saucerValue("portholes"))}
}

// deck is the height of the flat the dome stands on: the top of the
// upper slope.
func (size saucerSize) deck() float64 {
	return size.raised(size.bowl + size.band + size.rise)
}

// hull is the lens: out along the flat of the underside, up the bowl to
// the rim, up the rim's band, up the slope to the dome's foot, and in.
func (size saucerSize) hull() [][2]float64 {
	flat := size.raised(spaceZero)
	return [][2]float64{{spaceZero, flat}, {size.belly, flat},
		{size.radius, size.raised(size.bowl)},
		{size.radius, size.raised(size.bowl + size.band)},
		{size.dome, size.deck()}, {spaceZero, size.deck()}}
}

// glass is the dome, a hemisphere of glass with a wall: out along its
// foot, over its outside to the crown, down through the wall, and back
// down its inside to the foot.
//
// The foot is set a seat's depth into the deck, inside the hull, so a
// ray bent down through the glass meets the deck before the glass's
// floor. Standing on the deck, the two shared a plane and a ray met
// both at the same distance; which one it took was down to the mesh.
func (size saucerSize) glass() [][2]float64 {
	inner := size.dome - size.wall
	foot := size.deck() - size.seat
	points := [][2]float64{{inner, foot}, {size.dome, foot}}
	arc := func(radius float64, level int) [2]float64 {
		angle := math.Pi * spaceHalf * float64(level) / float64(size.levels)
		return [2]float64{radius * math.Cos(angle),
			size.deck() + radius*math.Sin(angle)}
	}
	for level := 1; level < size.levels; level++ {
		points = append(points, arc(size.dome, level))
	}
	points = append(points, [2]float64{spaceZero, size.deck() + size.dome},
		[2]float64{spaceZero, size.deck() + inner})
	for level := size.levels - 1; level > 0; level-- {
		points = append(points, arc(inner, level))
	}
	return points
}

// underneath is the disc set into the belly, hanging a little below it.
func (size saucerSize) underneath() [][2]float64 {
	low, flat := size.raised(-size.drop), size.raised(spaceZero)
	return [][2]float64{{spaceZero, low}, {size.disc, low},
		{size.disc, flat}, {spaceZero, flat}}
}

// ringPortholes rings the upper slope with short round windows, each square
// to the slope and standing proud of it.
func (size saucerSize) ringPortholes(assembly *instrumentAssembly) {
	across := size.radius + size.up*(size.dome-size.radius)
	height := size.raised(size.bowl + size.band + size.up*size.rise)
	// the slope's outward normal in a turned profile's radius and height
	normal := maths.V(size.rise, size.radius-size.dome, spaceZero).Unit()
	for each := range size.portholes {
		angle := fullTurn * float64(each) / float64(size.portholes)
		out := maths.V(normal.X*math.Cos(angle), normal.Y,
			-normal.X*math.Sin(angle))
		centre := maths.V(across*math.Cos(angle), height,
			-across*math.Sin(angle))
		assembly.rodAlong(centre.Sub(out.Scale(size.depth)),
			out.Scale(size.depth*bothSides), size.porthole, saucerPorthole)
	}
}

// extent is the saucer's box from its proportions, whatever its detail.
func (size saucerSize) extent() maths.Box {
	return maths.SegmentBox(
		maths.V(-size.radius, size.raised(-size.drop), -size.radius),
		maths.V(size.radius, size.deck()+size.dome, size.radius), spaceZero)
}

// made keeps quick as the saucer's default detail.
func (row saucer) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt builds the saucer at a detail: the hull, the dome, the belly
// disc and the portholes, and, landed, its legs and its stairs.
func (row saucer) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	assembly := &instrumentAssembly{fine: cut{
		around: int(saucerValue("sides-" + detail))},
		object: good{dressed: saucerLooks()}}
	size := saucerShape()
	if row.landed {
		// landed, the saucer's origin is the ground under its feet
		size.base = -landedGear().ground(size)
	}
	assembly.add(turnedProfile{points: size.hull()}, maths.Local(), body)
	assembly.add(turnedProfile{points: size.glass()}, maths.Local(),
		saucerDome)
	assembly.add(turnedProfile{points: size.underneath()}, maths.Local(),
		saucerBelly)
	size.ringPortholes(assembly)
	bounds := size.extent()
	if row.landed {
		bounds = landedGear().lower(assembly, size)
	}
	object, err := assembly.finish(bounds)
	if err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	return size.lit(*object)
}

// cabinName is the name a scene addresses the cabin light by.
const cabinName = "cabin"

// cabinLight is the soft light that fills the cabin under the dome: one
// red line, every way.
func cabinLight() (light, error) {
	colour, err := newLine(measure.Metres(saucerValue("cabin-wave")))
	if err != nil {
		return light{}, err
	}
	return newLight(colour, measure.Watts(saucerValue("cabin-power")),
		everyway{})
}

// saucerLooks is titanium for the hull (thing.Titanium) under the few
// nanometres of rutile it grows in air, satin and weathered, since a
// mirror in space shows only the dark it faces; clear glass for the
// dome; and greys for the rest.
//
// A clear surface shows its colour only past the tracer's refractions,
// so the dome's is dark: dark glass, not white paint.
func saucerLooks() looks {
	dark := maths.Grey(saucerValue("porthole-grey"))
	index := saucerValue("glass-index")
	// a porthole is a running light, so its glass glows, the colour
	// libtheme gives a body as hot as its lamp
	porthole := thing.Polished("saucer porthole", dark, spaceOne, index)
	porthole.Glow = swatches.Light(swatch.Planckian(
		saucerValue("rim-light-heat"))).Scale(saucerValue("porthole-glow"))
	return looks{
		body: &thing.Material{Name: "saucer hull",
			Colour: maths.Grey(saucerValue("weathered")), Gloss: spaceOne,
			Metal: thing.Titanium(),
			Film:  thing.Rutile(saucerValue("oxide")),
			Satin: saucerValue("satin")},
		saucerDome: &thing.Material{Name: "saucer dome", Colour: dark,
			Gloss: spaceOne, Index: index, Through: spaceOne},
		saucerBelly: thing.Diffuse("saucer belly disc",
			maths.Grey(saucerValue("structure"))),
		saucerPorthole: porthole,
	}
}
