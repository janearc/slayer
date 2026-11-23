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
// spot is drawn; its value is in picks.yaml.
var ovalness = pick("spot-ovalness")

// NewSpot is a spot, refused off the planet or with no size.
func NewSpot(latitude, longitude, radius, turn float64) (Spot, error) {
	if !(latitude >= -poleLatitude && latitude <= poleLatitude) ||
		!(radius > nothing && radius <= poleLatitude) ||
		math.IsNaN(longitude) || math.IsNaN(turn) ||
		endless(longitude) || endless(turn) {
		return Spot{}, fmt.Errorf("a spot at %g, %g of radius %g",
			latitude, longitude, radius)
	}
	return Spot{latitude: latitude, longitude: longitude, radius: radius,
		turn: turn}, nil
}

// anticyclone is whether a spot turns against the planet's turning:
// clockwise seen from above in the north, the other way in the south.
//
// these are the white ovals and the great spots of a gas giant, high haze
// lifted in their middles; the cyclones are its barges, cleared down to
// deeper cloud.
func (spot Spot) anticyclone() bool {
	return spot.turn*spot.latitude < nothing
}

// mostSpots is the most spots a swarm may hold, so a set cannot ask a cost
// without end; its value is in picks.yaml.
var mostSpots = pickCount("swarm-most-spots")

// swarmTurn is how far a swarm's spots turn at their middles, in half
// turns, and swarmHighest how near the poles they are placed, in degrees;
// swarmLargest is the largest radius, in degrees, a swarm's spots may be
// given. all three are in picks.yaml.
var (
	swarmTurn    = pick("swarm-turn")
	swarmHighest = pick("swarm-highest")
	swarmLargest = pick("swarm-largest")
)

// drawsPerSpot is defined as how many draws of a seeded hash a swarm's spot
// takes, and the four places they hold: latitude, longitude, radius and way.
const (
	// drawsPerSpot is defined as how many draws a swarm's spot takes: four.
	drawsPerSpot = 4
	// latitudeDraw is defined as the place of the draw for a spot's latitude.
	latitudeDraw = 0
	// longitudeDraw is defined as the place of the draw for its longitude.
	longitudeDraw = 1
	// radiusDraw is defined as the place of the draw for its radius.
	radiusDraw = 2
	// wayDraw is defined as the place of the draw for which way it turns.
	wayDraw = 3
)

// spotDraw is a draw from zero to one for one of a swarm's spots, by its
// place in the swarm and which of its draws it is.
func spotDraw(seed uint64, spot int, which int64) float64 {
	return noise.Hash(seed, int64(drawsPerSpot*spot)+which)
}

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
	if count < nothing || count > mostSpots || !(smallest > nothing) ||
		!(largest >= smallest && largest <= swarmLargest) ||
		!(slope > whole) || endless(slope) {
		return storms, fmt.Errorf("a swarm of %d spots from %g to %g "+
			"degrees, slope %g", count, smallest, largest, slope)
	}
	spots := append([]Spot(nil), storms.spots...)
	high := math.Sin(swarmHighest * maths.Degree)
	for i := range count {
		latitude := math.Asin((double*spotDraw(seed, i, latitudeDraw)-
			whole)*high) / maths.Degree
		longitude := fullTurnDegrees*spotDraw(seed, i, longitudeDraw) -
			halfTurnDegrees
		radius := sized(spotDraw(seed, i, radiusDraw), smallest, largest,
			slope)
		way := storms.turning(latitude)
		if way == nothing {
			way = whole
			if spotDraw(seed, i, wayDraw) < half {
				way = -whole
			}
		}
		spots = append(spots, Spot{latitude: latitude, longitude: longitude,
			radius: radius, turn: way * swarmTurn})
	}
	storms.spots = spots
	return storms, nil
}

// turning is which way the jets' shear turns the air at a latitude in degrees,
// one the planet's way and minus one against it; zero with no jets.
//
// the jet runs as the sine of the latitude, so the air's turning, against the
// jet's slope across the latitudes, is minus its cosine.
func (storms Storms) turning(latitude float64) float64 {
	if storms.shear == nothing {
		return nothing
	}
	if math.Cos(circle*latitude/storms.every) > nothing {
		return -whole
	}
	return whole
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
	placed := make([]vortex, nothing, len(storms.spots))
	for _, spot := range storms.spots {
		middle := ways.toward(spot.latitude, spot.longitude)
		east := ways.pole.Cross(middle)
		if east.Len() == nothing {
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
	if up.Dot(whirl.middle) <= nothing {
		return math.Inf(whole)
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
// lies, and collarWidth how wide it is. collarClearing is how much of a
// clearing the ring shows. all are in picks.yaml.
var (
	holding        = pick("vortex-holding")
	core           = pick("vortex-core")
	collar         = pick("vortex-collar")
	collarWidth    = pick("vortex-collar-width")
	collarClearing = pick("vortex-collar-clearing")
)

// swirled is a way turned about each vortex it falls in, by the spot's
// turn, most at its middle and none at its edge; and the marks the
// vortices leave there.
func swirled(up maths.Vec, whirls []vortex) (maths.Vec, marks) {
	var made marks
	for _, whirl := range whirls {
		in := whirl.within(up)
		if in >= whole {
			continue
		}
		share := whole - in
		if held := math.Min(whole, share/holding); held > made.held {
			made.held, made.ridden = held, whirl.spot.latitude
		}
		middle := math.Max(nothing, math.Min(whole,
			(whole-in)/(whole-core)))
		rung := math.Exp(-(in - collar) * (in - collar) /
			(collarWidth * collarWidth))
		if whirl.spot.anticyclone() {
			made.capped = math.Max(made.capped, middle)
			made.cleared = math.Max(made.cleared, collarClearing*rung)
		} else {
			made.cleared = math.Max(made.cleared, middle)
		}
		up = maths.Rotation(whirl.middle,
			whirl.spot.turn*math.Pi*share*share).Out(up)
	}
	return up, made
}
