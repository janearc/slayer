package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// discClose is how near a sun drawn as a disc is held to the same sun as
// a point where the two must read alike: a hundredth of the point's, as
// the satin floor's test holds them.
const discClose = 0.01

// wideSun is the size of a sun drawn wide enough for the small faces of
// rough water to see it by every one of their tilts, a little over a
// radian across.
const wideSun = 1.2

// sunOf is a sun of one toward a way, drawn as a disc of the size, or
// as a point.
func sunOf(toward maths.Vec, size float64, disc bool) film.Sun {
	return film.Sun{Toward: toward, Strength: maths.Grey(1), Size: size,
		Disc: disc}
}

// lookAt is the light of a look in a world, left out of the disc or not.
func lookAt(world *thing.World, ray maths.Ray, discless bool) float64 {
	scene := stage(world, 0)
	return bright(shade(scene, look{ray: ray,
		discless: discless}))
}

// A look that leaves the sun's disc out finds the sky without it, and
// one that does not finds it: the sun is a disc a quarter of a radian
// across straight overhead, and the look goes straight up.
func TestADisclessLookFindsNoDiscInTheSky(t *testing.T) {
	world := thing.NewWorld()
	toward := maths.V(0, 1, 0)
	world.Lamp(sunOf(toward, 0.25, true))
	up := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.V(0, 1, 0)}
	if got := lookAt(world, up, true); got != 0 {
		t.Errorf("the look that leaves the disc out finds %v", got)
	}
}

// Anti: the look that does not leave it out finds the disc, its light
// over its solid angle, some twenty times its strength.
func TestALookThatKeepsTheDiscFindsItInTheSky(t *testing.T) {
	world := thing.NewWorld()
	toward := maths.V(0, 1, 0)
	world.Lamp(sunOf(toward, 0.25, true))
	up := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.V(0, 1, 0)}
	if got := lookAt(world, up, false); !(got > 1) {
		t.Errorf("the look that keeps the disc finds %v, want over 1", got)
	}
}

// A pane the look passes on its way to the sky leaves it as it was: a
// look that leaves the disc out still leaves it out beyond a pane of
// glass, and finds none of the sun there.
func TestADisclessLookStaysDisclessThroughAClearPane(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Through: 1, Index: 1.5}})
	toward := maths.V(0, 1, 0)
	world.Lamp(sunOf(toward, 0.25, true))
	up := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.V(0, 1, 0)}
	if got := lookAt(world, up, true); !(got < 1e-6) {
		t.Errorf("the look finds %v of the disc beyond a pane", got)
	}
}

// Anti: the look that keeps the disc finds it beyond the same pane, by
// what the pane lets through.
func TestALookThatKeepsTheDiscFindsItBeyondAClearPane(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Through: 1, Index: 1.5}})
	toward := maths.V(0, 1, 0)
	world.Lamp(sunOf(toward, 0.25, true))
	up := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.V(0, 1, 0)}
	if got := lookAt(world, up, false); !(got > 1) {
		t.Errorf("the look finds %v of the disc beyond a pane, want "+
			"over 1", got)
	}
}

// glassFromWithin is a look inside a pane of glass going up at a
// grazing angle, past the critical angle, which the surface turns back
// whole, and a world with a sun drawn as a disc along the way it is
// turned back.
func glassFromWithin() (*thing.World, maths.Ray, *thing.Material) {
	glass := &thing.Material{Through: 1, Index: 1.5}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glass})
	way := maths.V(0.9, 0.436, 0).Unit()
	flipped := maths.V(way.X, -way.Y, way.Z)
	world.Lamp(sunOf(flipped, 0.25, true))
	return world, maths.Ray{At: maths.V(0, -1, 0), Dir: way}, glass
}

// A look turned back whole inside glass, having bent already, is still
// a look that leaves the disc out, and finds none of the sun along the
// way it is turned. A look that has not bent is mirrored by the glass
// instead, and the glass shows no highlight to leave the disc out for.
func TestADisclessLookTurnedBackInsideGlassFindsNoDisc(t *testing.T) {
	world, ray, glass := glassFromWithin()
	seen := look{ray: ray, discless: true, holding: 1, crossed: 1}
	seen.inside[0] = of(glass)
	scene := stage(world, 0)
	if got := bright(shade(scene, seen)); !(got < 1e-6) {
		t.Errorf("the turned look finds %v of the disc", got)
	}
}

// Anti: the look that keeps the disc finds it, whole, along the way it
// is turned.
func TestALookTurnedBackInsideGlassFindsTheDisc(t *testing.T) {
	world, ray, glass := glassFromWithin()
	seen := look{ray: ray, holding: 1, crossed: 1}
	seen.inside[0] = of(glass)
	scene := stage(world, 0)
	if got := bright(shade(scene, seen)); !(got > 1) {
		t.Errorf("the turned look finds %v of the disc, want over 1", got)
	}
}

// satinUnderAPane is a satin floor, glossy and black, under a pane of
// clear glass a metre over it that casts no shadow, and a sun toward the
// way the floor mirrors a look that comes down through the pane, drawn
// as a disc or as a point. It returns the light of that look.
func satinUnderAPane(satin float64, disc bool) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Gloss: 1, Satin: satin}})
	paneAt := resting(maths.V(0, 1, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Through: 1, Index: 1.0001},
		Attrs:    thing.Attrs{Clear: true},
		Motion:   paneAt})
	world.Lamp(film.Sun{Toward: maths.V(0, 0.6, 0.8),
		Strength: maths.Grey(1), Size: 0.53 * maths.Degree, Disc: disc})
	look := maths.Ray{At: maths.V(0, 3, -4), Dir: maths.V(0, -0.6, 0.8)}
	return bright(lone(world, look, 0))
}

// A satin floor seen through a pane shows the sun once, drawn as a disc
// or as a point. The look bends through the pane to the floor and the
// floor's mirrored look goes up to the pane, which does not pass a
// mirrored look: the highlight is the sun's only light there.
func TestASatinFloorUnderAPaneShowsTheSunOnceDiscOrPoint(t *testing.T) {
	point, disc := satinUnderAPane(0.3, false), satinUnderAPane(0.3, true)
	if !(point > 0) {
		t.Fatalf("the satin floor under a pane shows no highlight")
	}
	if !within(disc, point, discClose*point) {
		t.Errorf("a disc sun reads %v under the pane, a point sun %v",
			disc, point)
	}
}

// Anti: a floor with no satin has no highlight, and under a pane that
// does not pass its mirrored look it shows neither the disc nor the
// point: the sun is seen once only where something shows it.
func TestAMirrorFloorUnderAPaneShowsNoSunDiscOrPoint(t *testing.T) {
	point, disc := satinUnderAPane(0, false), satinUnderAPane(0, true)
	if point != 0 || disc != 0 {
		t.Errorf("a mirror floor under a pane reads %v of a disc sun and "+
			"%v of a point sun", disc, point)
	}
}

// chopAt is a world of a flat sea of rough water, whose waves it does
// not hold have the slope variance of 0.01, under a sun toward the way a
// look from above is mirrored, drawn wide as a disc or as a point, and
// the light of that look. The water's own highlight has the slope
// variance given.
func chopAt(variance float64, disc bool) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: chop{unseen: 0.01},
		Material: &thing.Material{Through: 1, Gloss: 1, Index: waterIndex,
			SlopeVariance: variance}})
	toward := maths.V(0, 0.8, 0.6)
	world.Lamp(sunOf(toward, wideSun, disc))
	ray := maths.Ray{At: maths.V(3, 4, -3), Dir: maths.V(0, -0.8, 0.6)}
	return bright(lone(world, ray, 0))
}

// Rough water shows the sun once, drawn as a disc or as a point, where
// its highlight is as wide as its faces' slopes: the faces' mirrored
// looks leave the disc out, and the highlight is the sun.
func TestRoughWaterShowsTheSunOnceWhereTheHighlightIsAsWideAsTheFaces(
	t *testing.T) {
	point, disc := chopAt(0.01, false), chopAt(0.01, true)
	if !(point > 0) {
		t.Fatalf("the rough water shows no highlight")
	}
	if !within(disc, point, discClose*point) {
		t.Errorf("a disc sun reads %v on rough water, a point sun %v",
			disc, point)
	}
}

// Anti: where the highlight's lobe is not as wide as the faces' slopes
// the disc is not dropped for it: the faces see the disc as they see
// the sky, and it reads over the point's.
func TestRoughWaterKeepsTheDiscWhereTheHighlightIsAnotherWidth(
	t *testing.T) {
	point, disc := chopAt(0.02, false), chopAt(0.02, true)
	if !(point > 0) || !(disc > (1+discClose)*point) {
		t.Errorf("a disc sun reads %v on rough water, a point sun %v, "+
			"want the disc over", disc, point)
	}
}

// Rough water with no highlight shows a sun drawn as a disc once, by its
// faces, which see it as they see the sky, and a sun drawn as a point
// not at all: it is in no sky. Its anti-test is the first of the rough
// water tests above, where a highlight as wide as the faces' slopes takes
// the disc from them and the water reads as the point's does.
func TestRoughWaterWithNoHighlightShowsTheDiscByItsFaces(t *testing.T) {
	point, disc := chopAt(0, false), chopAt(0, true)
	if point != 0 || !(disc > 0.01) {
		t.Errorf("a disc sun reads %v on rough water with no highlight, "+
			"a point sun %v", disc, point)
	}
}

// A satin floor under a sky shows the sky it mirrors whole when the sun,
// drawn as a disc, is elsewhere: the disc is left out of the mirrored
// sky and nothing else is, so the floor reads as it does under the same
// sun drawn as a point.
func TestASatinFloorKeepsTheSkyItMirrorsWithTheSunElsewhere(t *testing.T) {
	read := func(disc bool) float64 {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Gloss: 1, Satin: 0.3}})
		world.Sky = even(maths.Grey(1))
		world.Lamp(film.Sun{Toward: maths.V(0, 0.6, -0.8),
			Strength: maths.Grey(1), Size: 0.53 * maths.Degree,
			Disc: disc})
		look := maths.Ray{At: maths.V(0, 3, -4), Dir: maths.V(0, -0.6, 0.8)}
		return bright(lone(world, look, 0))
	}
	point, disc := read(false), read(true)
	if !(point > 0.04) || !within(disc, point, discClose*point) {
		t.Errorf("a disc sun elsewhere reads %v, and a point sun %v, "+
			"over four hundredths", disc, point)
	}
}

// Anti: a floor with no satin that mirrors the sky shows the disc where
// the sun is: the disc is left out for a highlight and for nothing else.
func TestAMirrorFloorShowsTheDiscAtItsMirroredWay(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Gloss: 1}})
	world.Sky = even(maths.Grey(1))
	world.Lamp(film.Sun{Toward: maths.V(0, 0.6, 0.8),
		Strength: maths.Grey(1), Size: 0.53 * maths.Degree, Disc: true})
	look := maths.Ray{At: maths.V(0, 3, -4), Dir: maths.V(0, -0.6, 0.8)}
	if got := bright(lone(world, look, 0)); !(got > 1) {
		t.Errorf("the mirror floor reads %v at the sun's disc, want over 1",
			got)
	}
}
