package smoke

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/sheet"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// noise is how far a float32 frame can be from exact: a sum stored once
// against two stored separately.
const noise = 1e-6

// shot is smoke two's scene under the lamps given, rendered small, at a
// sextant's shape.
func shot(lamps ...thing.Lamp) *frame.Frame {
	w, eye := lampsOnPaint(lamps...)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	return trace.Render(w, eye, lens, 0, 120, 60)
}

// frames are pictures a check looked at, with the names to print them by.
type frames []struct {
	name string
	f    *frame.Frame
}

// contact puts what a test rendered on one contact sheet, when SLAYER_SHEET
// names a folder: go test with it set, and every picture a check looked at
// is there to look at too. Without it, nothing is written.
func contact(t *testing.T, fs frames) {
	dir := os.Getenv("SLAYER_SHEET")
	if dir == "" {
		return
	}
	s := sheet.New(len(fs))
	for _, f := range fs {
		s.Add(f.name, colour.Develop(f.f, 1), 0.75)
	}
	path := filepath.Join(dir, t.Name()+".png")
	if err := s.PNG(path); err != nil {
		t.Fatal(err)
	}
	t.Log("contact sheet:", path)
}

// Smoke two, two lamps: every check reads as the smoke it belongs to.
func TestSmoke2TwoLamps(t *testing.T) {
	red, green, both := shot(redLamp), shot(greenLamp),
		shot(redLamp, greenLamp)
	contact(t, frames{{"red lamp", red}, {"green lamp", green},
		{"both lamps", both}})

	t.Run("red light has no green or blue in it", func(t *testing.T) {
		for _, p := range red.Pix {
			if p.G > noise || p.B > noise {
				t.Fatalf("a point lit only by red is %v", p)
			}
		}
	})
	t.Run("green light has no red or blue in it", func(t *testing.T) {
		for _, p := range green.Pix {
			if p.R > noise || p.B > noise {
				t.Fatalf("a point lit only by green is %v", p)
			}
		}
	})
	t.Run("light adds, everywhere", func(t *testing.T) {
		for i, p := range both.Pix {
			r, g := red.Pix[i], green.Pix[i]
			if math.Abs(float64(p.R-r.R-g.R)) > noise ||
				math.Abs(float64(p.G-r.G-g.G)) > noise {
				t.Fatalf("pixel %d: both lamps give %v, "+
					"red %v, green %v", i, p, r, g)
			}
		}
	})
	t.Run("red, green, yellow and black all show", func(t *testing.T) {
		var redOnly, greenOnly, yellow, black int
		for _, p := range both.Pix {
			lit := func(v float32) bool { return v > 0.05 }
			switch {
			case lit(p.R) && lit(p.G):
				yellow++
			case lit(p.R):
				redOnly++
			case lit(p.G):
				greenOnly++
			case p.Cover > 0:
				black++
			}
		}
		if redOnly == 0 || greenOnly == 0 || yellow == 0 || black == 0 {
			t.Errorf("red %d, green %d, yellow %d, black %d",
				redOnly, greenOnly, yellow, black)
		}
	})
}

// shotOf is a smoke rendered small, at a sextant's shape.
func shotOf(n int) *frame.Frame {
	s, _ := Get(n)
	w, eye := s.Scene()
	lens := film.Pinhole{Wide: s.Wide, Pixel: 0.75}
	return trace.Sampled(w, eye, lens, 0, 120, 60, s.Samples)
}

// Smoke three, a ball on a set: the floor reads as a floor.
func TestSmoke3ABallOnASet(t *testing.T) {
	f := shotOf(3)
	contact(t, frames{{"a ball on a set", f}})

	t.Run("the sky is there", func(t *testing.T) {
		seen := 0
		for _, p := range f.Pix {
			if math.IsInf(float64(p.Depth), 1) {
				seen++
				if p.G < 0.5 {
					t.Fatalf("a piece of sky is %v", p)
				}
			}
		}
		if seen == 0 {
			t.Error("no sky in the picture")
		}
	})
	t.Run("the near floor is checkered", func(t *testing.T) {
		var light, dark int
		for x := 0; x < f.W; x++ {
			switch p := f.At(x, f.H-1); {
			case p.G > 0.6:
				light++
			case p.G < 0.45:
				dark++
			}
		}
		if light == 0 || dark == 0 {
			t.Errorf("the bottom row has %d light and %d dark",
				light, dark)
		}
	})
	t.Run("far is hazier than near", func(t *testing.T) {
		// the spread of the floor's light, near and far: haze pulls
		// both squares toward the sky, so far away they differ less
		spread := func(lo, hi float32) float32 {
			least, most := float32(math.Inf(1)), float32(0)
			for _, p := range f.Pix {
				if p.Depth >= lo && p.Depth < hi {
					least = min(least, p.G)
					most = max(most, p.G)
				}
			}
			return most - least
		}
		near, far := spread(0, 5), spread(40, 80)
		if !(far < near/2) {
			t.Errorf("near the squares differ by %v, far by %v",
				near, far)
		}
	})
}

// clearShot is smoke four's ball and floor under the lamps given, with no
// sky and no haze, so what a check sees is the lamps' light and nothing
// else. air is not light that adds, so the arithmetic is checked without
// it.
func clearShot(lamps ...thing.Lamp) *frame.Frame {
	w, eye := fourUnder(false, lamps...)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	return trace.Render(w, eye, lens, 0, 120, 60)
}

// Smoke four, red, green and blue on the set.
func TestSmoke4RedGreenAndBlueOnTheSet(t *testing.T) {
	red, green, blue := clearShot(fourRed), clearShot(fourGreen),
		clearShot(fourBlue)
	all := clearShot(fourRed, fourGreen, fourBlue)
	contact(t, frames{{"red lamp", red}, {"green lamp", green},
		{"blue lamp", blue}, {"all three", all},
		{"on the set", shotOf(4)}})

	t.Run("each lamp is its own colour", func(t *testing.T) {
		for i := range all.Pix {
			r, g, b := red.Pix[i], green.Pix[i], blue.Pix[i]
			if r.G+r.B+g.R+g.B+b.R+b.G > noise {
				t.Fatalf("pixel %d: red %v, green %v, "+
					"blue %v", i, r, g, b)
			}
		}
	})
	t.Run("the three add, everywhere", func(t *testing.T) {
		for i, p := range all.Pix {
			r, g, b := red.Pix[i], green.Pix[i], blue.Pix[i]
			if math.Abs(float64(p.R-r.R)) > noise ||
				math.Abs(float64(p.G-g.G)) > noise ||
				math.Abs(float64(p.B-b.B)) > noise {
				t.Fatalf("pixel %d: all three give %v", i, p)
			}
		}
	})
	t.Run("every mixture shows", func(t *testing.T) {
		// by which of the three lamps reach a point: all, each pair
		// with one lamp in shadow, and none under the ball
		seen := map[[3]bool]int{}
		for _, p := range all.Pix {
			if p.Cover == 0 {
				continue
			}
			lit := func(v float32) bool { return v > 0.05 }
			seen[[3]bool{lit(p.R), lit(p.G), lit(p.B)}]++
		}
		for name, k := range map[string][3]bool{
			"white":   {true, true, true},
			"yellow":  {true, true, false},
			"cyan":    {false, true, true},
			"magenta": {true, false, true},
			"black":   {false, false, false},
		} {
			if seen[k] == 0 {
				t.Errorf("no %s anywhere", name)
			}
		}
	})
}

// Smoke five, a spotlight on the set: a warm pool, a cool fill, and no
// shadow black.
func TestSmoke5ASpotlightOnTheSet(t *testing.T) {
	spot, flood := clearShot(fiveSpot), clearShot(fiveFlood)
	both := clearShot(fiveSpot, fiveFlood)
	contact(t, frames{{"the spot", spot}, {"the flood", flood},
		{"both", both}, {"on the set", shotOf(5)}})

	t.Run("the two add, everywhere", func(t *testing.T) {
		for i, p := range both.Pix {
			s, f := spot.Pix[i], flood.Pix[i]
			if math.Abs(float64(p.R-s.R-f.R)) > noise ||
				math.Abs(float64(p.G-s.G-f.G)) > noise ||
				math.Abs(float64(p.B-s.B-f.B)) > noise {
				t.Fatalf("pixel %d: both give %v", i, p)
			}
		}
	})
	t.Run("the spot alone leaves shadow", func(t *testing.T) {
		black := 0
		for _, p := range spot.Pix {
			if p.Cover > 0 && p.R+p.G+p.B == 0 {
				black++
			}
		}
		if black == 0 {
			t.Error("the spot lights everything: no shadow")
		}
	})
	t.Run("the flood leaves nothing black", func(t *testing.T) {
		for i, p := range both.Pix {
			if p.Cover > 0 && p.R+p.G+p.B == 0 {
				t.Fatalf("pixel %d is black under the flood",
					i)
			}
		}
	})
	t.Run("the pool is warm, the shadow cool", func(t *testing.T) {
		var warm, cool int
		for i, p := range both.Pix {
			if p.Cover == 0 {
				continue
			}
			if spot.Pix[i].R > 0.3 && p.R > p.B {
				warm++
			}
			if spot.Pix[i].R == 0 && p.B > p.R {
				cool++
			}
		}
		if warm == 0 || cool == 0 {
			t.Errorf("%d warm lit pixels, %d cool shadowed ones",
				warm, cool)
		}
	})
}

// Smoke six, a world of hexes: tiles on the ball, lines between them, and
// a sky in facets that still meets at the seams.
func TestSmoke6AWorldOfHexes(t *testing.T) {
	w, eye := sixUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	ball := trace.Render(w, eye, lens, 0, 120, 60)
	whole := shotOf(6)
	contact(t, frames{{"the ball alone", ball}, {"on the set", whole}})

	t.Run("the ball is sea and moss", func(t *testing.T) {
		sea, moss := 0, 0
		for _, p := range ball.Pix {
			if p.Cover == 0 || p.Depth > 6 {
				continue // sky, or the floor
			}
			switch {
			case p.B > p.G && p.B > p.R:
				sea++
			case p.G > p.B && p.G > p.R:
				moss++
			}
		}
		if sea == 0 || moss == 0 {
			t.Errorf("%d sea pixels and %d moss", sea, moss)
		}
	})
	t.Run("lines run between the tiles", func(t *testing.T) {
		// a line pixel is much darker than one of its neighbours on
		// the same surface, which neither the paint nor the sun does
		// by itself across one pixel
		dark := 0
		for y := 1; y < ball.H-1; y++ {
			for x := 1; x < ball.W-1; x++ {
				p := ball.At(x, y)
				if p.Cover == 0 || p.Depth > 6 {
					continue
				}
				q := ball.At(x+1, y)
				if q.Cover == 0 || q.Depth > 6 {
					continue
				}
				a, b := p.Light().Luminance(), q.Light().Luminance()
				if a < 0.6*b || b < 0.6*a {
					dark++
				}
			}
		}
		if dark < 20 {
			t.Errorf("only %d pixels sit on a line", dark)
		}
	})
	t.Run("the sky is in facets that meet", func(t *testing.T) {
		w, _ := sixUnder(true)
		plain := sky.Gradient{
			Horizon: swatches.Light(swatches.Horizon),
			Zenith:  swatches.Light(swatches.Zenith),
			Ground:  swatches.Light(swatches.Horizon),
		}
		// up the middle of a tile the facet is flat and is not the
		// gradient; at the seam it is the gradient
		tile := sixSky.Nearest(maths.V(0.2, 1, 0.1))
		c := sixSky.Centre(tile)
		off := c.Add(maths.V(0.01, 0, 0.01)).Unit()
		if w.Sky.Light(c) != w.Sky.Light(off) {
			t.Errorf("the facet is not flat: %v then %v",
				w.Sky.Light(c), w.Sky.Light(off))
		}
		if w.Sky.Light(off) == plain.Light(off) {
			t.Errorf("the facet is the plain sky at %v", off)
		}
		nb := sixSky.Centre(sixSky.Neighbours(tile)[0])
		seam := c.Add(nb).Unit()
		got, want := w.Sky.Light(seam), plain.Light(seam)
		if math.Abs(float64(got.G-want.G)) > 1e-9 {
			t.Errorf("at the seam %v, the sky is %v", got, want)
		}
	})
}

// Smoke seven, a planet: sea, land and the pale biomes all show, and the
// world under the paint is the one the world package made.
func TestSmoke7APlanet(t *testing.T) {
	w, eye := sevenUnder(false, false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	alone := trace.Render(w, eye, lens, 0, 120, 60)
	contact(t, frames{{"the planet alone", alone}, {"on the set",
		shotOf(7)}})

	t.Run("sea, land and the pale biomes show", func(t *testing.T) {
		sea, land, pale := 0, 0, 0
		for _, p := range alone.Pix {
			if p.Cover == 0 {
				continue
			}
			l := p.Light()
			lo := min(l.R, l.G, l.B)
			hi := max(l.R, l.G, l.B)
			switch {
			case lo > 0.3 && hi-lo < 0.15*hi: // snow, ice, beach
				pale++
			case l.B > l.G && l.B > l.R:
				sea++
			case l.G > l.R && l.G > l.B:
				land++
			}
		}
		if sea == 0 || land == 0 || pale == 0 {
			t.Errorf("%d sea, %d land, %d pale pixels", sea, land, pale)
		}
	})
	t.Run("the paint is the world's biomes", func(t *testing.T) {
		planet := world.New(sevenTerrain)
		look := sevenLook()
		tx := planet.Painted(look)
		for _, tile := range planet.Tiles()[:50] {
			c := planet.Tiling.Centre(tile)
			if got, want := tx.At(c), look.Paints[planet.Biome(tile)]; got != want {
				t.Fatalf("tile %v: %v, want %v", tile, got, want)
			}
		}
	})
}

// Smoke eight, a floor and a ring in hexes: the plane tiling shows on the
// floor, the band tiling shows on the ring, inside and out.
func TestSmoke8AFloorAndARingInHexes(t *testing.T) {
	w, eye := eightUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 120, 60)
	contact(t, frames{{"bare", bare}, {"on the set", shotOf(8)}})

	t.Run("the floor is chalk and slate in tiles", func(t *testing.T) {
		light, dark := 0, 0
		for y := bare.H * 3 / 4; y < bare.H; y++ {
			for x := 0; x < bare.W; x++ {
				p := bare.At(x, y)
				if p.Cover == 0 || p.Depth > 12 {
					continue
				}
				switch l := p.Light().Luminance(); {
				case l > 0.5:
					light++
				case l > 0.05 && l < 0.3:
					dark++
				}
			}
		}
		if light == 0 || dark == 0 {
			t.Errorf("the near floor has %d light and %d dark pixels",
				light, dark)
		}
	})
	t.Run("the ring is sea and moss", func(t *testing.T) {
		sea, moss := 0, 0
		for _, p := range bare.Pix {
			if p.Cover == 0 {
				continue
			}
			l := p.Light()
			switch {
			case l.B > l.G && l.B > l.R && l.B > 0.05:
				sea++
			case l.G > l.R && l.G > l.B && l.G > 0.05:
				moss++
			}
		}
		if sea == 0 || moss == 0 {
			t.Errorf("%d sea and %d moss pixels on the ring", sea, moss)
		}
	})
	t.Run("the band wraps the tube without a seam", func(t *testing.T) {
		// every column of the band has a tile in every row, and the
		// tube's inside and outside at one angle are the same tile
		for _, tile := range eightRing.Tiles() {
			c := eightRing.Centre(tile)
			in := maths.V(c.X*0.95, c.Y, c.Z*0.95)
			out := maths.V(c.X*1.05, c.Y, c.Z*1.05)
			if eightRing.Nearest(in) != tile || eightRing.Nearest(out) != tile {
				t.Fatalf("tile %v differs inside and out", tile)
			}
		}
	})
}

// Smoke nine, smoke on the set: the puff is seen against the sky and is
// white, and where it hangs over the floor the checker is smoothed away.
func TestSmoke9SmokeOnTheSet(t *testing.T) {
	w, eye := nineUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 120, 60)
	contact(t, frames{{"bare", bare}, {"on the set", shotOf(9)}})
	plain, _ := nineUnder(false)
	plain.Volumes = nil
	ref := trace.Render(plain, eye, lens, 0, 120, 60)

	t.Run("the smoke is there and is white", func(t *testing.T) {
		hazy := 0
		for _, p := range bare.Pix {
			if p.Cover > 0.05 && p.Cover < 0.95 && math.IsInf(float64(p.Depth), 1) {
				hazy++
				l := p.Light()
				hi, lo := max(l.R, l.G, l.B), min(l.R, l.G, l.B)
				if hi == 0 || hi-lo > 0.2*hi {
					t.Fatalf("a wisp against the sky is %+v", p)
				}
			}
		}
		if hazy < 5 {
			t.Errorf("only %d part-covered sky pixels", hazy)
		}
	})
	t.Run("the smoke lights the floor and shadows it", func(t *testing.T) {
		// against the same floor with no smoke: some floor pixels are
		// brighter, lit smoke in front of them, and some are darker,
		// the smoke's shadow on them
		lit, shadowed := 0, 0
		for i, p := range bare.Pix {
			q := ref.Pix[i]
			if q.Cover == 0 || q.Depth > 12 {
				continue
			}
			x, y := p.Light().Luminance(), q.Light().Luminance()
			switch {
			case x > 1.1*y:
				lit++
			case x < 0.7*y:
				shadowed++
			}
		}
		if lit < 10 || shadowed < 10 {
			t.Errorf("%d floor pixels lit by the smoke, %d in its shadow",
				lit, shadowed)
		}
	})
}

// Smoke ten, a gas giant: it has no surface, its middle is opaque and its
// limb is not, and its bands differ by latitude.
func TestSmoke10AGasGiant(t *testing.T) {
	w, eye := tenUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 160, 80)
	contact(t, frames{{"bare", bare}, {"on the set", shotOf(10)}})

	t.Run("opaque in the middle, thin at the limb", func(t *testing.T) {
		mid := bare.At(80, 40)
		if mid.Cover < 0.99 || !math.IsInf(float64(mid.Depth), 1) {
			t.Fatalf("the middle of the giant: %+v", mid)
		}
		partial := 0
		for _, p := range bare.Pix {
			if p.Cover > 0.1 && p.Cover < 0.9 {
				partial++
			}
		}
		if partial < 20 {
			t.Errorf("only %d limb pixels are part covered", partial)
		}
	})
	t.Run("the storm shows, and it is red", func(t *testing.T) {
		red := 0
		for _, p := range bare.Pix {
			if p.Cover < 0.99 {
				continue
			}
			if l := p.Light(); l.R > 2*l.G && l.R > 0.05 {
				red++
			}
		}
		if red < 3 {
			t.Errorf("only %d storm pixels", red)
		}
	})
	t.Run("it has a night side", func(t *testing.T) {
		// the sun is up and to the right; the lower left of the disc is
		// darker than the upper right
		lit := bare.At(100, 25).Light().Luminance()
		dark := bare.At(60, 55).Light().Luminance()
		if !(dark < 0.6*lit) {
			t.Errorf("toward the sun %v, away %v", lit, dark)
		}
	})
	t.Run("the bands differ by latitude", func(t *testing.T) {
		seen := map[int]bool{}
		for y := 10; y < 70; y++ {
			p := bare.At(80, y)
			if p.Cover < 0.99 {
				continue
			}
			l := p.Light()
			seen[int(math.Round(l.R/math.Max(1e-9, l.B)*4))] = true
		}
		if len(seen) < 3 {
			t.Errorf("down the middle the bands are %d colours", len(seen))
		}
	})
}

// Smoke eleven, a ringed giant: the rings show on both sides of the
// giant, the side behind it from the sun lies in its shadow, and the rings
// fall as a shadow across the giant.
func TestSmoke11ARingedGiant(t *testing.T) {
	w, eye := elevenUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 180, 90)
	contact(t, frames{{"bare", bare}, {"on the set", shotOf(11)}})

	// ring pixels: part covered, nothing solid behind, left and right of
	// the middle. the giant leans away, so the near rings pass above it
	// and the far rings are the lower rows, where the giant's shadow
	// falls away from a sun in front and to the right
	var left, right, farLeft, farRight []float64
	for y := 0; y < bare.H; y++ {
		for x := 0; x < bare.W; x++ {
			p := bare.At(x, y)
			if p.Cover < 0.03 || p.Cover > 0.97 || !math.IsInf(float64(p.Depth), 1) {
				continue
			}
			l := p.Light().Luminance()
			far := y > bare.H/2+4
			if x < bare.W/2-20 {
				left = append(left, l)
				if far {
					farLeft = append(farLeft, l)
				}
			} else if x > bare.W/2+20 {
				right = append(right, l)
				if far {
					farRight = append(farRight, l)
				}
			}
		}
	}
	t.Run("rings on both sides", func(t *testing.T) {
		if len(left) < 20 || len(right) < 20 {
			t.Errorf("%d ring pixels left, %d right", len(left), len(right))
		}
	})
	t.Run("the giant's shadow lies across the rings", func(t *testing.T) {
		// the sun is to the right; the rings to the left pass behind
		// the giant from it, so their darkest pixels are darker than
		// the right's darkest, and the left is darker on the mean
		mean := func(xs []float64) float64 {
			s := 0.0
			for _, x := range xs {
				s += x
			}
			return s / float64(max(1, len(xs)))
		}
		if len(farLeft) < 10 || len(farRight) < 10 {
			t.Fatalf("%d far ring pixels left, %d right", len(farLeft),
				len(farRight))
		}
		if !(mean(farLeft) < 0.85*mean(farRight)) {
			t.Errorf("behind the giant the rings average %v on the left, "+
				"%v on the right", mean(farLeft), mean(farRight))
		}
	})
	t.Run("the rings fall as a shadow across the giant", func(t *testing.T) {
		plain, _ := elevenUnder(false)
		plain.Volumes = plain.Volumes[:1] // the giant alone
		ref := trace.Render(plain, eye, lens, 0, 180, 90)
		darker := 0
		for i, p := range bare.Pix {
			q := ref.Pix[i]
			// the giant's own face, solid in both, darker with the rings
			if q.Cover < 0.99 || p.Cover < 0.99 {
				continue
			}
			if p.Light().Luminance() < 0.7*q.Light().Luminance() {
				darker++
			}
		}
		if darker < 30 {
			t.Errorf("only %d of the giant's pixels lie in ring shadow", darker)
		}
	})
}

// Smoke twelve, a planet with features: the world under the paint has a
// volcano, a canyon, a shelf and a plateau where they were placed, and
// the lava shows in the picture.
func TestSmoke12APlanetWithFeatures(t *testing.T) {
	w, eye := twelveUnder(false)
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 160, 80)
	contact(t, frames{{"the planet alone", bare}, {"on the set",
		shotOf(12)}})

	t.Run("the features are on the world", func(t *testing.T) {
		planet := twelveWorld()
		counts := map[world.Biome]int{}
		for _, tile := range planet.Tiles() {
			counts[planet.Biome(tile)]++
		}
		if counts[world.Lava] == 0 || counts[world.Ash] == 0 {
			t.Errorf("lava %d, ash %d", counts[world.Lava], counts[world.Ash])
		}
		if counts[world.Rock] < 5 {
			t.Errorf("only %d rock tiles for a canyon and a cliff",
				counts[world.Rock])
		}
		if counts[world.Ice] == 0 {
			t.Error("no ice for the shelf")
		}
	})
	t.Run("the lava shows", func(t *testing.T) {
		lava := 0
		for _, p := range bare.Pix {
			if p.Cover == 0 {
				continue
			}
			if l := p.Light(); l.R > 2*l.G && l.R > 0.3 {
				lava++
			}
		}
		if lava < 3 {
			t.Errorf("only %d lava pixels", lava)
		}
	})
}

// Smoke thirteen, a ringed giant through a long lens: the rings span the
// share of the frame the lens says they should, and the long lens has
// flattened the picture against eleven's: the far side of the rings is
// nearly as wide as the near.
func TestSmoke13ThroughALongLens(t *testing.T) {
	w, eye := thirteen()
	w.Volumes = w.Volumes[:2] // the giant and the rings; no set
	w.Things, w.Sky = nil, nil
	w.Haze = thing.Haze{}
	lens := film.Lens{Focal: 200, Gauge: film.Super35, Pixel: 0.75}
	far := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"through a 200", far}, {"on the set", shotOf(13)}})

	// the leftmost and rightmost covered columns are the rings' edges
	span := func(f *frame.Frame) (int, int) {
		lo, hi := f.W, -1
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				if f.At(x, y).Cover > 0.05 {
					lo, hi = min(lo, x), max(hi, x)
				}
			}
		}
		return lo, hi
	}
	t.Run("the rings fill the share of the frame the lens says", func(t *testing.T) {
		lo, hi := span(far)
		got := float64(hi-lo+1) / float64(far.W)
		want := 2 * math.Atan(elevenRings.Outer/thirteenAway) / thirteenLens.Wide()
		if math.Abs(got-want) > 0.06 {
			t.Errorf("the rings span %.2f of the frame, the lens says %.2f",
				got, want)
		}
	})
	// the perspective the long lens takes out is in the picture and not
	// in a number: the giant's disc bounds the frame's extent the same
	// from anywhere, and the rings' near and far edges hide behind it.
	// look at the sheet beside eleven's.
}

// Smoke fourteen, a gas giant in detail: along a band there are many
// more colours than smoke ten's giant has, the storm is there, and so
// is the street of white ovals.
func TestSmoke14AGasGiantInDetail(t *testing.T) {
	w, eye := fourteenUnder(false)
	lens := film.Lens{Focal: 135, Gauge: film.Super35, Pixel: 0.75}
	bare := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"in detail", bare}, {"on the set", shotOf(14)}})

	t.Run("more detail across the face than ten's paint has", func(t *testing.T) {
		// the paints themselves, sampled round the giant at a degree a
		// step, which is about a pixel of the picture: free of the
		// shadow march's grain, which both pictures would share
		grainOf := func(tx texture.Texture) float64 {
			sum := 0.0
			var last maths.Light
			for i := 0; i <= 360; i++ {
				a := float64(i) * maths.Degree
				p := maths.V(math.Cos(a), 0.1, math.Sin(a)).Scale(tenGiant.Radius)
				l := tx.At(p)
				if i > 0 {
					sum += math.Abs(l.Luminance() - last.Luminance())
				}
				last = l
			}
			return sum / 360
		}
		if a, b := grainOf(tenBands), grainOf(fourteenBands); !(b > 1.25*a) {
			t.Errorf("ten's paint changes %.4f a degree, fourteen's %.4f", a, b)
		}
	})
	t.Run("the storm and the street show", func(t *testing.T) {
		red, white := 0, 0
		for _, p := range bare.Pix {
			if p.Cover < 0.99 {
				continue
			}
			l := p.Light()
			if l.R > 2*l.G && l.R > 0.05 {
				red++
			}
			// an oval is white: grey to within a tenth, and lit
			if lo, hi := min(l.R, l.G, l.B), max(l.R, l.G, l.B); hi > 0 &&
				lo/hi > 0.9 && lo > 0.15 {
				white++
			}
		}
		if red < 3 || white < 3 {
			t.Errorf("%d storm pixels, %d white oval pixels", red, white)
		}
	})
}

// Smoke fifteen, the giants: four bodies in a row, saturn's rings wider
// than jupiter, uranus and neptune small beside them, against stars.
func TestSmoke15TheGiants(t *testing.T) {
	w, eye := fifteen()
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	f := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"the giants", f}})

	if len(w.Volumes) != 3+4+2+1 {
		t.Errorf("%d volumes: jupiter, its aurora and io's plume; saturn, its aurora, rings and titan's air; uranus and rings; neptune",
			len(w.Volumes))
	}
	moons := 0
	for _, th := range w.Things {
		if _, ok := th.Motion.(thing.Orbit); ok {
			moons++
		}
	}
	if moons != 4+3+3+1 {
		t.Errorf("%d moons on orbits", moons)
	}
	// stars behind: lit pixels that are not bodies, scattered
	stars := 0
	for _, p := range f.Pix {
		if p.Cover == 0 && p.Light().Luminance() > 0.05 {
			stars++
		}
	}
	if stars < 20 {
		t.Errorf("only %d star pixels", stars)
	}
	// the row is wider than it is tall, and covers the middle
	lo, hi := f.W, -1
	for x := 0; x < f.W; x++ {
		if f.At(x, f.H/2).Cover > 0.5 {
			lo, hi = min(lo, x), max(hi, x)
		}
	}
	if hi-lo < f.W/2 {
		t.Errorf("the row of giants spans only %d of %d columns", hi-lo, f.W)
	}
}

// Smoke sixteen, the rocks: five worlds and a moon or three, the earth
// blue, mars rust, venus pale; pluto and charon a pair.
func TestSmoke16TheRocks(t *testing.T) {
	w, eye := sixteen()
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	f := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"the rocks", f}})

	names := map[string]bool{}
	for _, th := range w.Things {
		names[th.Material.Name] = true
	}
	for _, n := range []string{"mercury", "venus", "earth", "moon", "mars",
		"pluto", "charon"} {
		if !names[n] {
			t.Errorf("no %s", n)
		}
	}
	blue, rust := 0, 0
	for _, p := range f.Pix {
		if p.Cover < 0.99 {
			continue
		}
		l := p.Light()
		if l.B > 1.5*l.R && l.B > 0.05 {
			blue++
		}
		if l.R > 1.6*l.B && l.R > 1.2*l.G && l.R > 0.05 {
			rust++
		}
	}
	if blue < 20 || rust < 20 {
		t.Errorf("%d blue pixels for the earth, %d rust for mars", blue, rust)
	}
}

// Smoke seventeen, moons on orbits: ten hours of the giant and its moons
// on one sheet, a frame every two hours; between frames the moons have
// moved by what kepler says, and the picture has changed where they
// are and nowhere else.
func TestSmoke17MoonsOnOrbits(t *testing.T) {
	w, eye := seventeen()
	lens := film.Pinhole{Wide: 50 * maths.Degree, Pixel: 0.75}
	var fs frames
	var shots []*frame.Frame
	for h := 0; h <= 10; h += 2 {
		at := time.Duration(h) * time.Hour
		f := trace.Sampled(w, eye, lens, at, 180, 75, 2)
		shots = append(shots, f)
		fs = append(fs, struct {
			name string
			f    *frame.Frame
		}{fmt.Sprintf("t = %d h", h), f})
	}
	contact(t, fs)

	t.Run("the moons go round by kepler", func(t *testing.T) {
		j := seventeenGiant()
		moons := 0
		for _, th := range w.Things {
			o, ok := th.Motion.(thing.Orbit)
			if !ok {
				continue
			}
			moons++
			want := thing.Period(j.Mass, o.Radius/(1/j.Radius)*1000)
			if o.Period != want {
				t.Errorf("%s goes round in %v, kepler says %v",
					th.Material.Name, o.Period, want)
			}
			turned := o.Angle(2*time.Hour) - o.Angle(0)
			if math.Abs(turned-2*math.Pi*2/o.Period.Hours()) > 1e-9 {
				t.Errorf("%s turned %v in two hours", th.Material.Name, turned)
			}
			if o.Period.Hours() < 3 || o.Period.Hours() > 60 {
				t.Errorf("%s goes round in %.1f hours, which the sheet "+
					"cannot show", th.Material.Name, o.Period.Hours())
			}
		}
		if moons != 3 {
			t.Errorf("%d moons", moons)
		}
	})
	t.Run("the picture changes between frames", func(t *testing.T) {
		for i := 1; i < len(shots); i++ {
			changed := 0
			for k := range shots[i].Pix {
				if shots[i].Pix[k] != shots[i-1].Pix[k] {
					changed++
				}
			}
			if changed < 20 || changed > len(shots[i].Pix)/4 {
				t.Errorf("between frames %d and %d, %d pixels changed",
					i-1, i, changed)
			}
		}
	})
}

// Smoke eighteen, hubble looks at jupiter: the disc is a fifth of the
// field, as film.Hubble()'s lens makes it, which is not quite hubble's:
// film/lens.go gives that field as 220 arcseconds where WFC3's is 162,
// so the real disc would be 0.28 of its field and this one is about a
// quarter small. the moons are points beside it; io's shadow is on the
// disc; stars are behind.
func TestSmoke18HubbleLooksAtJupiter(t *testing.T) {
	w, eye := eighteen()
	lens := film.Hubble()
	lens.Pixel = 0.75
	f := trace.Render(w, eye, lens, 0, 358, 150)
	contact(t, frames{{"through hubble", f}})

	// the disc: the solid run across the middle row
	lo, hi := f.W, -1
	for x := 0; x < f.W; x++ {
		if f.At(x, f.H/2).Cover > 0.9 {
			lo, hi = min(lo, x), max(hi, x)
		}
	}
	t.Run("jupiter is a fifth of film's field", func(t *testing.T) {
		// the expected share is worked from film.Hubble().Wide(): this
		// checks the smoke against film's lens, and says nothing of
		// hubble's, whose field is 162 arcseconds (film/lens.go)
		got := float64(hi-lo+1) / float64(f.W)
		want := 2 * math.Atan(1/(eighteenAway*eighteenAU/template.Jupiter().Radius)) /
			film.Hubble().Wide()
		if math.Abs(got-want) > 0.015 {
			t.Errorf("the disc spans %.3f of the field, film's lens says %.3f", got, want)
		}
	})
	t.Run("the moons are points beside the disc", func(t *testing.T) {
		points := 0
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				p := f.At(x, y)
				if p.Cover > 0.3 && !math.IsInf(float64(p.Depth), 1) &&
					(x < lo-2 || x > hi+2) {
					points++
				}
			}
		}
		if points < 2 || points > 60 {
			t.Errorf("%d moon pixels beside the disc", points)
		}
	})
	t.Run("io's shadow is on the disc", func(t *testing.T) {
		plain, _ := eighteen()
		plain.Things = nil // the moons away
		g := trace.Render(plain, eye, lens, 0, 358, 150)
		darker := 0
		for i, p := range f.Pix {
			q := g.Pix[i]
			if q.Cover < 0.99 || math.IsInf(float64(p.Depth), 1) == false {
				continue
			}
			if p.Light().Luminance() < 0.6*q.Light().Luminance() {
				darker++
			}
		}
		// io is a pixel across at this scale, so its shadow is about two
		if darker < 2 {
			t.Errorf("only %d disc pixels lie in a moon's shadow", darker)
		}
	})
	t.Run("stars are behind", func(t *testing.T) {
		stars := 0
		for _, p := range f.Pix {
			if p.Cover == 0 && p.Light().Luminance() > 0.05 {
				stars++
			}
		}
		if stars < 5 {
			t.Errorf("only %d star pixels", stars)
		}
	})
}

// Smoke nineteen, a satellite looks down: the ground is in tiles a few
// kilometres across, many of them in the strip, in more than one class,
// with land and sea both in the picture.
func TestSmoke19ASatelliteLooksDown(t *testing.T) {
	w, eye := nineteen()
	lens := nineteenLens
	lens.Pixel = 0.75
	f := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"from four hundred kilometres", f}})

	t.Run("the ground is many tiles across", func(t *testing.T) {
		// along the middle row the paint changes at tile lines: count
		// the runs of one colour
		runs, last := 0, maths.Light{}
		for x := 0; x < f.W; x++ {
			l := f.At(x, f.H/2).Light()
			if x == 0 || math.Abs(l.R-last.R)+math.Abs(l.G-last.G)+math.Abs(l.B-last.B) > 0.02 {
				runs++
			}
			last = l
		}
		if runs < 6 {
			t.Errorf("the middle row is %d runs of colour", runs)
		}
	})
	t.Run("land and sea are both in the strip", func(t *testing.T) {
		sea, land := 0, 0
		for _, p := range f.Pix {
			l := p.Light()
			switch {
			case l.B > l.G && l.B > l.R:
				sea++
			case l.G > l.B || l.R > l.B:
				land++
			}
		}
		if sea < len(f.Pix)/10 || land < len(f.Pix)/10 {
			t.Errorf("%d sea pixels, %d land, of %d", sea, land, len(f.Pix))
		}
	})
	t.Run("everything is ground", func(t *testing.T) {
		for i, p := range f.Pix {
			if p.Cover < 0.99 {
				t.Fatalf("pixel %d is not ground: %+v", i, p)
			}
		}
	})
}

// Smoke twenty, a magnetar behind a giant: in visible light the giant
// hides the magnetar; in gamma the magnetar shines through the gas and
// the moon is black on it; in x-ray the pulsar shows through the limb
// only; and the three compose in false colour.
func TestSmoke20AMagnetarBehindAGiant(t *testing.T) {
	w, eye := twenty()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	gamma := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyGamma)
	xray := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyXRay)
	both := colour.False([]colour.Layer{{Frame: vis}, {Band: twentyGamma, Frame: gamma},
		{Band: twentyXRay, Frame: xray}})
	contact(t, frames{{"visible", vis}, {"gamma", gamma}, {"x-ray", xray}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(1)
		s.Add("false colour: visible, gamma, x-ray", both, 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-composite.png")); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("in visible light the giant hides the magnetar", func(t *testing.T) {
		// where the gamma is strongest, the visible shows the giant's
		// gas: covered, nothing solid, and lit by the sun, not glowing
		// the magnetar's blue
		x, y, top := 0, 0, float32(0)
		for yy := 0; yy < gamma.H; yy++ {
			for xx := 0; xx < gamma.W; xx++ {
				if g := gamma.At(xx, yy).R; g > top {
					top, x, y = g, xx, yy
				}
			}
		}
		p := vis.At(x, y)
		if p.Cover < 0.99 || p.B > p.R {
			t.Errorf("at the brightest gamma, the visible is %+v", p)
		}
	})
	t.Run("in gamma the magnetar shines through the gas", func(t *testing.T) {
		bright, through := 0, 0
		for _, p := range gamma.Pix {
			if p.R > 0.1 {
				bright++
			}
			if p.R > 0.1 && p.R < 1.25 {
				through++
			}
		}
		if bright < 20 || through < bright/2 {
			t.Errorf("%d bright gamma pixels, %d of them dimmed by gas", bright, through)
		}
	})
	t.Run("in gamma the moon is black", func(t *testing.T) {
		for i, p := range gamma.Pix {
			if q := vis.Pix[i]; !math.IsInf(float64(q.Depth), 1) && q.Depth < 6 && p.R != 0 {
				t.Fatalf("pixel %d is on the moon and gives %v in gamma", i, p.R)
			}
		}
	})
	t.Run("x-ray is its own picture", func(t *testing.T) {
		// the pulsar gives off x-rays and no gamma, so there are x-ray
		// pixels where gamma has none; and some of the pulsar's x-rays
		// come through the giant's limb, where the visible shows gas
		only, limb := 0, 0
		for i, p := range xray.Pix {
			if p.R > 0.05 && gamma.Pix[i].R == 0 {
				only++
				if q := vis.Pix[i]; q.Cover > 0.99 && q.Depth > 6 && p.R < 1.15 {
					limb++
				}
			}
		}
		if only < 10 || limb < 3 {
			t.Errorf("%d x-ray-only pixels, %d of them through the limb", only, limb)
		}
	})
}

// Smoke twenty-one, magellan maps venus: in visible light the disc is
// one pale colour, the clouds; in radio the ground shows under them,
// varied, bright where rough, and dark toward the limb.
func TestSmoke21MagellanMapsVenus(t *testing.T) {
	w, eye := twentyOne()
	lens := film.Pinhole{Wide: 30 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	radio := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyOneBand)
	contact(t, frames{{"visible", vis}, {"radio", radio}})

	// the disc: where the radio picture has ground
	var disc []int
	for i, p := range radio.Pix {
		if p.Cover > 0.99 {
			disc = append(disc, i)
		}
	}
	t.Run("in visible light the clouds hide the ground", func(t *testing.T) {
		var lo, hi float64 = 10, 0
		for _, i := range disc {
			l := vis.Pix[i].Light()
			if l.Luminance() < 0.05 {
				continue // the night side
			}
			// the clouds' hue: yellow, red over blue
			if l.R < l.B {
				t.Fatalf("pixel %d is not cloud: %v", i, l)
			}
			ratio := l.R / math.Max(1e-9, l.G)
			lo, hi = math.Min(lo, ratio), math.Max(hi, ratio)
		}
		if hi-lo > 0.15 {
			t.Errorf("the clouds' hue varies from %.2f to %.2f: ground shows", lo, hi)
		}
	})
	t.Run("in radio the ground shows, varied", func(t *testing.T) {
		seen := map[int]bool{}
		for _, i := range disc {
			seen[int(radio.Pix[i].R*40)] = true
		}
		if len(seen) < 8 {
			t.Errorf("the radio picture has %d levels on the disc", len(seen))
		}
	})
	t.Run("the limb returns less than the middle", func(t *testing.T) {
		mid := radio.At(radio.W/2, radio.H/2).R
		edge := float32(0)
		for x := 0; x < radio.W; x++ {
			if p := radio.At(x, radio.H/2); p.Cover > 0.99 {
				edge = p.R
				break
			}
		}
		if !(edge < 0.5*mid) {
			t.Errorf("the limb returns %v, the middle %v", edge, mid)
		}
	})
}

// Smokes twenty-two and twenty-three, the ringed giant on super 8 and
// on imax: the same 50 sees twelve times as much across on imax as on
// super 8, 70.41 mm of frame to 5.79; super 8 is barred narrower and
// grained, imax is barred less and clean.
func TestSmoke22And23TheGaugesSideBySide(t *testing.T) {
	w, eye := twentyTwo()
	shots := map[string]*frame.Frame{}
	for _, g := range []film.Gauge{film.Super8, film.IMAX} {
		lens := film.Lens{Focal: 50, Gauge: g, Pixel: 0.75}
		shots[g.Name] = trace.Render(w, eye, lens, 0, 240, 100)
	}
	contact(t, frames{{"a 50 on super 8", shots["super 8"]},
		{"a 50 on imax", shots["imax"]}})

	// the rings' span is measured with no set behind them, where the
	// giant and rings are the only cover
	bare, _ := elevenUnder(false)
	span := func(g film.Gauge) int {
		lens := film.Lens{Focal: 50, Gauge: g, Pixel: 0.75}
		f := trace.Render(bare, eye, lens, 0, 240, 100)
		lo, hi := f.W, -1
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				if f.At(x, y).Cover > 0.05 {
					lo, hi = min(lo, x), max(hi, x)
				}
			}
		}
		return hi - lo + 1
	}
	t.Run("a 50 sees twelve times as much on imax", func(t *testing.T) {
		// by hand: half a frame's tangent is its width over twice the
		// focal length, 5.79 over 100 on super 8 and 70.41 over 100 on
		// imax, so the ratio of what they see is 70.41 over 5.79, 12.16.
		// the rings' outer edge is 3.1 units at 54, a tangent of 0.0574
		// to each side, which fills 240 * 0.0574 / 0.0579 = 238
		// columns on super 8 and 240 * 0.0574 / 0.7041 = 19.6 on imax.
		//
		// a span counts the columns whose cover passes 0.05, and its soft
		// edge can put each end out by a column: 2 of 19.6 on imax, 10%,
		// and 1% on super 8. the worst integer spans, 236 on 21 and 239
		// on 18, give 11.2 and 13.3, both inside a tenth of 12.16; the
		// measured 238 and 20 give 11.9.
		a, b := span(film.Super8), span(film.IMAX)
		want := math.Tan(film.Lens{Focal: 50, Gauge: film.IMAX}.Wide()/2) /
			math.Tan(film.Lens{Focal: 50, Gauge: film.Super8}.Wide()/2)
		if got := float64(a) / float64(b); math.Abs(got-want) > 0.10*want {
			t.Errorf("the rings span %d on super 8 and %d on imax, a ratio of %.1f; the lenses say %.1f",
				a, b, got, want)
		}
	})
	t.Run("super 8 is grainy and imax is clean", func(t *testing.T) {
		grain := func(g film.Gauge) float64 {
			img := colour.Develop(shots[g.Name], 1)
			img = colour.Grained(img, g.Grain*float64(img.Bounds().Dx())/0.75, 0.35, 22)
			// the sky at the top is flat without grain: its spread is
			// the grain's
			var xs []float64
			for x := 0; x < img.Bounds().Dx(); x++ {
				xs = append(xs, float64(img.RGBAAt(x, 2).R))
			}
			mean := 0.0
			for _, x := range xs {
				mean += x
			}
			mean /= float64(len(xs))
			v := 0.0
			for _, x := range xs {
				v += (x - mean) * (x - mean)
			}
			return math.Sqrt(v / float64(len(xs)))
		}
		if s, i := grain(film.Super8), grain(film.IMAX); !(s > 3*i) {
			t.Errorf("super 8's sky spreads %.2f, imax's %.2f", s, i)
		}
	})
	t.Run("the frames are barred to their gauges", func(t *testing.T) {
		img := colour.Develop(shots["super 8"], 1)
		for _, g := range []film.Gauge{film.Super8, film.IMAX} {
			m := colour.Masked(img, g.Aspect(), 0.75)
			bars := 0
			for y := 0; y < m.Bounds().Dy(); y++ {
				if c := m.RGBAAt(0, y); c.R == 0 && c.G == 0 && c.B == 0 {
					bars++
				}
			}
			if bars != m.Bounds().Dy() {
				t.Errorf("%s: %d of %d rows barred at the edge", g.Name, bars, m.Bounds().Dy())
			}
		}
	})
}

// Smoke twenty-four, io looks up at jupiter: stars above, jupiter big
// in the upper frame, banded, with a lit side and a dark side, io's
// ground lit in sulfur hexes at the bottom, a plume on the horizon
// between, and half a circuit on, in jupiter's shadow, the ground dark.
func TestSmoke24IoLooksUpAtJupiter(t *testing.T) {
	w, eye := twentyFour()
	lens := film.Pinhole{Wide: twentyFourWide, Pixel: 0.75}
	day := trace.Render(w, eye, lens, 0, 240, 100)
	j := template.Jupiter()
	orbit := j.Orbit(0, maths.Vec{}, 1/j.Moons[0].Body.Radius)
	// where io sits straight behind jupiter from the sun.
	//
	// io starts at zero round its orbit from the plane's first axis, x,
	// toward its second, (0, 0.0546, -0.9985) with jupiter's tilt of 3.13
	// degrees; the sun (0.3, 0.05, 1) has 0.3 along the first and -0.9958
	// along the second, so away from it in the plane is atan2(0.9958,
	// -0.3), 1.8634 rad, 106.77 degrees from x toward the second axis.
	//
	// that is not an angle from the sun: io there is 174.3 degrees from it,
	// the other 5.7 being the sun's height out of the orbit's plane. 1.862
	// is 0.0014 rad, 0.08 degrees, short of the working.
	shadow := time.Duration(1.862 / (2 * math.Pi) * float64(orbit.Period))
	w2, eye2 := twentyFourAt(shadow)
	night := trace.Render(w2, eye2, lens, shadow, 240, 100)
	contact(t, frames{{"day", day}, {"in jupiter's shadow", night}})

	volume := func(p frame.Pixel) bool {
		return p.Cover > 0.05 && math.IsInf(float64(p.Depth), 1)
	}
	ground := func(p frame.Pixel) bool {
		return p.Cover > 0.5 && !math.IsInf(float64(p.Depth), 1)
	}
	horizon := -1
	for y := 0; y < day.H && horizon < 0; y++ {
		for x := 0; x < day.W; x++ {
			if ground(day.At(x, y)) {
				horizon = y
				break
			}
		}
	}
	if horizon < day.H*2/3 {
		t.Fatalf("the horizon is at row %d of %d", horizon, day.H)
	}
	t.Run("stars above", func(t *testing.T) {
		// the sky is sparse by day: the top twelve rows, any star
		stars, dark := 0, 0
		for y := 0; y < 12; y++ {
			for x := 0; x < day.W; x++ {
				p := day.At(x, y)
				if p.Cover > 0 {
					continue
				}
				if l := p.Light().Luminance(); l > 0.05 {
					stars++
				} else if l < 0.01 {
					dark++
				}
			}
		}
		if stars < 3 || dark < 12*day.W*8/10 {
			t.Errorf("%d star pixels, %d dark, in the top rows", stars, dark)
		}
	})
	// jupiter down the middle column: the first run of volume from the
	// top; the plume is a second run, below it
	top, bottom := -1, -1
	for y := 0; y < horizon; y++ {
		if !volume(day.At(day.W/2, y)) {
			if top >= 0 {
				break
			}
			continue
		}
		if top < 0 {
			top = y
		}
		bottom = y
	}
	t.Run("jupiter overhead, banded, lit and dark", func(t *testing.T) {
		if bottom-top < 40 {
			t.Fatalf("jupiter spans rows %d to %d", top, bottom)
		}
		hues := map[int]bool{}
		bright, dark := 0, 0
		y := (top + bottom) / 2
		for x := 0; x < day.W; x++ {
			p := day.At(x, y)
			if !volume(p) {
				continue
			}
			l := p.Light()
			hues[int(l.R*12)*10000+int(l.G*12)*100+int(l.B*12)] = true
			switch lum := l.Luminance(); {
			case lum > 0.08:
				bright++
			case lum < 0.02:
				dark++
			}
		}
		if len(hues) < 4 || bright < 20 || dark < 10 {
			t.Errorf("%d hues, %d bright and %d dark pixels across jupiter", len(hues), bright, dark)
		}
	})
	t.Run("the ground is io's, lit, in hexes", func(t *testing.T) {
		var sum maths.Light
		n := 0
		levels := map[int]bool{}
		for y := horizon; y < day.H; y++ {
			for x := 0; x < day.W; x++ {
				p := day.At(x, y)
				if !ground(p) {
					continue
				}
				l := p.Light()
				sum, n = sum.Add(l), n+1
				if y == day.H-1 {
					levels[int(l.R*40)*10000+int(l.G*40)*100+int(l.B*40)] = true
				}
			}
		}
		mean := sum.Scale(1 / float64(max(1, n)))
		if !(mean.R > mean.G && mean.G > mean.B) || mean.Luminance() < 0.1 {
			t.Errorf("the ground's mean colour is %v", mean)
		}
		if len(levels) < 5 {
			t.Errorf("the nearest row has %d levels: no hexes", len(levels))
		}
	})
	t.Run("a plume on the horizon under jupiter", func(t *testing.T) {
		lit := 0
		for y := bottom + 1; y < horizon; y++ {
			for x := 0; x < day.W; x++ {
				if p := day.At(x, y); volume(p) && p.Light().Luminance() > 0.02 {
					lit++
				}
			}
		}
		if lit < 10 {
			t.Errorf("%d lit plume pixels between jupiter and the horizon", lit)
		}
	})
	t.Run("in jupiter's shadow the ground is dark", func(t *testing.T) {
		mean := func(f *frame.Frame) float64 {
			sum, n := 0.0, 0
			for y := horizon; y < f.H; y++ {
				for x := 0; x < f.W; x++ {
					if p := f.At(x, y); ground(p) {
						sum, n = sum+p.Light().Luminance(), n+1
					}
				}
			}
			return sum / float64(max(1, n))
		}
		if d, s := mean(day), mean(night); !(s < 0.2*d) {
			t.Errorf("the ground is %.3f by day and %.3f in the shadow", d, s)
		}
	})
}

// lit is the lit volume pixels of a frame in a band of rows: how many,
// and their mean x and luminance.
func lit(f *frame.Frame, from, to int) (n int, meanX, meanLum float64) {
	sumX, sumL := 0.0, 0.0
	for y := from; y < to; y++ {
		for x := 0; x < f.W; x++ {
			p := f.At(x, y)
			if p.Cover > 0.05 && math.IsInf(float64(p.Depth), 1) {
				if l := p.Light().Luminance(); l > 0.02 {
					n, sumX, sumL = n+1, sumX+float64(x), sumL+l
				}
			}
		}
	}
	if n == 0 {
		return 0, 0, 0
	}
	return n, sumX / float64(n), sumL / float64(n)
}

// Smoke twenty-five, a quasar and its jets: a black hole in the
// middle, a disc blue-white inside and orange at the rim that hides the
// stars behind it and is brighter on the side coming toward the eye,
// jets above and below, narrow, the upper one brighter; in x-ray the
// jets alone.
func TestSmoke25AQuasarAndItsJets(t *testing.T) {
	w, eye := twentyFive()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	xray := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyXRay)
	contact(t, frames{{"visible", vis}, {"x-ray", xray}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(1)
		s.Add("through the tone", quasarTone.Develop(vis), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("the hole is black", func(t *testing.T) {
		if c := vis.At(120, 50); c.Light().Luminance() > 0.01 {
			t.Errorf("the middle is %v", c.Light())
		}
	})
	t.Run("the disc hides the stars and runs blue to orange", func(t *testing.T) {
		// along the middle row out from the hole to the right: the first
		// bright disc pixel and the last
		first, last := -1, -1
		for x := 121; x < vis.W; x++ {
			p := vis.At(x, 50)
			if p.Light().Luminance() > 0.05 && math.IsInf(float64(p.Depth), 1) {
				if first < 0 {
					first = x
				}
				last = x
			}
		}
		if first < 0 || last-first < 20 {
			t.Fatalf("the disc runs from %d to %d", first, last)
		}
		in, out := vis.At(first+2, 50), vis.At(last-2, 50)
		if in.Cover < 0.9 {
			t.Errorf("the inner disc lets the stars through: cover %v", in.Cover)
		}
		il, ol := in.Light(), out.Light()
		if !(il.B/il.R > ol.B/ol.R) || !(ol.R > ol.B) {
			t.Errorf("the inner disc is %v and the rim %v", il, ol)
		}
	})
	t.Run("the side coming toward the eye is brighter", func(t *testing.T) {
		left, right := 0.0, 0.0
		for y := 35; y < 65; y++ {
			for x := 0; x < vis.W; x++ {
				if p := vis.At(x, y); p.Cover > 0.5 && math.IsInf(float64(p.Depth), 1) {
					if x < 120 {
						left += p.Light().Luminance()
					} else {
						right += p.Light().Luminance()
					}
				}
			}
		}
		if ratio := math.Max(left, right) / math.Min(left, right); ratio < 1.3 {
			t.Errorf("the disc's halves are %.1f and %.1f: not beamed", left, right)
		}
	})
	t.Run("a narrow jet above, the counter-jet hidden", func(t *testing.T) {
		// the rows above and below the disc, which spans the middle: the
		// upper jet stands clear and bright; the lower one lies behind
		// the opaque disc from this side, as telescopes find them, but
		// for a faint stub the hole bends round the disc's edge, which
		// has under fifteen hundredths of the upper jet's light
		up, _, upLum := lit(vis, 0, 20)
		down, _, downLum := lit(vis, 80, 100)
		if up < 20 || upLum < 0.1 {
			t.Errorf("%d lit pixels above at %.3f: no jet", up, upLum)
		}
		if float64(down)*downLum > 0.15*float64(up)*upLum {
			t.Errorf("%d lit pixels below at %.3f: the counter-jet shows", down, downLum)
		}
		wide := 0
		for x := 0; x < vis.W; x++ {
			if p := vis.At(x, 10); p.Cover > 0.05 && p.Light().Luminance() > 0.05 {
				wide++
			}
		}
		if wide > 14 {
			t.Errorf("the upper jet is %d pixels wide ten rows down", wide)
		}
	})
	t.Run("in x-ray the jets alone", func(t *testing.T) {
		if x := xray.At(150, 50).R; x > 0.05 {
			t.Errorf("the disc shows in x-ray: %v", x)
		}
		n := 0
		for y := 0; y < 30; y++ {
			for x := 0; x < xray.W; x++ {
				if xray.At(x, y).R > 0.02 {
					n++
				}
			}
		}
		if n < 20 {
			t.Errorf("%d x-ray pixels in the upper jet", n)
		}
	})
}

// Smoke twenty-six, a blazar: the jet toward the eye outshines
// everything, a white blob wider than the disc, and the same in gamma.
func TestSmoke26ABlazarDownTheJet(t *testing.T) {
	w, eye := twentySix()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	gamma := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyGamma)
	contact(t, frames{{"visible", vis}, {"gamma", gamma}})
	// the blaze: the brightest pixel, five degrees off the middle
	bx, by, top := 0, 0, 0.0
	for y := 0; y < vis.H; y++ {
		for x := 0; x < vis.W; x++ {
			if l := vis.At(x, y).Light().Luminance(); l > top {
				top, bx, by = l, x, y
			}
		}
	}
	if c := vis.At(bx, by); c.R < 1 || c.G < 1 || c.B < 1 {
		t.Errorf("the blaze is %v: not blinding", c.Light())
	}
	wide := 0
	for x := 0; x < vis.W; x++ {
		if vis.At(x, by).Light().Luminance() > 0.9 {
			wide++
		}
	}
	if wide < 8 {
		t.Errorf("the blaze is %d pixels wide", wide)
	}
	if g := gamma.At(bx, by).R; g < 1 {
		t.Errorf("in gamma the blaze is %v", g)
	}
	// the blaze is beamed far past anything the quasar's jets reach
	// side-on, which is a few times white
	if top < 20 {
		t.Errorf("the blaze peaks at %.1f times white", top)
	}
}

// Smoke twenty-seven, a pulsar turns: a turn laid out in quarters, the
// beams sweeping, so that a quarter and three quarters in the upper
// beam leans opposite ways.
func TestSmoke27APulsarTurns(t *testing.T) {
	w, eye := twentySeven()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	p := template.Pulsar()
	var fs frames
	var quarters []*frame.Frame
	for i := range 4 {
		at := time.Duration(i) * p.Day / 4
		f := trace.Render(w, eye, lens, at, 240, 100)
		quarters = append(quarters, f)
		fs = append(fs, struct {
			name string
			f    *frame.Frame
		}{fmt.Sprintf("%d of 4", i), f})
	}
	contact(t, fs)
	n1, x1, _ := lit(quarters[1], 0, 45)
	n3, x3, _ := lit(quarters[3], 0, 45)
	if n1 < 30 || n3 < 30 {
		t.Fatalf("%d and %d lit pixels in the upper beam", n1, n3)
	}
	if !((x1-120)*(x3-120) < 0 && math.Abs(x1-120) > 8 && math.Abs(x3-120) > 8) {
		t.Errorf("the upper beam is at %.0f a quarter in and %.0f three quarters in", x1, x3)
	}
	star := 0.0
	for y := 48; y <= 51; y++ {
		for x := 118; x <= 121; x++ {
			if p := quarters[0].At(x, y); !math.IsInf(float64(p.Depth), 1) {
				star = math.Max(star, p.Light().Luminance())
			}
		}
	}
	if star < 0.3 {
		t.Errorf("the star is %.2f", star)
	}
}

// Smoke twenty-eight, a magnetar flares: over milliseconds the fireball
// grows, dim in visible light and bright in gamma, past the star.
func TestSmoke28AMagnetarFlares(t *testing.T) {
	w, eye := twentyEight()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	ages := []time.Duration{-1500 * time.Microsecond, 0, 2 * time.Millisecond}
	var fs frames
	var counts, gammas []int
	s := sheet.New(len(ages))
	for _, at := range ages {
		vis := trace.Render(w, eye, lens, at, 240, 100)
		gamma := trace.RenderIn(w, eye, lens, at, 240, 100, twentyGamma)
		name := fmt.Sprintf("%v after", at+2*time.Millisecond)
		fs = append(fs, struct {
			name string
			f    *frame.Frame
		}{name, vis})
		n, _, _ := lit(vis, 0, vis.H)
		g := 0
		for i := range gamma.Pix {
			if gamma.Pix[i].R > 0.1 {
				g++
			}
		}
		counts, gammas = append(counts, n), append(gammas, g)
		s.Add(name+", false colour", colour.False([]colour.Layer{{Frame: vis},
			{Band: twentyGamma, Frame: gamma}}), 0.75)
	}
	contact(t, fs)
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		if err := s.PNG(filepath.Join(dir, t.Name()+"-composite.png")); err != nil {
			t.Fatal(err)
		}
	}
	if !(counts[0] < counts[1] && counts[1] < counts[2]) {
		t.Errorf("the fireball covers %v pixels over time", counts)
	}
	if !(gammas[0] < gammas[1] && gammas[1] < gammas[2] && gammas[2] > 2*counts[2]/3) {
		t.Errorf("in gamma the fireball covers %v pixels, in visible %v", gammas, counts)
	}
}

// Smoke twenty-nine, the pillars of creation: columns standing apart
// at mid height, the lines glowing where the star reaches the gas and
// dark in the interiors, and the three lines each with a signal.
func TestSmoke29ThePillarsOfCreation(t *testing.T) {
	w, eye := twentyNine()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	fs := frames{{"visible", vis}}
	var layers []colour.Layer
	var lines []*frame.Frame
	for i, b := range twentyNineLines {
		f := trace.RenderIn(w, eye, lens, 0, 240, 100, b)
		lines = append(lines, f)
		layers = append(layers, colour.Layer{Band: b, Frame: f, Tint: twentyNinePalette[i]})
		fs = append(fs, struct {
			name string
			f    *frame.Frame
		}{twentyNineNames[i], f})
	}
	contact(t, fs)
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(1)
		s.Add("the hubble palette", twentyNineTone.False(layers), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-palette.png")); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("columns stand apart at mid height", func(t *testing.T) {
		runs, in := 0, false
		for x := 0; x < vis.W; x++ {
			c := vis.At(x, 50).Cover
			switch {
			case c > 0.5 && !in:
				runs, in = runs+1, true
			case c < 0.2 && in:
				in = false
			}
		}
		if runs < 2 {
			t.Errorf("%d columns at mid height", runs)
		}
	})
	t.Run("the lines glow at the tips more than down the columns", func(t *testing.T) {
		h := lines[1]
		first, last := h.H, -1
		for y := 0; y < h.H; y++ {
			for x := 0; x < h.W; x++ {
				if vis.At(x, y).Cover > 0.5 {
					first, last = min(first, y), max(last, y)
				}
			}
		}
		fifth := (last - first + 1) / 5
		mean := func(from, to int) float64 {
			sum, n := 0.0, 0
			for y := from; y < to; y++ {
				for x := 0; x < h.W; x++ {
					if vis.At(x, y).Cover > 0.5 {
						sum, n = sum+float64(h.At(x, y).R), n+1
					}
				}
			}
			return sum / float64(max(1, n))
		}
		// the top fifth of the columns against the middle fifth; the
		// wall at the foot has a lit top of its own
		mid := (first + last) / 2
		top, middle := mean(first, first+fifth), mean(mid-fifth/2, mid+fifth/2)
		if !(top > 0.05 && top > 1.2*middle) {
			t.Errorf("h alpha is %.3f at the tips and %.3f down the columns", top, middle)
		}
	})
	t.Run("the pillars are sulphur and the gas between is oxygen", func(t *testing.T) {
		// the hubble palette's reason: in the dense pillars, shaded,
		// sulphur outdoes oxygen; in the thin lit gas between, oxygen
		// comes up
		var denseS, denseO, thinS, thinO float64
		for y := 0; y < vis.H; y++ {
			for x := 0; x < vis.W; x++ {
				c := vis.At(x, y).Cover
				s, o := float64(lines[0].At(x, y).R), float64(lines[2].At(x, y).R)
				switch {
				case c > 0.5:
					denseS, denseO = denseS+s, denseO+o
				case c > 0.02 && c < 0.3:
					thinS, thinO = thinS+s, thinO+o
				}
			}
		}
		if denseS == 0 || thinS == 0 {
			t.Fatalf("sulphur sums: dense %v, thin %v", denseS, thinS)
		}
		if !(denseO/denseS < 0.9 && thinO/thinS > 1.5*denseO/denseS) {
			t.Errorf("oxygen over sulphur: %.2f in the pillars, %.2f between", denseO/denseS, thinO/thinS)
		}
	})
	t.Run("young stars at the tips, in every line", func(t *testing.T) {
		stars := 0
		for y := 0; y < vis.H; y++ {
			for x := 0; x < vis.W; x++ {
				p := vis.At(x, y)
				if !math.IsInf(float64(p.Depth), 1) && p.Light().Luminance() > 1 {
					stars++
					if lines[1].At(x, y).R < 0.5 {
						t.Fatalf("a star at %d,%d is dark in h alpha", x, y)
					}
				}
			}
		}
		if stars < 5 {
			t.Errorf("%d star pixels", stars)
		}
	})
	t.Run("three lines, each with a signal", func(t *testing.T) {
		for i, f := range lines {
			top := float32(0)
			for _, p := range f.Pix {
				top = max(top, p.R)
			}
			if top < 0.05 {
				t.Errorf("%s peaks at %v", twentyNineNames[i], top)
			}
		}
	})
}

// Smoke thirty, the earth by night, and by day: by day white cloud
// over blue and green and no city to be seen; by night, exposed two
// hundred times, dark ground pricked with warm city lights.
func TestSmoke30TheEarthByDayAndNight(t *testing.T) {
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	dayW, eye := thirtyLit(maths.V(1, 0.15, 0.1), false)
	day := trace.Render(dayW, eye, lens, 0, 240, 100)
	nightW, _ := thirty()
	night := trace.Render(nightW, eye, lens, 0, 240, 100)
	contact(t, frames{{"day", day}, {"night, as rendered", night}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(2)
		s.Add("by day", colour.Develop(day, 1), 0.75)
		s.Add("by night, exposed two hundred times", thirtyTone.Develop(night), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-exposed.png")); err != nil {
			t.Fatal(err)
		}
	}
	disc := func(p frame.Pixel) bool { return p.Cover > 0.5 }
	t.Run("clouds by day, and no cities", func(t *testing.T) {
		white, lit, warm := 0, 0, 0
		for y := 0; y < day.H; y++ {
			for x := 120; x < day.W; x++ {
				p := day.At(x, y)
				if !disc(p) {
					continue
				}
				l := p.Light()
				if l.Luminance() < 0.1 {
					continue
				}
				lit++
				if lo, hi := min(l.R, l.G, l.B), max(l.R, l.G, l.B); hi > 0.35 && lo/hi > 0.8 {
					white++
				}
				if l.R > 0.6 && l.R > 1.5*l.B {
					warm++
				}
			}
		}
		if lit < 1000 || white < lit/20 || white > lit*9/10 || warm > 5 {
			t.Errorf("%d white pixels of %d lit on the day side, %d warm", white, lit, warm)
		}
	})
	t.Run("cities by night", func(t *testing.T) {
		warm, dark := 0, 0
		for y := 0; y < night.H; y++ {
			for x := 0; x < night.W; x++ {
				p := night.At(x, y)
				if !disc(p) {
					continue
				}
				l := p.Light().Scale(200)
				switch {
				case l.Luminance() < 0.01:
					dark++
				case l.R > l.G && l.G > l.B && l.Luminance() > 0.05:
					warm++
				}
			}
		}
		if dark < 2000 || warm < 15 {
			t.Errorf("%d dark pixels and %d warm lights by night", dark, warm)
		}
	})
}

// Smoke thirty-one, the bullet cluster: galaxies in visible light, the
// hot gas in x-ray lagging toward the middle, the mass with the
// galaxies; the bullet's gas sits left of the bullet's mass.
func TestSmoke31TheBulletCluster(t *testing.T) {
	w, eye := thirtyOne()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	xray := trace.RenderIn(w, eye, lens, 0, 240, 100, thirtyOneBands[0])
	mass := trace.RenderIn(w, eye, lens, 0, 240, 100, thirtyOneBands[1])
	contact(t, frames{{"visible", vis}, {"x-ray", xray}, {"mass, drawn", mass}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(1)
		s.Add("the palette", thirtyOneTone.False([]colour.Layer{{Frame: vis},
			{Band: thirtyOneBands[0], Frame: xray, Tint: thirtyOnePalette[0]},
			{Band: thirtyOneBands[1], Frame: mass, Tint: thirtyOnePalette[1]}}), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-palette.png")); err != nil {
			t.Fatal(err)
		}
	}
	galaxies := 0
	for _, p := range vis.Pix {
		if !math.IsInf(float64(p.Depth), 1) && p.Light().Luminance() > 0.3 {
			galaxies++
		}
	}
	if galaxies < 30 {
		t.Errorf("%d galaxy pixels", galaxies)
	}
	// the centroid of a band over a span of columns
	centroid := func(f *frame.Frame, from, to int) float64 {
		sum, weight := 0.0, 0.0
		for y := 0; y < f.H; y++ {
			for x := from; x < to; x++ {
				v := float64(f.At(x, y).R)
				sum, weight = sum+v*float64(x), weight+v
			}
		}
		if weight == 0 {
			return -1
		}
		return sum / weight
	}
	gasR, massR := centroid(xray, 120, 240), centroid(mass, 120, 240)
	gasL, massL := centroid(xray, 0, 120), centroid(mass, 0, 120)
	if gasR < 0 || massR < 0 || gasL < 0 || massL < 0 {
		t.Fatalf("a band is empty: gas %.0f %.0f, mass %.0f %.0f", gasL, gasR, massL, massR)
	}
	if !(gasR < massR-4 && gasL > massL+4) {
		t.Errorf("the gas sits at %.0f and %.0f, the mass at %.0f and %.0f: the gas does not lag",
			gasL, gasR, massL, massR)
	}
}

// Smoke thirty-two, the aurora from orbit: green curtains between the
// eye's level and the limb, wide across the frame, cities on the dark
// ground; and jupiter's oval in blues over its pole.
func TestSmoke32TheAuroraFromOrbit(t *testing.T) {
	w, eye := thirtyTwo()
	lens := film.Pinhole{Wide: 60 * maths.Degree, Pixel: 0.75}
	earth := trace.Render(w, eye, lens, 0, 240, 100)
	jw, jeye := thirtyTwoJupiter()
	jlens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	jupiter := trace.Render(jw, jeye, jlens, 0, 240, 100)
	contact(t, frames{{"the earth", earth}, {"jupiter", jupiter}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(2)
		s.Add("the earth's aurora from orbit, a night exposure", thirtyTone.Develop(earth), 0.75)
		s.Add("jupiter's, in blues, exposed twenty times",
			(colour.Tone{Exposure: 20, Knee: 0.6, Top: 20, Bloom: 1}).Develop(jupiter), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-exposed.png")); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("green curtains across the frame, cities on the ground", func(t *testing.T) {
		green, warm := 0, 0
		left, right := false, false
		for y := 0; y < earth.H; y++ {
			for x := 0; x < earth.W; x++ {
				l := earth.At(x, y).Light().Scale(200)
				switch {
				case l.G > 1.3*l.R && l.G > 1.3*l.B && l.Luminance() > 0.1:
					green++
					if x < earth.W/4 {
						left = true
					}
					if x > earth.W*3/4 {
						right = true
					}
				case l.R > l.G && l.G > l.B && l.Luminance() > 0.05:
					warm++
				}
			}
		}
		if green < 200 || !left || !right || warm < 10 {
			t.Errorf("%d green pixels, left %v right %v; %d warm", green, left, right, warm)
		}
	})
	t.Run("jupiter's oval in blues over its pole", func(t *testing.T) {
		blue := 0
		for y := 0; y < jupiter.H; y++ {
			for x := 0; x < jupiter.W; x++ {
				l := jupiter.At(x, y).Light().Scale(20)
				if l.B > 1.2*l.R && l.B > 1.4*l.G && l.Luminance() > 0.1 {
					blue++
				}
			}
		}
		if blue < 50 {
			t.Errorf("%d blue pixels", blue)
		}
	})
}

// Smoke thirty-three, a world beside a magnetar: the aurora reaches
// far past the planet's disc, in green and red, the magnetar is a
// point too bright to hold, and in x-ray the aurora glows.
func TestSmoke33AWorldBesideAMagnetar(t *testing.T) {
	w, eye := thirtyThree()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	vis := trace.Render(w, eye, lens, 0, 240, 100)
	xray := trace.RenderIn(w, eye, lens, 0, 240, 100, twentyXRay)
	contact(t, frames{{"visible", vis}, {"x-ray", xray}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(2)
		s.Add("through the tone", thirtyThreeTone.Develop(vis), 0.75)
		s.Add("with the x-ray in cyan", thirtyThreeTone.False([]colour.Layer{{Frame: vis},
			{Band: twentyXRay, Frame: xray, Tint: maths.RGB(0.3, 0.9, 1)}}), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	// the planet's disc is about four degrees across at fourteen radii
	// through forty degrees: twenty-four pixels; the aurora must glow
	// well past twice that
	far, green, red := 0, 0, 0
	for y := 0; y < vis.H; y++ {
		for x := 0; x < vis.W; x++ {
			p := vis.At(x, y)
			if !(p.Cover > 0.05 && math.IsInf(float64(p.Depth), 1)) {
				continue
			}
			l := p.Light()
			if l.Luminance() < 0.05 {
				continue
			}
			dx, dy := float64(x-120), float64(y-50)*1.33
			if math.Hypot(dx, dy) > 30 {
				far++
			}
			switch {
			case l.G > 1.3*l.R && l.G > 1.3*l.B:
				green++
			case l.R > 1.3*l.G && l.R > 1.3*l.B:
				red++
			}
		}
	}
	if far < 1500 || green < 300 || red < 100 {
		t.Errorf("%d lit pixels far from the planet, %d green, %d red", far, green, red)
	}
	top, tx := float32(0), 0
	for y := 0; y < vis.H; y++ {
		for x := 0; x < vis.W; x++ {
			if l := vis.At(x, y).Light().Luminance(); float32(l) > top {
				top, tx = float32(l), x
			}
		}
	}
	if top < 5 || tx < 150 {
		t.Errorf("the brightest pixel is %v at column %d: no magnetar", top, tx)
	}
	glow := 0
	for _, p := range xray.Pix {
		if p.R > 0.05 {
			glow++
		}
	}
	if glow < 500 {
		t.Errorf("%d x-ray pixels", glow)
	}
}

// Smoke thirty-four, a hot jupiter: the sun's disc at the left, the
// planet small with its air streaming to the right away from the sun.
func TestSmoke34AHotJupiter(t *testing.T) {
	w, eye := thirtyFour()
	lens := film.Pinhole{Wide: 80 * maths.Degree, Pixel: 0.75}
	f := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"a hot jupiter", f}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(1)
		s.Add("through the tone", cometTone.Develop(f), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	// the planet: the surface pixels; its centre column
	sumX, n := 0, 0
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			if p := f.At(x, y); p.Cover > 0.5 && !math.IsInf(float64(p.Depth), 1) && p.Depth < 60 {
				sumX, n = sumX+x, n+1
			}
		}
	}
	if n < 10 {
		t.Fatalf("%d planet pixels", n)
	}
	planet := sumX / n
	tail, sun := 0, 0
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			p := f.At(x, y)
			l := p.Light().Luminance()
			switch {
			case p.Cover > 0.05 && math.IsInf(float64(p.Depth), 1) && l > 0.02 && x > planet+8:
				tail++
			case !math.IsInf(float64(p.Depth), 1) && l > 1 && x < f.W/3:
				sun++
			}
		}
	}
	if tail < 300 || sun < 200 {
		t.Errorf("%d tail pixels right of the planet at %d, %d sun pixels at the left", tail, planet, sun)
	}
}

// Smoke thirty-five, hale-bopp and hyakutake: a bright coma, a broad
// warm dust tail and a straight blue ion tail; and hyakutake's ion
// tail across the whole frame.
func TestSmoke35HaleBoppAndHyakutake(t *testing.T) {
	w, eye := thirtyFive()
	lens := film.Pinhole{Wide: 60 * maths.Degree, Pixel: 0.75}
	// two rays per pixel, so a head smaller than a pixel still lands
	hb := trace.Sampled(w, eye, lens, 0, 240, 100, 2)
	hw, heye := thirtyFiveHyakutake()
	hy := trace.Sampled(hw, heye, lens, 0, 240, 100, 2)
	contact(t, frames{{"hale-bopp", hb}, {"hyakutake", hy}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(2)
		s.Add("hale-bopp, through the tone", cometTone.Develop(hb), 0.75)
		s.Add("hyakutake, through the tone", cometTone.Develop(hy), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	count := func(f *frame.Frame) (blue, warm, span int) {
		first, last := f.W, -1
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				p := f.At(x, y)
				if !(p.Cover > 0.02 && math.IsInf(float64(p.Depth), 1)) {
					continue
				}
				l := p.Light()
				switch {
				case l.B > 1.3*l.R && l.Luminance() > 0.01:
					blue++
					first, last = min(first, x), max(last, x)
				case l.R > 1.05*l.B && l.Luminance() > 0.05:
					warm++
				}
			}
		}
		return blue, warm, last - first
	}
	if blue, warm, _ := count(hb); blue < 200 || warm < 150 {
		t.Errorf("hale-bopp: %d blue pixels, %d warm", blue, warm)
	}
	if blue, _, span := count(hy); blue < 300 || span < hy.W/2 {
		t.Errorf("hyakutake: %d blue pixels spanning %d columns", blue, span)
	}
}

// Smoke thirty-six, the photon ring: the shadow in the middle as wide
// as root twenty-seven says, the disc's far side bent over the top of
// it, which the same scene unbent does not show.
func TestSmoke36ThePhotonRing(t *testing.T) {
	w, eye := thirtySix()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	bent := trace.Render(w, eye, lens, 0, 240, 100)
	kw, _ := thirtySixSpun(true, 0.95)
	spun := trace.Render(kw, eye, lens, 0, 240, 100)
	sw, _ := thirtySixBent(false)
	straight := trace.Render(sw, eye, lens, 0, 240, 100)
	contact(t, frames{{"bent", bent}, {"spinning", spun}, {"straight", straight}})
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		s := sheet.New(3)
		s.Add("still, through the tone", quasarTone.Develop(bent), 0.75)
		s.Add("spinning, hexes on the horizon", quasarTone.Develop(spun), 0.75)
		s.Add("straight, through the tone", quasarTone.Develop(straight), 0.75)
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	dark := func(f *frame.Frame, x, y int) bool { return f.At(x, y).Light().Luminance() < 0.01 }
	t.Run("a spinning hole's shadow leans", func(t *testing.T) {
		// against a glowing sky, the dark's reach left and right of the
		// middle along the middle row: alike for a still hole, and for
		// a spinning one the side coming toward the eye flattened, so
		// the two differ by a fifth or more.
		//
		// the spin is up and the eye on +Z, and the gas goes round from
		// +X toward -Z, so the left of the picture, minus X, is the
		// side coming toward the eye, and its dark run is the shorter.
		//
		// the hexes are off, as Edge.Hexes zero: the hex ball is the
		// spun scene's only thing, so the two scenes differ by the spin
		reach := func(spin float64) (left, right int) {
			w, _ := thirtySixSpun(true, spin)
			w.Volumes, w.Things = nil, nil
			w.Sky = glow(0.5)
			f := trace.Render(w, eye, lens, 0, 240, 100)
			for x := 120; x >= 0 && dark(f, x, 50); x-- {
				left++
			}
			for x := 121; x < f.W && dark(f, x, 50); x++ {
				right++
			}
			return left, right
		}
		l, r := reach(0)
		if math.Abs(float64(l-r)) > 0.1*float64(l+r) {
			t.Errorf("the still hole's shadow reaches %d left and %d right", l, r)
		}
		l, r = reach(0.95)
		if math.Abs(float64(l-r)) < 0.2*float64(l+r)/2 {
			t.Errorf("the spinning hole's shadow reaches %d left and %d right: no lean", l, r)
		}
		if l >= r {
			t.Errorf("the spun shadow reaches %d left, the side "+
				"coming toward the eye, and %d right: the "+
				"wrong side is flat", l, r)
		}
	})
	t.Run("the shadow is as wide as root twenty-seven says", func(t *testing.T) {
		// against a glowing sky and no disc, the shadow is the dark
		// disc in the middle; up the middle column from the centre
		// until it is lit, its radius is root twenty-seven at fifty
		// units, through a lens of forty degrees, in rows
		w, _ := thirtySixBent(true)
		w.Volumes = nil
		w.Sky = glow(0.5)
		sky := trace.Render(w, eye, lens, 0, 240, 100)
		up := 0
		for y := 50; y >= 0 && dark(sky, 120, y); y-- {
			up++
		}
		want := math.Atan(math.Sqrt(27)/50) / (math.Atan(math.Tan(20*maths.Degree)*100/180) / 50)
		if math.Abs(float64(up)-want) > 0.15*want {
			t.Errorf("the shadow reaches %d rows up; root twenty-seven says %.0f", up, want)
		}
	})
	t.Run("the far side of the disc shows over the shadow, bent", func(t *testing.T) {
		// up the middle column: bent, the dark reaches only to the
		// shadow's edge and the disc's far side is lit beyond it. the
		// straight scene has no dark at all now, since a hole with no
		// field and no hexes has nothing there to be dark
		darkBent := 0
		for y := 50; y >= 0 && dark(bent, 120, y); y-- {
			darkBent++
		}
		if !(darkBent > 8 && darkBent < 45) {
			t.Errorf("the dark reaches %d rows up bent", darkBent)
		}
	})
}

// glow is a sky of one light everywhere.
type glow float64

// Light is the same everywhere.
func (g glow) Light(maths.Vec) maths.Light { return maths.Grey(float64(g)) }

// Smoke thirty-seven, falling in: a sheet of the fall, a frame at each
// radius, looking at the hole and looking back. The hole's dark grows
// as the camera falls until it fills the view ahead; looking back,
// light still arrives inside the horizon; and at three masses the
// falling camera sees a smaller dark ahead than a camera held still
// there would, since its speed crowds the sky forward.
func TestSmoke37FallingIn(t *testing.T) {
	lens := film.Pinhole{Wide: 90 * maths.Degree, Pixel: 0.75}
	shares := []float64{20, 12, 8, 5, 3, 2.05, 1.6, 1}
	darkShare := func(f *frame.Frame) float64 {
		n := 0
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				if f.At(x, y).Light().Luminance() < 0.01 && f.At(x, y).Cover > 0.5 {
					n++
				}
			}
		}
		return float64(n) / float64(f.W*f.H)
	}
	var fs frames
	var shares37, backs37 []float64
	s := sheet.New(4)
	for _, back := range []bool{false, true} {
		for _, share := range shares {
			w, eye := thirtySevenFrame(share, back, false)
			f := trace.Render(w, eye, lens, 0, 240, 100)
			way := "at the hole"
			if back {
				way = "looking back"
				backs37 = append(backs37, darkShare(f))
			} else {
				shares37 = append(shares37, darkShare(f))
			}
			label := fmt.Sprintf("%.2f masses, %s", share, way)
			fs = append(fs, frames{{label, f}}...)
			s.Add(fmt.Sprintf("%s, dark %.2f", label, darkShare(f)), quasarTone.Develop(f), 0.75)
		}
	}
	contact(t, fs)
	if dir := os.Getenv("SLAYER_SHEET"); dir != "" {
		if err := s.PNG(filepath.Join(dir, t.Name()+"-toned.png")); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("the dark grows as the camera falls", func(t *testing.T) {
		for i := 1; i < 6; i++ {
			if shares37[i] <= shares37[i-1] {
				t.Errorf("the dark is %.3f at %.2f and %.3f at %.2f", shares37[i-1], shares[i-1], shares37[i], shares[i])
			}
		}
	})
	t.Run("the view ahead fills with the dark inside the horizon", func(t *testing.T) {
		if d := shares37[7]; d < 0.99 {
			t.Errorf("at a mass from the middle the dark ahead is %.3f of the frame", d)
		}
	})
	t.Run("light still arrives inside the horizon, looking back", func(t *testing.T) {
		for i := 6; i < 8; i++ {
			if d := backs37[i]; d > 0.5 {
				t.Errorf("at %.2f masses, inside the horizon, the dark behind is %.3f of the frame", shares[i], d)
			}
		}
	})
	t.Run("the falling camera sees a smaller dark than a still one", func(t *testing.T) {
		w, eye := thirtySevenFrame(3, false, true)
		still := darkShare(trace.Render(w, eye, lens, 0, 240, 100))
		if falling := shares37[4]; falling >= still {
			t.Errorf("at three masses the falling camera's dark is %.3f and the still camera's %.3f", falling, still)
		}
	})
}

// A smoke's Reads opens its inputs out: a path passes through, a $NAME
// becomes the files in the folder the variable names, a $NAME/file that
// one file in it, sorted, and an unset name reads nothing. Smoke
// thirty-eight declares the blue marble.
func TestASmokeSaysWhatItReads(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"black.jpg", "blue.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SLAYER_MARBLE", dir)
	s := Smoke{Inputs: []string{"$SLAYER_MARBLE", "/etc/hosts"}}
	got, err := s.Reads()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/etc/hosts", filepath.Join(dir, "black.jpg"), filepath.Join(dir, "blue.jpg")}
	if len(got) != len(want) {
		t.Fatalf("reads %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reads %v, want %v", got, want)
		}
	}
	t.Setenv("SLAYER_MARBLE", "")
	if got, _ := s.Reads(); len(got) != 1 {
		t.Errorf("with the name unset, reads %v", got)
	}
	marble, _ := Get(38)
	if len(marble.Inputs) != 1 || marble.Inputs[0] != "$SLAYER_MARBLE/blue.jpg" {
		t.Errorf("the marble smoke declares %v", marble.Inputs)
	}
	for _, s := range All() {
		if s.Number != 38 && len(s.Inputs) > 0 {
			t.Errorf("smoke %d declares inputs %v", s.Number, s.Inputs)
		}
	}
}

// Smoke thirty-eight reads blue.jpg and nothing else in the folder, so
// that is all its Reads says: black.jpg, which only a test reads, and any
// other file there are not its inputs. The file is named whether or not
// it is there, since Reads opens nothing.
func TestTheBlueMarbleSmokeReadsBlueAlone(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"black.jpg", "blue.jpg", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, ok := Get(38)
	if !ok {
		t.Fatal("no smoke 38")
	}
	t.Setenv("SLAYER_MARBLE", dir)
	got, err := s.Reads()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "blue.jpg"); len(got) != 1 || got[0] != want {
		t.Errorf("reads %v, want just %v", got, want)
	}
	empty := t.TempDir()
	t.Setenv("SLAYER_MARBLE", empty)
	got, err = s.Reads()
	if want := filepath.Join(empty, "blue.jpg"); err != nil || len(got) != 1 || got[0] != want {
		t.Errorf("in a folder without it, reads %v, %v, want just %v", got, err, want)
	}
	t.Setenv("SLAYER_MARBLE", "")
	if got, err := s.Reads(); err != nil || len(got) != 0 {
		t.Errorf("with the name unset, reads %v, %v", got, err)
	}
}

// refusal is the error a call panics with, which is how a smoke refuses
// by name through placed; nil when the call does not panic. A panic that
// is not an error is returned as one that says so.
func refusal(call func()) (err error) {
	defer func() {
		if caught := recover(); caught != nil {
			var ok bool
			if err, ok = caught.(error); !ok {
				err = fmt.Errorf("panicked with %v, which is not an error", caught)
			}
		}
	}()
	call()
	return nil
}

// watched runs call on this goroutine and ends the whole test run, with status
// 2, if the call takes longer than limit or the heap passes 256 MB: a refusal
// that does not come would otherwise loop forever or fill the machine's memory,
// and a failing test cannot stop a loop.
func watched(limit time.Duration, call func()) {
	runtime.GC()
	start, done := time.Now(), make(chan struct{})
	defer close(done)
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > 256<<20 {
					fmt.Fprintf(os.Stderr, "watched: the heap is %d MB, past 256\n", m.HeapAlloc>>20)
					os.Exit(2)
				}
				if time.Since(start) > limit {
					fmt.Fprintf(os.Stderr, "watched: still running after %v\n", limit)
					os.Exit(2)
				}
			}
		}
	}()
	call()
}

// Smoke thirty-seven refuses a radius it cannot fall to: zero, below it, no
// number, infinite, or past where the chart carries the camera. Each is refused
// by name inside a second, where the fall to one of them would otherwise go on
// forever.
func TestSmoke37RefusesARadiusItCannotFallTo(t *testing.T) {
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1), 0.005} {
		var err error
		watched(time.Second, func() {
			err = refusal(func() { thirtySevenFrame(radius, false, false) })
		})
		if err == nil || !strings.Contains(err.Error(), "thirtySevenFrame") {
			t.Errorf("a radius of %v: refused with %v", radius, err)
		}
	}
}

// A Find that finds nothing is refused by name, with what was looked for
// and where, and a Find that finds a tile gives that tile.
func TestAFindThatFindsNothingIsRefused(t *testing.T) {
	plain := world.New(twelveTerrain)
	toward := maths.V(0.35, 0.1, 1)
	err := refusal(func() {
		found(plain, "a tile of no biome", toward, func(world.Biome) bool { return false })
	})
	if err == nil || !strings.Contains(err.Error(), "a tile of no biome") ||
		!strings.Contains(err.Error(), fmt.Sprint(toward)) {
		t.Errorf("a want nothing meets: refused with %v", err)
	}
	tile := plain.Tiling.Nearest(toward)
	got := found(plain, "the tile itself", toward, func(b world.Biome) bool {
		return b == plain.Biome(tile)
	})
	if got != tile {
		t.Errorf("found %v, the nearest tile is %v and wants itself", got, tile)
	}
}

// Smoke thirty-eight renders without the marble: the painted earth, lit
// from the upper right, most of the frame covered.
func TestSmoke38TheBlueMarble(t *testing.T) {
	t.Setenv("SLAYER_MARBLE", "")
	w, eye := thirtyEight()
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	f := trace.Render(w, eye, lens, 0, 240, 100)
	contact(t, frames{{"the painted earth", f}})
	covered := 0
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			if f.At(x, y).Cover > 0.5 {
				covered++
			}
		}
	}
	if covered < f.W*f.H/3 {
		t.Errorf("the earth covers %d of %d pixels", covered, f.W*f.H)
	}
	if l := f.At(150, 40).Light().Luminance(); l < 0.05 {
		t.Errorf("the lit side is dark: %v", l)
	}
}

// Smoke thirty-nine, the windscreen: the glass is brighter toward the
// top, where the look grazes it, than low, where it is face on, by
// fresnel's share of the sky; the same sheet dull is flat.
func TestSmoke39AWindscreenCatchesTheSky(t *testing.T) {
	lens := film.Pinhole{Wide: 60 * maths.Degree, Pixel: 0.75}
	w, eye := thirtyNine()
	glossy := trace.Render(w, eye, lens, 0, 240, 100)
	dw, _ := thirtyNineGloss(0)
	dull := trace.Render(dw, eye, lens, 0, 240, 100)
	contact(t, frames{{"glass", glossy}, {"the same sheet, dull", dull}})
	// up the middle column the glass fills the frame, the look
	// steeper at the bottom and grazing at the top
	glass := func(f *frame.Frame, y int) float64 { return f.At(120, y).Light().Luminance() }
	low, high := glass(glossy, 85), glass(glossy, 10)
	if !(high > 2*low) {
		t.Errorf("the glass is %.3f low and %.3f high: no fresnel", low, high)
	}
	if dl, dh := glass(dull, 70), glass(dull, 10); math.Abs(dh-dl) > 0.2*math.Max(dl, dh)+0.005 {
		t.Errorf("the dull sheet is %.3f low and %.3f high", dl, dh)
	}
	if glass(dull, 10) > 0.05 {
		t.Errorf("the dull sheet is bright: %.3f", glass(dull, 10))
	}
}
