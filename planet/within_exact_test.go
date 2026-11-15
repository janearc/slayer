package planet

import (
	"math"
	"math/rand"
	"sync"
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// oldWithin is Face.Within as it was written: a map of the pixel's bands
// made for each call.
func oldWithin(face *Face, p maths.Vec, footprint float64) maths.Light {
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
			at := sight{mixed: mixed, star: star, eye: eye, turn: turn,
				tilts: face.tilted(nil, turn)}
			sent = face.sent(&at, i)
		}
		lo, hi := band.Edges()
		lines[centre(band)] = (face.light.Power(band)*reached*sent +
			face.glowed(marked.cleared, lo, hi) +
			face.glowedAt(came, eye, heldIn(lo, hi))*front) * face.exposure
	}
	red, green, blue := srgb.Light(observer.Swatch(bands.Lines(lines)))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}

// Within is the old arithmetic bit for bit, on a bare face, a two-zone face
// under a layer, and a lit one at an angle, at seeded random points and
// footprints, from many goroutines sharing the pool.
func TestWithinIsTheOldArithmeticBitForBit(t *testing.T) {
	dark := mustGas(t, "dark", flat(0), 0.02).Absorbing(flat(1e-30))
	layer, err := NewLayer(0, 1e3, []Gas{dark}, []float64{1}, Haze{})
	if err != nil {
		t.Fatal(err)
	}
	star := maths.V(0.3, 0.2, 1).Unit()
	faces := map[string]*Face{
		"bare":    NewFace(bareBody(t, 1, nil), maths.V(0, 1, 0), maths.V(0, 0, 1), maths.V(0, 0, 1e9), even(), 2),
		"layered": NewFace(bareBody(t, 0.6, []Layer{layer}), maths.V(0, 1, 0), star, star.Scale(1e12), even(), 1),
		"offset":  NewFace(bareBody(t, 0.3, []Layer{layer}), maths.V(0.1, 1, 0).Unit(), star, maths.V(1e9, 3e8, 1e9), even(), 0.7),
	}
	var wg sync.WaitGroup
	for name, face := range faces {
		for g := 0; g < 4; g++ {
			wg.Add(1)
			seed := int64(g)*7 + int64(len(name))
			go func() {
				defer wg.Done()
				r := rand.New(rand.NewSource(seed))
				for n := 0; n < 1500; n++ {
					p := maths.V(r.NormFloat64(), r.NormFloat64(), r.NormFloat64()).Unit().Scale(6e7)
					foot := 0.0
					if n%2 == 0 {
						foot = math.Exp(r.Float64()*10) * 10
					}
					got, want := face.Within(p, foot), oldWithin(face, p, foot)
					if math.Float64bits(got.R) != math.Float64bits(want.R) ||
						math.Float64bits(got.G) != math.Float64bits(want.G) ||
						math.Float64bits(got.B) != math.Float64bits(want.B) {
						t.Errorf("%s at %v footprint %v: %+v, was %+v", name, p, foot, got, want)
						return
					}
				}
			}()
		}
	}
	wg.Wait()
}

// narrow is starlight of one watt from 500 to 600 nm: fewer bands than even.
func narrow() bands.Light {
	spectrum := map[int]float64{}
	for nm := 500; nm <= 600; nm++ {
		spectrum[nm] = 1
	}
	return bands.Split(spectrum)
}

// a face with fewer bands, asked after a face with more on the same goroutine,
// shows nothing of the wider face's bands: the pooled map is emptied before
// each pixel. the two faces alternate so a stale entry would change a result.
func TestWithinLeavesNothingOfTheLastFaceInThePooledMap(t *testing.T) {
	star := maths.V(0, 0, 1)
	eye := star.Scale(1e9)
	wide := NewFace(bareBody(t, 1, nil), maths.V(0, 1, 0), star, eye, even(), 2)
	thin := NewFace(bareBody(t, 1, nil), maths.V(0, 1, 0), star, eye, narrow(), 2)
	r := rand.New(rand.NewSource(3))
	for n := 0; n < 400; n++ {
		p := maths.V(r.Float64()-0.5, r.Float64()-0.5, 1).Unit().Scale(6e7)
		for _, face := range []*Face{wide, thin, thin, wide} {
			got, want := face.Within(p, 0), oldWithin(face, p, 0)
			if math.Float64bits(got.R) != math.Float64bits(want.R) ||
				math.Float64bits(got.G) != math.Float64bits(want.G) ||
				math.Float64bits(got.B) != math.Float64bits(want.B) {
				t.Fatalf("at %v: %+v, was %+v", p, got, want)
			}
		}
	}
}
