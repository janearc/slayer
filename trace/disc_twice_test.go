package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// sunInTheFloor is the light of a look at a black glossy floor of a
// satin, along the way that mirrors to a sun of one, the sun drawn as a
// disc or as a point.
func sunInTheFloor(satin float64, disc bool) float64 {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Gloss: 1, Satin: satin}})
	world.Lamp(film.Sun{Toward: maths.V(0, 0.6, 0.8),
		Strength: maths.Grey(1), Size: 0.53 * maths.Degree, Disc: disc})
	look := maths.Ray{At: maths.V(0, 3, -4), Dir: maths.V(0, -0.6, 0.8)}
	return bright(lone(world, look, 0))
}

// A satin surface shows a lamp as a highlight: the lamp's light spread
// by the surface's small faces, which is all of the lamp's light the
// surface mirrors toward the eye. Whether the sun is drawn as a disc in
// the sky or as a point changes nothing of how much light it sends the
// floor, so the look that mirrors to the sun reads the same either way.
//
// Drawn as a disc, the sun is also added to the sky a mirrored look
// meets, its light over its disc's solid angle, some 15,000 times its
// strength, by fresnel's share: the sun's mirrored light counted twice.
func TestASatinFloorShowsTheSunOnceDiscOrPoint(t *testing.T) {
	point, disc := sunInTheFloor(0.3, false), sunInTheFloor(0.3, true)
	if !(point > 0) {
		t.Fatalf("the satin floor shows no highlight of a point sun")
	}
	if !within(disc, point, 0.01*point) {
		t.Errorf("the satin floor shows a disc sun as %v and the same "+
			"sun as a point as %v", disc, point)
	}
}

// Anti: a floor with no satin has no highlight, and is a mirror. It
// shows a sun drawn as a disc and nothing of a sun drawn as a point,
// which is in no sky: there the disc is the only way the sun is seen.
func TestAMirrorFloorShowsOnlyASunDrawnAsADisc(t *testing.T) {
	point, disc := sunInTheFloor(0, false), sunInTheFloor(0, true)
	if point != 0 || !(disc > 1) {
		t.Errorf("the mirror floor shows a disc sun as %v and a point "+
			"sun as %v", disc, point)
	}
}
