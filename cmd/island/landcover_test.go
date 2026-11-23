package main

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// paintAt places a point on the test sphere at the given chart offsets and
// radial height and returns its position in world coordinates.
func paintAt(t *testing.T, site *siteBuilder, east, south, height float64) maths.Vec {
	t.Helper()
	up := site.chartUp(east, south)
	radius := site.frame.Radius().InMetres()
	return site.frame.Centre().Add(up.Scale(radius + height))
}

// forestedPaint is a palette over the test sphere with a forest of one
// hundred metres either way, heights 0 to 100 and a fade from 1 to 3 pixels.
func forestedPaint(t *testing.T) (altitudePaint, *siteBuilder) {
	t.Helper()
	site, _ := newTestSite(t, 0, 0)
	forest := &forestPaint{space: site.space, sphere: site.frame,
		bounds:  localBounds{MinEast: -100, MinSouth: -100, MaxEast: 100, MaxSouth: 100},
		minimum: 0, maximum: 100, fadeFrom: 1, fadeTo: 3, colour: maths.RGB(0, 1, 0)}
	paint := altitudePaint{frame: site.frame, bands: []heightBand{{Below: 200, Colour: [3]float64{1, 0, 0}}},
		base: maths.RGB(0.5, 0.5, 0.5), forest: forest}
	return paint, site
}

// TestAltitudePaintWithoutForest checks the palette colour at a height, and
// that the same value comes back from At and from Within at any footprint.
func TestAltitudePaintWithoutForest(t *testing.T) {
	paint, site := forestedPaint(t)
	paint.forest = nil
	low := paintAt(t, site, 0, 0, 50)
	if got := paint.At(low); got != maths.RGB(1, 0, 0) {
		t.Errorf("below the band edge: %v", got)
	}
	if got := paint.Within(low, 50); got != maths.RGB(1, 0, 0) {
		t.Errorf("a footprint changed a colour with no forest: %v", got)
	}
	if got := paint.At(paintAt(t, site, 0, 0, 500)); got != maths.RGB(0.5, 0.5, 0.5) {
		t.Errorf("above every band: %v", got)
	}
}

// TestAltitudePaintShiftsWithOrigin checks the paint keeps its world anchor
// when the render origin moves.
func TestAltitudePaintShiftsWithOrigin(t *testing.T) {
	paint, site := forestedPaint(t)
	paint.forest = nil
	paint.origin = maths.V(0, 1000, 0)
	world := paintAt(t, site, 0, 0, 50)
	if got := paint.At(world.Sub(paint.origin)); got != maths.RGB(1, 0, 0) {
		t.Errorf("a shifted local point: %v", got)
	}
}

// TestAltitudePaintBadPosition checks that a point at the sphere's centre has
// no height and gives not-a-number grey.
func TestAltitudePaintBadPosition(t *testing.T) {
	paint, site := forestedPaint(t)
	got := paint.At(site.frame.Centre())
	if !math.IsNaN(got.R) {
		t.Errorf("colour at the sphere centre = %v, want NaN", got)
	}
}

// TestAltitudePaintForestBlend checks the blend toward the aggregate colour
// at hand-worked footprints: nothing at or below the fade start, half at the
// middle of the fade, and the full cover beyond its end.
func TestAltitudePaintForestBlend(t *testing.T) {
	paint, site := forestedPaint(t)
	point := paintAt(t, site, 0, 0, 50)
	red, green := maths.RGB(1, 0, 0), maths.RGB(0, 1, 0)
	cases := []struct {
		footprint float64
		want      maths.Light
	}{
		{0.5, red}, {1, red},
		{2, red.Scale(0.5).Add(green.Scale(0.5))},
		{3, green}, {10, green},
	}
	for _, tc := range cases {
		got := paint.Within(point, tc.footprint)
		if !near(got.R, tc.want.R, 1e-9) || !near(got.G, tc.want.G, 1e-9) || !near(got.B, tc.want.B, 1e-9) {
			t.Errorf("footprint %v: %v, want %v", tc.footprint, got, tc.want)
		}
	}
	if paint.At(point) != red {
		t.Error("At must not blend")
	}
}

// TestAltitudePaintForestAreaCover checks that a sampled canopy area limits
// cover to one minus exp of minus the area density times the edge weight: an
// area density of ln 2 gives half cover and so a quarter blend at mid fade.
func TestAltitudePaintForestAreaCover(t *testing.T) {
	paint, site := forestedPaint(t)
	paint.forest.areaSampled, paint.forest.areaDensity = true, math.Ln2
	got := paint.Within(paintAt(t, site, 0, 0, 50), 2)
	if !near(got.R, 0.75, 1e-9) || !near(got.G, 0.25, 1e-9) {
		t.Errorf("blend = %v, want 0.75 red and 0.25 green", got)
	}
}

// TestAltitudePaintForestEdges checks the places the forest does not reach:
// beyond its bounds, outside its elevation band and inside a clearing; and the
// tapers of its edge and elevation fades.
func TestAltitudePaintForestEdges(t *testing.T) {
	paint, site := forestedPaint(t)
	red := maths.RGB(1, 0, 0)
	for name, point := range map[string]maths.Vec{
		"east of bounds":  paintAt(t, site, 150, 0, 50),
		"west of bounds":  paintAt(t, site, -150, 0, 50),
		"north of bounds": paintAt(t, site, 0, -150, 50),
		"south of bounds": paintAt(t, site, 0, 150, 50),
	} {
		if got := paint.Within(point, 10); got != red {
			t.Errorf("%s: %v, want the plain palette colour", name, got)
		}
	}
	paint.forest.minimum = 60
	if got := paint.Within(paintAt(t, site, 0, 0, 50), 10); got != red {
		t.Errorf("below the elevation band: %v", got)
	}
	paint.forest.minimum, paint.forest.maximum = 0, 40
	if got := paint.Within(paintAt(t, site, 0, 0, 50), 10); got != red {
		t.Errorf("above the elevation band: %v", got)
	}
	paint.forest.maximum = 100
	paint.forest.clearings = []clearing{{east: 10, south: 0, radius: 4}}
	paint.forest.clearance = 1
	if got := paint.Within(paintAt(t, site, 14.5, 0, 50), 10); got != red {
		t.Errorf("inside a clearing: %v", got)
	}
	if got := paint.Within(paintAt(t, site, 16, 0, 50), 10); got == red {
		t.Errorf("outside the clearing and its margin: %v", got)
	}
	paint.forest.clearings = nil
	paint.forest.edgeFade = 20
	if got := paint.Within(paintAt(t, site, 90, 0, 50), 10); !near(got.G, 0.5, 1e-6) {
		t.Errorf("halfway through the edge fade: %v, want half green", got)
	}
	paint.forest.edgeFade, paint.forest.heightFade = 0, 20
	if got := paint.Within(paintAt(t, site, 0, 0, 10), 10); !near(got.G, 0.5, 1e-6) {
		t.Errorf("halfway up the elevation fade: %v, want half green", got)
	}
}

// TestForestPaintChartAtPole checks the degenerate chart point: straight up
// from the site is the chart origin.
func TestForestPaintChartAtPole(t *testing.T) {
	_, site := forestedPaint(t)
	forest := forestPaint{space: site.space, sphere: site.frame}
	above := site.frame.Centre().Add(site.space.Orientation.Y.Scale(1e6))
	if got := forest.chart(above); got != (maths.Vec{}) {
		t.Errorf("chart of straight up = %v, want the origin", got)
	}
}
