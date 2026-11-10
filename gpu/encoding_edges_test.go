//go:build darwin

package gpu

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestEncodingKeepsHiddenObjectsOutAndHonoursExplicitPixelWidth(t *testing.T) {
	world := thing.NewWorld()
	world.Pixel = .002
	world.Add(thing.Thing{Attrs: thing.Attrs{Hidden: true}})
	encoded, err := encode(world, maths.Subspace{}, nil, 0, 4, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.standings) != 0 || encoded.counts.spheres != 0 {
		t.Fatal("hidden geometry reached the kernel")
	}
	if got := wideOf(world, maths.Subspace{}, film.Pinhole{}, 4, 3); got != .002 {
		t.Fatalf("explicit pixel width changed: %g", got)
	}
	if lens, ok := pinholeOf(nil); !ok || lens != (film.Pinhole{}) {
		t.Fatal("nil lens did not select the default pinhole")
	}
	if _, err := hazeOf(medium.Wispy{Of: medium.Giant{}}); err == nil ||
		!strings.Contains(err.Error(), "wispy medium over") {
		t.Fatalf("unsupported wispy medium: %v", err)
	}
}

func TestGroupEncodingRechecksMutableReferenceMeshNormals(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	reference, err := shape.NewMesh(points, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: reference}})
	if err != nil {
		t.Fatal(err)
	}
	encoded := newScene()
	if err := encoded.group(group, maths.Subspace{}, 0, false); err != nil {
		t.Fatal(err)
	}
	if len(encoded.groups) != 1 || len(encoded.faces.triangles) != 1 {
		t.Fatal("reference group lost or copied its face")
	}
	reference.Smooth = true
	reference.Normals = []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}
	if err := newScene().group(group, maths.Subspace{}, 0, false); err == nil ||
		!strings.Contains(err.Error(), "smooth reference mesh") {
		t.Fatalf("mutated reference mesh: %v", err)
	}
}

func TestFlamePairRequiresBodiesAtTheSamePlacement(t *testing.T) {
	grid := fireGridFixture(t, 5, 0)
	body := grid.med.(*flame.Body)
	for _, pair := range [][2]flame.Piece{
		{{}, {Body: body}}, {{Body: body}, {}},
		{{Body: body}, {Body: body, At: maths.XAxis}},
	} {
		if _, ok := pairedFlame(pair[0], pair[1]); ok {
			t.Fatalf("unpaired flame accepted: %+v", pair)
		}
	}
}

func TestMissingPackedFieldNamesTheContractFailure(t *testing.T) {
	defer func() {
		if got := recover(); got != "gpu: missing buffer field absent" {
			t.Fatalf("wrong layout failure: %v", got)
		}
	}()
	fieldOffset(volumeStruct, "absent")
}

func TestEmptyNativeTreeRangeContainsNoTriangles(t *testing.T) {
	encoded := newScene()
	encoded.faces.nodes = []node{{bounds: insideOut}}
	if first, count := encoded.triangleRange(0); first != 0 || count != 0 {
		t.Fatalf("empty tree claimed triangles %d..%d", first, count)
	}
}

func TestNativePlanRefusesInvalidPackedGeometryAndPlacements(t *testing.T) {
	for _, fault := range []string{"layout", "nonfinite", "direct range", "member range", "member transform", "group index", "world transform"} {
		t.Run(fault, func(t *testing.T) {
			encoded := newScene()
			encoded.faces.nodes = []node{{bounds: box{low: maths.Vec{}, high: maths.V(1, 1, 0)}, count: 1}}
			encoded.triangles.bytes = make([]byte, 3*16)
			encoded.standings = []standing{{}}
			want := ""
			switch fault {
			case "layout":
				encoded.triangles.bytes = []byte{1}
				want = "invalid triangle vertex layout"
			case "nonfinite":
				binary.LittleEndian.PutUint32(encoded.triangles.bytes,
					math.Float32bits(float32(math.Inf(1))))
				want = "nonfinite triangle position"
			case "direct range", "member range":
				encoded.faces.nodes[0].count = 2
				want = "triangle range outside"
			case "member transform", "world transform":
				encoded.standings[0].space.Scale = math.NaN()
				want = "invalid instance transform"
			case "group index":
				encoded.standings[0].group = 1
				want = "invalid shared group"
			}
			if fault == "member range" || fault == "member transform" {
				encoded.groups = []meshGroup{{members: append([]standing(nil), encoded.standings...)}}
				encoded.standings = []standing{{group: 1}}
			}
			if _, _, err := encoded.accelerationPlan(); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want %q, got %v", want, err)
			}
		})
	}
}
