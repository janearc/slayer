// Package lantern builds a star inside a Dyson swarm. Inward-facing panels on
// tilted orbits let light escape in shafts, like a pierced tin lantern. A point
// emitter supplies the star's illumination.
//
// Dimensions are relative to the star and scaled by Radius when placed. The
// baseline arrangement places panels near six stellar radii, in bands 0.4 radii
// wide. lantern_test.go measures about 87% light interception for that
// arrangement; other settings change the coverage.
package lantern

import (
	"math"
	"time"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Lantern is one star and its swarm, at a place and a size.
//
// Kelvin is the star's surface, its light a black body there; Green is how
// bright that light is at the star's own radius, in the green, so a lantern a
// thousand times larger lights its swarm as the small one did.
//
// Turn is how long a ring takes to go once round its pole; zero holds the swarm
// still. Every ring turns at the one rate, since they are all the same distance
// out, each about its own axis.
type Lantern struct {
	At             maths.Vec
	Radius         float64
	Orbits, Panels int
	Kelvin, Green  float64
	Turn           time.Duration
	// Wide is how wide each band of panels is, in radii of the star; zero
	// is 0.4
	Wide float64
	// Wear is how hard a life the panels have had, zero for new and one for
	// battered: chewed edges, holes and dents
	Wear float64
	// Skin is what the panels are made of on the outside; nil is a warm
	// metal, half a mirror. Lining, when set, is what they are made of
	// on the side that faces the star, and each panel is a thin slab of
	// the two.
	Skin, Lining *thing.Material
	// Beams, when set, is a polar emitter at the star, its two beams
	// read at whatever time Clock holds: a thing.Lamp has no time of
	// its own, so whoever draws a frame sets the clock first. Glare is
	// how much brighter along a beam than the star's own light.
	Beams *emit.Polar
	Clock *time.Duration
	Glare float64
	// Shells is the rings set apart, each Spread radii of the star
	// further out than the one before; from the inside out they turn
	// faster, to Fastest for the outermost, and narrow, to Narrowest.
	// Noughts keep every ring alike.
	Shells
}

// the swarm, in radii of the star: the innermost ring's distance out,
// and the bands' width when the lantern does not say
const (
	shell = 6.0
	wide  = 0.4
)

// Shells is how the rings change from the innermost to the outermost.
//
// Tumble, when set, is how long the innermost ring takes to tumble once
// round the star, the band itself turning end over end about an axis
// across it, as the hoops of a human gyroscope do; further out it
// quickens as the turn does.
type Shells struct {
	Spread    float64
	Fastest   time.Duration
	Narrowest float64
	Tumble    time.Duration
}

// mirrored is what the panels are made of: a warm metal, half a mirror
var mirrored = &thing.Material{Name: "panels",
	Colour: maths.RGB(0.55, 0.45, 0.25), Gloss: 0.55, Index: 6}

// Place puts the star, its light and its rings of panels into a world,
// each ring a thing of its own turning about its own pole, and says how
// many panels there are.
func (lantern Lantern) Place(world *thing.World) (int, error) {
	radius := lantern.Radius
	here := thing.Still(maths.Subspace{Origin: lantern.At})
	glowing := &thing.Material{Name: "the star",
		Glow: maths.RGB(3.2, 2.9, 2.4)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: radius},
		Material: glowing, Motion: here})
	world.Lamp(lantern.light().When(0))
	if lantern.Beams != nil && lantern.Clock != nil {
		world.Lamp(lantern.beams().Beams(lantern.Clock))
	}
	for number := range lantern.Orbits {
		if err := lantern.placeRing(world, number); err != nil {
			return 0, err
		}
	}
	return lantern.Orbits * lantern.Panels, nil
}

// shelled is one ring's distance out, width and time to turn, in the
// star's radii, by where it is from the innermost to the outermost
func (lantern Lantern) shelled(number int) (float64, float64,
	time.Duration) {
	band := lantern.Wide
	if band <= 0 {
		band = wide
	}
	out, turn := shell+float64(number)*lantern.Spread, lantern.Turn
	if lantern.Orbits < 2 {
		return out, band, turn
	}
	share := float64(number) / float64(lantern.Orbits-1)
	if lantern.Narrowest > 0 {
		band += (lantern.Narrowest - band) * share
	}
	if lantern.Fastest > 0 && turn > 0 {
		ratio := float64(lantern.Fastest) / float64(turn)
		turn = time.Duration(float64(turn) * math.Pow(ratio, share))
	}
	return out, band, turn
}

// quickened is a period for a ring, quickening outward as the turn
// does, from the innermost ring's period
func (lantern Lantern) quickened(period time.Duration,
	number int) time.Duration {
	if period <= 0 || lantern.Orbits < 2 || lantern.Fastest <= 0 ||
		lantern.Turn <= 0 {
		return period
	}
	share := float64(number) / float64(lantern.Orbits-1)
	ratio := float64(lantern.Fastest) / float64(lantern.Turn)
	return time.Duration(float64(period) * math.Pow(ratio, share))
}

// placeRing puts one ring into the world: its panels, as one sheet or
// as a slab of skin outside and lining in, turning about its pole
func (lantern Lantern) placeRing(world *thing.World, number int) error {
	ring := orbitOf(number, lantern.Orbits)
	out, band, turn := lantern.shelled(number)
	radius := lantern.Radius
	turning := spin{at: lantern.At, pole: ring.pole, turn: turn,
		across: ring.tumbleAxis(number),
		tumble: lantern.quickened(lantern.Tumble, number)}
	skin := lantern.Skin
	if skin == nil {
		skin = mirrored
	}
	sheets := []struct {
		out      float64
		material *thing.Material
	}{{out, skin}}
	if lantern.Lining != nil {
		sheets = append(sheets, struct {
			out      float64
			material *thing.Material
		}{out - 0.02, lantern.Lining})
	}
	// further out and narrower, a ring has more panels, so they stay
	// about as square as the innermost ring's
	first, firstBand, _ := lantern.shelled(0)
	count := int(math.Round(float64(lantern.Panels) * out / first *
		firstBand / band))
	for _, sheet := range sheets {
		panels, err := ringMesh(ring, number, count,
			sheet.out*radius, band*radius, lantern.Wear)
		if err != nil {
			return err
		}
		world.Add(thing.Thing{Shape: panels, Material: sheet.material,
			Motion: turning})
	}
	return nil
}

// light is the star's light: a black body at Kelvin, scaled so its green
// at the star's radius is Green
func (lantern Lantern) light() emit.Point {
	spectrum := bands.Split(swatch.Blackbody(lantern.Kelvin))
	unit := emit.Point{Light: spectrum}.When(0).Strength.G
	radius := lantern.Radius
	scale := lantern.Green * radius * radius / unit
	return emit.Point{At: lantern.At, Light: spectrum.Scale(scale),
		Beyond: radius * 1.02}
}

// beams is the polar emitter at the star, in the star's own light made
// Glare times brighter along a pole
func (lantern Lantern) beams() emit.Polar {
	polar := *lantern.Beams
	polar.At, polar.Beyond = lantern.At, lantern.Radius*1.02
	polar.Light = lantern.light().Light.Scale(max(1, lantern.Glare))
	return polar
}

// spin is a ring turning about its pole through the lantern's middle, once a
// turn; a turn of zero is still
type spin struct {
	at, pole, across maths.Vec
	turn, tumble     time.Duration
}

// At is where the ring is at a time: its panels turned round it about
// its pole, and the whole band tumbled about the axis across it, each
// by the share of its own period that time is
func (turning spin) At(at time.Duration) maths.Subspace {
	round := maths.About(turning.pole, angleAt(at, turning.turn))
	over := maths.About(turning.across, angleAt(at, turning.tumble))
	return maths.Subspace{Origin: turning.at, Orientation: over.Then(round)}
}

// angleAt is how far round a period has gone at a time, in radians; a period of
// zero never turns
func angleAt(at, period time.Duration) float64 {
	if period <= 0 {
		return 0
	}
	return 2 * math.Pi * float64(at) / float64(period)
}

// BlackGlass is black glass at an index, like a solar panel's face: inky where
// it is seen square on and a mirror where it is seen edge on; nil at an index
// of zero, for the lantern's own warm metal
func BlackGlass(index float64) *thing.Material {
	if index <= 0 {
		return nil
	}
	return &thing.Material{Name: "black glass",
		Colour: maths.RGB(0.01, 0.018, 0.06), Gloss: 1, Index: index}
}

// Matte is matte black, for the side of black glass panels that faces
// the star, as a solar panel's back is not its front; nil with no glass
func Matte(index float64) *thing.Material {
	if index <= 0 {
		return nil
	}
	return &thing.Material{Name: "matte black",
		Colour: maths.RGB(0.02, 0.02, 0.025)}
}
