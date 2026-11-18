package named

import (
	"io"
	"math"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// cityTriangle supplies a non-degenerate native open mesh for placement tests.
func cityTriangle(t *testing.T) *mesh.Mesh {
	t.Helper()
	made, err := mesh.New([]maths.Vec{maths.V(0, 0, 0), maths.V(50, 0, 0), maths.V(0, 0, 50)}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// This opt-in constructs the actual named orbital, with all ten thousand
// buildings, foliage, six fixtures and the independent magnetar flare. It
// exercises the same surface selection and tick wiring used by film jobs.
func TestOrbitalCityIntegratedScene(t *testing.T) {
	if os.Getenv("ORBITAL_CITY_FULL") != "1" {
		t.Skip("set ORBITAL_CITY_FULL=1 for the integrated procedural city")
	}
	for _, far := range []bool{false, true} {
		t.Run(map[bool]string{false: "near", true: "far"}[far], func(t *testing.T) {
			view, err := orbitalScene().Build(map[string]any{
				"floor-tiles": 256, "surface-structures": 1, "surface-city": true,
				"surface-tour": !far, "far": far, "zappy-flares": true,
				"day": "24h", "gas": 0,
			}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			world := view.World()
			if !far && len(world.Lamps) != 9 {
				t.Fatalf("near view has %d lamps, want six city fixtures, sun, fill and flare", len(world.Lamps))
			}
			if len(world.Lamps) < 7 {
				t.Fatalf("missing fixtures: %d lamps", len(world.Lamps))
			}
			var initial [6]emit.Lit
			for i := range initial {
				var ok bool
				initial[i], ok = world.Lamps[i].(emit.Lit)
				if !ok || initial[i].Strength.Luminance() <= 0 {
					t.Fatalf("city fixture %d not retained: %#v", i, world.Lamps[i])
				}
			}
			if err := view.Advance(2500 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
			for i, previous := range initial {
				current, ok := world.Lamps[i].(emit.Lit)
				if !ok || current.At == previous.At || current.Toward == previous.Toward || current.Strength != previous.Strength {
					t.Fatalf("city fixture %d failed to follow ring intact", i)
				}
			}
			flare, ok := world.Lamps[len(world.Lamps)-1].(emit.Lit)
			if !ok || flare.At != (maths.Vec{}) || flare.Strength.Luminance() <= 0 {
				t.Fatal("magnetar flare did not peak")
			}
			if err := view.Advance(5 * time.Second); err != nil {
				t.Fatal(err)
			}
			quiet := world.Lamps[len(world.Lamps)-1].(emit.Lit)
			if quiet.Strength.Luminance() >= flare.Strength.Luminance()*.001 {
				t.Fatal("flare did not fade independently of fixtures")
			}
			if err := view.Advance(2500 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
			repeated := world.Lamps[len(world.Lamps)-1].(emit.Lit)
			if repeated.At != flare.At || repeated.Toward != flare.Toward || repeated.Strength != flare.Strength {
				t.Fatal("flare changed after seeking")
			}
		})
	}
}

func TestOrbitalCityConformsUniqueChunksAndKeepsSharedTrees(t *testing.T) {
	ring := nested()[0]
	centre := maths.V(0, 0, ring.radius)
	chunk := cityTriangle(t)
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: chunk, Space: maths.Local()}})
	if err != nil {
		t.Fatal(err)
	}
	bent, err := ring.cityGroup(group, centre)
	if err != nil {
		t.Fatal(err)
	}
	wrapped := bent.Member(0).Mesh.(*mesh.Mesh)
	wanted := ring.surfacePoint(centre, orbitalCityOffset)
	if !wrapped.Points()[0].Near(wanted, 1e-9) {
		t.Fatalf("city base left native floor: %v / %v", wrapped.Points()[0], wanted)
	}
	if altitude := ring.surfaceRadius(math.Atan2(wanted.X, wanted.Z)) - math.Hypot(wanted.X, wanted.Z); math.Abs(altitude-10e-6) > 1e-8 {
		t.Fatalf("road offset lost: %g", altitude)
	}
	leaf := cityTriangle(t)
	a := maths.NewSubspace(maths.V(100, 0, 50), maths.Identity, 2)
	b := maths.NewSubspace(maths.V(-100, 0, 90), maths.Identity, 3)
	forest, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: leaf, Space: a}, {Mesh: leaf, Space: b}})
	if err != nil {
		t.Fatal(err)
	}
	kept, err := ring.cityGroup(forest, centre)
	if err != nil {
		t.Fatal(err)
	}
	if kept.StoredFaces() != leaf.FaceCount() || kept.FaceCount() != 2*leaf.FaceCount() {
		t.Fatal("repeated foliage was flattened or lost")
	}
	for i := 0; i < kept.MemberCount(); i++ {
		got := kept.Member(i)
		if got.Mesh != leaf {
			t.Fatal("shared leaf geometry was duplicated")
		}
		local := []maths.Subspace{a, b}[i]
		anchor := ring.surfacePoint(centre, local.Origin.Scale(orbitalCityScale).Add(orbitalCityOffset))
		if !got.Space.Origin.Near(anchor, 1e-9) {
			t.Fatalf("tree detached from floor: %v / %v", got.Space.Origin, anchor)
		}
		if math.Abs(got.Space.Scale-local.Scale*orbitalCityScale*1e-6) > 1e-12 {
			t.Fatal("tree scale changed")
		}
	}
}

// This opt-in exercises the real 10,000-building recipe, including foliage and
// night windows. The normal test remains inexpensive and deterministic.
func TestOrbitalCityFullRecipe(t *testing.T) {
	if os.Getenv("ORBITAL_CITY_FULL") != "1" {
		t.Skip("set ORBITAL_CITY_FULL=1 for the full procedural city")
	}
	ring := nested()[0]
	world := thing.NewWorld()
	centre := maths.V(0, 0, ring.radius)
	advance, err := ring.surfaceCity(world, centre, 1806, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Lamps) != 6 {
		t.Fatalf("city lost fixture lamps: %d", len(world.Lamps))
	}
	initial := world.Lamps[0].(emit.Lit)
	advance(time.Hour)
	moved := world.Lamps[0].(emit.Lit)
	if moved.At == initial.At || moved.Toward == initial.Toward {
		t.Fatal("city lamps did not turn with the ring")
	}
	var members, lit, shared int
	for _, part := range world.Things {
		group, ok := part.Shape.(*shape.MeshGroup)
		if !ok {
			t.Fatalf("unexpected city geometry %T", part.Shape)
		}
		members += group.MemberCount()
		if group.StoredFaces() < group.FaceCount() {
			shared++
		}
		if part.Material.Glow.Luminance() > 0 {
			lit++
		}
		if part.At(time.Hour) != ring.turning(24*time.Hour).At(time.Hour) {
			t.Fatal("city lost ring motion")
		}
	}
	if members < 100 || lit == 0 || shared == 0 {
		t.Fatalf("incomplete city: members=%d lit=%d shared=%d", members, lit, shared)
	}
	t.Logf("full city: %d material groups, %d members, %d lit, %d shared", len(world.Things), members, lit, shared)
}
