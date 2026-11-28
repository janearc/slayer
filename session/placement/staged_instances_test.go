package placement

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestWorkingInstancesShareGeometryAndKeepStagedTransforms(t *testing.T) {
	triangle, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	first := maths.NewSubspace(maths.V(1, 2, 3), maths.About(maths.YAxis, 0.4), 2)
	second := maths.NewSubspace(maths.V(-2, 4, 1), maths.About(maths.YAxis, -0.2), 0.5)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: triangle, Space: first}, {Mesh: triangle, Space: second}})
	if err != nil {
		t.Fatal(err)
	}
	parent := maths.NewSubspace(maths.V(8, 3, -4), maths.About(maths.YAxis, 0.7), 3)
	world := thing.NewWorld()
	part := world.Add(thing.Thing{Shape: group, Motion: thing.Still(parent)})
	author := thing.AuthorRef{Collection: "scene.set.things", Name: "example"}
	world.RecordAuthor(part, author)
	world.AddVolume(thing.Volume{})
	world.Add(thing.Thing{Shape: &arid.Sea{}})
	got, err := WorkingInstances(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Batches) != 2 || len(got.Batches[0].Places) != 2 || got.Volumes != 1 || got.OmittedSeas != 0 {
		t.Fatalf("instances do not share a mesh: %+v", got)
	}
	sea := got.Batches[1]
	if sea.Mesh.Notice != "Sea shown as a finite mean-level proxy; waves and coastline are not shown" {
		t.Fatal("sea proxy was not labeled")
	}
	if low, high, ok := sea.Mesh.Bounds(); !ok || low.Y != 0 || high.Y != 0 || low.X >= high.X || low.Z >= high.Z {
		t.Fatalf("sea proxy bounds = %v %v %v", low, high, ok)
	}
	point := maths.V(1, 0, 0)
	for i, local := range []maths.Subspace{first, second} {
		batch := got.Batches[0]
		if !batch.Authored || batch.Author != author {
			t.Fatalf("instance lost its author: %+v", batch)
		}
		if delta := batch.Places[i].Out(point).Sub(parent.Out(local.Out(point))).Len(); delta > 1e-9 {
			t.Fatalf("instance %d transform differs by %g", i, delta)
		}
	}
}

func TestWorkingInstancesSeaWithoutBoundedSurfaces(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: &arid.Sea{}})
	got, err := WorkingInstancesFrom(world, 0, maths.V(400, 10, 300))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Batches) != 1 || got.OmittedSeas != 0 {
		t.Fatalf("sea-only preview: %+v", got)
	}
	batch := got.Batches[0]
	if !batch.HasBounds || batch.Mesh.Notice != "Sea shown as a finite mean-level proxy around the camera; waves and coastline are not shown" {
		t.Fatalf("unlabeled or unbounded sea proxy: %+v", batch)
	}
	low, high, _ := batch.Mesh.Bounds()
	if low != maths.V(360, 0, 260) || high != maths.V(440, 0, 340) {
		t.Fatalf("proxy bounds = %v %v", low, high)
	}
}

func TestWorkingInstancesKeepSupportedSurfacesBesideAnalyticPlane(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}})
	part := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	author := thing.AuthorRef{Collection: "scene.set.things", Name: "sphere"}
	world.RecordAuthor(part, author)
	got, err := WorkingInstances(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.OmittedPlanes != 0 || len(got.Batches) != 2 || got.Batches[0].Author != author || !got.Batches[0].Authored {
		t.Fatalf("supported authored sphere lost: %+v", got)
	}
	scene := DisplayScene{Batches: got.Batches}
	ray := maths.NewRay(maths.V(0, 0, 3), maths.V(0, 0, -1))
	if hit, ok := scene.batchHit(ray, 0, math.Inf(1)); !ok || hit.ID != "scene.set.things/sphere" {
		t.Fatalf("supported authored sphere is not pickable: %+v", hit)
	}
	world.Things[0].Shape = (*shape.Plane)(nil)
	if _, err := WorkingInstances(world, 0); err == nil || !strings.Contains(err.Error(), "staged surface 0") {
		t.Fatalf("missing analytic plane did not report its staged index: %v", err)
	}
	world.Things[0].Shape = shape.Plane{}
	world.Things[0].Motion = thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, -1))
	if _, err := WorkingInstances(world, 0); err == nil || !strings.Contains(err.Error(), "invalid geometry or placement") {
		t.Fatalf("invalid analytic plane transform did not fail staging: %v", err)
	}
}

func TestWorkingInstancesPlaneFollowsStagedPlacementAndCamera(t *testing.T) {
	at := maths.NewSubspace(maths.V(12, 8, -4), maths.About(maths.ZAxis, 0.4), 0.01)
	world := thing.NewWorld()
	colour := maths.RGB(.2, .4, .6)
	part := world.Add(thing.Thing{Shape: &shape.Plane{}, Motion: thing.Still(at), Material: &thing.Material{Colour: colour}})
	world.RecordAuthor(part, thing.AuthorRef{Collection: "scene.set.things", Name: "floor"})
	eye := at.Out(maths.V(3000, 120000, 4000))
	got, err := WorkingInstancesFrom(world, 0, eye)
	if err != nil || len(got.Batches) != 1 || got.OmittedPlanes != 0 {
		t.Fatalf("tilted plane proxy: %+v, %v", got, err)
	}
	batch := got.Batches[0]
	if batch.Authored || !batch.HasBounds || batch.Mesh.Notice == "" || len(batch.Mesh.Indices) != 6 {
		t.Fatalf("plane does not have a bounded, labeled read-only proxy: %+v", batch)
	}
	centre := batch.Places[0].Origin
	expected := at.Out(maths.V(3000, 0, 4000))
	if centre.Sub(expected).Len() > 1e-9 {
		t.Fatalf("proxy centre = %v, want %v", centre, expected)
	}
	for _, vertex := range batch.Mesh.Vertices {
		position := batch.Places[0].Out(vertex.Position)
		if math.Abs(at.In(position).Y) > 1e-8 || position.Sub(centre).Len() > 3000 ||
			position.Sub(centre).Len() < 1000 || vertex.Colour != colour {
			t.Fatalf("proxy vertex = %+v at %v", vertex, position)
		}
	}
}

func TestWorkingInstancesOmitsTransmissiveAndUnrepresentablePlanes(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Through: 1}})
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Radiant: thing.Excited{Passes: 1}}})
	world.Add(thing.Thing{Shape: shape.Plane{}, Motion: thing.Still(maths.NewSubspace(maths.V(0, 1e20, 0), maths.Identity, 1))})
	sphere := world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Motion: thing.Still(maths.NewSubspace(maths.V(0, -3, 0), maths.Identity, 1))})
	world.RecordAuthor(sphere, thing.AuthorRef{Collection: "scene.set.things", Name: "sphere"})
	eye := maths.V(0, 5, 0)
	got, err := WorkingInstancesFrom(world, 0, eye)
	if err != nil || len(got.Batches) != 1 || got.OmittedPlanes != 3 {
		t.Fatalf("unsupported planes hid finite geometry or went unreported: %+v, %v", got, err)
	}
	scene := DisplayScene{Batches: got.Batches}
	if hit, ok := scene.batchHit(maths.NewRay(eye, maths.V(0, -1, 0)), 0, math.Inf(1)); !ok || hit.ID != "scene.set.things/sphere" {
		t.Fatalf("the transmissive plane obscured the sphere: %+v", hit)
	}
}

func TestWorkingPlanePicksUseInspectionCameraClipDepth(t *testing.T) {
	world := thing.NewWorld()
	wall := maths.NewSubspace(maths.Vec{}, maths.About(maths.XAxis, math.Pi/2), 1)
	world.Add(thing.Thing{Shape: shape.Plane{}, Motion: thing.Still(wall)})
	scene := DisplayScene{Width: 101, Height: 101}
	const pixel = .8
	ray, err := scene.displayRay(pixel, .5)
	if err != nil {
		t.Fatal(err)
	}
	part := world.Add(thing.Thing{Shape: shape.Sphere{Radius: .2}, Motion: thing.Still(maths.NewSubspace(ray.Point(2), maths.Identity, 1))})
	world.RecordAuthor(part, thing.AuthorRef{Collection: "scene.set.things", Name: "sphere"})
	staged, err := WorkingInstancesFrom(world, 0, maths.V(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	scene.Batches = staged.Batches
	for _, depth := range []float64{0, DisplayNear / 2} {
		scene.Eye.Origin.Z = depth
		if got := scene.Pick(pixel, .5); got != "scene.set.things/sphere" {
			t.Fatalf("plane inside near clip at eye z=%g hid sphere: %q", depth, got)
		}
	}
	scene.Eye.Origin.Z = DisplayNear * 2
	if hit, ok := scene.PickAt(pixel, .5); !ok || hit.ID != "scene.world" {
		t.Fatalf("visible plane did not block the sphere: %+v", hit)
	}

	// Keep the off-axis ray crossing the patch's centre while the eye moves
	// away. A ray-length cut would reject the first hit inside the far depth.
	scene.Batches = staged.Batches[1:]
	scene.Eye.Origin = maths.V((DisplayFar-1)*ray.Dir.X/ray.Dir.Z, 0, DisplayFar-1)
	if hit, ok := scene.PickAt(pixel, .5); !ok || hit.Distance <= DisplayFar || hit.ID != "scene.world" {
		t.Fatalf("plane inside far depth did not survive off-axis ray length: %+v", hit)
	}
	sphere := staged.Batches[0]
	sphere.Places[0].Origin = ray.Dir.Scale(-.5 / ray.Dir.Z)
	scene.Batches = []StagedBatch{sphere}
	stagedBounds(scene.Batches)
	if got := scene.Pick(pixel, .5); got != "scene.set.things/sphere" {
		t.Fatalf("authored sphere inside far depth did not survive off-axis ray length: %q", got)
	}
	scene.Batches = staged.Batches[1:]
	scene.Eye.Origin = maths.V((DisplayFar+1)*ray.Dir.X/ray.Dir.Z, 0, DisplayFar+1)
	if hit, ok := scene.PickAt(pixel, .5); ok {
		t.Fatalf("plane beyond far clip was picked: %+v", hit)
	}
}
