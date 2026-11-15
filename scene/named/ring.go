package named

import (
	"io"
	"math"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// The ring: a band of triangles round a magnetar, mirror on its inner
// face and dark on its outer. The magnetar is sgr 1806-20's, placed at a
// thousandth of a unit a kilometre, so one unit is a thousand kilometres
// and the ring is 10,000 km across. Every number in the ring's files was
// picked for the picture when cmd/ring was written, and moved here
// unchanged.

// the ring, in thousands of kilometres: its radius, its width and
// thickness, the rows across each face, and the magnetar's scale, a
// thousandth of a unit a kilometre, and the detail its ground is tiled at
const (
	ringRadius    = 10.0
	ringWidth     = 4.0
	ringThick     = 0.2
	ringRows      = 4
	ringScale     = 1e-3
	magnetarTiles = 4
)

// the fewest steps round the ring that close it as a solid, and the steps
// cmd/ring took by default
const (
	ringFewest = 8
	ringAround = 1024
)

// the ring's sky: its darkness, and the seed of its stars
const (
	ringSkyRed, ringSkyGreen, ringSkyBlue = 0.004, 0.004, 0.008
	ringSkySeed                           = 1806
)

// the ring's mirror: its colour, gloss and index, and how much it glows
// in the infrared, warm from the flare
const (
	mirrorRed, mirrorGreen, mirrorBlue = 0.06, 0.06, 0.08
	mirrorGloss, mirrorIndex           = 1, 8
	mirrorWarmth                       = 0.3
)

// the ring as cmd/ring drew it: the eye, outside and above, looking in
// across the near rim; the lens's width in degrees; the moment, after
// the flare leaves at a second; the picture's size and rays a side; and
// the tone's knee and top
const (
	ringEyeY, ringEyeZ                  = 9, 26
	ringLookY                           = -1
	ringWideDegrees                     = 55
	ringAt                              = 1200 * time.Millisecond
	ringPictureWidth, ringPictureHeight = 960, 540
	ringRays                            = 2
	ringKnee, ringTop                   = 0.7, 8
)

// matteLamp is how many times stronger a sun is made than the strength
// the ring names, for a matte surface to show its colour times that
// strength. trace reads a lamp's strength as irradiance, so a matte
// surface shows its colour over pi times it; the ring's strengths are
// written for colour times strength.
//
// A flood is not multiplied: trace shows its colour times its strength.
const matteLamp = math.Pi

// ringScene is the mirror ring round a flaring magnetar.
func ringScene() Scene {
	return Scene{name: "ring",
		about: "a ring of triangles round a flaring magnetar, mirror " +
			"on its inner face and dark on its outer",
		unit:       "a thousand kilometres",
		renderers:  []Renderer{CPU},
		parameters: ringParameters(),
		defaults: Defaults{width: ringPictureWidth,
			height: ringPictureHeight, rays: ringRays,
			wide: ringWideDegrees * maths.Degree, at: ringAt,
			tone: colour.Tone{Knee: ringKnee, Top: ringTop}},
		build: buildRing}
}

// ringParameters are what the ring takes, as cmd/ring's flags were. Its
// false-colour flags make a second picture, not a second scene, and stay
// in the command.
func ringParameters() []Parameter {
	return []Parameter{
		whole("around", ringAround, atLeast(ringFewest), "steps round "+
			"the ring; fewer shows the triangles in the mirror"),
		toggle("sun", true,
			"the small far sun that gives the hull a shape"),
		toggle("dress", false, "dress the inner face in hexes, in the "+
			"earth's paints, and light it"),
		toggle("night", false, "with dress, put the ring's sun out: "+
			"only what glows shows, the cities and the lava"),
	}
}

// buildRing is the magnetar at the middle, the ring round it with its
// axis up, the stars, and the eye outside and above, looking in across
// the near rim at the mirror on the far side.
func buildRing(held values, said io.Writer) (*Built, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(ringSkyRed, ringSkyGreen,
		ringSkyBlue), Seed: ringSkySeed}
	magnetar, err := template.Magnetar()
	if err != nil {
		return nil, err
	}
	err = magnetar.Place(world, maths.V(0, 0, 0), ringScale, magnetarTiles)
	if err != nil {
		return nil, err
	}
	dress := held.toggle("dress")
	if dress {
		if err := dressRing(world, said); err != nil {
			return nil, err
		}
	}
	skin, err := ringSolid(held.whole("around"),
		func(strip int) bool { return !dress || strip != 0 })
	if err != nil {
		return nil, err
	}
	world.Add(thing.Thing{Shape: skin, Material: ringMirror()})
	ringLamps(world, held.toggle("sun"), dress, held.toggle("night"))
	from := maths.V(0, ringEyeY, ringEyeZ)
	return &Built{world: world, eye: maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, ringLookY, 0))}}, nil
}

// ringMirror is one material for the whole solid: a mirror, so the inner
// face throws the flare round, and the outer one shows it too, faintly.
//
// warm from the flare: a hull beside a giant flare heats, and a warm
// thing glows in the infrared. set by hand, not worked out from a
// temperature, since nothing in slayer does that yet
func ringMirror() *thing.Material {
	return &thing.Material{Name: "ring",
		Colour: maths.RGB(mirrorRed, mirrorGreen, mirrorBlue),
		Gloss:  mirrorGloss, Index: mirrorIndex,
		Radiant: &thing.Spectrum{Gives: map[radiation.Range]float64{
			radiation.Infrared(): mirrorWarmth}}}
}

// the ring's lamps: the far sun bare, the dressed ring's own sun with
// its fill, and the night's whisper of starlight, each a way to the sun
// and a light
const (
	bareX, bareY, bareZ                = -0.4, 1, 0.3
	bareRed, bareGreen, bareBlue       = 0.35, 0.33, 0.3
	dressedX, dressedY, dressedZ       = 0.25, 0.45, 1
	dressedRed, dressedGreen           = 1.1, 1.05
	dressedBlue                        = 0.95
	fillRed, fillGreen, fillBlue       = 0.08, 0.09, 0.12
	starlitRed, starlitGreen, starBlue = 0.01, 0.012, 0.02
)

// ringLamps lights the ring: a far, small sun, so the hull has a shape
// where nothing mirrors; dressed, the ring's own sun, low and behind the
// eye, so the far inner face is lit, and a dim dome of sky to fill the
// shadows; at night, a whisper of starlight, so the coasts read by their
// lights.
func ringLamps(world *thing.World, sun, dress, night bool) {
	if dress && night {
		world.Lamp(film.Flood{Strength: maths.RGB(starlitRed,
			starlitGreen, starBlue)})
		return
	}
	if dress {
		world.Lamp(matteSun(maths.V(dressedX, dressedY, dressedZ),
			maths.RGB(dressedRed, dressedGreen, dressedBlue)))
		world.Lamp(film.Flood{Strength: maths.RGB(fillRed, fillGreen,
			fillBlue)})
		return
	}
	if sun {
		world.Lamp(matteSun(maths.V(bareX, bareY, bareZ),
			maths.RGB(bareRed, bareGreen, bareBlue)))
	}
}

// matteSun is a sun from toward whose light is named, made matte lamp
// times stronger.
func matteSun(toward maths.Vec, light maths.Light) thing.Lamp {
	return thing.Sun(toward, light.Scale(matteLamp))
}
