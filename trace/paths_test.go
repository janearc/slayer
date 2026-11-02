package trace

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// overTheScene is the eye the golden solid is drawn from.
var overTheScene = lookingAt(maths.V(0, 2, 6), centre)

// Sampled with one ray a pixel is Render, pixel for pixel.
func TestSampledOnceIsRender(t *testing.T) {
	lens := film.Pinhole{}
	once := sampled(smoking(), overTheScene, lens, 0, 16, 16, 1)
	plain := draw(smoking(), overTheScene, lens, 0, 16, 16)
	samePixels(t, once, plain, "sampled once,")
}

// meanOf is the mean of the four fine pixels under one coarse pixel,
// as light and cover, with the nearest of their depths.
func meanOf(fine *frame.Frame, x, y int) frame.Pixel {
	mean := frame.Pixel{Depth: float32(math.Inf(1))}
	four := []frame.Pixel{fine.At(2*x, 2*y), fine.At(2*x+1, 2*y),
		fine.At(2*x, 2*y+1), fine.At(2*x+1, 2*y+1)}
	for _, each := range four {
		mean.R += each.R / 4
		mean.G += each.G / 4
		mean.B += each.B / 4
		mean.Cover += each.Cover / 4
		mean.Depth = min(mean.Depth, each.Depth)
	}
	return mean
}

// alike is whether two pixels have the same light and cover to a
// millionth, and the same depth.
func alike(got, want frame.Pixel) bool {
	near := func(got, want float32) bool {
		return within(float64(got), float64(want), 1e-6)
	}
	return near(got.R, want.R) && near(got.G, want.G) &&
		near(got.B, want.B) && near(got.Cover, want.Cover) &&
		got.Depth == want.Depth
}

// Sampled with two rays per pixel each way is, at every pixel, the mean of the
// four rays' light and cover and the nearest of their depths. On the ball's
// edge that is a cover between none and all.
func TestSampledTwiceIsTheMeanOfFour(t *testing.T) {
	lens := film.Pinhole{}
	twice := sampled(smoking(), overTheScene, lens, 0, 16, 16, 2)
	fine := draw(smoking(), overTheScene, lens, 0, 32, 32)
	edges := 0
	for i, got := range twice.Pix {
		want := meanOf(fine, i%16, i/16)
		if !alike(got, want) {
			t.Fatalf("pixel %d is %v, its four's mean %v", i, got,
				want)
		}
		if got.Cover > 0 && got.Cover < 1 {
			edges++
		}
	}
	if edges == 0 {
		t.Error("no pixel lies across an edge, so the mean of cover " +
			"was never asked")
	}
}

// footprint is a texture that paints how wide it was asked to paint:
// what comes back says what pixel width the tracer handed down.
type footprint struct{}

// At is black: asked with no width.
func (footprint) At(maths.Vec) maths.Light { return maths.Light{} }

// Within is the width asked for, as light.
func (footprint) Within(_ maths.Vec, wide float64) maths.Light {
	return maths.Grey(wide)
}

// widthAsked is the pixel width a ball two units from the eye is asked to paint
// at, at the picture's middle, in a world whose Pixel is set or left at zero.
// the ball's matte face, square to a sun of one, gives back its colour over pi,
// so the width is pi times what comes back, over the distance.
func widthAsked(pixel float64) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Texture: footprint{}}})
	world.Lamp(thing.Sun(maths.V(0, 0, 1), maths.Grey(1)))
	world.Pixel = pixel
	eye := lookingAt(maths.V(0, 0, 3), maths.Vec{})
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	const side, middle, away = 65, 32, 2.0
	return math.Pi * float64(draw(world, eye, lens, 0, side, side).
		At(middle, middle).R) / away
}

// A pixel's angle is the angle between the middle pixel's ray and its
// neighbour's. For a pinhole sixty degrees wide over sixty-five pixels
// that is the arctangent of one pixel's share of the picture's width.
// A world's own Pixel stands in for it when set.
func TestThePixelAngleIsThePicturesOrTheWorlds(t *testing.T) {
	share := 2 * math.Tan(30*maths.Degree) / 65
	want := math.Atan(share)
	if got := widthAsked(0); !within(got, want, 1e-4*want) {
		t.Errorf("the picture's own pixel is %v, want %v", got, want)
	}
	if got := widthAsked(0.01); !within(got, 0.01, 1e-7) {
		t.Errorf("with the world's Pixel at 0.01 the width is %v", got)
	}
}

// A hidden thing is not drawn and casts no shadow, and a hidden volume
// is not drawn either.
func TestHiddenThingsAndVolumesAreNotThere(t *testing.T) {
	world := smoking()
	world.Things[1].Attrs.Hidden = true
	at := maths.Ray{At: centre.Add(sunwards.Scale(10)), Dir: sunwards.Neg()}
	if seen := lone(world, at, 0); !(float64(seen.Depth) >= 10.5) {
		t.Errorf("the hidden ball is drawn: %+v", seen)
	}
	shadow := centre.Sub(sunwards.Scale(centre.Y / sunwards.Y))
	onto := maths.Ray{At: shadow.Add(maths.V(0, 1e-3, 0)),
		Dir: maths.V(0, -1, 0)}
	// the chalk lit by the sun's slant, matte: its colour over pi
	if got := bright(lone(world, onto, 0)); !within(got,
		0.8*sunwards.Y/math.Pi, 1e-6) {
		t.Errorf("under the hidden ball the floor is %v", got)
	}
	smoke := puffed(2)
	smoke.Volumes[0].Attrs.Hidden = true
	got := lone(smoke, straightDown(3), 0)
	want := lone(glowing(), straightDown(3), 0)
	if got != want {
		t.Errorf("through hidden smoke the floor is %v, not %v", got,
			want)
	}
}

// A unit ball on a motion of scale three is three across: from five
// away its near side is two away.
func TestAThingsScaleIsItsSize(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Motion: thing.Still(maths.Subspace{Scale: 3})})
	from := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}
	if got := lone(world, from, 0); !within(float64(got.Depth), 2, 1e-6) {
		t.Errorf("the ball of scale three is %v away", got.Depth)
	}
}

// creeping is a shape that says its surface is a thousandth further on until a
// point is within half a unit of the plane z of zero, where it says the surface
// is there. A march from far off needs thousands of steps to reach it: one that
// stops at marchSteps gives up, and one with no such stop would arrive.
type creeping struct{}

// Bounds is everywhere.
func (creeping) Bounds() maths.Box { return maths.Everywhere() }

// Distance is a thousandth, and zero within half a unit of the plane.
func (creeping) Distance(point maths.Vec) float64 {
	if math.Abs(point.Z) <= 0.5 {
		return 0
	}
	return 1e-3
}

// A march finds a ball it is aimed at, from outside and from inside,
// misses one it only grazes, and gives up as a miss when its steps run
// out before it arrives.
func TestAMarchHitsMissesAndGivesUp(t *testing.T) {
	ball := struct{ shape.Shape }{shape.Sphere{Radius: 1}}
	inward := maths.Ray{At: maths.V(0, 0, 5), Dir: maths.V(0, 0, -1)}
	if at, met := march(ball, inward, 0, 100); !met ||
		!within(at, 4, 2*touch) {
		t.Errorf("aimed at the ball the march met %v, %v", at, met)
	}
	inside := maths.Ray{At: maths.Vec{}, Dir: maths.V(0, 0, 1)}
	if at, met := march(ball, inside, 0, 100); !met ||
		!within(at, 1, 2*touch) {
		t.Errorf("from inside the march met %v, %v", at, met)
	}
	grazing := maths.Ray{At: maths.V(1.01, 0, 5), Dir: maths.V(0, 0, -1)}
	if at, met := march(ball, grazing, 0, 100); met {
		t.Errorf("a ray a hundredth clear of the ball met it at %v", at)
	}
	// from five out, 4500 steps of a thousandth to the surface
	if at, met := march(creeping{}, inward, 0, 100); met {
		t.Errorf("a march out of steps met a surface at %v", at)
	}
}

// resting is a motion that stays at a place.
func resting(at maths.Vec) thing.Motion {
	return thing.Still(maths.Subspace{Origin: at})
}

// mirrors is a glossy black floor under a white sky, with a glossy
// black ball over it or not.
func mirrors(ball bool) *thing.World {
	glass := &thing.Material{Name: "glass", Gloss: 1}
	world := thing.NewWorld()
	world.Sky = even(maths.Grey(1))
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glass})
	if ball {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: glass, Motion: resting(maths.V(0, 3, 0))})
	}
	return world
}

// A mirror seen in a mirror is not followed: a ray mirrors once. The
// floor under a glossy ball shows the ball as black, with no lamp to
// light it, and beside it the floor shows the sky by fresnel's share.
func TestAMirrorInAMirrorIsNotFollowed(t *testing.T) {
	steep := maths.Ray{At: maths.V(0.3, 1, 0),
		Dir: maths.V(-0.3, -1, 0).Unit()}
	share := (&thing.Material{Gloss: 1}).Mirrors(-steep.Dir.Y)
	open := bright(lone(mirrors(false), steep, 0))
	if !within(open, share, 1e-6) {
		t.Fatalf("with no ball the floor mirrors %v, want %v", open,
			share)
	}
	under := lone(mirrors(true), steep, 0).Light()
	if under != (maths.Light{}) {
		t.Errorf("the ball in the floor is %v: mirrored twice", under)
	}
}

// echoing is a ball that returns radio, seen from plus z, under a radar
// up and behind the eye, with a wall between the radar and the ball's
// near point or not.
func echoing(wall bool) float64 {
	world := radarWorld(radarGround(nil), maths.V(0, 10, 10))
	if wall {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Name: "wall"},
			Motion:   resting(maths.V(0, 5, 6))})
	}
	return echoOf(world)
}

// A radar's echo comes back by how squarely the beam lands, and not at
// all from a point a wall stands between the radar and.
func TestAWallBetweenARadarAndItsTargetStopsTheEcho(t *testing.T) {
	near, radar := maths.V(0, 0, 2), maths.V(0, 10, 10)
	square := near.Unit().Dot(radar.Sub(near).Unit())
	if got := echoing(false); !within(got, 0.8*square, 1e-6) {
		t.Fatalf("with no wall the echo is %v, not %v", got, 0.8*square)
	}
	if got := echoing(true); got != 0 {
		t.Errorf("behind the wall the echo is %v", got)
	}
}

// In a band a gas passes e to the minus its own extinction there times
// its density times the way through, and gives off nothing.
func TestAGasInABandIsBeersLaw(t *testing.T) {
	const density, deep = 2e4, 2.0
	air, red := gas.Air(), eyeBands[0]
	gassy := thing.Volume{Medium: gas.Of{Shape: plate(density, -1, 1),
		In: air, Unit: 1}}
	got := throughIn(staging(gassy), alongZ, math.Inf(1), red)
	keep, added := got.keep, got.added
	want := math.Exp(-air.Extinction(red) * density * deep)
	if !(want < 0.99) || !within(keep, want, 1e-9) || added != 0 {
		t.Errorf("in the band the gas keeps %v and adds %v, want %v "+
			"and nought", keep, added, want)
	}
}

// behindBalls is a gamma source seen through a row of balls that each
// pass all of it.
func behindBalls(count int) float64 {
	passing := map[radiation.Range]float64{radiation.Gamma(): 1}
	world := thing.NewWorld()
	world.Add(source())
	for i := range count {
		at := maths.V(0, 0, -float64(i))
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4},
			Material: &thing.Material{Name: "glass",
				Radiant: thing.Spectrum{Passes: passing}},
			Motion: resting(at)})
	}
	eye := tenBack()
	lens := film.Pinhole{Wide: 2 * maths.Degree}
	picture := drawIn(world, eye, lens, 0, 3, 3, gammaBand)
	return float64(picture.At(1, 1).R)
}

// A ray in a band passes through as many things as passes allows and
// no more: through eight clear balls the source shows whole, through
// nine it is dark.
func TestARayPassesThroughOnlySoManyThings(t *testing.T) {
	if got := behindBalls(passes); !within(got, 5, 1e-6) {
		t.Errorf("through %d clear balls the source is %v, not 5",
			passes, got)
	}
	if got := behindBalls(passes + 1); got != 0 {
		t.Errorf("through %d clear balls the source is %v, want dark",
			passes+1, got)
	}
}

// A thing with no shape is not there: a world holding one draws as the
// empty world does, under the same sky, and nothing panics.
func TestAThingWithNoShapeIsNotThere(t *testing.T) {
	lens := film.Pinhole{}
	empty := thing.NewWorld()
	empty.Sky = even(maths.Grey(0.5))
	shapeless := thing.NewWorld()
	shapeless.Sky = even(maths.Grey(0.5))
	shapeless.Add(thing.Thing{})
	want := draw(empty, overTheScene, lens, 0, 8, 8)
	got := draw(shapeless, overTheScene, lens, 0, 8, 8)
	samePixels(t, got, want, "with a shapeless thing,")
}

// A subspace with no scale given has a scale of one, and one with a
// scale has that.
func TestUnitScale(t *testing.T) {
	if got := unitScale(maths.Subspace{}); got != 1 {
		t.Errorf("a subspace with no scale has %v", got)
	}
	if got := unitScale(maths.Subspace{Scale: 3}); got != 3 {
		t.Errorf("a subspace of scale three has %v", got)
	}
}

// A floor's normal faces whoever looks at it: up for a ray coming
// down, and down for a ray coming up from beneath.
func TestANormalFacesTheRay(t *testing.T) {
	floor := place(glowing(), 0)[0]
	falling, rising := maths.V(0.2, -1, 0).Unit(), maths.V(0.2, 1, 0).Unit()
	if got := floor.facing(maths.Vec{}, falling); got != maths.Up {
		t.Errorf("from above the floor's normal is %v", got)
	}
	if got := floor.facing(maths.Vec{}, rising); got != maths.Up.Neg() {
		t.Errorf("from beneath the floor's normal is %v", got)
	}
}

// ticking is a texture that paints the seconds it was ticked to.
type ticking struct{ seconds float64 }

// At is the seconds, as grey.
func (clock ticking) At(maths.Vec) maths.Light {
	return maths.Grey(clock.seconds)
}

// Tick is the texture at a moment.
func (clock ticking) Tick(moment time.Duration) texture.Texture {
	return ticking{seconds: moment.Seconds()}
}

// A placement is where something is, its scale with zero read as one, and its
// material's three textures each ticked to the picture's moment; with no
// material it has none of the three. A thing and a volume are placed by the
// same rule.
func TestAPlacementTicksAllThreeTextures(t *testing.T) {
	moment := 3 * time.Second
	worn := &thing.Material{Texture: ticking{}, Shine: ticking{},
		Rough: ticking{}}
	put := newPlacement(maths.Subspace{}, worn, moment)
	for name, each := range map[string]texture.Texture{
		"paint": put.paint, "shine": put.shine, "rough": put.rough} {
		if each == nil || each.At(maths.Vec{}).R != 3 {
			t.Errorf("the %s is not ticked to three seconds: %v",
				name, each)
		}
	}
	bare := newPlacement(maths.Subspace{Scale: 2}, nil, moment)
	if put.scale != 1 || bare.scale != 2 || bare.paint != nil ||
		bare.shine != nil || bare.rough != nil {
		t.Errorf("scales %v and %v; a bare placement %+v", put.scale,
			bare.scale, bare)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: worn})
	world.AddVolume(thing.Volume{Medium: plate(1, 0, 1), Material: worn})
	scene := stage(world, moment)
	thingAt := scene.crowd.things[0].rough.At(maths.Vec{}).R
	volumeAt := scene.volumes[0].rough.At(maths.Vec{}).R
	if thingAt != 3 || volumeAt != 3 {
		t.Errorf("a thing's roughness is ticked to %v and a volume's "+
			"to %v, not three", thingAt, volumeAt)
	}
}

// glassBefore is a gamma source seen through one ball that passes a
// half of gamma a unit, a unit in radius, at a scale.
func glassBefore(scale float64) float64 {
	passing := map[radiation.Range]float64{radiation.Gamma(): 0.5}
	world := thing.NewWorld()
	world.Add(source())
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "glass",
			Radiant: thing.Spectrum{Passes: passing}},
		Motion: thing.Still(maths.Subspace{Scale: scale})})
	eye := tenBack()
	lens := film.Pinhole{Wide: 2 * maths.Degree}
	picture := drawIn(world, eye, lens, 0, 3, 3, gammaBand)
	return float64(picture.At(1, 1).R)
}

// A thing passes a band by its Through to the power of the way through
// it in its own units. A ball two units through passes a quarter of a
// source of five, and passes the same at twice the size: a thing
// scaled up keeps its look, as a medium does.
func TestAThingPassesABandByItsOwnUnits(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		got := glassBefore(scale)
		if !within(got, 5*0.25, 1e-6) {
			t.Errorf("at scale %v the source through the ball is "+
				"%v, want 1.25", scale, got)
		}
	}
}
