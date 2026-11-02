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
	world := glowing()
	world.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 0.5, Thick: thick},
		Material: &thing.Material{Name: "smoke", Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	return world
}

// Looking down through the smoke, less of the floor gets through than
// beside it, and the thicker the smoke the less; a ray that misses the
// puff is exactly what it was.
func TestSmokeDimsWhatIsBehindIt(t *testing.T) {
	bare := lone(glowing(), straightDown(3), 0)
	thin := lone(puffed(1), straightDown(3), 0)
	thick := lone(puffed(8), straightDown(3), 0)
	if !(thick.R < thin.R && thin.R < bare.R) {
		t.Fatalf("clear %v, thin %v, thick %v", bare.R, thin.R, thick.R)
	}
	beside := lone(puffed(8), maths.Ray{At: maths.V(2, 3, 0),
		Dir: maths.V(0, -1, 0)}, 0)
	if beside != bare {
		t.Fatalf("beside the puff %v, clear %v", beside, bare)
	}
	if thin.Depth != bare.Depth || thin.Cover != 1 {
		t.Fatalf("the floor through smoke: %+v", thin)
	}
}

// Under a lamp the smoke itself is lit, so a ray into the sky through it
// brings back light where the empty sky brought none, and covers the
// pixel by as much as it hid.
func TestSmokeIsLitAndCoversBySoMuch(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 2},
		Material: &thing.Material{Colour: maths.RGB(1, 0.5, 0.25)},
	})
	ray := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 1, 0)}
	dark := lone(world, ray, 0)
	if dark.R != 0 || !(dark.Cover > 0 && dark.Cover < 1) ||
		!math.IsInf(float64(dark.Depth), 1) {
		t.Fatalf("unlit smoke against an empty sky: %+v", dark)
	}
	world.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	lit := lone(world, ray, 0)
	if !(lit.R > 0 && lit.G > 0 && lit.B > 0) || lit.Cover != dark.Cover {
		t.Fatalf("lit smoke: %+v", lit)
	}
	// the smoke scatters its own colour: more red than green than blue
	if !(lit.R > lit.G && lit.G > lit.B) {
		t.Fatalf("the smoke's colour is lost: %+v", lit)
	}
}

// blueGlow is a puff that glows blue and scatters nothing, over a grey
// floor, seen from straight above; or the same under a sun with a
// black ball beneath the puff, hiding the floor.
func blueGlow(lampAndBall bool) frame.Pixel {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	world.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 3},
		Material: &thing.Material{Glow: maths.RGB(0, 0, 1)},
		Motion:   resting(maths.V(0, 2, 0)),
	})
	if lampAndBall {
		world.Lamp(thing.Sun(maths.V(1, 2, 1), maths.Grey(1)))
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5},
			Material: unlit, Motion: resting(maths.V(0, 0.5, 0))})
	}
	return lone(world, straightDown(5), 0)
}

// A volume that glows needs no lamp: with none, over a dark floor, its
// blue is there and nothing else is. A lamp adds nothing to a volume
// that scatters nothing: under a sun, with a black ball hiding the
// floor beneath it, the blue is the same.
func TestAGlowingVolumeNeedsNoLamp(t *testing.T) {
	alone := blueGlow(false)
	if !(alone.B > 0.3) || alone.R != 0 {
		t.Fatalf("a blue glow over a dark floor: %+v", alone)
	}
	if lamped := blueGlow(true); lamped.B != alone.B || lamped.R != 0 {
		t.Errorf("under a sun the glow is %+v, with none %+v", lamped,
			alone)
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
	bare := lone(glowing(), beside, 0)
	if lone(small, beside, 0) != bare {
		t.Fatal("the small puff reaches a unit aside")
	}
	if !(lone(big, beside, 0).R < bare.R) {
		t.Fatal("the big puff does not reach a unit aside")
	}
	little := lone(small, straightDown(3), 0)
	large := lone(big, straightDown(3), 0)
	if !within(float64(little.R), float64(large.R), 0.03) {
		t.Fatalf("through the middle, small leaves %v and big %v",
			little.R, large.R)
	}
}

// litFloor is a white floor under a sun straight overhead, for shadows to fall
// on.
func litFloor() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(1)}})
	world.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	return world
}

// A volume shadows a surface: the floor under a puff is darker than the
// floor beside it, seen from the side so the puff is not in the way.
func TestAVolumeShadowsASurface(t *testing.T) {
	world := litFloor()
	world.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1, Thick: 4},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 3, 0)}),
	})
	// a ray that comes in low, under the puff, to the floor at the origin
	// and at a point four units aside
	side := func(x float64) frame.Pixel {
		return lone(world, maths.Ray{At: maths.V(x+6, 0.1, 0),
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
	for _, each := range []*thing.World{open, shaded} {
		each.AddVolume(thing.Volume{
			Medium:   medium.Puff{Radius: 1, Thick: 2},
			Material: &thing.Material{Colour: maths.Grey(1)},
		})
		each.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	}
	shaded.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.5},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 4, 0)})})
	ray := maths.Ray{At: maths.V(5, 0, 0), Dir: maths.V(-1, 0, 0)}
	inOpen, inShade := lone(open, ray, 0), lone(shaded, ray, 0)
	if !(inShade.R < 0.2*inOpen.R) {
		t.Fatalf("in the open %v, under the ball %v", inOpen.R,
			inShade.R)
	}
}

// A volume shadows another volume: a puff under a thicker puff is darker
// than the same puff alone.
func TestAVolumeShadowsAVolume(t *testing.T) {
	alone, under := thing.NewWorld(), thing.NewWorld()
	for _, each := range []*thing.World{alone, under} {
		each.AddVolume(thing.Volume{
			Medium:   medium.Puff{Radius: 1, Thick: 2},
			Material: &thing.Material{Colour: maths.Grey(1)},
		})
		each.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	}
	under.AddVolume(thing.Volume{
		Medium:   medium.Puff{Radius: 1.5, Thick: 6},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 4, 0)}),
	})
	ray := maths.Ray{At: maths.V(5, 0, 0), Dir: maths.V(-1, 0, 0)}
	one, other := lone(alone, ray, 0), lone(under, ray, 0)
	if !(other.R < 0.5*one.R) {
		t.Fatalf("alone %v, under a thicker puff %v", one.R, other.R)
	}
}

// shineByHeight glows more the higher the point.
type shineByHeight struct{}

func (shineByHeight) At(point maths.Vec) maths.Light {
	return maths.Grey(point.Y + 1)
}

// Shine is glow from place to place: on a surface and in a volume.
func TestShineGlowsByPlace(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Shine: shineByHeight{}}})
	leftward := maths.V(-1, 0, 0)
	top := lone(world, straightDown(5), 0)
	side := lone(world, maths.Ray{At: maths.V(5, 0, 0), Dir: leftward}, 0)
	if !within(float64(top.R), 2, 1e-6) ||
		!within(float64(side.R), 1, 1e-6) {
		t.Errorf("the top shines %v, the side %v", top.R, side.R)
	}
	second := thing.NewWorld()
	second.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 3},
		Material: &thing.Material{Shine: shineByHeight{}}})
	above := maths.Ray{At: maths.V(5, 0.6, 0), Dir: leftward}
	below := maths.Ray{At: maths.V(5, -0.6, 0), Dir: leftward}
	high, low := lone(second, above, 0), lone(second, below, 0)
	if !(high.R > low.R && low.R > 0) {
		t.Errorf("through the puff, high %v and low %v", high.R, low.R)
	}
}
