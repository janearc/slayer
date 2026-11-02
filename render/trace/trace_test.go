package trace

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	skies "github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// smoking: a clay ball on a chalk floor, one sun up and to the right.
var toward = maths.V(1, 2, 1).Unit()

func smoking() *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape:    shape.Plane{},
		Material: &thing.Material{Name: "chalk", Colour: maths.Grey(0.8)},
	})
	w.Add(thing.Thing{
		Shape:    shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "clay", Colour: maths.RGB(0.7, 0.3, 0.2)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	w.Lamp(thing.Sun(toward, maths.Grey(1)))
	return w
}

var centre = maths.V(0, 1, 0)

func bright(p frame.Pixel) float64 { return p.Light().Luminance() }

func TestTheSideFacingTheSunIsBrighter(t *testing.T) {
	w := smoking()
	lit := Trace(w, maths.Ray{At: centre.Add(toward.Scale(10)), Dir: toward.Neg()}, 0)
	dark := Trace(w, maths.Ray{At: centre.Sub(toward.Scale(10)), Dir: toward}, 0)
	if lit.Cover != 1 || dark.Cover != 1 {
		t.Fatalf("both rays should meet the ball: cover %v and %v", lit.Cover, dark.Cover)
	}
	if !(bright(lit) > bright(dark)) {
		t.Fatalf("lit side %v is not brighter than the far side %v", bright(lit), bright(dark))
	}
	// facing the sun square, the pixel is the clay's own colour times the sun
	if got := lit.Light(); math.Abs(got.R-0.7) > 1e-3 || math.Abs(got.G-0.3) > 1e-3 {
		t.Fatalf("lit side is %v, want the clay at full sun", got)
	}
}

// the ball's shadow falls where a line from its centre, away from the sun,
// comes down to the floor.
func TestTheFloorInTheShadowIsDarker(t *testing.T) {
	w := smoking()
	shadow := centre.Sub(toward.Scale(centre.Y / toward.Y))
	open := maths.V(5, 0, 5)
	down := func(p maths.Vec) frame.Pixel {
		return Trace(w, maths.Ray{At: p.Add(maths.V(0, 1e-3, 0)), Dir: maths.V(0, -1, 0)}, 0)
	}
	in, out := down(shadow), down(open)
	if in.Cover != 1 || out.Cover != 1 {
		t.Fatalf("both rays should meet the floor: cover %v and %v", in.Cover, out.Cover)
	}
	if !(bright(in) < bright(out)) {
		t.Fatalf("shadow %v is not darker than open floor %v", bright(in), bright(out))
	}
}

func TestARayThatMeetsNothingHasNoCoverAndNoEnd(t *testing.T) {
	p := Trace(smoking(), maths.Ray{At: maths.V(0, 5, 0), Dir: maths.Up}, 0)
	if p.Cover != 0 || !math.IsInf(float64(p.Depth), 1) {
		t.Fatalf("cover %v depth %v; want 0 and +Inf", p.Cover, p.Depth)
	}
}

// the same answers, from a whole picture: the sky above the horizon is empty
// and the ball is in the middle.
func TestAPictureOfTheScene(t *testing.T) {
	from := maths.V(0, 2, 6)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, centre)}
	f := Render(smoking(), eye, film.Pinhole{Pixel: 0.75}, 0, 64, 36)
	if f.At(0, 0).Cover != 0 {
		t.Fatal("the top corner should be sky")
	}
	mid := f.At(32, 18)
	if mid.Cover != 1 || math.Abs(float64(mid.Depth)-from.Sub(centre).Len()+1) > 0.05 {
		t.Fatalf("the middle should be the near side of the ball, got %+v", mid)
	}
}

// a shape with no exact answer is marched, and has to land in the same place.
func TestAMarchedBallMatchesTheExactOne(t *testing.T) {
	exact, walk := thing.NewWorld(), thing.NewWorld()
	exact.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	// wrapped so only Bounds and Distance show: no exact hit, no exact normal
	walk.Add(thing.Thing{Shape: struct{ shape.Shape }{shape.Sphere{Radius: 1}}})
	r := maths.Ray{At: maths.V(0.3, 0.2, 5), Dir: maths.V(0, 0, -1)}
	a, b := Trace(exact, r, 0), Trace(walk, r, 0)
	if math.Abs(float64(a.Depth-b.Depth)) > 1e-3 {
		t.Fatalf("exact %v, marched %v", a.Depth, b.Depth)
	}
}

// A clear thing is drawn and casts no shadow: a ball of light over a
// floor leaves the floor lit beneath it.
func TestAClearThingCastsNoShadow(t *testing.T) {
	floor := func(clear bool) float32 {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Name: "floor", Colour: maths.Grey(0.8)}})
		w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Name: "ball", Glow: maths.Grey(2)},
			Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 2, 0)}),
			Attrs:    thing.Attrs{Clear: clear}})
		w.Lamp(thing.Sun(maths.Up, maths.Grey(1)))
		from := maths.V(0, 1.2, 8)
		eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
		f := Render(w, eye, film.Pinhole{Wide: 30 * maths.Degree}, 0, 41, 41)
		// the floor straight under the ball: the middle of the
		// picture, where the eye looks, is the floor's origin
		return f.At(20, 20).R
	}
	if lit, shadowed := floor(true), floor(false); !(lit > 0.3 && shadowed < 0.5*lit) {
		t.Errorf("under a clear ball the floor is %v, under a solid one %v", lit, shadowed)
	}
}

// A glossy floor mirrors the sky by fresnel's share: looked at steeply
// it is mostly its own colour, at a grazing angle mostly sky, and a
// dull floor under a sun is its own lit colour both ways.
func TestAGlossyFloorMirrorsTheSkyMoreAtAGraze(t *testing.T) {
	floor := func(gloss float64) *thing.World {
		w := thing.NewWorld()
		w.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Colour: maths.Grey(0.1), Gloss: gloss}})
		w.Sky = skies.Gradient{Horizon: maths.Grey(1), Zenith: maths.Grey(1)}
		w.Lamps = nil
		return w
	}
	steep := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0.1).Unit()}
	graze := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -0.05, 1).Unit()}
	glossy := floor(1)
	s, g := Trace(glossy, steep, 0).Light().Luminance(), Trace(glossy, graze, 0).Light().Luminance()
	if !(g > 0.5) || !(s < 0.1) || !(g > 3*s) {
		t.Errorf("the glossy floor is %.3f steep and %.3f at a graze", s, g)
	}
	dull := floor(0)
	dull.Lamp(thing.Sun(maths.Up, maths.Grey(1)))
	ds, dg := Trace(dull, steep, 0).Light().Luminance(), Trace(dull, graze, 0).Light().Luminance()
	if ds != dg || ds <= 0.05 {
		t.Errorf("the dull floor is %.3f steep and %.3f at a graze", ds, dg)
	}
	// glass mirrors a twenty-fifth face on and everything at a graze
	m := &thing.Material{Gloss: 1}
	if f := m.Mirrors(1); math.Abs(f-0.04) > 0.001 {
		t.Errorf("glass face on mirrors %.3f", f)
	}
	if f := m.Mirrors(0); math.Abs(f-1) > 1e-9 {
		t.Errorf("glass at a graze mirrors %.3f", f)
	}
}

// The tracer ticks what moves: a thin disc with an orbital rate, drawn at two
// times, differs, and at time zero it is a disc standing still, byte for byte.
func TestTheTracerTicksAMovingDisc(t *testing.T) {
	disc := func(omega float64) *thing.World {
		w := thing.NewWorld()
		w.Lamps = nil
		w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
			Shine: texture.Thin{Inner: 1, Outer: 10, Hot: 6000, Bright: 1,
				Streaks: 1, Seed: 3, Omega: omega}}})
		return w
	}
	from := maths.V(0, 12, 6)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	still := Render(disc(0), eye, lens, 0, 32, 32)
	now := Render(disc(1), eye, lens, 0, 32, 32)
	later := Render(disc(1), eye, lens, 20*time.Second, 32, 32)
	same, moved := true, false
	for i := range now.Pix {
		same = same && now.Pix[i] == still.Pix[i]
		moved = moved || later.Pix[i] != now.Pix[i]
	}
	if !same {
		t.Error("at nought the moving disc is not the still one")
	}
	if !moved {
		t.Error("twenty seconds on the disc has not moved")
	}
}
