package planet

import (
	"math"
	"math/rand"
	"slices"
	"sync"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// the reference below is Face.Within, and everything it calls that was
// hoisted, written out as they were before the face worked out its bands once.
// each takes the optics a face holds for a column and band, so a face's
// stale or wrong numbers cannot reach it.

// refRayleigh is a molecule's phase, as it was.
func refRayleigh(turn float64) float64 { return 0.75 * (1 + turn*turn) }

// refHenyey is the henyey-greenstein phase, as it was.
func refHenyey(lean, turn float64) float64 {
	square := lean * lean
	return (1 - square) / math.Pow(1+square-2*lean*turn, 1.5)
}

// refPhase is a layer's phase, as it was.
func refPhase(layer optics, turn float64) float64 {
	gas, haze := layer.scatter, layer.haze*layer.hazeAlbedo
	if gas+haze == 0 {
		return 0
	}
	return (gas*refRayleigh(turn) + haze*refHenyey(layer.hazeLean, turn)) /
		(gas + haze)
}

// refReflected is a column's reflection, as it was, from the optics each
// leaf was made of.
func refReflected(stack []leaf, deck, star, eye, turn float64) float64 {
	slant := 1/star + 1/eye
	above, light := 0.0, 0.0
	for _, held := range stack {
		layer := held.raw
		tau := layer.depth()
		if tau > 0 {
			albedo := layer.scattered() / tau
			light += albedo * refPhase(layer, turn) / (4 * math.Pi) *
				star / (star + eye) * math.Exp(-above*slant) *
				-math.Expm1(-tau*slant)
		}
		above += tau
	}
	return light + deck*star/math.Pi*math.Exp(-above*slant)
}

// refFrom is one column's light in a band, as it was.
func refFrom(face *Face, mixed mix, column, i int, star, eye,
	turn float64) float64 {
	if face.ladders == nil || column >= len(face.ladders) {
		return refReflected(face.stacks[column][i], face.deck,
			star, eye, turn)
	}
	rungs := face.ladders[column]
	below := refReflected(rungs[mixed.rung][i], face.deck, star,
		eye, turn)
	if mixed.toward == 0 {
		return below
	}
	above := refReflected(rungs[mixed.rung+1][i], face.deck, star,
		eye, turn)
	return below + mixed.toward*(above-below)
}

// refSent is the mix's light in a band, as it was.
func refSent(face *Face, mixed mix, i int, star, eye, turn float64) float64 {
	sent := refFrom(face, mixed, mixed.here, i, star, eye, turn)
	for _, each := range mixed.blends() {
		other := refFrom(face, mixed, each.column, i, star, eye, turn)
		sent += each.share * (other - sent)
	}
	return sent
}

// refRadiance is planck's law, as it was.
func refRadiance(nm, kelvin float64) float64 {
	if !(kelvin > 0) {
		return 0
	}
	wave := nm * 1e-9
	return 2 * planck * light * light / math.Pow(wave, 5) /
		math.Expm1(planck*light/(wave*boltzmann*kelvin))
}

// refGlowed is a clearing's heat in a band, as it was.
func refGlowed(face *Face, cleared, lo, hi float64) float64 {
	if cleared == 0 || face.hottest == 0 {
		return 0
	}
	kelvin := face.top + (face.hottest-face.top)*cleared
	return cleared * refRadiance(math.Sqrt(lo*hi), kelvin) * (hi - lo) * 1e-9
}

// refGlowedAt is the airglow in a band, as it was.
func refGlowedAt(face *Face, latitude, eye, lo, hi float64) float64 {
	if face.airglow == 0 || math.Abs(latitude) >= face.airband {
		return 0
	}
	held := 0.0
	for _, line := range []float64{sodiumD2, sodiumD1} {
		if line >= lo && line < hi {
			held += 0.5
		}
	}
	return held * face.airglow / math.Max(eye, grazing)
}

// refWithin is Within as it was before a face's bands were worked out once:
// a map of the pixel's bands made for each call and given to the observer.
func refWithin(face *Face, p maths.Vec, footprint float64) maths.Light {
	up := p.Unit()
	toward := face.eye.Sub(p).Unit()
	star, eye := up.Dot(face.star), up.Dot(toward)
	if !(eye > 0) {
		return maths.Light{}
	}
	mixed, marked, came := face.airAt(up, footprint/p.Len())
	front := 0.0
	if face.airglow > 0 && math.Abs(came) < face.airband {
		front = face.front(up, footprint/p.Len())
	}
	reached := 0.0
	if star > 0 {
		reached = shadowed(face.rings, p, face.star, face.pole)
	}
	turn := -face.star.Dot(toward)
	first, _ := face.light.Span()
	lines := make(map[float64]float64, len(face.stacks[mixed.here]))
	for i := range face.stacks[mixed.here] {
		band := first + bands.Band(i)
		sent := 0.0
		if reached > 0 {
			sent = refSent(face, mixed, i, star, eye, turn)
		}
		lo, hi := band.Edges()
		lines[centre(band)] = (face.light.Power(band)*reached*sent +
			refGlowed(face, marked.cleared, lo, hi) +
			refGlowedAt(face, came, eye, lo, hi)*front) * face.exposure
	}
	spectrum := bands.Lines(lines)
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

// a probe is a point of a body and the footprint to ask it at.
type probe struct {
	at        maths.Vec
	footprint float64
}

// probesOf is n seeded points of a face's body, a third spread evenly, and the
// rest on the limb, the terminator, the poles and the night side, with a
// footprint of none for every other one and a wide range for the rest.
func probesOf(face *Face, seed int64, n int) []probe {
	r := rand.New(rand.NewSource(seed))
	radius := 6e7
	around := func(centre maths.Vec, along float64) maths.Vec {
		side := maths.V(r.NormFloat64(), r.NormFloat64(), r.NormFloat64())
		across := side.Dot(centre)
		side = side.Sub(centre.Scale(across)).Unit()
		return side.Scale(math.Sqrt(1 - along*along)).Add(centre.Scale(along))
	}
	eyeWay := face.eye.Unit()
	limb := radius / face.eye.Len()
	made := make([]probe, n)
	for k := range made {
		var up maths.Vec
		switch k % 6 {
		case 0, 1:
			up = maths.V(r.NormFloat64(), r.NormFloat64(),
				r.NormFloat64()).Unit()
			if lit := eyeWay.Add(face.star); k%6 == 1 && up.Dot(lit) < 0 {
				up = up.Scale(-1)
			}
		case 2:
			up = around(eyeWay, limb+(r.Float64()-0.5)*2e-3)
		case 3:
			up = around(face.star, (r.Float64()-0.5)*2e-3)
		case 4:
			spin := float64(1 - 2*(k/6%2))
			up = around(face.pole.Scale(spin),
				1-r.Float64()*1e-3)
		default:
			up = around(face.star.Scale(-1), r.Float64()*0.9)
		}
		foot := 0.0
		if k%2 == 0 {
			foot = math.Exp(r.Float64()*10) * 10
		}
		made[k] = probe{at: up.Unit().Scale(radius), footprint: foot}
	}
	return made
}

// sameBits is whether two lights are equal in every bit.
func sameBits(a, b maths.Light) bool {
	return math.Float64bits(a.R) == math.Float64bits(b.R) &&
		math.Float64bits(a.G) == math.Float64bits(b.G) &&
		math.Float64bits(a.B) == math.Float64bits(b.B)
}

// differing is how many of the probes the face draws differently from the
// reference, in any bit.
func differing(face *Face, probes []probe) int {
	count := 0
	for _, each := range probes {
		got := face.Within(each.at, each.footprint)
		if !sameBits(got, refWithin(face, each.at, each.footprint)) {
			count++
		}
	}
	return count
}

// aSwirl is a flow of its own: the latitude its air came from moved by a
// wave round the planet, spin that changes sign, and an age.
type aSwirl struct{}

// At is the swirl at a latitude and longitude in radians.
func (aSwirl) At(latitude, longitude float64) (origin, spin, age float64) {
	origin = latitude + 0.15*math.Sin(3*longitude)*math.Cos(latitude)
	spin = 0.7 * math.Sin(5*longitude+2*latitude)
	age = 1e6 * (1 + math.Sin(longitude+latitude))
	return origin, spin, age
}

// hoistGas is a gas with a cross-section falling as the fourth power, and
// one that absorbs in a red line, as methane does.
func hoistGas(t testing.TB, name string, area, molar float64,
	absorbs bool) Gas {
	t.Helper()
	gas, err := NewGas(name, func(nm float64) float64 {
		return area * math.Pow(550/nm, 4)
	}, molar)
	if err != nil {
		t.Fatal(err)
	}
	if absorbs {
		gas = gas.Absorbing(func(nm float64) float64 {
			return 4e-27 * math.Exp(-math.Pow((nm-620)/40, 2))
		})
	}
	return gas
}

// hoistLayer is a layer of two gases under a haze.
func hoistLayer(t testing.TB, top, bottom, haze, albedo, lean,
	blue float64) Layer {
	t.Helper()
	air := hoistGas(t, "air", 5e-31, 0.0023, false)
	methane := hoistGas(t, "methane", 2e-31, 0.016, true)
	cloud, err := NewHaze(haze, albedo, lean)
	if err != nil {
		t.Fatal(err)
	}
	if blue >= 0 {
		if cloud, err = cloud.Tinted(blue); err != nil {
			t.Fatal(err)
		}
	}
	layer, err := NewLayer(top, bottom, []Gas{air, methane},
		[]float64{0.9, 0.1}, cloud)
	if err != nil {
		t.Fatal(err)
	}
	return layer
}

// hoistZones is three zones of two layers each, differing from one to the
// next in haze, lean and tint.
func hoistZones(t testing.TB) []Zone {
	t.Helper()
	var zones []Zone
	edges := []float64{-90, -25, 25, 90}
	for i := 0; i < 3; i++ {
		share := float64(i + 1)
		column := []Layer{
			hoistLayer(t, 1e3, 4e4, 0.2*share, 0.9-0.1*share, 0.3*share-0.5, 0.2*share),
			hoistLayer(t, 4e4, 3e5, 0.1*share, 0.5, -0.2*share, -1),
		}
		zone, err := NewZone(edges[i], edges[i+1], column)
		if err != nil {
			t.Fatal(err)
		}
		zones = append(zones, zone)
	}
	return zones
}

// stormyBody is a body as full as the giant's: storms with jets and a swarm
// of spots, a churn, a cap and a clearing, cooked air, a clearing's heat and
// an airglow, with a flow of its own where asked.
func stormyBody(t testing.TB, flowing bool) Body {
	t.Helper()
	body, err := NewBody(6e7, 24.8, 0.8, hoistZones(t))
	if err != nil {
		t.Fatal(err)
	}
	spot, err := NewSpot(-22, 30, 10, 6)
	if err != nil {
		t.Fatal(err)
	}
	storms, err := NewStorms(7, 0.8, 24, 4, 6, 3, []Spot{spot})
	if err != nil {
		t.Fatal(err)
	}
	if storms, err = storms.Jetted(30, 30); err != nil {
		t.Fatal(err)
	}
	if storms, err = storms.Swarmed(23, 40, 0.4, 6, 2.5); err != nil {
		t.Fatal(err)
	}
	body = body.Stormy(storms)
	churn := []Layer{hoistLayer(t, 1e3, 1e5, 0.5, 0.4, 0.5, 0.1)}
	cap := []Layer{hoistLayer(t, 1e3, 1e5, 1.5, 1, 0.8, -1)}
	clearing := []Layer{hoistLayer(t, 1e3, 1e5, 0.01, 0.2, 0, -1)}
	if body, err = body.Churned(churn, 1); err != nil {
		t.Fatal(err)
	}
	if body, err = body.Marked(cap, clearing); err != nil {
		t.Fatal(err)
	}
	if body, err = body.Heated(150, 1800); err != nil {
		t.Fatal(err)
	}
	if body, err = body.Airglowing(50, 70, 0.05); err != nil {
		t.Fatal(err)
	}
	cooked := []Layer{hoistLayer(t, 1e3, 4e4, 0.6, 0.7, 0.6, 0.3)}
	if body, err = body.Cooked(cooked, 1e6); err != nil {
		t.Fatal(err)
	}
	if flowing {
		body = body.Flowing(aSwirl{})
	}
	return body
}

// wide is starlight from 300 to 900 nm, a few ultraviolet and infrared bands
// the eye weighs nothing in, and more bands than even.
func wide() bands.Light {
	spectrum := map[int]float64{}
	for nm := 300; nm <= 900; nm++ {
		spectrum[nm] = 1 + float64(nm%7)/10
	}
	return bands.Split(spectrum)
}

// hoistFaces are faces that differ in everything a face works out once: air,
// band count, storms or none, a flow or none, glows, and where the eye and
// the star stand.
func hoistFaces(t testing.TB) map[string]*Face {
	t.Helper()
	dark, err := NewLayer(0, 1e3, []Gas{hoistGas(t, "dark", 0, 0.02, true)},
		[]float64{1}, Haze{})
	if err != nil {
		t.Fatal(err)
	}
	star := maths.V(0.3, 0.2, 1).Unit()
	plain := bareBody(t, 0.6, []Layer{dark})
	return map[string]*Face{
		"bare": NewFace(bareBody(t, 1, nil), maths.V(0, 1, 0),
			maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 2),
		"dark": NewFace(plain, maths.V(0.1, 1, 0).Unit(), star,
			maths.V(1e9, 3e8, 1e9), narrow(), 0.7),
		"stormy": NewFace(stormyBody(t, false), maths.V(0.2, 1, 0.1),
			maths.V(-0.5, 0.3, 1), maths.V(2e8, -1e8, 1.4e8), even(), 1.3),
		"flowing": NewFace(stormyBody(t, true), maths.V(0, 1, 0.4),
			maths.V(0.8, 0.1, 0.5), maths.V(-1.4e8, 3e7, 5e7), wide(), 0.9),
		"far": NewFace(stormyBody(t, true), maths.V(1, 0.3, 0),
			maths.V(0, 0.3, -1), maths.V(3e11, 1e11, 2e11), even(), 2),
	}
}

// Within is the reference's arithmetic bit for bit on faces that differ in
// every input a hoisted value depends on, at points on the limb, the
// terminator, the poles and the night side, on one goroutine and on eight.
func TestWithinIsTheStartingArithmeticBitForBit(t *testing.T) {
	for name, face := range hoistFaces(t) {
		if !face.quick {
			t.Errorf("%s: the face's sums do not agree with the observer's", name)
		}
		seed := int64(len(name))
		probes := probesOf(face, seed, 2400)
		if got := differing(face, probes); got != 0 {
			t.Errorf("%s: %d of %d points differ, on one goroutine", name,
				got, len(probes))
		}
		var wg sync.WaitGroup
		bad := make([]int, 8)
		for g := range bad {
			wg.Add(1)
			go func() {
				defer wg.Done()
				probes := probesOf(face, int64(g)+100, 600)
				bad[g] = differing(face, probes)
			}()
		}
		wg.Wait()
		for g, got := range bad {
			if got != 0 {
				t.Errorf("%s: goroutine %d: %d points differ", name, g, got)
			}
		}
	}
}

// spoiled is a copy of a face whose worked-out values a function has made
// wrong, as a hoist gone wrong would: stale from another face, or worked out
// for the wrong band. the copy shares nothing it changes with the face.
func spoiled(face *Face, spoil func(*Face)) *Face {
	bad := *face
	bad.shades = slices.Clone(face.shades)
	bad.lobes = slices.Clone(face.lobes)
	spoil(&bad)
	return &bad
}

// swapped is a face's shades with two bands' worked-out values exchanged.
func swapped(face *Face) {
	last := len(face.shades) - 1
	face.shades[0], face.shades[last] = face.shades[last], face.shades[0]
}

// holder is the place of the first band that holds a d line, or the last.
func holder(face *Face) int {
	for i, each := range face.shades {
		if each.held > 0 {
			return i
		}
	}
	return len(face.shades) - 1
}

// leavesAt is a face with its stacks and ladders copied, and a function
// given each leaf of the band at the middle of its light, so it may spoil it.
func leavesAt(face *Face, spoil func(*leaf)) {
	face.stacks = slices.Clone(face.stacks)
	for column := range face.stacks {
		face.stacks[column] = cloneBands(face.stacks[column], spoil)
	}
	face.ladders = slices.Clone(face.ladders)
	for column := range face.ladders {
		face.ladders[column] = slices.Clone(face.ladders[column])
		for rung := range face.ladders[column] {
			face.ladders[column][rung] = cloneBands(face.ladders[column][rung], spoil)
		}
	}
}

// cloneBands is a column's stacks copied, with a function given each leaf of
// the middle band's.
func cloneBands(stacks [][]leaf, spoil func(*leaf)) [][]leaf {
	made := slices.Clone(stacks)
	middle := len(made) / 2
	for i := range made {
		made[i] = slices.Clone(made[i])
	}
	for k := range made[middle] {
		spoil(&made[middle][k])
	}
	return made
}

// elsewhere is a leaf taken from the next band over, as a leaf worked out for
// the wrong band would be, keeping the optics it was made from, which the
// reference reads.
func elsewhere(face *Face) func(*leaf) {
	return func(each *leaf) {
		other := face.stacks[len(face.stacks)-1][len(face.stacks[0])/2+1]
		raw := each.raw
		*each = other[0]
		each.raw = raw
	}
}

// heated is the place of a band the eye sees, where a clearing's glow is
// most.
func heated(face *Face) int { return len(face.shades) / 2 }

// bandSpoilings are the ways to make wrong a value hoisted for one band of a
// face, or the face's scale of sums, by name.
func bandSpoilings() map[string]func(*Face) {
	return map[string]func(*Face){
		"the bands' values for the wrong bands": swapped,
		"a band's power": func(face *Face) {
			face.shades[holder(face)].power *= 1.01
		},
		"a band's middle, where the observer is asked": func(face *Face) {
			face.quick = false
			face.shades[holder(face)].centre *= 1.2
		},
		"a band's weight in X": func(face *Face) {
			face.shades[holder(face)].wx *= 1.01
		},
		"a band's weight in Y": func(face *Face) {
			face.shades[holder(face)].wy *= 1.01
		},
		"a band's weight in Z": func(face *Face) {
			face.shades[holder(face)].wz *= 1.01
		},
		"the scale of the sums": func(face *Face) { face.unit *= 1.01 },
		"a band's warmth scale": func(face *Face) {
			face.shades[heated(face)].warm.scale *= 1.01
		},
		"a band's warmth cool": func(face *Face) {
			face.shades[heated(face)].warm.cool *= 1.01
		},
		"a band's width": func(face *Face) {
			face.shades[heated(face)].warm.width *= 1.01
		},
		"how much of the d lines a band holds": func(face *Face) {
			face.shades[holder(face)].held = 0
		},
	}
}

// leafSpoilings adds to spoilings the ways to make wrong one value of every
// layer's leaf, by name.
func leafSpoilings(spoilings map[string]func(*Face)) {
	for what, change := range map[string]func(*leaf){
		"a leaf's depth":    func(each *leaf) { each.tau *= 1.01 },
		"a leaf's albedo":   func(each *leaf) { each.albedo *= 1.01 },
		"a leaf being open": func(each *leaf) { each.open = !each.open },
		"a leaf's gas":      func(each *leaf) { each.gas *= 1.01 },
		"a leaf's haze":     func(each *leaf) { each.haze *= 1.01 },
		"a leaf's total":    func(each *leaf) { each.together *= 1.01 },
	} {
		spoilings["every layer's "+what] = func(face *Face) { leavesAt(face, change) }
	}
}

// lobeSpoilings adds to spoilings the ways to make wrong one value of every
// lobe, by name.
func lobeSpoilings(spoilings map[string]func(*Face)) {
	for what, change := range map[string]func(*lobe){
		"rest":  func(each *lobe) { each.rest *= 1.01 },
		"near":  func(each *lobe) { each.near *= 1.01 },
		"twice": func(each *lobe) { each.twice += 0.01 },
	} {
		spoilings["every lobe's "+what] = func(face *Face) {
			for k := range face.lobes {
				change(&face.lobes[k])
			}
		}
	}
}

// placeSpoilings adds to spoilings the ways to put lobes and layers in the
// wrong places, by name.
func placeSpoilings(spoilings map[string]func(*Face)) {
	spoilings["lobes for the wrong faces"] = func(face *Face) {
		if last := len(face.lobes) - 1; last > 0 {
			face.lobes[0], face.lobes[last] = face.lobes[last], face.lobes[0]
		}
	}
	spoilings["every layer's lobe place"] = func(face *Face) {
		leavesAt(face, func(each *leaf) {
			each.slot = (each.slot + 1) % len(face.lobes)
		})
	}
	spoilings["a layer from the wrong band"] = func(face *Face) {
		leavesAt(face, elsewhere(face))
	}
}

// each hoisted value, made wrong for one band of a face, shows in some face's
// picture: the test above would fail on it.
func TestWithinNoticesAHoistedValueGoneWrong(t *testing.T) {
	spoilings := bandSpoilings()
	leafSpoilings(spoilings)
	lobeSpoilings(spoilings)
	placeSpoilings(spoilings)
	faces := hoistFaces(t)
	for what, spoil := range spoilings {
		seen := 0
		for name, face := range faces {
			bad := spoiled(face, spoil)
			seed := int64(len(name))
			seen += differing(bad, probesOf(face, seed, 1200))
		}
		if seen == 0 {
			t.Errorf("%s: no face's picture changed", what)
		}
	}
}

// a face that asks the observer for every point, as one does whose sums do not
// agree with the observer's, draws every point as the reference does.
func TestWithinAskingTheObserverIsTheStartingArithmetic(t *testing.T) {
	for name, face := range hoistFaces(t) {
		asking := spoiled(face, func(each *Face) { each.quick = false })
		if got := differing(asking, probesOf(face, 9, 600)); got != 0 {
			t.Errorf("%s: %d points differ", name, got)
		}
	}
}

// the sums are made for the faces the tests build, and a weight, the scale
// or a band's place that is wrong is found out when the face is made.
func TestAFaceChecksItsSumsAgainstTheObserver(t *testing.T) {
	light := even()
	shades := shadesOf(light)
	unit := whiteUnit()
	if !agreeing(light, shades, unit) {
		t.Fatal("the sums do not agree with the observer's on the eye's bands")
	}
	wrong := map[string]func([]shade) float64{
		"a weight": func(each []shade) float64 {
			each[len(each)/2].wy *= 1.0000001
			return unit
		},
		"the scale": func([]shade) float64 { return unit * 1.0000001 },
		"a middle": func(each []shade) float64 {
			each[1].centre = each[3].centre
			return unit
		},
	}
	for what, spoil := range wrong {
		changed := slices.Clone(shades)
		if got := spoil(changed); agreeing(light, changed, got) {
			t.Errorf("%s changed and the check did not notice", what)
		}
	}
}
