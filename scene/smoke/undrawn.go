package smoke

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/materials"
	"github.com/janearc/slayer/maths"
	ninja "github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/render/shortcut"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// The numbers every undrawn smoke leans on: the numbers of arithmetic.
const (
	nought   = 0   // defined: no amount
	unity    = 1   // defined: one
	pair     = 2   // defined: two, as in half and twice
	fullTurn = 360 // defined: degrees in a full turn
)

// unscaled is the scale of a frame that is not scaled: zero is read as one.
const unscaled = nought // defined: a subspace's zero scale means one

// middle is the middle of the world, which most of the smokes look at.
var middle = maths.V(nought, nought, nought) // defined: the origin

// colourless is the colour of a material that has none of its own: a
// volume's is read from its medium.
var colourless = maths.Grey(nought) // defined: no colour is no light

// stillAt is the motion of a thing that stands at a place, unturned and
// not scaled.
func stillAt(at maths.Vec) thing.Motion {
	space := maths.NewSubspace(at, maths.Identity, unscaled)
	return thing.Still(space)
}

// eyeToward is an eye at a place looking at another.
func eyeToward(from, at maths.Vec) maths.Subspace {
	return maths.NewSubspace(from, maths.LookAt(from, at), unscaled)
}

// painted is a material of a name and a colour, with no gloss and no
// refraction.
func painted(name string, colour maths.Light) *thing.Material {
	return thing.Polished(name, colour, nought, nought)
}

// slateFloor is a thing that is the floor: a plane of slate, with no
// motion of its own.
func slateFloor() thing.Thing {
	return thing.Thing{
		Shape:    shape.Plane{},
		Material: painted("slate", swatches.Light(swatches.Slate)),
	}
}

// volumeAt is a medium with a material, standing at a place.
func volumeAt(field medium.Medium, material *thing.Material,
	at maths.Vec) thing.Volume {
	volume := thing.NewVolume(field, material)
	volume.Motion = stillAt(at)
	return volume
}

// thingAt is a shape with a material, standing at a place.
func thingAt(of shape.Shape, material *thing.Material,
	at maths.Vec) thing.Thing {
	return thing.Thing{Shape: of, Material: material, Motion: stillAt(at)}
}

// daylight is the daylight swatch, a share of its strength.
func daylight(share float64) maths.Light {
	return swatches.Light(swatches.Daylight).Scale(share)
}

// airBall is a ball of a gas as a medium: a puff of a radius, thick in the
// middle and thinning to nothing at its edge, whose unit is metres metres.
func airBall(radius, thick float64, air gas.Gas, metres float64) gas.Of {
	return gas.Of{
		Shape: medium.Puff{Radius: radius, Thick: thick},
		In:    air,
		Unit:  metres,
	}
}

// fortyBalls is how far apart smoke forty's four balls of air stand, in
// units, and fortyRadius how big each is.
const (
	fortyApart  = 2.6 // picked: four balls fill the frame at fifty degrees from eleven units
	fortyRadius = 1.0 // picked: a ball a unit in radius, as smoke one's
)

// fortyMetres is how many metres a unit of any of the four balls is, the
// same for all: the gases are compared at one size, so the thicker the
// gas the brighter and whiter its ball.
const fortyMetres = 5e4 // picked: earth's air a little under one optical depth through the middle, at green

// fortyThick is the density at the middle of each ball, times the gas's
// own extinction per metre and fortyMetres to give its optical depth.
const fortyThick = 1.0 // defined: the gas's numbers are per metre at unit density

// fortyStrength is how strong smoke forty's sun is. A gas is lit by the
// lamp's light as it is, with no division by pi, so the strength is the
// light the gas scatters from, before its scattering per metre.
const fortyStrength = 24.0 // picked: puts the lit side of the thinnest ball well above the stars

// fortyAirs are the gases of smoke forty, left to right: the earth's air,
// a continental haze, venus and mars.
func fortyAirs() []gas.Gas {
	return []gas.Gas{gas.Air(), gas.Haze(), gas.Venus(), gas.Mars()}
}

// fortyBall is a ball of the gas as a medium: a puff, thick in the middle
// and thinning to nothing at its edge, whose unit is fortyMetres metres.
func fortyBall(air gas.Gas) gas.Of {
	return airBall(fortyRadius, fortyThick, air, fortyMetres)
}

// fortyFirst is the middle of the first of the four balls, in a row along
// x: the row is centred, so the first of four is one and a half gaps left.
const fortyFirst = -1.5 // defined: a centred row of four starts one and a half gaps left

// fortyAt is the middle of the ball at index i of four, in a row along x.
func fortyAt(i int) maths.Vec {
	return maths.V((fortyFirst+float64(i))*fortyApart, nought, nought)
}

// fortySunToward and fortyEye are where smoke forty's sun is toward and
// where its eye stands.
var (
	fortySunToward = maths.V(-1, 0.3, -0.6) // picked: from the left and a little behind, so each ball's lit side shows
	fortyEye       = maths.V(0, 0, 11)      // picked: eleven units from the row, which fills the frame at fifty degrees
)

// forty is the fortieth milestone: four atmospheres. a ball each of the
// earth's air, a haze, venus's air and mars's, side by side in the dark,
// each the same thickness at green, lit by a sun from the left and a
// little behind.
//
// nothing in the picture is painted: each ball's colour is what its gas
// scatters and absorbs at each band. the air scatters the short end and
// is blue; the haze scatters all bands alike and is white; venus is
// yellow, its blue taken by an absorber; mars is red, its dust scattering
// flat and absorbing the blue.
func forty() (*thing.World, maths.Subspace) {
	stage := dark()
	for i, air := range fortyAirs() {
		material := painted(air.Name, colourless)
		ball, at := fortyBall(air), fortyAt(i)
		stage.AddVolume(volumeAt(ball, material, at))
	}
	sun := daylight(fortyStrength)
	stage.Lamp(thing.Sun(fortySunToward, sun))
	return stage, eyeToward(fortyEye, middle)
}

// fortyOne's numbers: a laser a quarter of a unit wide, a haze eleven
// units across, and the metres a unit of the haze is.
const (
	fortyOneWidth  = 0.25 // picked: wide enough to be more than a pixel's width at the shot's distance
	fortyOneHaze   = 5.0  // picked: the haze ball is radius five, with the beam and the floor's spot inside it
	fortyOneMetres = 6e3  // picked: the haze is about two optical depths through its middle
)

// fortyOneStrength is the laser's strength: red, with a little green so a
// white floor shows it as a red spot and not a pink one. It is a lamp's
// irradiance, so a matte floor shows its colour over pi times it.
var fortyOneStrength = maths.RGB(150, 5, 3) // picked: a floor spot near white-hot and a beam that shows

// fortyOneAt and fortyOneAim are where the laser stands and which way it
// points, and fortyOneHazeAt where the middle of the haze is.
var (
	fortyOneAt     = maths.V(-4, 3, 0)    // picked: upper left of the frame, out of the haze's middle
	fortyOneAim    = maths.V(1, -0.75, 0) // picked: down and to the right, to the floor inside the haze
	fortyOneHazeAt = maths.V(0, 1.5, 0)   // picked: the haze ball sits on the floor with its middle above it
)

// fortyOne is the forty-first milestone: a laser in haze. a red beam
// from the upper left down to the floor, through a ball of haze, a spot
// where it lands.
//
// the beam is a lamp, a thing.Lamp whose light is only inside its width,
// so on the floor it is a spot. in the clear it is invisible; in a
// scattering gas the tracer gathers its light at every step and the beam
// shows along its length. there is no other light: the floor is lit by
// the spot alone, and the haze by the beam alone.
func fortyOne() (*thing.World, maths.Subspace) {
	stage := set(false)
	ball := airBall(fortyOneHaze, fortyDepthOne, gas.Haze(), fortyOneMetres)
	haze := painted("haze", colourless)
	stage.AddVolume(volumeAt(ball, haze, fortyOneHazeAt))
	stage.Lamp(film.Laser{At: fortyOneAt, Aim: fortyOneAim,
		Width: fortyOneWidth, Strength: fortyOneStrength})
	return stage, setEye()
}

// fortyDepthOne is the haze's density at its middle, times the haze's own
// extinction per metre and fortyOneMetres its optical depth per unit.
const fortyDepthOne = 1.0 // defined: the gas's numbers are per metre at unit density

// fortyTwo's hill, in units: the square it stands on, the samples across
// it, how high it is at its middle, and how much a ripple of it rises.
const (
	fortyTwoExtent = 2.6  // picked: a hill that fits left of the middle of a frame fifty degrees wide
	fortyTwoSample = 41   // picked: forty cells, each under a tenth of a unit, so the ripple is resolved
	fortyTwoHeight = 0.9  // picked: lower than the gem beside it, so the two are not one outline
	fortyTwoRipple = 0.06 // picked: a bump a fifteenth of a unit, which a ball's smooth side would not have
	fortyTwoWaves  = 7.0  // picked: radians a unit, about a bump to every two-fifths of a unit
)

// fortyTwoHeights is the hill's heights, row by row: a bell that falls to
// zero at a round edge half the extent out, so the hill meets the floor,
// ridged by the ripple. Its middle is the hill's height exactly, as the
// ripple is zero there.
func fortyTwoHeights() []float64 {
	reach := fortyTwoExtent / pair
	step := fortyTwoExtent / float64(fortyTwoSample-unity)
	heights := make([]float64, nought, fortyTwoSample*fortyTwoSample)
	for row := range fortyTwoSample {
		for column := range fortyTwoSample {
			x := float64(column)*step - reach
			z := float64(row)*step - reach
			heights = append(heights, fortyTwoBell(x, z, reach))
		}
	}
	return heights
}

// fortyTwoBell is the hill's height at x and z, for a hill that ends
// reach out from its middle: the square of what is left of one minus the
// square of the share of reach, so it is flat where it meets the floor.
func fortyTwoBell(x, z, reach float64) float64 {
	left := unity - (x*x+z*z)/(reach*reach)
	if left <= nought {
		return nought
	}
	ripple := fortyTwoRipple * math.Sin(fortyTwoWaves*x) * math.Sin(fortyTwoWaves*z)
	return left * left * (fortyTwoHeight + ripple)
}

// fortyTwoApart is how far left and right of the middle the hill and the
// gem stand, and fortyTwoSunk how far the hill's square is set below the
// floor, so that the flat edge of the field is under the floor and only
// the hill shows above it.
const (
	fortyTwoApart = 1.3  // picked: the two clear of each other and of the frame's edges
	fortyTwoSunk  = 0.02 // picked: more than the ripple at the edge, which is zero, and less than a pixel at this distance
)

// fortyTwoGem's numbers: how many sides it has, how wide its girdle is
// and how far its two points stand above and below it.
const (
	fortyTwoSides  = 7    // picked: odd, so no face stands square to the eye
	fortyTwoGirdle = 0.9  // picked: a unit's width, near the hill's
	fortyTwoPoint  = 1.15 // picked: taller than the hill, a closed solid that is plainly not a ball
)

// faceCorners is how many corners a face of a mesh has.
const faceCorners = 3 // defined: a face is a triangle

// face is the three corners of a triangle of a mesh, by their index.
type face = [faceCorners]int

// fortyTwoRing is the girdle of the gem: sides corners in a ring about the
// middle, in a plane level with it.
func fortyTwoRing() []maths.Vec {
	corners := make([]maths.Vec, nought, fortyTwoSides+pair)
	for i := range fortyTwoSides {
		turn := pair * math.Pi * float64(i) / fortyTwoSides
		corner := maths.V(fortyTwoGirdle*math.Cos(turn), nought,
			-fortyTwoGirdle*math.Sin(turn))
		corners = append(corners, corner)
	}
	return corners
}

// fortyTwoFaces are the faces of the gem: every side a triangle to the
// top point and one to the bottom point, wound outward.
func fortyTwoFaces(top, bottom int) []face {
	faces := make([]face, nought, pair*fortyTwoSides)
	for i := range fortyTwoSides {
		next := (i + unity) % fortyTwoSides
		faces = append(faces, face{i, next, top}, face{i, bottom, next})
	}
	return faces
}

// fortyTwoGem is a closed solid of flat faces: a ring of sides corners
// about the middle and a point above and one below, every side a triangle
// to each point. Each face winds outward, so the solid is closed and has
// an inside.
func fortyTwoGem() (*shape.Mesh, error) {
	corners := fortyTwoRing()
	top, bottom := len(corners), len(corners)+unity
	corners = append(corners, maths.V(nought, fortyTwoPoint, nought),
		maths.V(nought, -fortyTwoPoint, nought))
	gem, err := shape.NewMesh(corners, fortyTwoFaces(top, bottom), nil, false)
	if err != nil {
		return nil, err
	}
	if !gem.Closed() || !gem.Inside(middle) {
		return nil, fmt.Errorf("smoke: the gem of smoke 42 is not a closed " +
			"solid with its middle inside it")
	}
	return gem, nil
}

// matteSunToward and matteSunShare are the sun of the smokes that light a
// matte set: where it is toward, and its share of daylight.
var matteSunToward = maths.V(1, 2, 1) // picked: from the upper right and the front, as smoke three's

// matteSunShare is the share of daylight a matte set's sun is as strong as.
const matteSunShare = 1.1 // picked: a little over daylight, so a matte surface is not dimmed by its colour

// matteDaySun is the sun of the matte sets, a thing.Lamp.
func matteDaySun() thing.Lamp {
	return matteSun(matteSunToward, daylight(matteSunShare))
}

// fortyTwo is the forty-second milestone: a hill beside a closed solid.
// a height field, a bell of ground with a ripple across it, beside a mesh,
// a gem of flat faces that sits on the floor on its lower point, in smoke
// three's set and light.
//
// the hill is the ground as a surface given by heights on a grid; its
// ripple is why it cannot be a ball. the gem is the other way to a solid
// shape: triangles, closed, so that a point can be inside it.
func fortyTwo() (*thing.World, maths.Subspace) {
	stage := set(true)
	heights := fortyTwoHeights()
	hill := sure(shape.NewHeightField(heights, fortyTwoSample, fortyTwoExtent))
	moss := painted("hill", swatches.Light(swatches.Moss))
	hillAt := maths.V(-fortyTwoApart, -fortyTwoSunk, nought)
	stage.Add(thingAt(hill, moss, hillAt))
	gem := sure(fortyTwoGem())
	sea := painted("gem", swatches.Light(swatches.Sea))
	gemAt := maths.V(fortyTwoApart, fortyTwoPoint, nought)
	stage.Add(thingAt(gem, sea, gemAt))
	stage.Lamp(matteDaySun())
	return stage, setEye()
}

// fortyThree's column: how wide it is at its foot, how high it rises, how
// thick it is at the foot's middle, how much the churn takes away and its
// seed, in units, and the density the bubbles are made of.
const (
	fortyThreeRadius = 0.4  // picked: a glass's width, about the gem's girdle of smoke forty-two
	fortyThreeHeight = 2.0  // picked: tall enough to fill the frame from the floor up
	fortyThreeThick  = 4.0  // picked: thick at the foot, where the light stops in it, and thin by the top
	fortyThreeChurn  = 0.75 // picked: ragged, with the thick of it left in the middle
	fortyThreeEye    = 4.5  // picked: far enough that the column fills the frame from floor to top at fifty degrees
	fortyThreeRise   = 0.3  // picked: a third of a unit a second, a slow fizz
	fortyThreeSeed   = 43   // defined: the smoke's number, as the others'
)

// fortyThreeFizz is the column of bubbles: a medium of its own, one that
// moves with time.
func fortyThreeFizz() medium.Fizz {
	return medium.Fizz{Radius: fortyThreeRadius, Height: fortyThreeHeight,
		Thick: fortyThreeThick, Churn: fortyThreeChurn,
		Rise: fortyThreeRise, Seed: fortyThreeSeed}
}

// fortyThree is the forty-third milestone: a glass fizzing. the haze of
// bubbles too small to see, a column that stands on the floor, thick at its
// foot and thinning as it climbs, widening a little and ragged with the
// churn of the water.
//
// it stands on a dark floor under smoke three's sun, and it is white as a foam is, since a
// bubble scatters every band alike. the column is a medium of its own, one
// that moves with time: at a time it is the churn risen with the bubbles.
func fortyThree() (*thing.World, maths.Subspace) {
	stage := thing.NewWorld()
	stage.Add(slateFloor())
	paint := painted("fizz", swatches.Light(swatches.Paint))
	fizz := fortyThreeFizz()
	stage.AddVolume(volumeAt(fizz, paint, middle))
	stage.Lamp(matteDaySun())
	halfway := maths.V(nought, fortyThreeHeight/pair, nought)
	from := maths.V(nought, fortyThreeHeight/pair, fortyThreeEye)
	return stage, eyeToward(from, halfway)
}

// fortyFour's lights picture: how many pixels wide it is, how many high
// (half as many, for an equirectangular picture), how many degrees apart
// its lit lines are and how many pixels thick they are.
const (
	fortyFourWide  = 720 // picked: two pixels a degree, finer than the sphere shows at this distance
	fortyFourHigh  = fortyFourWide / pair
	fortyFourEvery = 30 // picked: twelve meridians and five parallels, a graticule an eye can count
	fortyFourThick = 3  // picked: a line a degree and a half across
)

// fortyFourEye is how far from the earth's middle the eye stands, in
// units of its radius, so the whole of the globe is in the frame.
const fortyFourEye = 5.0 // picked: the globe fills about two thirds of a frame forty degrees across

// fortyFourGlow is the share of the sun's light a pixel of the lights
// picture gives at its brightest on the night side.
const fortyFourGlow = 0.004 // picked: a few thousandths of the sun's, as smoke thirty's cities

// fortyFourSunToward and fortyFourSunShare are where smoke forty-four's sun
// is toward and its share of daylight.
var fortyFourSunToward = maths.V(0, 0.1, -1) // picked: behind the globe and a little above, so only the lights show

// fortyFourSunShare is the share of daylight smoke forty-four's sun is, and
// smoke forty-five's.
const fortyFourSunShare = 1.2 // picked: a little over daylight, so the lit rim of the globe reads

// The earth's night sky and where the earth is placed: the stars' seed,
// their share, size and brightness, and the detail of the earth's ground.
const (
	nightSeed   = 15    // picked: the seed of smoke thirty's stars
	nightShare  = 0.06  // picked: sparse, so the graticule reads against the sky
	nightSize   = 0.35  // picked: stars a pixel or two across
	nightBright = 0.004 // picked: as faint as the lights, so neither drowns the other
	nightDark   = 0.002 // picked: the dark between the stars of smoke forty-five, barely above black
	earthDetail = 10    // picked: tiles fine enough that the ground reads as a sphere
)

// fortyFourLights is a picture of the ground's lights at night, drawn in
// code so that the smoke reads nothing from disk: a graticule, a white line
// on black at every thirty degrees of longitude and latitude, the picture's
// left edge the antimeridian and its middle row the equator.
func fortyFourLights() image.Image {
	lights := image.NewRGBA(image.Rect(nought, nought, fortyFourWide, fortyFourHigh))
	pixelsEvery := fortyFourWide / (fullTurn / fortyFourEvery)
	for y := range fortyFourHigh {
		for x := range fortyFourWide {
			onMeridian := x%pixelsEvery < fortyFourThick
			onParallel := (y+fortyFourThick/pair)%pixelsEvery < fortyFourThick
			if onMeridian || onParallel {
				lights.Set(x, y, color.White)
			}
		}
	}
	return lights
}

// fortyFourEarth is the earth with its moons, clouds and cities taken off,
// and in their place a picture of the ground's lights.
func fortyFourEarth() bodies.Body {
	earth := sure(template.Earth()).WithoutMoons().WithoutClouds().WithoutCities()
	ground, worn := earth.Ground()
	if !worn {
		placed(fmt.Errorf("smoke: the earth template wears no ground"))
	}
	lights, glow := fortyFourLights(), maths.Share(fortyFourGlow)
	lit := sure(ground.NightLit(lights, glow))
	return with(earth, bodies.Part(lit))
}

// fortyFour is the forty-fourth milestone: the earth by night from a
// picture of its lights. smoke thirty's earth with its cities taken off,
// and in their place a picture of the ground's lights, here a lit graticule,
// wrapped round the sphere and scaled by a share of the sun's light, the
// sun behind it so only the lights show.
//
// the picture is how a real night side is drawn: nasa's black marble lights
// are such a picture. here it is a graticule drawn in code, so the lines
// can be told from any city, and the smoke reads nothing from disk.
func fortyFour() (*thing.World, maths.Subspace) {
	stage := dark()
	stage.Sky = sky.Stars{Seed: nightSeed, Share: nightShare, Size: nightSize,
		Bright: nightBright}
	sun := daylight(fortyFourSunShare)
	stage.Lamp(thing.Sun(fortyFourSunToward, sun))
	earth := fortyFourEarth()
	scale := unity / earth.Radius()
	placed(earth.Place(stage, middle, scale, earthDetail))
	from := maths.V(nought, nought, fortyFourEye)
	return stage, eyeToward(from, middle)
}

// fortyFive's numbers: how far from the eye the track is, how many
// degrees either side of straight ahead the outer planets stand, how
// finely their ground is tiled and how long a circuit of the track takes.
const (
	fortyFiveTrack  = 7.0               // picked: three planets a unit in radius clear of each other, at fifty degrees
	fortyFiveSpread = 20 * maths.Degree // picked: the two outer planets in frame at sixty degrees, clear of the middle one
	fortyFiveDetail = 8                 // picked: tiles a few degrees across, so a continent is a shape
	fortyFivePeriod = 24 * time.Hour    // picked: a circuit a day; a locked planet keeps the face whatever the period
)

// fortyFiveBeachNear is the direction the beach the three planets show the
// eye is sought nearest.
var fortyFiveBeachNear = maths.V(0.3, 0.2, 1) // picked: toward the eye and a little up and to the right

// fortyFiveSunToward is where the sun of smoke forty-five is toward.
var fortyFiveSunToward = maths.V(0.2, 0.3, 1) // picked: from behind the eye, so the three planets are lit full face

// fortyFiveFacing is the world's tile centre the three planets show the
// eye: the beach nearest the direction given, found in the world the
// planet's ground is made from, so the face they keep is a coast.
func fortyFiveFacing(planet bodies.Body) maths.Vec {
	ground := sure(planet.World(fortyFiveDetail))
	tile := found(ground, "beach", fortyFiveBeachNear, isBeach)
	return ground.Tiling.Centre(tile)
}

// isBeach is whether a biome is a beach, the dry ground beside the sea.
func isBeach(biome world.Biome) bool { return biome == world.Beach }

// fortyFiveTurn is the turn that takes the direction from to the
// direction to, about the axis square to both, the short way.
func fortyFiveTurn(from, to maths.Vec) maths.Orientation {
	axis := from.Cross(to)
	if axis == middle {
		placed(fmt.Errorf("smoke: no turn takes %v to %v: they lie "+
			"along one line", from, to))
	}
	cosine := from.Unit().Dot(to.Unit())
	return maths.Rotation(axis, math.Acos(cosine))
}

// fortyFiveTrackAt is the orbit of the planet that stands a spread from
// straight ahead, keeping the face that looks to the middle of the track.
func fortyFiveTrackAt(along float64, face maths.Vec) thing.Orbit {
	phase := math.Pi/pair + along
	// the track's first axis is x and its second is minus z, so the
	// planet at phase is at (cos, 0, -sin) of the track, and its
	// way to the middle is the opposite
	across, ahead := -math.Cos(phase), math.Sin(phase)
	toward := maths.V(across, nought, ahead)
	return thing.Orbit{Radius: fortyFiveTrack, Period: fortyFivePeriod,
		Phase: phase, Locked: true,
		Facing: fortyFiveTurn(face, toward)}
}

// fortyFive is the forty-fifth milestone: a planet on a track of the
// caller's. the earth, its clouds and moons taken off, three times on one
// circle round the eye, each carried by an orbit the smoke makes and
// not the body: each keeps one face to the middle of the circle, the
// face of the land the planet's own ground is made with, so each shows the
// eye the same coast.
//
// a body that places itself turns on its own axis, and would show three
// longitudes. one carried on a motion of the caller's is not turned, so
// where the caller points a face, it stays.
func fortyFive() (*thing.World, maths.Subspace) {
	stage := thing.NewWorld()
	stage.Sky = sky.Stars{Seed: nightSeed, Size: nightSize, Dark: maths.Grey(nightDark)}
	earth := sure(template.Earth()).WithoutMoons().WithoutClouds().
		WithoutAurora().WithoutCities()
	face := fortyFiveFacing(earth)
	scale := unity / earth.Radius()
	for _, along := range []float64{-fortyFiveSpread, nought, fortyFiveSpread} {
		track := fortyFiveTrackAt(along, face)
		placed(earth.PlaceOn(stage, track, scale, fortyFiveDetail))
	}
	sun := daylight(fortyFourSunShare)
	stage.Lamp(thing.Sun(fortyFiveSunToward, sun))
	return stage, maths.Local()
}

// fortySix's puffs: where the two stand, in units, either side of the
// middle and a little above the floor, and what the plan says about the
// one that is baked.
const (
	fortySixApart = 1.4         // picked: two puffs of radius one clear of each other and the frame's edges
	fortySixUp    = 1.3         // picked: a little above the puff's radius, so the floor shows under it
	fortySixPlan  = "baked(64)" // picked: sixty-four cells a side, a cell a thirtieth of a unit
)

// fortySixBaked is the name of the puff the plan bakes, and fortySixExact
// the name of the one it leaves.
const (
	fortySixBaked = "baked puff"
	fortySixExact = "exact puff"
)

// fortySixPuff's numbers: its radius, how thick it is, and the seed, scale
// and amount of its ragged edge.
const (
	fortySixRadius = 1.0 // picked: a unit, as smoke nine's puff
	fortySixThick  = 2.5 // picked: thinner than smoke nine's, so the lit side does not blow out
	fortySixSeed   = 9   // picked: smoke nine's seed, so the noise is the same
	fortySixScale  = 0.3 // picked: smoke nine's scale of noise
	fortySixAmount = 0.7 // picked: smoke nine's amount, so the ragged shows
)

// fortySixPuff is the puff of both: smoke nine's, thinner, so the ragged
// shows in it and the lit side does not blow out.
var fortySixPuff = medium.Wispy{
	Of:     medium.Puff{Radius: fortySixRadius, Thick: fortySixThick},
	Seed:   fortySixSeed,
	Scale:  fortySixScale,
	Amount: fortySixAmount,
}

// fortySixAdd puts a puff of smoke forty-six in the world, with a name,
// standing at x on the row.
func fortySixAdd(stage *thing.World, name string, x float64) {
	paint := painted(name, swatches.Light(swatches.Paint))
	at := maths.V(x, fortySixUp, nought)
	stage.AddVolume(volumeAt(fortySixPuff, paint, at))
}

// fortySix is the forty-sixth milestone: a baked puff beside the exact
// one. smoke nine's puff of ragged smoke twice, side by side over a floor,
// the same noise in both: the left evaluates its noise at every step of
// every ray, the right was computed once into a grid of cells and is read
// back by blending the eight round a point.
//
// the plan says which puff is baked, by its name, and the world handed
// back by shortcut.Apply is a copy with only that one baked. the two
// should look the same to an eye: that is a bake's budget, an oklab
// distance of two hundredths, and the smoke's test measures it.
func fortySix() (*thing.World, maths.Subspace) {
	stage := thing.NewWorld()
	stage.Add(slateFloor())
	fortySixAdd(stage, fortySixExact, -fortySixApart)
	fortySixAdd(stage, fortySixBaked, fortySixApart)
	stage.Lamp(matteDaySun())
	taken := sure(shortcut.Parse(fortySixPlan))
	if !taken.Any() {
		placed(fmt.Errorf("smoke: the plan %q of smoke 46 takes no shortcut",
			fortySixPlan))
	}
	plan := shortcut.Plan{Global: shortcut.None()}.With(fortySixBaked, taken)
	return shortcut.Apply(stage, plan), setEye()
}

// fortySeven's lamps' numbers: how strong the sun is of the day's, where
// it is toward, the laser's place, its aim, how wide it is and how strong,
// and the radar's place and strength.
const (
	fortySevenSun      = 0.25 // picked: a fill from the right, a quarter of smoke one's sun, so the spot is still the key
	fortySevenLaserLen = 20   // picked: more than the distance to the ball, so the beam reaches it
	fortySevenWidth    = 0.12 // picked: a dot on the ball a tenth of its width
	fortySevenRadar    = 5.0  // defined: a strength; a radar lights nothing the eye sees, so its number is not seen
)

var (
	fortySevenSunToward = maths.V(2, 1, -1)       // picked: from the right and behind, so it rims the ball
	fortySevenLaserAt   = maths.V(3.5, 1.6, 3)    // picked: front right of the ball, so the dot faces the eye
	fortySevenLaserAim  = maths.V(-3.5, -0.6, -3) // picked: at the middle of the ball from the laser's place
	fortySevenRadarAt   = maths.V(-4, 3, 0)       // picked: out of frame, to the left
	fortySevenLaserLit  = maths.RGB(10, 0.4, 0.3) // picked: a red dot well above the paint's white, before the matte factor
)

// fortySevenLamps are the five lamps of smoke forty-seven, all made by
// ninjafilm: a sun, a flood, a spot, a laser and a radar. The spot and the
// flood are smoke five's, number for number.
func fortySevenLamps() []thing.Lamp {
	fill := matte(daylight(fortySevenSun))
	laser := matte(fortySevenLaserLit)
	return []thing.Lamp{
		ninja.NewSun(fortySevenSunToward, fill),
		ninja.NewFlood(fiveFlood.Up, fiveFlood.Strength),
		ninja.NewSpot(fiveSpot.At, fiveSpot.Aim, fiveSpot.Strength,
			fiveSpot.Cone, fiveSpot.Soft),
		ninja.NewLaser(fortySevenLaserAt, fortySevenLaserAim, fortySevenWidth,
			laser, fortySevenLaserLen),
		ninja.NewRadar(fortySevenRadarAt, radiation.Radio(), fortySevenRadar),
	}
}

// fortySeven is the forty-seventh milestone: smoke five lit by ninjafilm's
// own lamps. the stage spot and the cool flood of smoke five, made by
// ninjafilm and not by slayer's film package, with a fill of sun from the
// right, a red laser that lands a dot on the ball, and a radar, which gives
// no light to the eye and so changes nothing in the picture.
//
// the lamps are the same lamps to the tracer, which takes any: so the spot
// and the flood alone give smoke five's picture, number for number.
func fortySeven() (*thing.World, maths.Subspace) {
	return fourUnder(true, fortySevenLamps()...)
}

// fortyEight's numbers: the length of one of the scene's units, the balls
// and how far apart they stand, the source behind them, and the eye.
const (
	fortyEightUnit   = 0.01 // picked: one unit of the scene is a centimetre, in metres
	fortyEightRadius = 1.0  // picked: a ball two centimetres across, so aluminium's middle passes about a fifth of the beam and water's two thirds
	fortyEightApart  = 2.6  // picked: middle to middle, so the three balls stand clear of each other
	fortyEightSource = 30.0 // picked: the source's radius, wide enough to fill the frame behind the balls
	fortyEightBehind = 40.0 // picked: how far behind the balls the source's middle is, so its face is ten units back
	fortyEightGives  = 1.0  // defined: the source's strength is the white of the film
	fortyEightAway   = 9.0  // picked: how far the eye stands, so the three balls fill the frame's width
)

// fortyEightBalls are the three balls of smoke forty-eight, left to right,
// each named for the material it is made of: the heaviest first.
var fortyEightBalls = []string{"lead", "aluminium", "water"}

// fortyEightBand is the band smoke forty-eight is seen in: the photon of
// the film energy, 60 keV, about 0.021 nm, an x-ray.
var fortyEightBand = fortyEightSeen()

// fortyEightSeen is the film energy's band, read off lead's own
// coefficient, so the band and the materials cannot disagree about the
// wavelength.
func fortyEightSeen() maths.Band {
	lead := sure(materials.ByName("lead"))
	row, there := lead.Attenuations().At(materials.FilmEnergy)
	if !there {
		panic("lead has no coefficient at the film energy")
	}
	return maths.Band{Nanometres: row.Wavelength()}
}

// fortyEight is the forty-eighth milestone: lead, aluminium and water on
// an x-ray film. three balls, each two centimetres across, stand in a dark
// room in front of an x-ray source and are seen at 60 keV.
//
// no lamp is lit, so the eye's own picture is black and the balls have no
// look in the light we see. the picture is the x-rays' alone: white where
// the source's beam arrives whole, darker where a ball took some of it.
//
// what each ball takes is not picked. it is derived from the material's
// row of NIST's tables: e to the minus mu over rho, times the density,
// times the way through. so lead is a black disc; aluminium is a shadow
// that deepens toward its middle, where the way through is longest; and
// water is the lightest shadow: its middle takes a third of the beam.
func fortyEight() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: fortyEightSource},
		Material: &thing.Material{Name: "x-ray source",
			Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{
				radiation.XRay(): fortyEightGives}}},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(nought, nought, -fortyEightBehind)}),
	})
	for at, name := range fortyEightBalls {
		kind := sure(materials.ByName(name))
		w.Add(thing.Thing{
			Shape: shape.Sphere{Radius: fortyEightRadius},
			Material: &thing.Material{Name: name,
				Radiant: sure(thing.Stopping(kind, fortyEightUnit))},
			Motion: thing.Still(maths.Subspace{Origin: maths.V(
				(float64(at)-unity)*fortyEightApart, nought, nought)}),
		})
	}
	from := maths.V(nought, nought, fortyEightAway)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, middle)}
}
