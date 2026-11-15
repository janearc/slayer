package lantern

import (
	_ "embed"
	"fmt"
	"math/big"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Zappy is the lantern grown up, by its own name: a star of 9,000
// km in nested shells of blue-black glass panels that tumble, with two
// beams from its poles that spin every second and a half and precess
// every five. It was designed for the orbital's far view and lived there
// as constants.
//
// It is defined here, once, beside the lantern it grew from, as an
// arrangement of a Lantern with beams. Its values are data, in
// zappy.yaml, each with who picked it.

//go:embed zappy.yaml
var zappyRaw []byte

// Beamed is a lantern with two beams from its star's poles: zappy, or
// one like it. Lengths are in the unit its world is in, angles in
// degrees.
type Beamed struct {
	// Radius is the star's radius; Kelvin its surface and Green how
	// bright its light is at its own radius, in the green.
	Radius, Kelvin, Green float64
	// Orbits is how many rings of panels, Panels how many to a ring.
	Orbits, Panels int
	// Turn is how long the innermost ring takes to go round its pole;
	// zero holds the swarm still. Fastest is the outermost ring's turn and
	// Tumble the innermost ring's tumble end over end.
	Turn, Fastest, Tumble time.Duration
	// Wide is each ring's width and Narrowest the outermost's, Spread
	// how far each ring is out from the one inside it, all in radii of
	// the star. Wear is how battered the panels are, nought to one, and
	// Glass the index of their black glass, both sides.
	Wide, Narrowest, Spread, Wear, Glass float64
	// Glare is how much brighter along a beam than the star's own light.
	Glare float64
	// Axis is the beams' mean spin axis.
	Axis maths.Vec
	// Beams are the beams' shape and motion.
	Beams Beams
}

// Beams are a beamed lantern's two beams: how far off its pole a beam is
// at full strength (Width) and how much further it fades (Soft); how far
// the second pole is turned off the first's opposite (Offset); how far
// the poles sit off the spin axis (Tilt) and how long they take round it
// (Spin); how far the spin axis leans off the mean axis (Cone) and how
// long it takes round it (Precess). Angles are in degrees.
type Beams struct {
	Width, Soft, Offset, Tilt, Cone float64
	Spin, Precess                   time.Duration
}

// Zappy is zappy as defined in zappy.yaml, a copy the caller may change.
// The file is part of the package, so a file that will not read is a
// fault in the package, and stops it.
func Zappy() Beamed {
	read, err := numbers.Read("zappy.yaml", zappyRaw)
	if err != nil {
		panic(err)
	}
	value := func(name string) float64 { return zappyValue(read, name) }
	seconds := func(name string) time.Duration {
		return time.Duration(value(name) * float64(time.Second))
	}
	return Beamed{Radius: value("radius"), Kelvin: value("kelvin"),
		Green: value("green"), Orbits: int(value("orbits")),
		Panels: int(value("panels")), Turn: seconds("turn"),
		Fastest: seconds("fastest"), Tumble: seconds("tumble"),
		Wide: value("band-width"), Narrowest: value("narrowest"),
		Spread: value("spread"), Wear: value("wear"),
		Glass: value("glass"), Glare: value("glare"),
		Axis: maths.V(value("axis-x"), value("axis-y"), value("axis-z")),
		Beams: Beams{Width: value("beam-width"), Soft: value("beam-soft"),
			Offset: value("beam-offset"), Tilt: value("beam-tilt"),
			Cone: value("beam-cone"), Spin: seconds("beam-spin"),
			Precess: seconds("beam-precess")}}
}

// zappyValue is the value of the number of that name; a name the file
// does not hold is a fault in the package.
func zappyValue(read numbers.Numbers, name string) float64 {
	number, err := read.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}

// Lantern is the lantern that draws the beamed one: its star and glass
// swarm, and its beams read at whatever time clock holds. A nil clock
// is refused, since beams with no time cannot be drawn.
func (beamed Beamed) Lantern(clock *time.Duration) (Lantern, error) {
	if clock == nil {
		return Lantern{}, fmt.Errorf("lantern: a beamed lantern needs " +
			"a clock for its beams; give it the time each frame is drawn")
	}
	glass := BlackGlass(beamed.Glass)
	beams := &emit.Polar{Width: radians(beamed.Beams.Width),
		Soft:   radians(beamed.Beams.Soft),
		Offset: radians(beamed.Beams.Offset),
		Tilt:   radians(beamed.Beams.Tilt), Spin: beamed.Beams.Spin,
		Axis: beamed.Axis.Unit(), Cone: radians(beamed.Beams.Cone),
		Precess: beamed.Beams.Precess}
	return Lantern{Radius: beamed.Radius, Orbits: beamed.Orbits,
		Panels: beamed.Panels, Kelvin: beamed.Kelvin, Green: beamed.Green,
		Turn: beamed.Turn, Wide: beamed.Wide, Wear: beamed.Wear,
		Skin: glass, Lining: glass,
		Shells: Shells{Spread: beamed.Spread, Fastest: beamed.Fastest,
			Narrowest: beamed.Narrowest, Tumble: beamed.Tumble},
		Beams: beams, Clock: clock, Glare: beamed.Glare}, nil
}

// Place puts the beamed lantern into a world at the middle, its beams
// read at whatever time clock holds, and says how many panels it has.
func (beamed Beamed) Place(world *thing.World,
	clock *time.Duration) (int, error) {
	made, err := beamed.Lantern(clock)
	if err != nil {
		return 0, err
	}
	return made.Place(world)
}

// piDigits is pi as Go's math package writes it, defined, so that an
// angle made from degrees here is the one Go makes of the same degrees
// written as a constant.
const piDigits = "3.14159265358979323846264338327950" +
	"288419716939937510582097494459"

// halfTurnDegrees is the degrees in a half turn, and precision the bits
// the conversion carries before it rounds once to a float64: defined, as
// more than Go's own constant arithmetic carries.
const (
	halfTurnDegrees = 180
	precision       = 512
)

// radians is an angle in degrees as radians, rounded once, exactly as Go
// rounds degrees times maths.Degree written as a constant. Multiplying at
// run time rounds twice, and 15 degrees then lands a bit away.
func radians(degrees float64) float64 {
	pi, _, err := big.ParseFloat(piDigits, 10, precision, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	angle := new(big.Float).SetPrec(precision).SetFloat64(degrees)
	angle.Mul(angle, pi)
	angle.Quo(angle, new(big.Float).SetPrec(precision).
		SetInt64(halfTurnDegrees))
	rounded, _ := angle.Float64()
	return rounded
}
