package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// lockedLayer is the test layer over a world locked to a sun that stands
// toward a direction in the medium's frame.
func lockedLayer(t *testing.T, toward maths.Vec) Clouds {
	t.Helper()
	layer := cloudsLayer(t)
	layer.Locked, layer.Toward = true, toward.Unit()
	return layer
}

// awayFrom is a unit direction a number of radians of arc from toward, at
// an azimuth about it, in radians.
func awayFrom(toward maths.Vec, arc, azimuth float64) maths.Vec {
	first, second := CloudWeather{Toward: toward.Unit()}.frame()
	ring := first.Scale(math.Cos(azimuth)).Add(second.Scale(math.Sin(azimuth)))
	return toward.Unit().Scale(math.Cos(arc)).Add(ring.Scale(math.Sin(arc)))
}

// clouded is the share of the azimuths round a circle of arc from the
// substellar point, over every test sky, that are clouded in the layer.
func clouded(t *testing.T, toward maths.Vec, arc float64) float64 {
	t.Helper()
	const azimuths = 120
	var count, asked int
	for _, seed := range weatherSeeds {
		layer := lockedLayer(t, toward)
		layer.Seed = seed
		middle := layer.Radius + layer.Low + (layer.High-layer.Low)/10
		for step := range azimuths {
			at := awayFrom(toward, arc, 2*math.Pi*float64(step)/azimuths)
			asked++
			if layer.Density(at.Scale(middle)) > cloudVisible*layer.Thick {
				count++
			}
		}
	}
	return float64(count) / float64(asked)
}

// A locked world is clouded almost all over: a solid deck at its substellar
// point, dense and churning cloud about its terminator, and cloud carried
// onto its night side, thinner there than anywhere else. Over the arcs from
// the point to the point opposite, most of the sphere is cloud. The
// anti-test: the night is not solid, so the carried cloud is thinner than
// the deck and the belt, and the night has some, so it is carried.
func TestALockedWorldIsDeckedAtItsSubstellarPointAndCloudedAlmostAllOver(
	t *testing.T) {
	toward := maths.V(0.6, 0.5, 0.62)
	deck := clouded(t, toward, 0.1)
	terminator := clouded(t, toward, math.Pi/2)
	night := clouded(t, toward, math.Pi-0.1)
	var share float64
	const arcs = 18
	for step := range arcs {
		share += math.Sin(math.Pi*(float64(step)+0.5)/arcs) *
			clouded(t, toward, math.Pi*(float64(step)+0.5)/arcs)
	}
	share /= 2 * arcs / math.Pi
	t.Logf("clouded: deck %.2f, terminator %.2f, night %.2f, over the "+
		"sphere %.2f", deck, terminator, night, share)
	if deck < 0.95 {
		t.Errorf("the substellar deck is clouded %.2f, want solid", deck)
	}
	if terminator < 0.8 {
		t.Errorf("the terminator is clouded %.2f, want dense", terminator)
	}
	if !(night > 0.05 && night < terminator && night < deck) {
		t.Errorf("the night %.2f is not thinner than the terminator %.2f "+
			"and the deck %.2f, and not none", night, terminator, deck)
	}
	if share < 0.6 {
		t.Errorf("the sphere is clouded %.2f, want most of it", share)
	}
}

// A locked world's towers are tallest at its substellar point and lower
// toward its terminator, and differ from one another; they are never taller
// than the layer. The anti-test: the sky of a world that is not locked is
// one everywhere, every cloud as tall as its patch says.
func TestALockedWorldsTowersAreTallestAtItsSun(t *testing.T) {
	toward := maths.V(0.6, 0.5, 0.62)
	weather := lockedLayer(t, toward).Weather()
	mean := func(arc float64) (average, least, most float64) {
		least = math.Inf(1)
		const azimuths = 360
		for step := range azimuths {
			top := weather.Top(awayFrom(toward, arc,
				2*math.Pi*float64(step)/azimuths))
			average += top / azimuths
			least, most = math.Min(least, top), math.Max(most, top)
		}
		return average, least, most
	}
	near, _, _ := mean(0.1)
	far, least, most := mean(math.Pi / 2)
	t.Logf("towers %.2f at the sun and %.2f at the terminator, %.2f to "+
		"%.2f round it", near, far, least, most)
	if !(near > far+0.15) {
		t.Errorf("towers are %.2f at the sun and %.2f at the terminator",
			near, far)
	}
	if most-least < 0.15 || most > 1 || least <= 0 {
		t.Errorf("towers round the terminator run %v to %v", least, most)
	}
	plain := cloudsLayer(t).Weather()
	for _, at := range []maths.Vec{onSphere(0.3, 1), onSphere(-1, 4)} {
		if top := plain.Top(at); top != 1 {
			t.Errorf("a world not locked has a tower of %v at %v", top, at)
		}
	}
}

// The anti-test: the substellar point does not drift. Wherever the sun
// stands in the medium's frame, the cover is greatest at it: the
// direction of greatest cover over a coarse sweep of the sphere is within
// a tenth of a radian of the substellar point, and a world not locked
// has no such point.
func TestALockedWorldsDeckStaysAtItsSun(t *testing.T) {
	for _, toward := range []maths.Vec{maths.V(1, 0, 0), maths.V(0, 0, 1),
		maths.V(0.6, 0.5, 0.62), maths.V(-0.3, -0.8, 0.5)} {
		weather := lockedLayer(t, toward).Weather()
		best, where := math.Inf(-1), maths.Vec{}
		for latitude := -80.0; latitude <= 80; latitude += 5 {
			for longitude := 0.0; longitude < 360; longitude += 5 {
				at := onSphere(latitude*math.Pi/180, longitude*math.Pi/180)
				if cover, _ := weather.Locking(at); cover > best {
					best, where = cover, at
				}
			}
		}
		if arc := math.Acos(where.Dot(weather.Toward)); arc > 0.1 {
			t.Errorf("sun toward %v: the deck is %v radians from it",
				toward, arc)
		}
	}
	plain := cloudsLayer(t).Weather()
	if plain.Locked || plain.Table()[weatherNumbers+stormCount*stormNumbers] != 0 {
		t.Errorf("a world that is not locked says it is")
	}
}

// A locked world's whorls turn round its terminator, each the way the
// rotation turns it in its hemisphere: nearly square to the sun, and
// counter-clockwise only where they are north.
func TestALockedWorldsWhorlsTurnRoundItsTerminator(t *testing.T) {
	weather := lockedLayer(t, maths.V(0.6, 0.5, 0.62)).Weather()
	least, most := math.Inf(1), 0.0
	for index, storm := range weather.Storms {
		arc := math.Acos(storm.Centre.Dot(weather.Toward))
		if math.Abs(arc-weather.LockedStorm) > lockedStormSpread+1e-9 {
			t.Errorf("whorl %d is %v radians from the sun", index, arc)
		}
		if north := storm.Centre.Y >= 0; north != (storm.Sign > 0) {
			t.Errorf("whorl %d at height %v turns %v", index,
				storm.Centre.Y, storm.Sign)
		}
		strength := math.Abs(storm.Sign)
		if math.Abs(strength-1) > weather.LockedStrength+1e-9 {
			t.Errorf("whorl %d has a strength of %v", index, strength)
		}
		least, most = math.Min(least, strength), math.Max(most, strength)
	}
	t.Logf("whorls of strength %.2f to %.2f", least, most)
	if most-least < 0.2 {
		t.Errorf("the whorls are all of a strength, %v to %v: the north "+
			"and the south are mirrors", least, most)
	}
}

// The anti-test: a sky that is not locked has whorls of one strength, one
// or minus one, as it had before the locked ones varied.
func TestAWorldNotLockedHasWhorlsOfOneStrength(t *testing.T) {
	for index, storm := range cloudsLayer(t).Weather().Storms {
		if math.Abs(storm.Sign) != 1 {
			t.Errorf("whorl %d of a turning world has strength %v", index,
				storm.Sign)
		}
	}
}

// topShare is how much of its full density a cloud has a twentieth of its
// own height below its top, at the most, over the directions of a sphere
// that are solid at the bottom of the layer: the share of Thick left in
// the last twentieth is what makes a top a step or a fade.
func topShare(layer Clouds) (most float64, solid int) {
	const steps, slice = 2000, 0.05
	for _, at := range sphereDirections(300) {
		density := func(height float64) float64 {
			radius := layer.Radius + layer.Low + height*(layer.High-layer.Low)
			return layer.Density(at.Scale(radius)) / layer.Thick
		}
		if density(0.02) < 0.99 {
			continue
		}
		solid++
		top := 0.0
		for step := range steps {
			if height := float64(step) / steps; density(height) > 0 {
				top = height
			}
		}
		most = math.Max(most, density(top*(1-slice)))
	}
	return most, solid
}

// sphereDirections is a spiral of unit directions over the sphere.
func sphereDirections(count int) []maths.Vec {
	points := make([]maths.Vec, count)
	golden := math.Pi * (math.Sqrt(5) - 1)
	for index := range points {
		height := 1 - 2*(float64(index)+0.5)/float64(count)
		flat := math.Sqrt(1 - height*height)
		points[index] = maths.V(flat*math.Cos(golden*float64(index)),
			height, flat*math.Sin(golden*float64(index)))
	}
	return points
}

// A locked world's towers thin to nothing at their tops, so that their tops
// and edges are soft: a twentieth of its height below its top a cloud has
// under a twentieth of its full density. The anti-test: a world that turns
// keeps a share of its density to its top, a step there, over a tenth a
// twentieth below it, which is what drew round three's towers as rock.
func TestALockedWorldsTowersThinToNothingAtTheirTops(t *testing.T) {
	locked := lockedLayer(t, maths.V(0.6, 0.5, 0.62))
	locked.Cover = 3
	turning := cloudsLayer(t)
	turning.Cover = 3
	soft, solid := topShare(locked)
	step, found := topShare(turning)
	t.Logf("a twentieth below the top a locked cloud has %.3f of its "+
		"density over %d solid directions, a turning one %.3f over %d",
		soft, solid, step, found)
	if solid < 30 || found < 30 {
		t.Fatalf("only %d and %d directions are solid", solid, found)
	}
	if soft > 0.05 {
		t.Errorf("a locked cloud has %v of its density at its top", soft)
	}
	if step < 0.1 {
		t.Errorf("a turning cloud has %v of its density at its top, want a "+
			"step", step)
	}
}

// The kernel's table carries the locking after the storms: its flag, the
// way to the sun, and the numbers the deck, terminator and night are laid
// by, and its length is the one the kernel reads.
func TestTheTableCarriesTheLocking(t *testing.T) {
	weather := lockedLayer(t, maths.V(0, 0, 2)).Weather()
	table := weather.Table()
	first := weatherNumbers + stormCount*stormNumbers
	if len(table) != first+lockedNumbers {
		t.Fatalf("a table of %d numbers, want %d", len(table),
			first+lockedNumbers)
	}
	if table[first] != 1 || table[first+1] != 0 || table[first+2] != 0 ||
		table[first+3] != 1 || table[first+4] != weather.Deck ||
		table[first+11] != weather.Broken ||
		table[first+12] != weather.TowerBase ||
		table[first+15] != weather.TowerFrequency ||
		table[first+18] != weather.FarWidth ||
		table[first+lockedNumbers-2] != weather.TowerBillow ||
		table[first+lockedNumbers-1] != weather.WhorlThin {
		t.Errorf("the locking in the table is %v", table[first:])
	}
}

// A whorl in a locked sky is relief and not cover: its arms are ridges of
// taller cloud and its lanes lower cloud, between whorlLowest and
// whorlHighest of the cloud's own height. The anti-test: a sky that turns
// has no relief anywhere, and a locked one has both ridges and lanes.
func TestAWhorlIsRidgesAndLanesInALockedSky(t *testing.T) {
	const asked = 3000 // directions over the sphere, enough to meet a whorl
	weather := lockedLayer(t, maths.V(0.6, 0.5, 0.62)).Weather()
	var least, most float64 = 1, 0
	for _, direction := range sphereDirections(asked) {
		kept := weather.Relief(direction)
		least, most = math.Min(least, kept), math.Max(most, kept)
		if kept < whorlLowest || kept > whorlHighest {
			t.Fatalf("a cloud keeps %.2f of its height at %v", kept, direction)
		}
	}
	if least > 0.8 || most < 1.2 {
		t.Errorf("whorls leave between %.2f and %.2f of the height", least,
			most)
	}
	plain := weather
	plain.Locked = false
	for _, direction := range sphereDirections(asked) {
		if plain.Relief(direction) != 1 {
			t.Fatalf("a sky that turns has relief at %v", direction)
		}
	}
}
