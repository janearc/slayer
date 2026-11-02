package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// puffed is a glowing floor with a puff of smoke a unit above it, white
// smoke that gives off nothing of its own.
func puffed(thick float64) *thing.World {
	w := glowing()
	w.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 0.5, Thick: thick},
		Material: &thing.Material{Name: "smoke", Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	return w
}

// Looking down through the smoke, less of the floor gets through than
// beside it, and the thicker the smoke the less; a ray that misses the
// puff is exactly what it was.
func TestSmokeDimsWhatIsBehindIt(t *testing.T) {
	clear := Trace(glowing(), down(3), 0)
	thin := Trace(puffed(1), down(3), 0)
	thick := Trace(puffed(8), down(3), 0)
	if !(thick.R < thin.R && thin.R < clear.R) {
		t.Fatalf("clear %v, thin %v, thick %v", clear.R, thin.R, thick.R)
	}
	beside := Trace(puffed(8), maths.Ray{At: maths.V(2, 3, 0),
		Dir: maths.V(0, -1, 0)}, 0)
	if beside != clear {
		t.Fatalf("beside the puff %v, clear %v", beside, clear)
	}
	if thin.Depth != clear.Depth || thin.Cover != 1 {
		t.Fatalf("the floor through smoke: %+v", thin)
	}
}

// Under a lamp the smoke itself is lit, so a ray into the sky through it
// brings back light where the empty sky brought none, and covers the
// pixel by as much as it hid.
func TestSmokeIsLitAndCoversBySoMuch(t *testing.T) {
	w := thing.NewWorld()
	w.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 2},
		Material: &thing.Material{Colour: maths.RGB(1, 0.5, 0.25)},
	})
	r := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 1, 0)}
	dark := Trace(w, r, 0)
	if dark.R != 0 || !(dark.Cover > 0 && dark.Cover < 1) ||
		!math.IsInf(float64(dark.Depth), 1) {
		t.Fatalf("unlit smoke against an empty sky: %+v", dark)
	}
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	lit := Trace(w, r, 0)
	if !(lit.R > 0 && lit.G > 0 && lit.B > 0) || lit.Cover != dark.Cover {
		t.Fatalf("lit smoke: %+v", lit)
	}
	// the smoke scatters its own colour: more red than green than blue
	if !(lit.R > lit.G && lit.G > lit.B) {
		t.Fatalf("the smoke's colour is lost: %+v", lit)
	}
}

// A volume that glows needs no lamp, and a surface's shadow does not
// reach it: the glow is there behind a ball that hides the floor.
func TestAGlowingVolumeNeedsNoLamp(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	w.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 3},
		Material: &thing.Material{Glow: maths.RGB(0, 0, 1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 2, 0)}),
	})
	p := Trace(w, down(5), 0)
	if !(p.B > 0.3) || p.R != 0 {
		t.Fatalf("a blue glow over a dark floor: %+v", p)
	}
}

// A volume's own space counts: scaled up three times, the same puff is
// three times as wide, so a ray that passed beside the small one goes
// through the big one; and through the middle it hides the same, since
// density is per unit of the medium's own space and the look is kept.
func TestAVolumeLivesInItsSubspace(t *testing.T) {
	small, big := puffed(2), glowing()
	big.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 0.5, Thick: 2},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0),
			Scale: 3}),
	})
	beside := maths.Ray{At: maths.V(1, 3, 0), Dir: maths.V(0, -1, 0)}
	clear := Trace(glowing(), beside, 0)
	if Trace(small, beside, 0) != clear {
		t.Fatal("the small puff reaches a unit aside")
	}
	if !(Trace(big, beside, 0).R < clear.R) {
		t.Fatal("the big puff does not reach a unit aside")
	}
	a, b := Trace(small, down(3), 0), Trace(big, down(3), 0)
	if math.Abs(float64(a.R-b.R)) > 0.03 {
		t.Fatalf("through the middle, small leaves %v and big %v", a.R, b.R)
	}
}

// lit is a white floor under a sun straight overhead, for shadows to fall
// on.
func lit() *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	return w
}

// A volume shadows a surface: the floor under a puff is darker than the
// floor beside it, seen from the side so the puff is not in the way.
func TestAVolumeShadowsASurface(t *testing.T) {
	w := lit()
	w.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 4},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 3, 0)}),
	})
	// a ray that comes in low, under the puff, to the floor at the origin
	// and at a point four units aside
	side := func(x float64) frame.Pixel {
		return Trace(w, maths.Ray{At: maths.V(x+6, 0.1, 0),
			Dir: maths.V(-6, -0.1, 0).Unit()}, 0)
	}
	under, open := side(0), side(4)
	if under.Cover != 1 || !(under.R < 0.5*open.R) {
		t.Fatalf("under the puff %+v, in the open %+v", under, open)
	}
}

// A surface shadows a volume: a puff with a ball between it and the sun
// is darker than the same puff in the open.
func TestASurfaceShadowsAVolume(t *testing.T) {
	open, shaded := thing.NewWorld(), thing.NewWorld()
	for _, w := range []*thing.World{open, shaded} {
		w.AddVolume(thing.Volume{
			Medium:   medium.Puff{Radius: 1, Thick: 2},
			Material: &thing.Material{Colour: maths.Grey(1)},
		})
		w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	}
	shaded.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.5},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 4, 0)})})
	r := maths.Ray{At: maths.V(5, 0, 0), Dir: maths.V(-1, 0, 0)}
	a, b := Trace(open, r, 0), Trace(shaded, r, 0)
	if !(b.R < 0.2*a.R) {
		t.Fatalf("in the open %v, under the ball %v", a.R, b.R)
	}
}

// A volume shadows another volume: a puff under a thicker puff is darker
// than the same puff alone.
func TestAVolumeShadowsAVolume(t *testing.T) {
	alone, under := thing.NewWorld(), thing.NewWorld()
	for _, w := range []*thing.World{alone, under} {
		w.AddVolume(thing.Volume{
			Medium:   medium.Puff{Radius: 1, Thick: 2},
			Material: &thing.Material{Colour: maths.Grey(1)},
		})
		w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	}
	under.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1.5, Thick: 6},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 4, 0)}),
	})
	r := maths.Ray{At: maths.V(5, 0, 0), Dir: maths.V(-1, 0, 0)}
	a, b := Trace(alone, r, 0), Trace(under, r, 0)
	if !(b.R < 0.5*a.R) {
		t.Fatalf("alone %v, under a thicker puff %v", a.R, b.R)
	}
}

// shineByHeight glows more the higher the point.
type shineByHeight struct{}

func (shineByHeight) At(p maths.Vec) maths.Light { return maths.Grey(p.Y) }

// Shine is glow from place to place: on a surface and in a volume.
func TestShineGlowsByPlace(t *testing.T) {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Shine: shineByHeight{}}})
	top := Trace(w, maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}, 0)
	side := Trace(w, maths.Ray{At: maths.V(5, 0, 0), Dir: maths.V(-1, 0, 0)}, 0)
	if math.Abs(float64(top.R)-1) > 1e-6 || math.Abs(float64(side.R)) > 1e-6 {
		t.Errorf("the top shines %v, the side %v", top.R, side.R)
	}
	v := thing.NewWorld()
	v.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 3},
		Material: &thing.Material{Shine: shineByHeight{}}})
	high := Trace(v, maths.Ray{At: maths.V(5, 0.6, 0), Dir: maths.V(-1, 0, 0)}, 0)
	low := Trace(v, maths.Ray{At: maths.V(5, -0.6, 0), Dir: maths.V(-1, 0, 0)}, 0)
	if !(high.R > 0 && low.R < 0) {
		t.Errorf("through the puff, high %v and low %v", high.R, low.R)
	}
}
