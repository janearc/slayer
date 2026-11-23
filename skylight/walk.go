package skylight

import (
	"math"
	"math/rand/v2"
)

// a census is what a walk counted besides power: the events, which are
// scatterings and reflections, and the photons that reached the ground
// at least once.
type census struct {
	events, grounded int64
}

// a photon is one packet of sunlight on its walk: where it is, its way,
// a unit vector, the layer it is in, its weight, whether it has been
// scattered or reflected, whether it has reached the ground, and how
// many events it has had.
type photon struct {
	position, heading   vector
	layer               int
	weight              float64
	scattered, grounded bool
	events              int
}

// advance moves the photon along its way by distance.
func (light *photon) advance(distance float64) {
	light.position = light.position.plus(light.heading.scaled(distance))
}

// an ending is how a photon's travel stopped: at an event in the air,
// on the ground, or out of the top.
type ending int

// The endings of travel.
const (
	inAir   ending = iota // defined: an event in a layer
	landed                // defined: at the ground
	escaped               // defined: above the top
)

// a walker is one worker's walk: the air, its generator, the seed and
// the number of its first photon, the cap on events, its tally, and its
// budget and census, in weights of a photon's power.
type walker struct {
	air     *air
	random  *rand.PCG
	seed    uint64
	first   uint64
	most    int
	tally   tally
	spent   Budget
	counted census
}

// newWalker is the walker whose photons are numbered from first, each
// drawing from the generator seeded by the seed and its own number,
// with a tally of the grid's shape.
func newWalker(made *air, grid Grid, seed, first uint64,
	most int) *walker {
	return &walker{air: made, random: rand.NewPCG(seed, first), seed: seed,
		first: first, most: most, tally: newTally(grid)}
}

// begin seeds the generator for the photon of a number: from the seed
// and that number alone, so the photon draws the same numbers whatever
// the photons before it drew.
//
// A column walked at two wavelengths from one seed so walks each photon
// from the same numbers at both, and their noise moves together: the
// sky's colour, a ratio of wavelengths, keeps less of it than either.
func (w *walker) begin(number uint64) {
	w.random.Seed(w.seed, number)
}

// uniform draws a number from nought up to, not including, one.
func (w *walker) uniform() float64 {
	return float64(w.random.Uint64()>>exponentBits) / fraction
}

// depth draws an optical depth to the next event: minus the log of a
// uniform draw from above nought to one.
func (w *walker) depth() float64 { return -math.Log1p(-w.uniform()) }

// run walks photons one after another, each to its end, each from its
// own number's draws.
func (w *walker) run(photons int) {
	var light photon
	for number := range uint64(photons) {
		w.begin(w.first + number)
		w.launch(&light)
		w.walk(&light)
		w.counted.events += int64(light.events)
		if light.grounded {
			w.counted.grounded++
		}
	}
}

// launch starts a photon on the top sphere, at a place drawn evenly
// over the band of the disc the sphere shows the sun, going away from
// the sun.
func (w *walker) launch(light *photon) {
	top := w.air.top
	reach := w.air.band.reach(w.uniform())
	across, along := math.Sincos(fullTurn * w.uniform())
	height := math.Sqrt(max(none, (top-reach)*(top+reach)))
	*light = photon{position: vector{reach * along, reach * across, height},
		heading: vector{none, none, -one},
		layer:   len(w.air.layers) - one, weight: one}
}

// walk follows a photon from event to event until it escapes, is
// absorbed, is capped or loses at roulette.
func (w *walker) walk(light *photon) {
	for {
		var going bool
		switch w.travel(light, w.depth()) {
		case escaped:
			w.spent.Escaped += light.weight
			return
		case landed:
			light.events++
			going = w.reflect(light)
		default:
			light.events++
			going = w.scatter(light)
		}
		if !going || !w.survive(light) {
			return
		}
	}
}

// travel moves the photon through the layers, each of its own
// extinction, until the optical depth runs out inside one, or it
// reaches the ground or the top, and says which.
func (w *walker) travel(light *photon, depth float64) ending {
	for {
		through := &w.air.layers[light.layer]
		distance, outward := exit(light.position, light.heading, through)
		if spent := through.extinction * distance; spent > depth {
			light.advance(depth / through.extinction)
			return inAir
		}
		depth -= through.extinction * distance
		light.advance(distance)
		if stop, done := w.cross(light, outward); done {
			return stop
		}
	}
}

// cross takes the photon over the boundary of its layer it has reached,
// outward or inward, tallying it there, and says whether that was the
// ground or the top.
func (w *walker) cross(light *photon, outward bool) (ending, bool) {
	boundary, next := light.layer, light.layer-one
	if outward {
		boundary, next = light.layer+one, light.layer+one
	}
	w.record(light, boundary)
	if next < none {
		return landed, true
	}
	if next == len(w.air.layers) {
		return escaped, true
	}
	light.layer = next
	return inAir, false
}

// reflect is the ground's event: it absorbs its share, and sends the
// rest up in a way drawn by the cosine law about the local vertical,
// tallied on its way out. It says whether any light is left.
func (w *walker) reflect(light *photon) bool {
	w.spent.Ground += light.weight * (one - w.air.ground)
	light.weight *= w.air.ground
	light.grounded = true
	if light.weight == none {
		return false
	}
	up := light.position.unit()
	light.heading = turned(up, math.Sqrt(w.uniform()), w.uniform())
	light.scattered = true
	w.record(light, none)
	return true
}

// scatter is an event in the air: the layer absorbs its share, and the
// rest turns by the gas's phase or the particles', chosen by their
// shares of the scattering. It says whether any light is left.
func (w *walker) scatter(light *photon) bool {
	through := &w.air.layers[light.layer]
	w.spent.Air += light.weight * (one - through.albedo)
	light.weight *= through.albedo
	if light.weight == none {
		return false
	}
	var cosine float64
	if w.uniform() < through.gasShare {
		cosine = rayleighCosine(w.uniform())
	} else {
		cosine = henyeyCosine(through.asymmetry, w.uniform())
	}
	light.heading = turned(light.heading, cosine, w.uniform())
	light.scattered = true
	return true
}

// survive is whether the photon walks on after an event: not at the
// cap, which takes what it carries, and, under the lightest weight, by
// winning at roulette, which doubles it.
func (w *walker) survive(light *photon) bool {
	if light.events >= w.most {
		w.spent.Capped += light.weight
		return false
	}
	if light.weight >= lightest {
		return true
	}
	if w.uniform() < half {
		return false
	}
	light.weight *= two
	return true
}

// record tallies a photon crossing the boundary, when the boundary
// keeps radiance and the photon is sky: scattered or reflected at least
// once. Unscattered light is the sun's beam, which Transmittance gives.
func (w *walker) record(light *photon, boundary int) {
	keep := w.air.kept[boundary]
	if keep == unkept || !light.scattered {
		return
	}
	w.tally.add(keep, light.position, light.heading, light.weight)
}
