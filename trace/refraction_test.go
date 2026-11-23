package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// water is a clear surface with seawater's visible index, black of its
// own, so whatever a test sees came through or off the surface.
func water() *thing.Material {
	return &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1}
}

// pooled is a clear plane at nought over a floor two under it that
// glows with luminance one: a pool with its own light at the bottom,
// no lamps and no sky, so every number is fresnel's.
func pooled(floor *thing.Material) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: water()})
	floorAt := resting(maths.V(0, -2, 0))
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: floor,
		Motion: floorAt})
	return world
}

// waterHead is the share water reflects at square incidence, for the
// index water() carries, 1.33, worked by hand from Fresnel's equations:
//
//	((n - 1) / (n + 1))^2 = (0.33 / 2.33)^2 = 0.141631^2 = 0.0200593
const waterHead = 0.0200593

// Straight down into still water, the ray does not bend, and the pool
// floor shows by one less the share water reflects at square incidence,
// 0.0200593 at an index of 1.33: 1 - 0.0200593 = 0.9799407.
func TestAPoolFloorShowsThroughStillWater(t *testing.T) {
	glow := &thing.Material{Glow: maths.Grey(1)}
	down := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0)}
	want := 1 - waterHead
	pool := pooled(glow)
	if got := bright(lone(pool, down, 0)); !within(got, want, 1e-6) {
		t.Fatalf("the floor shows %v through the surface, want %v",
			got, want)
	}
}

// A slanted ray bends at the surface by snell's law: a glowing ball
// hung on the refracted path lights the pixel, and the same ball hung
// on the unbent path does not.
func TestWaterBendsASlantedRayBySnell(t *testing.T) {
	slant := maths.Ray{At: maths.V(-1, 1, 0),
		Dir: maths.V(1, -1, 0).Unit()}
	// the bent way down from the entry at the origin
	sine := math.Sin(math.Pi/4) / 1.33
	bent := maths.V(sine, -math.Sqrt(1-sine*sine), 0)
	onPath := bent.Scale(1.5 / -bent.Y) // the bent point at y = -1.5
	offPath := maths.V(1.5, -1.5, 0)    // the unbent point there
	for _, place := range []struct {
		at   maths.Vec
		lit  bool
		what string
	}{{onPath, true, "on the bent path"},
		{offPath, false, "on the unbent path"}} {
		world := pooled(&thing.Material{})
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.2},
			Material: &thing.Material{Glow: maths.Grey(1)},
			Motion:   resting(place.at)})
		got := bright(lone(world, slant, 0))
		if place.lit && got <= 0 {
			t.Errorf("a ball %s is dark: the ray did not bend there",
				place.what)
		}
		if !place.lit && got > 0 {
			t.Errorf("a ball %s shows %v: the ray went straight",
				place.what, got)
		}
	}
}

// Two clear panes pass a straight ray with fresnel's toll at each:
// one less the share, squared. Glass of the default index, 1.5, reflects
// (0.5 / 2.5)^2 = 0.2^2 = 0.04 at square incidence, so the floor shows
// (1 - 0.04)^2 = 0.96^2 = 0.9216.
func TestTwoClearPanesEachTakeFresnelsShare(t *testing.T) {
	glass := &thing.Material{Gloss: 1, Through: 1} // index 1.5, the default
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glass})
	glassAt := resting(maths.V(0, -1, 0))
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glass,
		Motion: glassAt})
	floorAt := resting(maths.V(0, -3, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   floorAt})
	down := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0)}
	want := 0.9216
	if got := bright(lone(world, down, 0)); !within(got, want, 1e-6) {
		t.Fatalf("two panes pass %v, want %v", got, want)
	}
}

// An opaque surface is untouched by the new split: Through nought
// renders exactly as before, whatever the gloss.
func TestAnOpaqueSurfaceIsUntouchedByThrough(t *testing.T) {
	down := maths.Ray{At: maths.V(0, 1, 0), Dir: maths.V(0, -1, 0)}
	solid := pooled(&thing.Material{Glow: maths.Grey(1)})
	solid.Things[0].Material = &thing.Material{Name: "paint",
		Colour: maths.Grey(0.25), Gloss: 1, Index: 1.33}
	got := lone(solid, down, 0).Light()
	// with no lamps and no sky, an opaque unlit surface is black
	if got != (maths.Light{}) {
		t.Fatalf("an opaque surface leaks %v through itself", got)
	}
}

// A level height field of water reads exactly as the plane pool: the
// march meets the field, the bend is the same bend.
func TestAHeightFieldPoolReadsAsThePlanePool(t *testing.T) {
	level := make([]float64, 64*64)
	field, err := shape.NewHeightField(level, 64, 10)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: field, Material: water()})
	floorAt := resting(maths.V(0, -2, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   floorAt})
	down := maths.Ray{At: maths.V(0.4, 1, 0.3), Dir: maths.V(0, -1, 0)}
	want := 1 - waterHead
	if got := bright(lone(world, down, 0)); !within(got, want, 1e-4) {
		t.Fatalf("the floor shows %v through the field, want %v",
			got, want)
	}
}

func TestASpectralHeightFieldBoundaryRefractionUsesItsGeometricSide(t *testing.T) {
	core, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	xSlopes, zSlopes := make([]float64, 9), make([]float64, 9)
	for index := range xSlopes {
		xSlopes[index] = 10
	}
	field, err := core.WithApproximateSpectralNormals(xSlopes, zSlopes)
	if err != nil {
		t.Fatal(err)
	}
	material := water()
	placed := placedThing{thing: &thing.Thing{Shape: field,
		Material: material}, placement: placement{at: maths.Local(), scale: 1}}
	met := hit{thing: placed}
	point := maths.V(0, 0, 0)

	// This above-water-oriented spectral normal would falsely predict TIR
	// for a ray that the exact plane geometry transmits into air.
	exiting := maths.V(0.6, 0.8, 0)
	shading := met.normal(point, exiting)
	if _, total := refracted(exiting, shading, 1/material.Dense()); !total {
		t.Fatal("fixture did not make the shading normal predict false TIR")
	}
	optical := met.boundaryNormal(point, exiting)
	if !optical.Near(maths.V(0, -1, 0), 1e-14) {
		t.Fatalf("water-side optical normal = %v, want downward geometry normal", optical)
	}
	bent, total := refracted(exiting, optical, 1/material.Dense())
	if total || !(bent.Y > 0) {
		t.Fatalf("geometric exit refraction = %v, TIR=%t; want transmission into air", bent, total)
	}
	airOrigin := point.Add(optical.Scale(-lift))
	if gap := field.Distance(airOrigin); !(gap > 0) {
		t.Errorf("exit origin lies on the water side: signed gap %g", gap)
	}
	seen := look{holding: 1, inside: [mostHeld]held{of(material)}}
	boundary := seen.meets(material)
	if !boundary.real || boundary.entering {
		t.Fatalf("exit boundary = %+v", boundary)
	}
	if next := seen.passed(maths.Ray{At: airOrigin, Dir: bent}, material,
		boundary); next.holding != 0 || next.lastClearTIR {
		t.Errorf("after geometric exit the look holds %d media and last-TIR=%t",
			next.holding, next.lastClearTIR)
	}

	// A real water-to-air critical-angle ray still turns back into water
	// when classified with the exact interface normal.
	shallow := maths.V(math.Sqrt(.99), .1, 0)
	reflected, total := refracted(shallow, optical, 1/material.Dense())
	if !total {
		t.Fatal("a ray past the geometric critical angle did not TIR")
	}
	normalPart := optical.Scale(2 * shallow.Dot(optical))
	reflected = shallow.Sub(normalPart).Unit()
	waterOrigin := point.Add(optical.Scale(lift))
	if gap := field.Distance(waterOrigin); !(gap < 0) || !(reflected.Y < 0) {
		t.Errorf("TIR left the water side: gap=%g reflected=%v", gap, reflected)
	}
	if next := seen.turnedBack(maths.Ray{At: waterOrigin, Dir: reflected}); next.holding != 1 || !next.lastClearTIR {
		t.Errorf("TIR changed water stack to %d or last-TIR=%t",
			next.holding, next.lastClearTIR)
	}
}

// submerged is a look that starts inside a clear thing, as one that has
// bent into it once: the thing is among those it is inside, and one of
// its bends is spent. It is the eye's own look still, not a later leg.
func submerged(seen look, one *thing.Thing) look {
	inside := seen.passed(seen.ray, one.Material,
		boundary{real: true, entering: true})
	inside.onward = false
	return inside
}

// refracted turns a ray back whole past the critical angle, and bends
// it by snell's law inside it.
func TestRefractedKnowsTheCriticalAngle(t *testing.T) {
	// sixty degrees up inside water, against a surface facing down
	steep := maths.V(math.Sin(maths.Degree*60), math.Cos(maths.Degree*60), 0)
	if _, total := refracted(steep, maths.V(0, -1, 0), 1/1.33); !total {
		t.Error("sixty degrees inside water should reflect whole")
	}
	shallow := maths.V(math.Sin(maths.Degree*30), math.Cos(maths.Degree*30), 0)
	bent, total := refracted(shallow, maths.V(0, -1, 0), 1/1.33)
	if total {
		t.Fatal("thirty degrees inside water should leave")
	}
	if want := 1.33 * math.Sin(maths.Degree*30); !within(bent.X, want, 1e-9) {
		t.Errorf("the leaving sine is %v, want %v", bent.X, want)
	}
}

// layer is a layer of water one deep, a plane at nought and another at
// minus one, over a floor that glows with luminance one at minus three:
// a ray crosses the top into the water and the bottom out of it.
func layer() *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: water()})
	waterAt := resting(maths.V(0, -1, 0))
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: water(),
		Motion: waterAt})
	floorAt := resting(maths.V(0, -3, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   floorAt})
	return world
}

// A ray slanted at forty-five degrees meets the top of the water from
// above and the bottom of it from below, and each surface reflects the
// unpolarised share of Fresnel's equations, worked by hand at an index
// of 1.33.
//
// From above, the cosine of incidence is 0.707107, the sine of the
// refracted angle is 0.707107 / 1.33 = 0.531659, and the cosine of it
// is the root of 1 - 0.282659, 0.847 (0.846958). With i the cosine of
// incidence and t the cosine of the refracted angle:
//
//	rs = (i - n t) / (i + n t) = (0.707107 - 1.126455) / (0.707107 + 1.126455)
//	   = -0.419348 / 1.833562 = -0.228707
//	rp = (n i - t) / (n i + t) = (0.940452 - 0.846958) / (0.940452 + 0.846958)
//	   = 0.093494 / 1.787410 = 0.052307
//	R  = (rs^2 + rp^2) / 2 = (0.052307 + 0.002736) / 2 = 0.027521
//
// From below the ray is in the water at the refracted angle, whose
// cosine is 0.846958, and by reciprocity the share at that angle from
// below is the share at 45 degrees from above: 0.027521 again. Each
// surface passes 1 - 0.027521 = 0.972479, and the floor shows
// 0.972479^2 = 0.945714.
func TestAWaterLayerPassesFresnelsShareAtEachFace(t *testing.T) {
	slant := maths.Ray{At: maths.V(-1, 1, 0),
		Dir: maths.V(1, -1, 0).Unit()}
	const want = 0.945714
	stack := layer()
	if got := bright(lone(stack, slant, 0)); !within(got, want, 2e-6) {
		t.Fatalf("the floor shows %v through the layer, want %v", got, want)
	}
}

// A ray inside water, sixty degrees from the normal, meets the top past
// the critical angle, asin(1 / 1.33) = 48.75 degrees, since the sine of
// sixty degrees, 0.866, is more than 1 / 1.33 = 0.752. Nothing leaves:
// the whole of the light reflects, back down onto the floor, which
// shows its luminance of one with nothing taken.
func TestTotalInternalReflectionCarriesAllTheLight(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: water()})
	floorAt := resting(maths.V(0, -3, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   floorAt})
	up := maths.Ray{At: maths.V(0, -1, 0),
		Dir: maths.V(math.Sin(maths.Degree*60), math.Cos(maths.Degree*60), 0)}
	inside := submerged(look{ray: up}, world.Things[0])
	scene := stage(world, 0)
	if got := bright(shade(scene, inside)); !within(got, 1, 1e-6) {
		t.Fatalf("the floor shows %v after total internal reflection, want 1",
			got)
	}
}

// A ray inside water, thirty degrees from the normal, is under the
// critical angle and leaves into the air, where a glowing plane at
// three over the water shows what it passes. The sine of the leaving
// angle is 1.33 sin(30) = 0.665, whose cosine is 0.746843. With i the
// cosine of incidence, 0.866025, and t the cosine of the leaving angle,
// for light going from index 1.33 to 1:
//
//	rs = (n i - t) / (n i + t) = (1.151813 - 0.746843) / (1.151813 + 0.746843)
//	   = 0.404970 / 1.898656 = 0.213293
//	rp = (i - n t) / (i + n t) = (0.866025 - 0.993292) / (0.866025 + 0.993292)
//	   = -0.127267 / 1.859317 = -0.068453
//	R  = (rs^2 + rp^2) / 2 = (0.045493 + 0.004686) / 2 = 0.025090
//
// so the glowing plane shows 1 - 0.025090 = 0.974910.
func TestWaterLetsLightOutBelowTheCriticalAngle(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: water()})
	planeAt := resting(maths.V(0, 3, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   planeAt})
	up := maths.Ray{At: maths.V(0, -1, 0),
		Dir: maths.V(math.Sin(maths.Degree*30), math.Cos(maths.Degree*30), 0)}
	inside := submerged(look{ray: up}, world.Things[0])
	const want = 0.974910
	scene := stage(world, 0)
	if got := bright(shade(scene, inside)); !within(got, want, 2e-6) {
		t.Fatalf("the plane shows %v out of the water, want %v", got, want)
	}
}

// A slanted ray at forty-five degrees into the pool floor shows by one
// less the share water reflects at that angle, 0.027521 at an index of
// 1.33, worked by hand from Fresnel's equations in the layer test above:
// 1 - 0.027521 = 0.972479. Schlick's approximation gives 0.022172 here,
// a floor of 0.977828.
func TestAPoolFloorShowsObliquelyByFresnelsShare(t *testing.T) {
	glow := &thing.Material{Glow: maths.Grey(1)}
	slant := maths.Ray{At: maths.V(-1, 1, 0),
		Dir: maths.V(1, -1, 0).Unit()}
	const want = 0.972479
	pool := pooled(glow)
	if got := bright(lone(pool, slant, 0)); !within(got, want, 2e-6) {
		t.Fatalf("the floor shows %v through the surface, want %v", got, want)
	}
}

// A look that has bent through as many surfaces as it may, and meets
// water from inside past the critical angle, finds the surface opaque
// and lit as itself: here a surface that glows with luminance one
// quarter, so the look shows one quarter, with nothing taken by a share
// that nothing follows.
func TestATotalReflectorPastTheBudgetShadesAsItself(t *testing.T) {
	glowing := &thing.Material{Name: "water", Gloss: 1, Index: 1.33,
		Through: 1, Glow: maths.Grey(0.25)}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: glowing})
	up := maths.Ray{At: maths.V(0, -1, 0),
		Dir: maths.V(math.Sin(maths.Degree*60), math.Cos(maths.Degree*60), 0)}
	spent := submerged(look{ray: up}, world.Things[0])
	spent.crossed = refractions
	scene := stage(world, 0)
	if got := bright(shade(scene, spent)); !within(got, 0.25, 1e-6) {
		t.Fatalf("the spent look shows %v, want 0.25", got)
	}
}

// A surface of index 0.75 in air is the inside of a bubble in water,
// and a ray in from the air at sixty degrees is past its critical angle:
// the sine of sixty degrees, 0.866, is more than 0.75. It reflects
// whole, once, off the first surface: a glowing plane three over it
// shows luminance one, and not the two that the mirror and a reflected
// bend each carrying the whole would add to.
func TestAFirstSurfaceThatReflectsWholeFromTheAirIsFollowedOnce(t *testing.T) {
	bubble := &thing.Material{Name: "bubble", Gloss: 1, Index: 0.75,
		Through: 1}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: bubble})
	planeAt := resting(maths.V(0, 3, 0))
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Glow: maths.Grey(1)},
		Motion:   planeAt})
	down := maths.Ray{At: maths.V(0, 1, 0),
		Dir: maths.V(math.Sin(maths.Degree*60), -math.Cos(maths.Degree*60), 0)}
	if got := bright(lone(world, down, 0)); !within(got, 1, 1e-6) {
		t.Fatalf("the plane shows %v off the bubble, want 1", got)
	}
}

// A ball of water of index 1.33 in a surround that glows with
// luminance one reads less than one through its middle and less again
// at its rim, because the share the ball's far side reflects back into
// the ball is not followed. A look reads the share the near side
// mirrors, R, and the share that passes both sides, (1 - R)^2, and by
// reciprocity the far side's share is the near side's.
//
// Head on, R = 0.0200593 and the look reads 0.0200593 + 0.9799407^2 =
// 0.0200593 + 0.960284 = 0.980343.
//
// At 85 degrees of incidence, near the rim, the cosine is 0.087156, the
// refracted sine is 0.996195 / 1.33 = 0.749019 and its cosine
// 0.662556:
//
//	rs = (0.087156 - 0.881200) / (0.087156 + 0.881200) = -0.819992
//	rp = (0.115917 - 0.662556) / (0.115917 + 0.662556) = -0.702197
//	R  = (0.672387 + 0.493080) / 2 = 0.582733
//
// and the look reads 0.582733 + 0.417267^2 = 0.582733 + 0.174111 =
// 0.756844. Whole light, all of it counted, would read one.
func TestAWaterBallInASurroundReadsLessThanTheSurround(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: water()})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 50},
		Material: &thing.Material{Glow: maths.Grey(1)}})
	for _, c := range []struct {
		what   string
		impact float64
		want   float64
		tol    float64
	}{
		{"head on", 0, 0.980343, 1e-5},
		// near the rim the share changes fast with the point the ray
		// meets, so the reading is held to two ten-thousandths
		{"near the rim, at 85 degrees", math.Sin(maths.Degree * 85), 0.756844, 2e-4},
	} {
		ray := maths.Ray{At: maths.V(c.impact, 0, 5), Dir: maths.V(0, 0, -1)}
		if got := bright(lone(world, ray, 0)); !within(got, c.want, c.tol) {
			t.Errorf("%s: the ball reads %v, want %v", c.what, got, c.want)
		}
	}
}
