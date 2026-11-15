//go:build draw && darwin

package gpu

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/occultation"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// the planet and ring the tests draw, in metres, as a giant's are: a ball of
// 60 thousand km, a ring from 80 to 140 thousand km across its equator, a
// slab 200 km thick as hillside draws one, of an optical depth of 0.8, an
// albedo of 0.8 and a lean of minus 0.3; seen from 350 thousand km through
// a lens 30 degrees wide.
const (
	ringBall      = 6e7
	ringInner     = 8e7
	ringOuter     = 1.4e8
	ringThickness = 1e6
	ringDepth     = 0.8
	ringAlbedo    = 0.8
	ringLean      = -0.3
	ringRange     = 3.5e8
	ringView      = 30 * maths.Degree
	ringWide      = 96
	ringHigh      = 64
)

// plainRing is a ring of an even depth, grey, with nothing in it.
func plainRing(t *testing.T) planet.Ring {
	t.Helper()
	ring, err := planet.NewRing(ringInner, ringOuter, ringDepth, ringAlbedo,
		ringLean)
	if err != nil {
		t.Fatal(err)
	}
	return ring
}

// ringBody is the ball the rings go about, with none yet.
func ringBody(t *testing.T) planet.Body {
	t.Helper()
	zone, err := planet.NewZone(-90, 90, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := planet.NewBody(ringBall, 25, 0.8, []planet.Zone{zone})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// ringEyeAt is an eye so many degrees over the ring's plane, in the plane's
// own space, ringRange from the ball's middle, on the z axis's side.
func ringEyeAt(degrees float64) maths.Vec {
	up := degrees * maths.Degree
	return maths.V(0, ringRange*math.Sin(up), ringRange*math.Cos(up))
}

// ringSeen is the ring as seen from an eye by rays one pixel wide, so its
// grains finer than a pixel are drawn as the mean of what it covers.
func ringSeen(t *testing.T, ring planet.Ring, eye maths.Vec) planet.Ring {
	t.Helper()
	body, err := ringBody(t).Ringed([]planet.Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	body, err = body.Seen(eye, float64(ringView)/ringWide)
	if err != nil {
		t.Fatal(err)
	}
	return body.Rings()[0]
}

// ringWorld is the ball with a ring slab for each ring, lit by a star far
// off toward a way.
func ringWorld(t *testing.T, toward maths.Vec,
	rings ...planet.Ring) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: ringBall},
		Material: &thing.Material{Name: "giant", Colour: maths.Grey(0.6)}})
	for _, ring := range rings {
		slab, err := planet.NewRingSlab(ring, ringThickness)
		if err != nil {
			t.Fatal(err)
		}
		world.AddVolume(thing.Volume{Medium: slab})
	}
	star, err := emit.NewDistant(toward, maths.RGB(3, 2.8, 2.5))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(star)
	return world
}

// the tolerance a ring is drawn to: the exact tolerance's mean, and a largest
// pixel and a few strays beyond it. A ring has hard edges, its rims and a
// speckle's seams, and a ray's sample that lies within float32's reach of
// one falls on the other side of it from where the tracer's float64 puts it,
// in about one pixel of two hundred on an edge. Such a pixel is wrong by one
// step of its forty-eight, which is two percent of the ring's light there
// and twice what the exact tolerance allows the largest pixel; measured on
// 2026-11-10 at no more than 2.4e-3 of the brightest and three pixels in
// 6144 whose cover differs.
var (
	ringLargest = 3e-3
	ringStray   = 4
)

// ringFindings lists what is wrong with a difference to a ring's tolerance:
// the exact mean, the ring's largest, and no more strays than it allows.
func ringFindings(found difference, mean float64) []string {
	strays := found.cover + found.depth
	found.cover, found.depth = 0, 0
	problems := findings(found, tolerance{mean: mean, largest: ringLargest})
	if strays > ringStray {
		problems = append(problems, fmt.Sprintf("%d pixels' cover or depth "+
			"differ, more than the %d a hard edge may", strays, ringStray))
	}
	return problems
}

// ringAgainstWith draws a world both ways from an eye so many degrees over
// the ring's plane, and fails when the mean difference is over a mean.
func ringAgainstWith(t *testing.T, world *thing.World, degrees,
	mean float64) {
	t.Helper()
	from := ringEyeAt(degrees)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	lens := film.Pinhole{Wide: ringView}
	drawn, err := Sampled(world, eye, lens, 0, ringWide, ringHigh, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, ringWide, ringHigh, 1)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, ringWide, ringHigh)
	t.Logf("apart: %+v", found)
	for _, problem := range ringFindings(found, mean) {
		t.Error(problem)
	}
}

// ringAgainst is ringAgainstWith at the exact tolerance's mean.
func ringAgainst(t *testing.T, world *thing.World, degrees float64) {
	t.Helper()
	ringAgainstWith(t, world, degrees, exact.mean)
}

// the star the tests light the ring by: from the side, a little above the
// plane and a little toward the eye.
var ringStar = maths.V(1, 0.3, 0.2)

// a plain ring seen from 30 degrees above its plane, lit from the side: its
// slab's gas draws as the tracer draws it, and its inner and outer edges
// fall where the tracer's do.
func TestTheGPUDrawsAPlainRingAsSlayerDoes(t *testing.T) {
	ringAgainst(t, ringWorld(t, ringStar, plainRing(t)), 30)
}

// the same ring seen nearly edge on, a degree over its plane: a ray crosses
// the slab for a long way and the march's steps are far apart.
func TestTheGPUDrawsAnEdgeOnRingAsSlayerDoes(t *testing.T) {
	ringAgainst(t, ringWorld(t, ringStar, plainRing(t)), 1)
}

// the same ring seen from below with the star above: the light that reaches
// the eye has come through the ring.
func TestTheGPUDrawsARingSeenFromBelowAsSlayerDoes(t *testing.T) {
	ringAgainst(t, ringWorld(t, ringStar, plainRing(t)), -30)
}

// ringleted is a ring with ringlets, of the strength and the coarsest width
// given.
func ringleted(t *testing.T, ring planet.Ring, strength,
	width float64) planet.Ring {
	t.Helper()
	made, err := ring.Ringleted(11, strength, width)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a ring with ringlets, five layers of seeded gradient noise by radius.
func TestTheGPUDrawsARingWithRingletsAsSlayerDoes(t *testing.T) {
	ring := ringleted(t, plainRing(t), 0.85, 3e6)
	ringAgainst(t, ringWorld(t, ringStar, ring), 30)
}

// profileTable is a table of an occultation's profile as the pds writes one:
// bins 100 km wide from 10 thousand km out, each a depth from zero to two
// by a hash, one in eight opaque.
func profileTable(bins int) string {
	var table strings.Builder
	for k := range bins {
		depth := 2 * noise.Hash(5, int64(k))
		if noise.Hash(6, int64(k)) < 0.125 {
			depth = 99
		}
		fmt.Fprintf(&table, "%9.2f, 10.000, 1.000,%7.4f, 0.0000, 1.0000\r\n",
			float64(k)*100+10050, depth)
	}
	return table.String()
}

// measuredRing is a ring laid over a stretch of that profile.
func measuredRing(t *testing.T, ring planet.Ring, from,
	to float64) planet.Ring {
	t.Helper()
	profile, err := occultation.Read(strings.NewReader(profileTable(256)))
	if err != nil {
		t.Fatal(err)
	}
	made, err := ring.Measured(profile, from, to)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a ring laid over a measured profile: the baked table, read at the width
// the ray sees, as the tracer reads it.
func TestTheGPUDrawsAMeasuredRingAsSlayerDoes(t *testing.T) {
	eye := ringEyeAt(30)
	ring := ringSeen(t, measuredRing(t, plainRing(t), 1.2e7, 3.4e7), eye)
	ringAgainst(t, ringWorld(t, ringStar, ring), 30)
}

// grained is a ring with grains in a pattern, clumps from 1.5 to 12 thousand
// km, so a pixel of the picture sees some of its layers and not others.
func grained(t *testing.T, ring planet.Ring, pattern string) planet.Ring {
	t.Helper()
	made, err := ring.Grained(pattern, 7, 1.5e6, 1.2e7, 3, 3, 0.7)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// speckleMean is the mean difference a speckle's grains are held to: its
// cells are counted round a circle of a whole number of them, which steps by
// one at radii where a float32's radius cannot tell which side of the step
// it is on, and there the cell is another whole hash. Measured on 2026-11-10
// at 3.2e-5, three times the exact tolerance's.
const speckleMean = 5e-5

// a ring with grains in each of the four patterns, seen by rays a pixel wide.
func TestTheGPUDrawsAGrainedRingAsSlayerDoes(t *testing.T) {
	eye := ringEyeAt(30)
	for _, pattern := range planet.Patterns() {
		t.Run(pattern, func(t *testing.T) {
			ring := ringSeen(t, grained(t, plainRing(t), pattern), eye)
			mean := exact.mean
			if pattern == "speckle" {
				mean = speckleMean
			}
			ringAgainstWith(t, ringWorld(t, ringStar, ring), 30, mean)
		})
	}
}

// moonleted is a ring with moonlets, some opening gaps, some propellers.
func moonleted(t *testing.T, ring planet.Ring) planet.Ring {
	t.Helper()
	body, err := ringBody(t).Ringed([]planet.Ring{ring})
	if err != nil {
		t.Fatal(err)
	}
	made, err := ring.Cleared(3, 12, 2e5, 2e6, 2.5, 1e6, body.Mass())
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a ring with moonlets, their gaps and propellers cleared from it.
func TestTheGPUDrawsARingWithMoonletsAsSlayerDoes(t *testing.T) {
	ringAgainst(t, ringWorld(t, ringStar, moonleted(t, plainRing(t))), 30)
}

// a ring and the ball in each other's shadow: the star above the plane at
// 15 degrees, the eye 25 below it, so the ring's shadow lies across the
// ball's southern face and the ball's across the ring on its far side.
func TestTheGPUDrawsARingAndABallInEachOthersShadowAsSlayerDoes(t *testing.T) {
	star := maths.V(1, 0.27, 0.05)
	ringAgainst(t, ringWorld(t, star, plainRing(t)), -25)
}

// a ring with all its terms at once, the way a giant's rings are.
func TestTheGPUDrawsARingWithEveryTermAsSlayerDoes(t *testing.T) {
	eye := ringEyeAt(30)
	ring := measuredRing(t, plainRing(t), 1.2e7, 3.4e7)
	ring = moonleted(t, grained(t, ring, "smooth"))
	ring = ringSeen(t, ring, eye)
	ringAgainst(t, ringWorld(t, ringStar, ring), 30)
}

// the x-rays the glow tests shine on the ring's phosphors: 1e22 watts from a
// band 5 degrees either side of the equator, which makes the glow about a
// sixtieth of the frame's light.
const (
	xrayPower = 1e22
	xrayBand  = 5
)

// glowing is a ring holding phosphors, a fifth zns:ag and a fifth csi:tl by
// mass, the rest ice.
func glowing(t *testing.T, ring planet.Ring) planet.Ring {
	t.Helper()
	made, err := ring.Glowing(map[string]float64{"zns-ag": 0.2, "csi-tl": 0.2})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// glowSlab is the ring's glow as a volume, with its shine.
func glowSlab(t *testing.T, ring planet.Ring) thing.Volume {
	t.Helper()
	xrays, err := planet.NewXrays(xrayPower, xrayBand)
	if err != nil {
		t.Fatal(err)
	}
	slab, shine, err := planet.NewRingGlow(ring, ringThickness, ringBall,
		xrays, 1)
	if err != nil {
		t.Fatal(err)
	}
	return thing.Volume{Medium: slab,
		Material: &thing.Material{Name: "glow", Shine: shine}}
}

// a ring's glow beside its slab, as hillside draws one: a slab a trillionth
// as dense, with a shine that reads the ring's share and the x-rays that
// fall on it, so its light is its share of the ring's.
func TestTheGPUDrawsARingsGlowAsSlayerDoes(t *testing.T) {
	ring := glowing(t, ringleted(t, plainRing(t), 0.85, 3e6))
	world := ringWorld(t, ringStar, ring)
	world.AddVolume(glowSlab(t, ring))
	ringAgainst(t, world, 30)
}
