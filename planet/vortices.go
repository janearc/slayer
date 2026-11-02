package planet

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// A Spot is a vortex: where its middle is, latitude and longitude in degrees,
// longitude zero toward the eye; its radius, half its height north to south, in
// degrees; and how far it turns the bands at its middle, in half turns, the
// sign saying which way, more than zero the way the planet turns.
//
// it is drawn ovalness times wider east to west than north to south.
type Spot struct {
	latitude, longitude, radius, turn float64
}

// ovalness is how many times wider east to west than north to south a
// spot is drawn: picked, near the great red spot's of recent decades,
// not pinned.
const ovalness = 1.7

// NewSpot is a spot, refused off the planet or with no size.
func NewSpot(latitude, longitude, radius, turn float64) (Spot, error) {
	if !(latitude >= -90 && latitude <= 90) || !(radius > 0 &&
		radius <= 90) || math.IsNaN(longitude) || math.IsNaN(turn) {
		return Spot{}, fmt.Errorf("a spot at %g, %g of radius %g",
			latitude, longitude, radius)
	}
	return Spot{latitude: latitude, longitude: longitude, radius: radius,
		turn: turn}, nil
}

// anticyclone is whether a spot turns against the planet's turning:
// clockwise seen from above in the north, the other way in the south.
//
// on jupiter these are the white ovals and the great red spot, high
// haze lifted in their middles; the cyclones are the brown barges,
// cleared down to deeper cloud.
func (spot Spot) anticyclone() bool { return spot.turn*spot.latitude < 0 }

// mostSpots is the most spots a swarm may hold: picked, so a set cannot
// ask a cost without end.
const mostSpots = 2000

// swarmTurn is how far a swarm's spots turn at their middles, in half
// turns, and highest how near the poles they are placed, in degrees:
// picked.
const (
	swarmTurn    = 2.5
	swarmHighest = 75.0
)

// Swarmed is the storms with a swarm of spots more: so many, from a
// seed, their radii from smallest to largest in degrees, so many more
// of the small as a power law of a slope says, scattered evenly over the
// sphere below swarmHighest.
//
// each turns the way the jets' shear turns the air where it stands, as only
// a vortex turning with its shear lasts in it; with no jets, either way.
// the jets are set first.
func (storms Storms) Swarmed(seed uint64, count int, smallest, largest,
	slope float64) (Storms, error) {
	if count < 0 || count > mostSpots || !(smallest > 0) ||
		!(largest >= smallest && largest <= 30) || !(slope > 1) {
		return storms, fmt.Errorf("a swarm of %d spots from %g to %g "+
			"degrees, slope %g", count, smallest, largest, slope)
	}
	spots := append([]Spot(nil), storms.spots...)
	for i := range count {
		drawn := func(k int64) float64 { return noise.Hash(seed, int64(4*i)+k) }
		high := math.Sin(swarmHighest * maths.Degree)
		latitude := math.Asin((2*drawn(0)-1)*high) / maths.Degree
		longitude := 360*drawn(1) - 180
		radius := powerLaw(drawn(2), smallest, largest, slope)
		way := storms.turning(latitude)
		if way == 0 {
			way = 1
			if drawn(3) < 0.5 {
				way = -1
			}
		}
		spots = append(spots, Spot{latitude: latitude, longitude: longitude,
			radius: radius, turn: way * swarmTurn})
	}
	storms.spots = spots
	return storms, nil
}

// powerLaw is a size from smallest to largest drawn by a share from zero to
// one, so many more of the small as a slope says.
func powerLaw(share, smallest, largest, slope float64) float64 {
	low, high := math.Pow(smallest, 1-slope), math.Pow(largest, 1-slope)
	return math.Pow(low+share*(high-low), 1/(1-slope))
}

// turning is which way the jets' shear turns the air at a latitude in degrees,
// one the planet's way and minus one against it; zero with no jets.
//
// the jet runs as the sine of the latitude, so the air's turning, against the
// jet's slope across the latitudes, is minus its cosine.
func (storms Storms) turning(latitude float64) float64 {
	if storms.shear == 0 {
		return 0
	}
	if math.Cos(2*math.Pi*latitude/storms.every) > 0 {
		return -1
	}
	return 1
}

// a vortex is a spot placed on the face: its middle, the ways east and
// north there, and its radius in radians.
type vortex struct {
	spot                Spot
	middle, east, north maths.Vec
	radius              float64
}

// vortices is the storms' spots placed in a frame, once a picture.
func (storms Storms) vortices(ways frame) []vortex {
	placed := make([]vortex, 0, len(storms.spots))
	for _, spot := range storms.spots {
		middle := ways.toward(spot.latitude, spot.longitude)
		east := ways.pole.Cross(middle)
		if east.Len() == 0 {
			east = ways.east
		}
		east = east.Unit()
		placed = append(placed, vortex{spot: spot, middle: middle,
			east: east, north: middle.Cross(east).Unit(),
			radius: spot.radius * maths.Degree})
	}
	return placed
}

// within is how far into a vortex a way is, on its oval: zero at its middle,
// one at its edge, and more outside it.
func (whirl vortex) within(up maths.Vec) float64 {
	if up.Dot(whirl.middle) <= 0 {
		return math.Inf(1)
	}
	across := up.Dot(whirl.east) / ovalness
	return math.Hypot(across, up.Dot(whirl.north)) / whirl.radius
}

// marks are what the vortices leave at a point: how much one holds its air
// against the jets, zero to one, and the latitude it rides at; how much of a
// white cap shows, and how much of a dark clearing; and, read from a flow, how
// many seconds the air there has cooked.
type marks struct {
	held, ridden    float64
	capped, cleared float64
	age             float64
}

// holding is how far in from a vortex's edge, as a share of its radius,
// it holds its air whole against the jets; core, where its cap or
// clearing is full; and collar, how far out a white oval's dark ring
// lies, and how wide it is: picked, after juno's pictures.
const (
	holding = 0.3
	core    = 0.6
	collar  = 0.92
	ring    = 0.08
)

// swirled is a way turned about each vortex it falls in, by the spot's
// turn, most at its middle and none at its edge; and the marks the
// vortices leave there.
func swirled(up maths.Vec, whirls []vortex) (maths.Vec, marks) {
	var made marks
	for _, whirl := range whirls {
		in := whirl.within(up)
		if in >= 1 {
			continue
		}
		share := 1 - in
		if held := math.Min(1, share/holding); held > made.held {
			made.held, made.ridden = held, whirl.spot.latitude
		}
		middle := math.Max(0, math.Min(1, (1-in)/(1-core)))
		rung := math.Exp(-(in - collar) * (in - collar) / (ring * ring))
		if whirl.spot.anticyclone() {
			made.capped = math.Max(made.capped, middle)
			made.cleared = math.Max(made.cleared, 0.6*rung)
		} else {
			made.cleared = math.Max(made.cleared, middle)
		}
		up = maths.Rotation(whirl.middle,
			whirl.spot.turn*math.Pi*share*share).Out(up)
	}
	return up, made
}
