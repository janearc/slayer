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

// sunwards is the way to smoking's sun: up and to the right.
var sunwards = maths.V(1, 2, 1).Unit()

// smoking is a clay ball on a chalk floor under one sun.
func smoking() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk",
			Colour: maths.Grey(0.8)},
	})
	world.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "clay",
			Colour: maths.RGB(0.7, 0.3, 0.2)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	world.Lamp(thing.Sun(sunwards, maths.Grey(1)))
	return world
}

var centre = maths.V(0, 1, 0)

func bright(pixel frame.Pixel) float64 { return pixel.Light().Luminance() }

// The side of the ball facing the sun square is the clay's own colour
// over pi times the sun, in all three channels, as a matte surface
// gives back what lands on it; and the far side is black: the world
// has no other light.
func TestTheSideFacingTheSunIsLitAndTheFarSideBlack(t *testing.T) {
	world := smoking()
	sunward := maths.Ray{At: centre.Add(sunwards.Scale(10)),
		Dir: sunwards.Neg()}
	away := maths.Ray{At: centre.Sub(sunwards.Scale(10)), Dir: sunwards}
	lit, dark := lone(world, sunward, 0), lone(world, away, 0)
	if lit.Cover != 1 || dark.Cover != 1 {
		t.Fatalf("both rays should meet the ball: cover %v and %v",
			lit.Cover, dark.Cover)
	}
	got := lit.Light()
	if !within(got.R, 0.7/math.Pi, 1e-6) ||
		!within(got.G, 0.3/math.Pi, 1e-6) ||
		!within(got.B, 0.2/math.Pi, 1e-6) {
		t.Errorf("lit side is %v, want the clay at full sun", got)
	}
	if dark.Light() != (maths.Light{}) {
		t.Errorf("the far side is %v, want no light", dark.Light())
	}
}

// The ball's shadow falls where a line from its centre, away from the
// sun, comes down to the floor. There the floor is black; in the open
// it is the chalk over pi times how squarely the sun lands on it.
func TestTheFloorInTheShadowIsBlack(t *testing.T) {
	world := smoking()
	shadow := centre.Sub(sunwards.Scale(centre.Y / sunwards.Y))
	open := maths.V(5, 0, 5)
	onto := func(point maths.Vec) frame.Pixel {
		above := point.Add(maths.V(0, 1e-3, 0))
		falling := maths.Ray{At: above, Dir: maths.V(0, -1, 0)}
		return lone(world, falling, 0)
	}
	in, out := onto(shadow), onto(open)
	if in.Cover != 1 || out.Cover != 1 {
		t.Fatalf("both rays should meet the floor: cover %v and %v",
			in.Cover, out.Cover)
	}
	if in.Light() != (maths.Light{}) {
		t.Errorf("the floor in shadow is %v, want no light", in.Light())
	}
	if want := 0.8 * sunwards.Y / math.Pi; !within(bright(out), want,
		1e-6) {
		t.Errorf("the open floor is %v, want %v", bright(out), want)
	}
}

func TestARayThatMeetsNothingHasNoCoverAndNoEnd(t *testing.T) {
	rising := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.Up}
	pixel := lone(smoking(), rising, 0)
	if pixel.Cover != 0 || !math.IsInf(float64(pixel.Depth), 1) {
		t.Fatalf("cover %v depth %v; want 0 and +Inf", pixel.Cover,
			pixel.Depth)
	}
}

// the same answers, from a whole picture: the sky above the horizon is empty
// and the ball is in the middle.
func TestAPictureOfTheScene(t *testing.T) {
	from := maths.V(0, 2, 6)
	eye := lookingAt(from, centre)
	picture := draw(smoking(), eye, film.Pinhole{Pixel: 0.75}, 0, 64, 36)
	if picture.At(0, 0).Cover != 0 {
		t.Fatal("the top corner should be sky")
	}
	mid := picture.At(32, 18)
	near := from.Sub(centre).Len() - 1
	if mid.Cover != 1 || !within(float64(mid.Depth), near, 0.05) {
		t.Fatalf("the middle should be the near side of the ball, got "+
			"%+v", mid)
	}
}

// a shape with no exact answer is marched, and has to land in the same place.
func TestAMarchedBallMatchesTheExactOne(t *testing.T) {
	exact, walk := thing.NewWorld(), thing.NewWorld()
	exact.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	// wrapped so only Bounds and Distance show: no exact hit and no
	// exact normal
	wrapped := struct{ shape.Shape }{shape.Sphere{Radius: 1}}
	walk.Add(thing.Thing{Shape: wrapped})
	ray := maths.Ray{At: maths.V(0.3, 0.2, 5), Dir: maths.V(0, 0, -1)}
	direct, marched := lone(exact, ray, 0), lone(walk, ray, 0)
	if !within(float64(direct.Depth), float64(marched.Depth), 1e-3) {
		t.Fatalf("exact %v, marched %v", direct.Depth, marched.Depth)
	}
}

// A clear thing is drawn and casts no shadow: a ball of light over a
// floor leaves the floor lit beneath it.
func TestAClearThingCastsNoShadow(t *testing.T) {
	floor := func(shadowless bool) float32 {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "floor",
				Colour: maths.Grey(0.8)}})
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Name: "ball",
				Glow: maths.Grey(2)},
			Motion: resting(maths.V(0, 2, 0)),
			Attrs:  thing.Attrs{Clear: shadowless}})
		world.Lamp(thing.Sun(maths.Up, maths.Grey(1)))
		from := maths.V(0, 1.2, 8)
		eye := lookingAt(from, maths.Vec{})
		lens := film.Pinhole{Wide: 30 * maths.Degree}
		picture := draw(world, eye, lens, 0, 41, 41)
		// the floor straight under the ball: the middle of the
		// picture, where the eye looks, is the floor's origin
		return picture.At(20, 20).R
	}
	// the sun is overhead, so the lit floor is its own 0.8 over pi,
	// matte, and the shadowed floor has no light at all
	lit, shadowed := floor(true), floor(false)
	if !within(float64(lit), 0.8/math.Pi, 1e-6) || shadowed != 0 {
		t.Errorf("under a clear ball the floor is %v, under a solid "+
			"one %v", lit, shadowed)
	}
}

// A glossy floor mirrors the sky by fresnel's share, Mirrors of the
// ray's cosine: little looked at steeply, most at a graze. The rest is
// its matte colour lit by the sky, which under a sky of one everywhere
// is the colour itself. A dull floor under a sun is its own lit colour
// both ways.
func TestAGlossyFloorMirrorsTheSkyMoreAtAGraze(t *testing.T) {
	floor := func(gloss float64) *thing.World {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Colour: maths.Grey(0.1),
				Gloss: gloss}})
		world.Sky = skies.Gradient{Horizon: maths.Grey(1),
			Zenith: maths.Grey(1)}
		return world
	}
	above := maths.V(0, 1, 0)
	steep := maths.Ray{At: above, Dir: maths.V(0, -1, 0.1).Unit()}
	graze := maths.Ray{At: above, Dir: maths.V(0, -0.05, 1).Unit()}
	glossy := floor(1)
	glass := &thing.Material{Gloss: 1}
	for _, aim := range []maths.Ray{steep, graze} {
		got := lone(glossy, aim, 0).Light()
		share := glass.Mirrors(-aim.Dir.Y)
		want := share + (1-share)*0.1
		if !within(got.R, want, 1e-6) || !within(got.G, want, 1e-6) ||
			!within(got.B, want, 1e-6) {
			t.Errorf("looking %v the glossy floor is %v, want %v",
				aim.Dir, got, want)
		}
	}
	dull := floor(0)
	dull.Lamp(thing.Sun(maths.Up, maths.Grey(1)))
	dullSteep := bright(lone(dull, steep, 0))
	dullGraze := bright(lone(dull, graze, 0))
	if dullSteep != dullGraze || dullSteep <= 0.05 {
		t.Errorf("the dull floor is %.3f steep and %.3f at a graze",
			dullSteep, dullGraze)
	}
	// glass mirrors a twenty-fifth face on and everything at a graze
	mirror := &thing.Material{Gloss: 1}
	if share := mirror.Mirrors(1); !within(share, 0.04, 0.001) {
		t.Errorf("glass face on mirrors %.3f", share)
	}
	if share := mirror.Mirrors(0); !within(share, 1, 1e-9) {
		t.Errorf("glass at a graze mirrors %.3f", share)
	}
}

// The tracer ticks what moves: a thin disc with an orbital rate, drawn at two
// times, differs, and at time zero it is a disc standing still, byte for byte.
func TestTheTracerTicksAMovingDisc(t *testing.T) {
	disc := func(omega float64) *thing.World {
		world := thing.NewWorld()
		shine := texture.Thin{Inner: 1, Outer: 10, Hot: 6000, Bright: 1,
			Streaks: 1, Seed: 3, Omega: omega}
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Shine: shine}})
		return world
	}
	from := maths.V(0, 12, 6)
	eye := lookingAt(from, maths.Vec{})
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	still := draw(disc(0), eye, lens, 0, 32, 32)
	now := draw(disc(1), eye, lens, 0, 32, 32)
	later := draw(disc(1), eye, lens, 20*time.Second, 32, 32)
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

// The sky lights a matte floor by as much of it as the floor can see.
// Under a sky of one everywhere, an open floor sees half the sky and
// gives back its own colour, as a white furnace does; the floor under a
// ball, which hides part of that half, gives back less.
func TestTheSkyLightsWhatItCanSee(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Colour: maths.Grey(0.5)},
		Motion:   resting(maths.V(0, 1.5, 0))})
	world.Sky = skies.Gradient{Horizon: maths.Grey(1),
		Zenith: maths.Grey(1)}
	down := func(x float64) float64 {
		at := maths.Ray{At: maths.V(x, 1e-3, 0), Dir: maths.V(0, -1, 0)}
		return bright(lone(world, at, 0))
	}
	open, under := down(50), down(0)
	if !within(open, 0.5, 1e-6) || !(under < open) {
		t.Errorf("the open floor is %v, want 0.5; under the ball %v",
			open, under)
	}
}
