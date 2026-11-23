package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The matters a boat is made of. The slots are only keys of its looks.
const (
	boatPaint slot = iota
	boatUnder
	boatDeck
	boatHouse
	boatCanopy
	boatRubber
	boatSteel
	boatGlass
	boatCrate
	boatNet
	boatRust
	boatWindow
	boatRedLens
	boatGreenLens
	boatWhiteLens
	boatWarmLens
	boatBoot
	boatLetter
	boatBamboo
	boatLifebuoy
	boatBucket
	boatSeat
)

// The ranks of detail a part first appears at, by the place of the
// detail in Complexities: a silhouette has the hull and the house alone.
const (
	boatClumps = 1 // defined: the second of the complexities
	boatQuick  = 2 // defined: the third
	boatDemo   = 3 // defined: the fourth
)

// The places a boat's seed is drawn from, besides its ranges: each a
// stream of its own, so that adding a crate moves no window.
const (
	boatDrawPaint   = 900 // chosen: which paint the hull wears
	boatDrawCrates  = 0   // chosen: the first of a stream per crate
	boatDrawNets    = 100 // chosen: the first of a stream per net
	boatDrawWindows = 200 // chosen: the first of a stream per window
	boatDrawFloods  = 300 // chosen: the first of a stream per floodlight
	boatDrawGear    = 400 // chosen: the first of a stream per piece of deck gear
	boatDrawName    = 800 // chosen: the number and the name painted on the hull
)

// boatPaintGloss is how much of a mirror a hull's paint is, and
// boatWearShare how much of the way a worn paint has gone to rust at
// the most. Both chosen.
const (
	boatPaintGloss = 0.2
	boatWearShare  = 0.5
)

// boat is a boat being made: its row, its design's figures, its hull,
// the seed it grows from and the rank of its detail, the assembly it is
// put in, and where its lights stand, which its fittings decide.
type boat struct {
	row      boatRow
	design   boatDesign
	hull     boatHull
	from     seed
	rank     int
	assembly *instrumentAssembly
	mounts   boatMounts
	posted   map[[3]int64]bool
}

// craft is the boat a row makes from a seed at a rank of detail, before
// any of it is built.
func (row boatRow) craft(from seed, rank int) *boat {
	design := boatRead.Kinds[row.design]
	return &boat{row: row, design: design, from: from, rank: rank,
		hull: newBoatHull(design, from, rank)}
}

// detailed is whether the detail is at least a rank.
func (craft *boat) detailed(rank int) bool { return craft.rank >= rank }

// worn is the paint the seed picks, faded toward rust by its wear.
func (craft *boat) worn() *thing.Material {
	paints := craft.design.Paints
	which := int(craft.from.draw(housing, boatDrawPaint, 0) *
		float64(len(paints)))
	paint := paints[min(which, len(paints)-spaceOne)]
	rust := boatRead.Looks["rust"].Colour
	wear := craft.design.pick("wear", craft.from) * boatWearShare
	mixed := make([]float64, alfaChannels)
	for channel := range mixed {
		mixed[channel] = paint.Colour[channel] +
			(rust[channel]-paint.Colour[channel])*wear
	}
	return boatPlain(boatMixed("hull paint", mixed), boatPaintGloss)
}

// boatMixed is a paint of a name and a colour, which is not any of the
// data's: the one a seed has mixed.
func boatMixed(name string, colour []float64) boatColour {
	return boatColour{Name: name, Colour: colour, Basis: boatPicked}
}

// boatPlain is a paint as a look of a gloss.
func boatPlain(paint boatColour, gloss float64) *thing.Material {
	return &thing.Material{Name: paint.Name, Gloss: gloss,
		Colour: maths.RGB(paint.Colour[0], paint.Colour[1], paint.Colour[2])}
}

// dressed is what the boat wears: its worn paint, its design's underbody
// and deck, and the shared looks of its fittings and its lenses.
func (craft *boat) dressed() looks {
	canopy := "matting"
	if craft.design.chance("canopy-tarpaulin", craft.from) {
		canopy = "tarpaulin"
	}
	return looks{boatPaint: craft.worn(),
		boatUnder:     boatPlain(craft.design.Underbody, boatPaintGloss),
		boatDeck:      boatRead.material(craft.design.Deck),
		boatHouse:     boatRead.material("house"),
		boatCanopy:    boatRead.material(canopy),
		boatRubber:    boatRead.material("rubber"),
		boatSteel:     boatRead.material("steel"),
		boatGlass:     boatRead.material("glass"),
		boatCrate:     boatRead.material("crate"),
		boatNet:       boatRead.material("net"),
		boatRust:      boatRead.material("rust"),
		boatWindow:    boatRead.material("lit window"),
		boatRedLens:   boatRead.material("red lens"),
		boatGreenLens: boatRead.material("green lens"),
		boatWhiteLens: boatRead.material("white lens"),
		boatWarmLens:  boatRead.material("warm lens"),
		boatBoot:      boatRead.material("boot topping"),
		boatLetter:    boatRead.material("lettering"),
		boatBamboo:    boatRead.material("bamboo"),
		boatLifebuoy:  boatRead.material("lifebuoy"),
		boatBucket:    boatRead.material("bucket"),
		boatSeat:      boatRead.material("seat")}
}

// build puts every part of the boat in its assembly: the hull, the
// fittings of its design, and the housings and lenses of its lights.
func (craft *boat) build() {
	hull := craft.hull
	craft.assembly.add(hull.lofted(hull.lowerRing), maths.Local(), boatUnder)
	craft.assembly.add(hull.lofted(hull.upperRing), maths.Local(), boatPaint)
	craft.assembly.add(hull.lofted(hull.deckRing), maths.Local(), boatDeck)
	craft.hullWork()
	switch craft.row.design {
	case boatSampan:
		craft.sampan()
	case boatFishing:
		craft.fishing()
	default:
		craft.ferry()
	}
	craft.navigation()
}

// block is a box by its middle across and along, its foot's height, and
// its width, height and length.
func (craft *boat) block(across, up, along, wide, tall, long float64,
	material slot) {
	craft.assembly.box(wide, tall, long,
		maths.V(across-wide*spaceHalf, up, along-long*spaceHalf), material)
}

// rod is a rod between two places of the boat. It runs a little past each
// place, so that two rods that meet at one never have the same middle of the
// cap at their ends: their caps would share an edge, and a mesh whose faces
// run one edge one way is refused.
func (craft *boat) rod(from, to maths.Vec, radius float64, material slot) {
	run := to.Sub(from)
	if run.Len() > 0 {
		past := run.Unit().Scale(radius * boatRodPast)
		from, to = from.Sub(past), to.Add(past)
	}
	craft.assembly.rod(from, to, radius, material)
}

// boatRodPast is how far a rod runs past each end, as a share of its
// radius: chosen, enough to move the middle of its cap and not enough to be
// seen.
const boatRodPast = 0.5

// boatSide is a side of the boat for an index: one for starboard, minus
// one for port, each in turn.
func boatSide(index int) float64 {
	if index%2 == 0 {
		return spaceOne
	}
	return -spaceOne
}

// loaded lays crates or nets on the deck, as many as the design says
// and a stream each: between two places along the boat, not so far from
// the centreline that they would hang over the side, each size a little
// from its seed.
func (craft *boat) loaded(name string, first, last float64, stream int,
	size maths.Vec, material slot) {
	if !craft.detailed(boatQuick) {
		return
	}
	for index := range craft.design.count(name, craft.from) {
		along := first + (last-first)*craft.from.draw(housing, stream+index, 0)
		across := craft.hull.room(along-size.Z, along+size.Z) *
			boatLoadReach * craft.from.swing(housing, stream+index, 1)
		grown := spaceOne + boatLoadGrowth*craft.from.draw(housing,
			stream+index, 2)
		craft.block(across, craft.hull.deckAt(along)-boatDeckLift, along,
			size.X*grown, size.Y, size.Z*grown, material)
	}
}

// The reach of a load: the share of the room across the deck a crate
// may stand out from the centreline by, and how much bigger than its
// size one may grow. Chosen.
const (
	boatLoadReach  = 0.6
	boatLoadGrowth = 0.4
)
