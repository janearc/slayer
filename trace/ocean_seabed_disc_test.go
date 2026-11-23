package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// seabedTilt is how far the seabed is turned about x, a twentieth of a
// radian: enough that the sea's own mirror of the sun, along the look's
// mirrored way, is a long way off the way the seabed's mirrored look
// leaves the sea, and misses a disc half a degree across.
const seabedTilt = 0.05

// seabedExit is the way a look from above the sea along 0, -0.6, 0.8 leaves
// it again after the seabed mirrors it: bent into the water by snell,
// mirrored by the seabed turned by the tilt, and bent out again.
func seabedExit() maths.Vec {
	across := 0.8 / waterIndex
	water := maths.V(0, -math.Sqrt(1-across*across), across)
	normal := maths.V(0, math.Cos(seabedTilt), math.Sin(seabedTilt))
	normalPart := normal.Scale(2 * water.Dot(normal))
	back := water.Sub(normalPart)
	out := back.Z * waterIndex
	return maths.V(0, math.Sqrt(1-out*out), out)
}

// seabedSun is the light of a look from above a flat tagged sea, nearly
// straight down, over a tilted seabed of the satin given, under a sun
// along the way the seabed's mirror leaves the sea, drawn as a disc or as
// a point, and under a sky of the light given. The sea casts no shadow,
// so the highlight at the seabed shows the sun.
func seabedSun(t *testing.T, satin float64, disc bool, sky float64) float64 {
	t.Helper()
	world := seabedWorld(t, 2, true)
	world.Sky = even(maths.Grey(sky))
	world.Things[0].Attrs.Clear = true
	bed := world.Things[1]
	bed.Material.Satin = satin
	tilt := maths.Rotation(maths.V(1, 0, 0), seabedTilt)
	bed.Motion = thing.Still(maths.Subspace{Origin: maths.V(0, -5, 0),
		Orientation: tilt})
	world.Lamp(film.Sun{Toward: seabedExit(), Strength: maths.Grey(1),
		Size: 0.53 * maths.Degree, Disc: disc})
	ray := maths.Ray{At: maths.V(3, 3, -4), Dir: maths.V(0, -0.6, 0.8)}
	return bright(lone(world, ray, 0))
}

// A satin seabed under the tagged sea shows the sun once, drawn as a
// disc or as a point. The seabed's mirrored look leaves the sea along
// the sun, through the sea's clear surface that the sea's event policy
// lets a mirrored look pass, and finds the disc there if the look is not
// told to leave it out: the highlight is the sun already.
func TestASatinSeabedUnderTheTaggedSeaShowsTheSunOnceDiscOrPoint(
	t *testing.T) {
	point := seabedSun(t, 0.3, false, 1)
	disc := seabedSun(t, 0.3, true, 1)
	if !(point > 0) {
		t.Fatalf("the satin seabed shows no highlight of a point sun")
	}
	if !within(disc, point, discClose*point) {
		t.Errorf("a disc sun reads %v over the satin seabed, a point sun "+
			"%v", disc, point)
	}
}

// The two paths by which a sky of one reaches a look over a mirror
// seabed, with the sun a point and so in no sky, worked by hand. The look
// meets the sea at a cosine of 0.6, near Brewster's angle for 1.333, and
// the sea mirrors the sky by 0.039. The rest, 0.961, bends in, the
// seabed's dielectric mirror takes about 0.04 of it, and it bends out
// again by about 0.96: 0.961 x 0.04 x 0.96 = 0.037, which with the first
// is 0.076; the seabed's own share is held to 0.036, so 0.075.
const (
	seaMirrorOfSky  = 0.039
	seabedPathOfSky = 0.036
	skyOverSeabed   = seaMirrorOfSky + seabedPathOfSky
	// skyOverSeabedClose is how near the reading is held to that: three
	// parts in a hundred, the seabed's share being known to a hundredth.
	skyOverSeabedClose = 0.002
)

// Anti: a seabed with no satin has no highlight, and its mirror shows
// the disc along the way it leaves the sea. A point sun reads the sky's
// two paths only, 0.075. The disc adds its radiance, its strength over
// its solid angle 2 pi (1 - cos 0.265 degrees), some 14,900, by the
// seabed's path, 0.036: about 536 more, which is the double count the
// satin test would report. It also shows the sun is where the satin test
// puts it.
func TestAMirrorSeabedUnderTheTaggedSeaShowsTheDiscByItsMirror(
	t *testing.T) {
	point, disc := seabedSun(t, 0, false, 1), seabedSun(t, 0, true, 1)
	radiance := 1 / (2 * math.Pi * (1 - math.Cos(0.53*maths.Degree/2)))
	if !within(point, skyOverSeabed, skyOverSeabedClose) {
		t.Errorf("a point sun reads %v over the mirror seabed, want the "+
			"sky's two paths, %v", point, skyOverSeabed)
	}
	if want := seabedPathOfSky * radiance; !within(disc-point, want,
		0.05*want) {
		t.Errorf("a disc sun adds %v over the mirror seabed, want %v",
			disc-point, want)
	}
}

// Anti: under a sky of nothing the point sun reads nothing over the
// mirror seabed: the 0.075 above is the sky by the two paths and no
// light leaks from the sun, the sea or the seabed.
func TestAMirrorSeabedUnderTheTaggedSeaAndNoSkyShowsNoPointSun(
	t *testing.T) {
	if got := seabedSun(t, 0, false, 0); got != 0 {
		t.Errorf("a point sun under no sky reads %v over the mirror "+
			"seabed", got)
	}
}
