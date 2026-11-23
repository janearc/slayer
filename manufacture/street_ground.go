package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// the places a street's picks are drawn from in its seed.
const (
	// streetDishPlace is the place the dishes' draws are made from.
	streetDishPlace = 41
	// streetDishAcrossDraw, and the draws after it, say which number of a
	// dish a draw is: where it lies along and across the street, its
	// widths and its depth.
	streetDishAcrossDraw = iota - 1
	streetDishAlongDraw
	streetDishWideDraw
	streetDishLongDraw
	streetDishDeepDraw
	streetDishSideDraw
)

// the numbers the road's low places are made with.
const (
	// streetDishSpacing is about how far apart the dishes lie along the
	// street, in metres. picked.
	streetDishSpacing = 5.0
	// streetDishNearest is how near the street's middle a dish lies, in
	// metres. picked: rain pools in the wheel tracks and the gutter, and
	// not on the crown. streetDishMargin is how far a dish keeps from
	// the kerb, in metres.
	streetDishNearest = 1.0
	streetDishMargin  = 0.05
	// streetDishWideLeast and streetDishWideMost are the least and most
	// half widths of a dish across the street, and the Long ones along
	// it, in metres. picked: a dish is longer along the street, as a rut
	// is.
	streetDishWideLeast = 0.5
	streetDishWideMost  = 1.2
	streetDishLongLeast = 0.9
	streetDishLongMost  = 2.3
	// streetDishDeepLeast and streetDishDeepMost are how far a dish
	// sinks below the road around it, in metres. picked: two to five
	// centimetres, the depth of a rut in worn asphalt.
	streetDishDeepLeast = 0.02
	streetDishDeepMost  = 0.045
)

// the numbers puddles are made with.
const (
	// streetPuddleFirst and streetPuddleFull are the wetness at which
	// the first rain stands in the dishes and at which they are as full
	// as they get. picked.
	streetPuddleFirst = 0.35
	streetPuddleFull  = 0.95
	// streetPuddleBrim is the share of a dish's depth to its lowest rim
	// that the water reaches when it is as full as it gets. picked:
	// water that reached the rim would be running off.
	streetPuddleBrim = 0.9
	// streetPuddleSides is how many sides a puddle's outline has.
	streetPuddleSides = 28
	// streetPuddleSearch is how many steps along a ray from a puddle's
	// middle the search for its edge takes.
	streetPuddleSearch = 60
	// streetFloodFirst is the wetness at which water begins to rise
	// from the gutters over the whole road, and streetFloodUp how far
	// above the crown it stands when the street is flooded, in metres.
	// picked: a centimetre of water over the crown.
	streetFloodFirst = 0.9
	streetFloodUp    = 0.012
)

// streetDish is a low place in the road: an ellipse of road sunk by Deep
// at its middle, with a smooth rim, longer along the street than across.
type streetDish struct{ x, z, across, along, deep float64 }

// streetGround is the street's surface of the road, as a height from its
// crown: its camber and its dishes, and how wet it is, from nought to one.
type streetGround struct {
	length float64
	wet    float64
	dishes []streetDish
}

// newStreetGround is the ground of a street of this length and wetness,
// with the dishes the seed lays down it, a few to every stretch of it.
func newStreetGround(length, wet float64, from seed) streetGround {
	count := max(2, int(length/streetDishSpacing))
	ground := streetGround{length: length, wet: wet}
	for index := range count {
		ground.dishes = append(ground.dishes,
			streetDishOf(from, index, count, length))
	}
	return ground
}

// streetDishOf is the seed's dish number index of count along a street
// of this length. It lies in its own stretch of the street, off the kerb
// and clear of the street's ends, so that its water has a road to rest on.
func streetDishOf(from seed, index, count int, length float64) streetDish {
	draw := func(which int) float64 {
		return from.draw(streetDishPlace, index, which)
	}
	across := lerp(streetDishWideLeast, streetDishWideMost,
		draw(streetDishWideDraw))
	side := 1.0
	if draw(streetDishSideDraw) < 0.5 {
		side = -1
	}
	farthest := streetRoadHalf - across - streetDishMargin
	stretch := (float64(index) + draw(streetDishAlongDraw)) / float64(count)
	return streetDish{
		x: side * lerp(streetDishNearest, farthest,
			draw(streetDishAcrossDraw)),
		z: lerp(streetDishLongMost-length/2, length/2-streetDishLongMost,
			stretch),
		across: across,
		along: lerp(streetDishLongLeast, streetDishLongMost,
			draw(streetDishLongDraw)),
		deep: lerp(streetDishDeepLeast, streetDishDeepMost,
			draw(streetDishDeepDraw)),
	}
}

// lerp is the number a share of the way from one to the other.
func lerp(from, to, share float64) float64 { return from + (to-from)*share }

// reach is how far from the dish's middle a point is, as a share of the
// dish's half widths, squared: below one the point is in the dish.
func (dish streetDish) reach(x, z float64) float64 {
	across, along := (x-dish.x)/dish.across, (z-dish.z)/dish.along
	return across*across + along*along
}

// dip is how far the dish sinks the road at a point, in metres: Deep at
// its middle and nought, with no slope, at its rim.
func (dish streetDish) dip(x, z float64) float64 {
	reach := dish.reach(x, z)
	if reach >= 1 { // defined: outside the dish
		return 0 // defined: no sinking
	}
	return -dish.deep * (1 - reach) * (1 - reach)
}

// slope is the dish's rise of the road, per metre across and along.
func (dish streetDish) slope(x, z float64) (float64, float64) {
	reach := dish.reach(x, z)
	if reach >= 1 { // defined: outside the dish
		return 0, 0 // defined: no slope
	}
	scale := 4 * dish.deep * (1 - reach)
	return scale * (x - dish.x) / (dish.across * dish.across),
		scale * (z - dish.z) / (dish.along * dish.along)
}

// camber is the road's height at a distance from its middle, from the
// crown at nought, and its slope there: a parabola falling to the gutters.
func camber(x float64) (float64, float64) {
	share := x / streetRoadHalf
	return -streetCamber * share * share,
		-2 * streetCamber * x / (streetRoadHalf * streetRoadHalf)
}

// height is the road's height above its crown at a point, in metres.
func (ground streetGround) height(x, z float64) float64 {
	rise, _ := camber(x)
	for _, dish := range ground.dishes {
		rise += dish.dip(x, z)
	}
	return rise
}

// slope is the road's rise per metre across and along at a point.
func (ground streetGround) slope(x, z float64) (float64, float64) {
	_, across := camber(x)
	var along float64
	for _, dish := range ground.dishes {
		more, further := dish.slope(x, z)
		across, along = across+more, along+further
	}
	return across, along
}

// fill is how full of rain the dishes are, from nought to one.
func (ground streetGround) fill() float64 {
	return streetRamp(ground.wet, streetPuddleFirst, streetPuddleFull)
}

// puddles is the water standing in each dish: a flat fan of faces whose
// rim lies where the road reaches the water's level. A dry street has
// none.
func (ground streetGround) puddles() surface {
	var water surface
	if ground.fill() <= 0 { // defined: no rain has stood
		return water
	}
	for _, dish := range ground.dishes {
		water = append(water, ground.puddle(dish)...)
	}
	return water
}

// level is the height the water stands at in a dish: from the dish's
// bottom, the share the street's fill gives of the way to its lowest rim.
func (ground streetGround) level(dish streetDish) float64 {
	bottom := ground.height(dish.x, dish.z)
	spill := math.Inf(1)
	for side := range streetPuddleSides {
		angle := math.Pi * 2 * float64(side) / streetPuddleSides
		rim, _ := camber(dish.x + dish.across*math.Cos(angle))
		spill = math.Min(spill, rim)
	}
	return bottom + ground.fill()*streetPuddleBrim*(spill-bottom)
}

// puddle is the water in one dish, a fan about its middle, wound up.
func (ground streetGround) puddle(dish streetDish) surface {
	level := ground.level(dish)
	rim := make([]maths.Vec, streetPuddleSides)
	for side := range rim {
		angle := math.Pi * 2 * float64(side) / streetPuddleSides
		reach := ground.edge(dish, level, math.Cos(angle), math.Sin(angle))
		rim[side] = maths.V(dish.x+reach*dish.across*math.Cos(angle), level,
			dish.z+reach*dish.along*math.Sin(angle))
	}
	middle := maths.V(dish.x, level, dish.z)
	var water surface
	for side := range rim {
		following := rim[(side+1)%streetPuddleSides]
		water = append(water, face{middle, following, rim[side]})
	}
	return water
}

// edge is how far out a ray from the dish's middle, in shares of its
// half widths, the water at this level reaches before the road comes up
// to it: the first step of the search whose road is not below the water.
func (ground streetGround) edge(dish streetDish, level, across,
	along float64,
) float64 {
	for step := 1; step <= streetPuddleSearch; step++ {
		reach := float64(step) / streetPuddleSearch
		x := dish.x + reach*dish.across*across
		z := dish.z + reach*dish.along*along
		if ground.height(x, z) >= level {
			return reach
		}
	}
	return 1
}

// flood is the sheet of water that rises over the whole carriageway as
// the street floods: from the gutter, where it is deepest, to a centimetre
// over the crown. Where the road is above it the road shows through. A
// street less wet than streetFloodFirst has none.
func (ground streetGround) flood() surface {
	rise := streetRamp(ground.wet, streetFloodFirst, 1)
	if rise <= 0 { // defined: the gutters have not risen
		return nil
	}
	gutter, _ := camber(streetRoadHalf)
	level := lerp(gutter, streetFloodUp, rise)
	back, front := -ground.length/2, ground.length/2
	return quad(nil, maths.V(-streetRoadHalf, level, back),
		maths.V(-streetRoadHalf, level, front),
		maths.V(streetRoadHalf, level, front),
		maths.V(streetRoadHalf, level, back))
}
