package main

import (
	"context"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// diamond builds an octahedron whose horizontal extent is half a unit on x
// and z and whose height runs from zero to two. Its vertical shadow is the
// square |x| + |z| <= 1, whose area is 2, half of the 4 square metre bounds.
func diamond(t *testing.T) *mesh.Mesh {
	t.Helper()
	var points []maths.Vec
	var triangles [][3]int
	bin := crownBin{bounds: maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}, used: true}
	bin.append(&points, &triangles)
	made, err := mesh.New(points, triangles, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// diamondPrototype wraps the diamond as a one-material prototype.
func diamondPrototype(t *testing.T) prototype {
	t.Helper()
	made := diamond(t)
	return prototype{parts: []thing.Thing{{Shape: made, Material: &thing.Material{Name: "bark"}}}, bounds: made.Bounds()}
}

// TestCrownBinInclude checks that the first point sets a degenerate box and
// later points widen it on each axis.
func TestCrownBinInclude(t *testing.T) {
	var bin crownBin
	bin.include(maths.V(1, 2, 3))
	if !bin.used || bin.bounds.Min != maths.V(1, 2, 3) || bin.bounds.Max != maths.V(1, 2, 3) {
		t.Fatalf("first point: %+v", bin)
	}
	bin.include(maths.V(-1, 5, 3))
	bin.include(maths.V(0, 0, 9))
	if bin.bounds.Min != maths.V(-1, 0, 3) || bin.bounds.Max != maths.V(1, 5, 9) {
		t.Errorf("bounds = %+v", bin.bounds)
	}
}

// TestCrownBinAppend checks the octahedron of an occupied bin and that empty
// and flat bins add nothing.
func TestCrownBinAppend(t *testing.T) {
	var points []maths.Vec
	var triangles [][3]int
	(crownBin{}).append(&points, &triangles)
	flat := crownBin{bounds: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(4, 0, 4)}, used: true}
	flat.append(&points, &triangles)
	if len(points) != 0 || len(triangles) != 0 {
		t.Fatalf("an empty or flat bin added %d points, %d faces", len(points), len(triangles))
	}
	full := crownBin{bounds: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(2, 4, 6)}, used: true}
	full.append(&points, &triangles)
	if len(points) != 6 || len(triangles) != 8 {
		t.Fatalf("%d points and %d faces, want 6 and 8", len(points), len(triangles))
	}
	centre := maths.V(1, 2, 3)
	want := []maths.Vec{centre.Add(maths.V(1, 0, 0)), centre.Add(maths.V(0, 0, -3)), centre.Add(maths.V(-1, 0, 0)),
		centre.Add(maths.V(0, 0, 3)), centre.Add(maths.V(0, 2, 0)), centre.Add(maths.V(0, -2, 0))}
	for index := range want {
		if points[index] != want[index] {
			t.Errorf("point %d = %v, want %v", index, points[index], want[index])
		}
	}
	full.append(&points, &triangles)
	if triangles[8][0] != 6 || len(points) != 12 {
		t.Errorf("a second bin did not offset its indices: %v", triangles[8])
	}
}

// TestCrownPoints checks the eight corners of a mesh, a group's transformed
// members, and the refusal of other shapes.
func TestCrownPoints(t *testing.T) {
	made := diamond(t)
	count := 0
	seen := map[maths.Vec]bool{}
	visit := func(point maths.Vec) { count++; seen[point] = true }
	if err := crownPoints(made, maths.Local(), visit); err != nil {
		t.Fatal(err)
	}
	if count != 8 || len(seen) != 8 || !seen[maths.V(-1, 0, -1)] || !seen[maths.V(1, 2, 1)] {
		t.Errorf("%d corners, %d distinct", count, len(seen))
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: made, Space: maths.NewSubspace(maths.V(10, 0, 0), maths.Identity, 1)}})
	if err != nil {
		t.Fatal(err)
	}
	seen = map[maths.Vec]bool{}
	if err := crownPoints(group, maths.NewSubspace(maths.V(0, 100, 0), maths.Identity, 1), visit); err != nil {
		t.Fatal(err)
	}
	if !seen[maths.V(9, 100, -1)] || !seen[maths.V(11, 102, 1)] {
		t.Errorf("group corners were not carried through both poses: %v", seen)
	}
	if err := crownPoints(distanceOnly{}, maths.Local(), visit); err == nil {
		t.Error("an unsupported shape was accepted")
	}
	if err := crownPoints(&shape.MeshGroup{}, maths.Local(), visit); err != nil {
		t.Errorf("an empty group: %v", err)
	}
}

// TestCrownRefusals checks flat sources, unsupported shapes and parts whose
// corners all fall in flat bins.
func TestCrownRefusals(t *testing.T) {
	flat := prototype{bounds: maths.Box{Min: maths.V(0, 1, 0), Max: maths.V(1, 1, 1)}}
	if _, _, err := crown(flat); err == nil || !containsText(err.Error(), "needs a specimen with height") {
		t.Errorf("flat source: %v", err)
	}
	odd := prototype{parts: []thing.Thing{{Shape: distanceOnly{}}}, bounds: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 1)}}
	if _, _, err := crown(odd); err == nil || !containsText(err.Error(), "unsupported crown source") {
		t.Errorf("unsupported shape: %v", err)
	}
	corners := diamondPrototype(t)
	if _, _, err := crown(corners); err == nil || !containsText(err.Error(), "no spatial extent") {
		t.Errorf("corners in separate bins: %v", err)
	}
	empty := prototype{parts: []thing.Thing{{Shape: &shape.MeshGroup{}}}, bounds: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 1)}}
	if _, _, err := crown(empty); err == nil || !containsText(err.Error(), "no spatial extent") {
		t.Errorf("empty part: %v", err)
	}
}

// TestVerticalArea checks the sampled shadow of the diamond against the
// exact area of 2 and the refusals for bad sampling.
func TestVerticalArea(t *testing.T) {
	value := diamondPrototype(t)
	got, err := verticalArea(t.Context(), value, 64)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-2) > 0.1 {
		t.Errorf("area = %v, want about 2", got)
	}
	for _, side := range []int{7, 129, 0} {
		if _, err := verticalArea(t.Context(), value, side); err == nil || !containsText(err.Error(), "sampling side") {
			t.Errorf("side %d: %v", side, err)
		}
	}
	flat := prototype{bounds: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 0, 1)}}
	if _, err := verticalArea(t.Context(), flat, 8); err == nil || !containsText(err.Error(), "sampling bounds") {
		t.Errorf("flat bounds: %v", err)
	}
	opaque := prototype{parts: []thing.Thing{{Shape: distanceOnly{}}}, bounds: value.bounds}
	if _, err := verticalArea(t.Context(), opaque, 8); err == nil || !containsText(err.Error(), "exact mesh intersections") {
		t.Errorf("non-mesh part: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := verticalArea(ctx, value, 8); err == nil {
		t.Error("a cancelled context was ignored")
	}
}

// TestProjectedPartAreas checks per-material shadows on one grid, and its
// refusals.
func TestProjectedPartAreas(t *testing.T) {
	value := diamondPrototype(t)
	value.parts = append(value.parts, thing.Thing{Shape: diamond(t), Material: &thing.Material{Name: "leaf"},
		Motion: thing.Still(maths.NewSubspace(maths.V(0.5, 0, 0), maths.Identity, 1))})
	domain := maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1.5, 2, 1)}
	areas, err := projectedPartAreas(t.Context(), value, domain, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 || math.Abs(areas[0]-2) > 0.1 || math.Abs(areas[1]-2) > 0.1 {
		t.Errorf("areas = %v, want two near 2", areas)
	}
	bad := []struct {
		name   string
		domain maths.Box
		side   int
	}{
		{"side small", domain, 4}, {"side large", domain, 200},
		{"no width", maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(0, 1, 1)}, 8},
		{"no depth", maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 0)}, 8},
		{"no height", maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 0, 1)}, 8},
		{"unbounded", maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(math.Inf(1), 1, 1)}, 8},
	}
	for _, tc := range bad {
		if _, err := projectedPartAreas(t.Context(), value, tc.domain, tc.side); err == nil || !containsText(err.Error(), "probe grid") {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	opaque := prototype{parts: []thing.Thing{{Shape: distanceOnly{}}}}
	if _, err := projectedPartAreas(t.Context(), opaque, domain, 8); err == nil || !containsText(err.Error(), "exact mesh intersections") {
		t.Errorf("non-mesh part: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := projectedPartAreas(ctx, value, domain, 8); err == nil {
		t.Error("a cancelled context was ignored")
	}
}

// TestShrinkCrownPart checks that scaling narrows x and z about the axis and
// leaves height, and the refusals.
func TestShrinkCrownPart(t *testing.T) {
	part := thing.Thing{Shape: diamond(t), Material: &thing.Material{Name: "bark"}}
	same, err := shrinkCrownPart(part, 1)
	if err != nil || same.Shape != part.Shape {
		t.Fatalf("scale one: %v", err)
	}
	half, err := shrinkCrownPart(part, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	box := half.Shape.(*mesh.Mesh).Bounds()
	if box.Min != maths.V(-0.5, 0, -0.5) || box.Max != maths.V(0.5, 2, 0.5) || half.Material != part.Material {
		t.Errorf("shrunk bounds = %+v", box)
	}
	if got := part.Shape.(*mesh.Mesh).Bounds(); got.Max.X != 1 {
		t.Errorf("the source mesh was altered: %+v", got)
	}
	for _, scale := range []float64{0, -1, 1.5, math.NaN(), math.Inf(1)} {
		if _, err := shrinkCrownPart(part, scale); err == nil || !containsText(err.Error(), "invalid crown area scale") {
			t.Errorf("scale %v: %v", scale, err)
		}
	}
	if _, err := shrinkCrownPart(thing.Thing{Shape: distanceOnly{}}, 0.5); err == nil || !containsText(err.Error(), "flat crown mesh") {
		t.Errorf("non-mesh part: %v", err)
	}
	flat := thing.Thing{Shape: diamond(t)}
	if _, err := shrinkCrownPart(flat, 1e-320); err == nil {
		t.Log("a vanishing scale produced a mesh")
	}
}

// distanceOnly is a shape that answers distance queries but no exact rays.
type distanceOnly struct{}

// Bounds returns a unit box.
func (distanceOnly) Bounds() maths.Box {
	return maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 1)}
}

// Distance returns zero everywhere.
func (distanceOnly) Distance(maths.Vec) float64 { return 0 }

// TestCrownBinsCorners checks a source whose corners share one bin: two
// diamonds in the same quadrant and height slice give exactly one octahedron
// that spans their combined corners.
func TestCrownBinsCorners(t *testing.T) {
	made := diamond(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: made, Space: maths.NewSubspace(maths.V(2, 0, 2), maths.Identity, 1)},
		{Mesh: made, Space: maths.NewSubspace(maths.V(5, 0, 5), maths.Identity, 1)}})
	if err != nil {
		t.Fatal(err)
	}
	material := &thing.Material{Name: "bark"}
	source := prototype{parts: []thing.Thing{{Shape: group, Material: material}},
		bounds: maths.Box{Min: maths.V(1, 0, 1), Max: maths.V(6, 32, 6)}}
	proxy, faces, err := crown(source)
	if err != nil {
		t.Fatal(err)
	}
	if faces != 8 || len(proxy.parts) != 1 || proxy.parts[0].Material != material {
		t.Fatalf("faces %d parts %d", faces, len(proxy.parts))
	}
	box := proxy.parts[0].Shape.(*mesh.Mesh).Bounds()
	if box.Min != maths.V(1, 0, 1) || box.Max != maths.V(6, 2, 6) {
		t.Errorf("octahedron bounds = %+v, want x and z from 1 to 6 and y from 0 to 2", box)
	}
	if proxy.bounds != source.bounds {
		t.Errorf("proxy bounds = %+v, want the source bounds", proxy.bounds)
	}
}

// widened gives a prototype sharing the source's materials whose parts are the
// diamond stretched horizontally by factor, so its shadow grows by factor squared.
func widened(t *testing.T, source prototype, factor float64) prototype {
	t.Helper()
	part, err := shrinkCrownPart(source.parts[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	points := diamond(t).Points()
	for index := range points {
		points[index].X *= factor
		points[index].Z *= factor
	}
	made, err := mesh.New(points, diamond(t).Faces(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	part.Shape = made
	return prototype{parts: []thing.Thing{part}, bounds: made.Bounds()}
}

// TestFitCrownAreaShrinks checks a proxy with a shadow of 4.5 against a source
// with a shadow of 2: the horizontal scale is the square root of 2 over 4.5,
// which is two thirds, and the fitted shadow lands on the source's.
func TestFitCrownAreaShrinks(t *testing.T) {
	source := diamondPrototype(t)
	proxy := widened(t, source, 1.5)
	fitted, err := fitCrownArea(t.Context(), source, proxy, 96)
	if err != nil {
		t.Fatal(err)
	}
	fit := fitted.fit
	if fit == nil || fit.Side != 96 || len(fit.Parts) != 1 || fit.Bounds != proxy.bounds {
		t.Fatalf("fit = %+v", fit)
	}
	part := fit.Parts[0]
	if part.Material != "bark" || part.Limited != "" {
		t.Errorf("part = %+v", part)
	}
	if math.Abs(part.Scale-2.0/3) > 0.03 {
		t.Errorf("scale = %v, want about 0.667", part.Scale)
	}
	if math.Abs(part.Source-2) > 0.15 || math.Abs(part.Before-4.5) > 0.3 || math.Abs(part.After-2) > 0.2 {
		t.Errorf("areas source %v before %v after %v, want near 2, 4.5 and 2", part.Source, part.Before, part.After)
	}
	if proxy.fit != nil {
		t.Error("the proxy passed in was altered")
	}
}

// TestFitCrownAreaLimits checks the two cases that keep the original crown: a
// proxy with less shadow than its source, and one that casts none.
func TestFitCrownAreaLimits(t *testing.T) {
	source := diamondPrototype(t)
	small := widened(t, source, 0.5)
	small.bounds = maths.Box{Min: maths.V(-1, 0, -1), Max: maths.V(1, 2, 1)}
	fitted, err := fitCrownArea(t.Context(), source, small, 32)
	if err != nil {
		t.Fatal(err)
	}
	if part := fitted.fit.Parts[0]; part.Scale != 1 || !containsText(part.Limited, "needs expansion") {
		t.Errorf("small proxy: %+v", part)
	}
	away := widened(t, source, 1)
	away.parts[0].Motion = thing.Still(maths.NewSubspace(maths.V(50, 0, 0), maths.Identity, 1))
	fitted, err = fitCrownArea(t.Context(), source, away, 32)
	if err != nil {
		t.Fatal(err)
	}
	if part := fitted.fit.Parts[0]; part.Scale != 1 || !containsText(part.Limited, "zero sampled area") {
		t.Errorf("absent proxy: %+v", part)
	}
}

// TestFitCrownAreaRefusals checks each refusal.
func TestFitCrownAreaRefusals(t *testing.T) {
	source := diamondPrototype(t)
	proxy := widened(t, source, 1)
	cases := []struct {
		name          string
		source, proxy prototype
		side          int
		want          string
	}{
		{"no proxy parts", source, prototype{bounds: proxy.bounds}, 16, "correspondence is missing"},
		{"part count differs", prototype{parts: append(append([]thing.Thing(nil), source.parts...), source.parts...)}, proxy, 16, "correspondence is missing"},
		{"axis outside the envelope", source, func() prototype {
			p := proxy
			p.bounds = maths.Box{Min: maths.V(1, 0, 1), Max: maths.V(2, 2, 2)}
			return p
		}(), 16, "root axis"},
		{"bad side", source, proxy, 4, "probe grid"},
		{"material changed", source, func() prototype {
			p := proxy
			p.parts = []thing.Thing{{Shape: proxy.parts[0].Shape, Material: &thing.Material{Name: "bark"}}}
			return p
		}(), 16, "correspondence changed"},
		{"material missing", source, func() prototype {
			p := proxy
			p.parts = []thing.Thing{{Shape: proxy.parts[0].Shape}}
			return p
		}(), 16, "correspondence changed"},
	}
	for _, tc := range cases {
		if _, err := fitCrownArea(t.Context(), tc.source, tc.proxy, tc.side); err == nil || !containsText(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want one containing %q", tc.name, err, tc.want)
		}
	}
	other := prototype{parts: []thing.Thing{{Shape: distanceOnly{}, Material: source.parts[0].Material}}, bounds: proxy.bounds}
	if _, err := fitCrownArea(t.Context(), other, proxy, 16); err == nil {
		t.Error("a source without exact intersections was accepted")
	}
	if _, err := fitCrownArea(t.Context(), source, other, 16); err == nil {
		t.Error("a proxy without exact intersections was accepted")
	}
}
