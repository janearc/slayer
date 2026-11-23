package trace

// The mirror of water rougher than a pixel can show. The slope the sea's
// bands do not carry, and the slope of the bands a pixel is too wide to
// show, are waves all the same: a look that meets them is mirrored by many
// small faces, each its own way, and sees the sky and the scene by all of
// them. The sun's highlight takes that slope in its lobe; every other
// mirrored look takes it here.

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// roughLooks is how many small faces a mirrored look off rough water is
// sent by. Picked: four, one in each quarter turn about the surface's own
// normal, the fewest that tilt the mirror every way.
const roughLooks = 4

// The two multipliers of the SplitMix64 mixing step, and its three shifts
// (Steele, Lea and Flood 2014, Fast splittable pseudorandom number
// generators, OOPSLA; the constants as in Vigna's splitmix64.c).
const (
	mixFirst    = 0xbf58476d1ce4e5b9
	mixSecond   = 0x94d049bb133111eb
	mixStep     = 0x9e3779b97f4a7c15
	mixHigh     = 30
	mixMiddle   = 27
	mixLow      = 31
	mixMantissa = 53
	mixKept     = 64 - mixMantissa
)

// roughLeast is the slope variance under which water mirrors by its one
// normal: picked, a tilt of a thousandth, which no pixel shows.
const roughLeast = 1e-6

// roughStrata is which quarter of the slopes, by size, each quarter turn
// takes: every size once, and no two neighbouring turns neighbouring
// sizes.
var roughStrata = [roughLooks]float64{0, 2, 1, 3}

// facet is one small face of rough water: the way it faces, and the share
// of the arriving light it mirrors toward the eye, which is fresnel's at
// its own tilt times how much of what the eye sees of the surface it is.
type facet struct {
	normal maths.Vec
	share  float64
}

// facets are the small faces a look along ray meets a sea by at point,
// where normal is the sea's own normal there on the look's side and lost
// the slope variance the pixel cannot show; and whether the surface is
// rough water at all.
//
// It is only for a look that enters a sea from above
// with some slope unseen, the sea's own (roughWater) and the pixel's
// together: anything else mirrors by its one normal. A sea under a wind
// that leaves it like a mirror says none, and mirrors as a mirror.
//
// The slopes are drawn from the Gaussian of slopes about the surface's
// own that Cox and Munk measured the sea to have, the same Beckmann's lobe
// takes for the sun, the variance split evenly along and across.
//
// A face is
// seen by the area it shows the eye, its cosine to the look over its
// cosine to the surface, so a face leaning toward a grazing eye counts for
// more and one facing away for nothing. Faces hidden behind other faces
// are not found: there is no shadowing among them.
func (met hit) facets(material *thing.Material, ray maths.Ray, point,
	normal maths.Vec, lost float64, at boundary) ([roughLooks]facet, bool) {
	var faces [roughLooks]facet
	_, sea := met.thing.thing.Shape.(roughWater)
	if !sea || !at.real || !at.entering || material.Metal != nil ||
		!(material.Through > 0) {
		return faces, false
	}
	variance := met.facetVariance(lost)
	if !(variance > roughLeast) {
		return faces, false
	}
	way := ray.Dir.Unit()
	first, second := dithered(point, way)
	var seen float64
	for index := range faces {
		tilted := tilt(normal, variance, index, first, second)
		facing := -way.Dot(tilted)
		if !(facing > 0) || !(tilted.Dot(normal) > 0) {
			continue
		}
		shown := facing / tilted.Dot(normal)
		faces[index] = facet{normal: tilted,
			share: shown * material.SplitsBetween(facing, at.relative)}
		seen += shown
	}
	return shared(faces, seen)
}

// facetVariance is the slope variance rough water's faces are tilted by:
// what the sea says its waves do not hold and what the pixel cannot show.
// Nothing for a surface that is not a rough sea.
func (met hit) facetVariance(lost float64) float64 {
	water, sea := met.thing.thing.Shape.(roughWater)
	if !sea {
		return 0
	}
	return water.Unseen() + lost
}

// shared is faces with each share over seen, the area all of them show
// the eye together, so that the shares are of one surface; and whether
// any face showed the eye anything.
func shared(faces [roughLooks]facet, seen float64) ([roughLooks]facet,
	bool) {
	if !(seen > 0) {
		return faces, false
	}
	for index := range faces {
		faces[index].share /= seen
	}
	return faces, true
}

// facetShare is the share of the arriving light the faces mirror, all of
// them together: fresnel's share of the rough surface.
func facetShare(faces [roughLooks]facet) float64 {
	var sum float64
	for _, face := range faces {
		sum += face.share
	}
	return sum
}

// tilt is the normal of the index-th small face about normal, for a
// slope variance and two numbers from nought up to one that place the
// faces: its quarter turn begun at first of a quarter, and its slope's
// size, by the Gaussian's own share of sizes, taken at second within its
// stratum. A Gaussian in two slopes of variance v together has the square
// of its size spread as an exponential of mean v.
func tilt(normal maths.Vec, variance float64, index int, first,
	second float64) maths.Vec {
	turn := 2 * math.Pi * (float64(index) + first) / roughLooks
	within := (roughStrata[index] + second) / roughLooks
	size := math.Sqrt(-variance * math.Log(1-within))
	along, across := tangents(normal)
	sin, cos := math.Sincos(turn)
	return normal.Sub(along.Scale(size * cos)).Sub(
		across.Scale(size * sin)).Unit()
}

// tangents are two unit ways that lie in the surface whose normal is
// given, square to each other.
func tangents(normal maths.Vec) (along, across maths.Vec) {
	east := maths.V(1, 0, 0)
	if math.Abs(normal.X) > math.Abs(normal.Z) {
		east = maths.V(0, 0, 1)
	}
	along = normal.Cross(east).Unit()
	return along, normal.Cross(along).Unit()
}

// dithered is two numbers from nought up to one, the same every time
// for the same point and way and unlike for others: where among the
// slopes this look's faces fall. A draw is the same draw each time it is
// made.
func dithered(point, way maths.Vec) (first, second float64) {
	bits := mixed(math.Float64bits(point.X))
	for _, part := range [...]float64{point.Y, point.Z, way.X, way.Y,
		way.Z} {
		bits = mixed(bits ^ math.Float64bits(part))
	}
	return unitOf(bits), unitOf(mixed(bits + mixStep))
}

// mixed is bits stirred by SplitMix64's mixing step, so that near numbers
// give unlike ones.
func mixed(bits uint64) uint64 {
	bits ^= bits >> mixHigh
	bits *= mixFirst
	bits ^= bits >> mixMiddle
	bits *= mixSecond
	return bits ^ bits>>mixLow
}

// unitOf is a number from nought up to one made of the top bits of bits,
// as many as a float64 holds exactly.
func unitOf(bits uint64) float64 {
	return float64(bits>>mixKept) / (1 << mixMantissa)
}

// roughMirror is the light rough water mirrors toward the eye at point:
// each small face's own mirrored look followed through the scene, by the
// face's share. Every look leaves from the same place, kept above the
// surface's own side.
func (scene *staged) roughMirror(seen look, met hit, ray maths.Ray, point,
	side maths.Vec, faces [roughLooks]facet) maths.Light {
	var sum maths.Light
	for _, face := range faces {
		if !(face.share > 0) {
			continue
		}
		departure := scene.bounced(ray, point, face.normal, met.face(side))
		departure.At = scene.interfaceOrigin(met.thing, point, side, ray.At)
		off := shade(scene, seen.mirrored(departure))
		sum = sum.Add(off.Light().Scale(face.share))
	}
	return sum
}

// roughSky is the sky rough water mirrors toward the eye along dir when
// the look has used its bounces: lastMirror's, by every small face and
// its share, met by nothing on the way.
func (scene *staged) roughSky(dir maths.Vec,
	faces [roughLooks]facet) maths.Light {
	var sum maths.Light
	for _, face := range faces {
		if !(face.share > 0) {
			continue
		}
		mirrored := dir.Sub(face.normal.Scale(2 * dir.Dot(face.normal)))
		sum = sum.Add(scene.skyAt(mirrored).Scale(face.share))
	}
	return sum
}
