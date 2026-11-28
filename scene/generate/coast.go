package generate

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/world"
)

// Cameras are generated from the system and not placed by hand
// (doc/system-generators-design.md, "Cameras"), so a change to the
// terrain moves them with the coast. This file is the coast's part: a
// camera low over the shore looking down, and one high over the sea
// looking along it, both found by walking the planet's own ground.

// Camera is where a camera stands and what it looks at, in metres from
// the planet's middle, in the set's space: the planet's tilt is applied.
type Camera struct {
	At, LookAt maths.Vec
}

// Cameras are the two coast cameras of a planet: Low is above the shore
// looking straight down at it, and High is over the sea, well back,
// looking at the shore across the water.
type Cameras struct {
	Low, High Camera
}

// ErrNoCoast is what FindCoast gives when the walk meets no shore: the
// planet has no sea, or no land, along the circle walked.
var ErrNoCoast = errors.New("generate: no coast along the walk")

// Shore is a place where land meets sea, as unit directions from the
// planet's middle in the planet's own frame, before its tilt: the point
// itself, the way to the sea from it, and the way the coast runs there.
type Shore struct {
	Point, Seaward, Along maths.Vec
}

// How the walk goes, and where the cameras stand. By eye (2026-11-26),
// from the hand placed cameras they replace: a high camera sixteen times
// as high as the low one, its height back over the sea, looking at the
// shore, so the coast crosses the middle of its frame.
const (
	// coastMix is the mix the ground is painted with, as fineMix.
	coastMix = fineMix
	// coastStep is a step of the walk as a share of a coarse spacing,
	// short enough not to step over a narrow strait.
	coastStep = 0.25
	// coastBisections halves the last step this many times: about six
	// metres on a planet the Earth's size.
	coastBisections = 24
	// halfLand is the land share at which a place counts as land.
	halfLand = 0.5
	// fullTurn is the walk's length, once round the planet.
	fullTurn = 2 * math.Pi
	// nearPole is how short a tangent is before the start is a pole.
	nearPole = 1e-9
	// lowPixels is how many pixels wide the low camera sees the ground
	// beneath it: its height over this is a pixel's width, the finest
	// the walk works the ground to. By eye (2026-11-26): an 800 pixel
	// frame across fifty degrees.
	lowPixels = 860
	// highRise is how many times the low camera's height the high one
	// stands at.
	highRise = 16
	// highBack is how far back over the sea it stands, in its heights,
	// looking down at the shore at half a right angle. It was half a
	// height back looking twice its height along the coast, which put
	// the shore found outside the frame (2026-11-26).
	highBack = 1
	// clearEnough is the most of the sky round a shore that may be
	// clouded for the walk to stop there; past it the walk goes on, and
	// takes the clearest shore it found. By eye (2026-11-26): a shore
	// under a fifth of cloud shows in both frames.
	clearEnough = 0.2
	// skyLift is how far up the cloud layer, as a share of it, the sky
	// is asked: inside it, where a patch is whole.
	skyLift = 0.01
	// openSea is the most land a pixel under the high camera may hold for
	// it to stand over open sea: a pixel half land was a coast, which
	// a finer eye saw as land (2026-11-26).
	openSea = 0.1
)

// skyReaches are how far from a shore, in the low camera's heights, the
// sky is asked, each way along the coast and across it: about the low
// frame's half width, and halfway along the high camera's way back.
var skyReaches = [...]float64{1, highRise * highBack / 2}

// FindCoast works out a planet's ground and clouds, walks a great circle
// over it from a direction, and gives the coast cameras where land first
// meets sea under a clear sky, or under the least cloud of any shore on
// the circle: Low at height metres over it, High sixteen times that. The
// circle goes through the planet's north and the start, so a start at a
// pole walks along the x axis' meridian. detail is how finely the ground
// is tiled, as the set says it. A planet with no ground, or whose circle
// meets no coast, is refused: ErrNoCoast for the circle.
func FindCoast(planet Planet, detail int, from maths.Vec,
	height float64) (Cameras, error) {
	derived, err := Derive(planet)
	if err != nil {
		return Cameras{}, err
	}
	body, err := Make(planet)
	if err != nil {
		return Cameras{}, err
	}
	ground, err := body.World(detail)
	if err != nil {
		return Cameras{}, err
	}
	radius := derived.Radius.InKilometres() * metresPerKilometre
	found, err := FindCoastOn(worldGround{made: ground, sky: skyOf(body)},
		radius, planet.Tilt, from, height)
	if errors.Is(err, ErrNoCoast) {
		return Cameras{}, fmt.Errorf("%w: %s", ErrNoCoast, planet.Name)
	}
	return found, err
}

// Ground is a planet's classification as a coast is found from it: how
// much of a place is land, and how coarse the ground's own scale is. It
// is all the finder knows of the ground, so a ground made another way
// serves as long as it answers these.
type Ground interface {
	// LandShare is how much of the ground under a pixel is land at a unit
	// direction from the middle, from nought to one. pixel is its width as
	// an angle at the middle in radians; nought is as fine as it goes.
	LandShare(p maths.Vec, pixel float64) float64
	// Spacing is the ground's coarsest scale as an angle, which the walk's
	// steps are a share of.
	Spacing() float64
}

// Sky is the cloud over a planet's ground as the finder asks it: how
// much of the sky over a place, a unit direction from the middle in the
// planet's own frame, is clouded, from nought to one. A Ground that is
// also a Sky has its coast found under clear sky where there is one; a
// Ground that is not is clear everywhere.
type Sky interface {
	Clouded(p maths.Vec) float64
}

// worldGround is a world.World as a Ground: the ground painted by, and
// the planet's clouds over it, none where it has none.
type worldGround struct {
	made *world.World
	sky  *medium.Clouds
}

// skyOf is a body's cloud layer as the renderer lays it, in kilometres,
// a patch's heart of one, so its density is how clouded a place is; none
// where the body has no clouds.
func skyOf(body bodies.Body) *medium.Clouds {
	clouds, has := body.Clouds()
	if !has {
		return nil
	}
	turning, soft, turn, hasTurn := clouds.WeatherChoice()
	toward, locked := body.Facing()
	return &medium.Clouds{Radius: body.Radius(), Low: clouds.Low(),
		High: clouds.High(), Cover: clouds.Cover(), Thick: 1,
		Seed: clouds.Seed(), Locked: locked && !turning, Toward: toward,
		Soft: soft, Turn: turn, HasTurn: hasTurn}
}

// Clouded is the clouds' patch over a place, low in the layer, where a
// patch is whole; none under no clouds.
func (ground worldGround) Clouded(p maths.Vec) float64 {
	if ground.sky == nil {
		return 0
	}
	sky := ground.sky
	lifted := sky.Radius + sky.Low + skyLift*(sky.High-sky.Low)
	return sky.Density(p.Unit().Scale(lifted))
}

// LandShare is the world's, at the mix the ground is painted with.
func (ground worldGround) LandShare(p maths.Vec, pixel float64) float64 {
	return ground.made.LandShare(p, pixel, coastMix)
}

// Spacing is the world's tiling's.
func (ground worldGround) Spacing() float64 {
	return ground.made.Tiling.Spacing()
}

// FindCoastOn is FindCoast over any Ground of a planet of radius metres
// whose axis leans by tilt: the cameras are in the set's space. height
// and from are refused if they are no height or no direction.
func FindCoastOn(ground Ground, radius float64, tilt measure.Angle,
	from maths.Vec, height float64) (Cameras, error) {
	if !(height > 0) || math.IsInf(height, 0) || from.Len() == 0 ||
		math.IsNaN(from.Len()) {
		return Cameras{}, fmt.Errorf("generate: a coast is found from a "+
			"direction and a height over nought, not %v and %v", from,
			height)
	}
	pixel := height / lowPixels / radius
	shore, found := newWalk(ground, from.Unit(), pixel).shore()
	if !found {
		return Cameras{}, ErrNoCoast
	}
	lean := maths.Rotation(maths.XAxis, tilt.InRadians())
	return shore.cameras(radius, height, lean), nil
}

// walk is a great circle over a ground, from a start toward the north.
type walk struct {
	ground        Ground
	from, tangent maths.Vec
	// pixel is the width, as an angle at the planet's middle, the ground
	// is worked at: a speck narrower than that is not a coast.
	pixel float64
	// sky is the cloud over the ground, when the ground has one
	sky Sky
}

// newWalk is the walk from a direction over a ground, worked down to a
// pixel's width as an angle, under the ground's sky if it is one.
func newWalk(ground Ground, from maths.Vec, pixel float64) walk {
	sky, _ := ground.(Sky)
	return walk{ground: ground, from: from, tangent: meridianTangent(from),
		pixel: pixel, sky: sky}
}

// landAt is whether the place an angle along the walk is land, by the
// ground's own reckoning.
func (along walk) landAt(angle float64) bool {
	here := onCircle(along.from, along.tangent, angle)
	return along.ground.LandShare(here, along.pixel) >= halfLand
}

// shore is the first shore along the walk fit for its cameras, or where
// none is, the least unfit: a coast camera under cloud shows no coast,
// and the high one stands over the sea. A shore is where the class
// changes and is held for two steps before and two after, so a lone
// islet or lake is passed and the coast is of a landmass and a sea.
func (along walk) shore() (Shore, bool) {
	step := coastStep * along.ground.Spacing()
	best, least, found := Shore{}, math.Inf(1), false
	for angle := step; angle < fullTurn+step; angle += step {
		if least <= clearEnough {
			break
		}
		before := along.landAt(angle - step)
		if along.landAt(angle) == before ||
			along.landAt(angle-2*step) != before ||
			along.landAt(angle+step) == before {
			continue
		}
		shore := along.between(angle-step, angle, before)
		if unfit := along.unfit(shore); unfit < least {
			best, least, found = shore, unfit, true
		}
	}
	return best, found
}

// unfit is how far a shore falls short of what its cameras need: the
// share of the sky round it that is clouded, and one more where the high
// camera, its height back over the sea, would not stand over open sea,
// as a pixel of its own, highRise times the low one's, sees it.
func (along walk) unfit(shore Shore) float64 {
	back := highBack * highRise * along.pixel * lowPixels
	seat := onCircle(shore.Point, shore.Seaward, back)
	short := along.clouded(shore)
	if along.ground.LandShare(seat, highRise*along.pixel) > openSea {
		short++
	}
	return short
}

// clouded is how much of the sky round a shore is clouded, on the whole:
// over the shore, and each way along the coast and across it as far as
// each of skyReaches. Under no sky it is none.
func (along walk) clouded(shore Shore) float64 {
	if along.sky == nil {
		return 0
	}
	reach := along.pixel * lowPixels
	ways := [...]maths.Vec{shore.Seaward, shore.Seaward.Neg(), shore.Along,
		shore.Along.Neg()}
	sum, count := along.sky.Clouded(shore.Point), 1
	for _, far := range skyReaches {
		for _, way := range ways {
			sum += along.sky.Clouded(onCircle(shore.Point, way, far*reach))
			count++
		}
	}
	return sum / float64(count)
}

// between narrows a shore to the step between two angles, of which the
// first is of the class before, by halving the step again and again.
func (along walk) between(near, far float64, before bool) Shore {
	for range coastBisections {
		middle := (near + far) / 2
		if along.landAt(middle) == before {
			near = middle
		} else {
			far = middle
		}
	}
	point := onCircle(along.from, along.tangent, near)
	heading := along.tangent.Scale(math.Cos(near)).Sub(
		along.from.Scale(math.Sin(near)))
	if !before {
		heading = heading.Neg()
	}
	return Shore{point, heading, point.Cross(heading).Unit()}
}

// onCircle is the point an angle along the great circle that starts at
// from and sets out toward tangent, which is a unit and square to from.
func onCircle(from, tangent maths.Vec, angle float64) maths.Vec {
	return from.Scale(math.Cos(angle)).Add(tangent.Scale(math.Sin(angle)))
}

// meridianTangent is the unit square to from that points to the north,
// so the circle through from and it is a meridian; at a pole, toward x.
func meridianTangent(from maths.Vec) maths.Vec {
	tangent := maths.Up.Sub(from.Scale(maths.Up.Dot(from)))
	if tangent.Len() < nearPole {
		tangent = maths.XAxis.Sub(from.Scale(maths.XAxis.Dot(from)))
	}
	return tangent.Unit()
}

// cameras are the two cameras at a shore on a planet of radius metres,
// the low one height over it and the high one highRise times as high and
// back over the sea, both looking at it, turned by lean into the set's
// space.
func (shore Shore) cameras(radius, height float64,
	lean maths.Orientation) Cameras {
	rise := height * highRise
	back := onCircle(shore.Point, shore.Seaward, highBack*rise/radius)
	ground := lean.Out(shore.Point.Scale(radius))
	return Cameras{
		Low: Camera{At: lean.Out(shore.Point.Scale(radius + height)),
			LookAt: ground},
		High: Camera{At: lean.Out(back.Scale(radius + rise)),
			LookAt: ground}}
}
