package placement

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Discover shape candidates from the producer packages rather than keeping
// a second hand-written list of the representations the renderer can use.
func TestProducerSurfaceCoverage(t *testing.T) {
	covered := make(map[string]bool)
	for kind, disposition := range surfaceCoverage {
		covered[kind.String()] = disposition != unclassifiedSurface
	}
	for _, dir := range []string{"scene/shape", "mesh", "arid", "globe"} {
		t.Run(dir, func(t *testing.T) {
			pkg, err := build.ImportDir(filepath.Join("..", "..", dir), 0)
			if err != nil {
				t.Fatal(err)
			}
			const bounds, distance = 1, 2
			methods := make(map[string]int)
			files := token.NewFileSet()
			for _, name := range pkg.GoFiles {
				file, err := parser.ParseFile(files, filepath.Join(pkg.Dir, name), nil, parser.SkipObjectResolution)
				if err != nil {
					t.Fatal(err)
				}
				for _, decl := range file.Decls {
					method, ok := decl.(*ast.FuncDecl)
					if !ok || method.Recv == nil {
						continue
					}
					receiver := method.Recv.List[0].Type
					switch method.Name.Name {
					case "Bounds":
						methods[surfaceReceiver(pkg.Name, receiver)] |= bounds
					case "Distance":
						methods[surfaceReceiver(pkg.Name, receiver)] |= distance
					}
				}
			}
			for kind, found := range methods {
				if found == bounds|distance && !covered[kind] {
					t.Errorf("producer type %s needs a working-view disposition", kind)
				}
			}
		})
	}
}

func surfaceReceiver(pkg string, receiver ast.Expr) string {
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		return "*" + surfaceReceiver(pkg, pointer.X)
	}
	if name, ok := receiver.(*ast.Ident); ok {
		return pkg + "." + name.Name
	}
	return ""
}

type unclassifiedSphere struct{ shape.Sphere }

func TestUnsupportedSurfaceKeepsOtherGeometry(t *testing.T) {
	ball, err := globe.New(12)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	world.Add(thing.Thing{Shape: ball})
	world.Add(thing.Thing{Shape: unclassifiedSphere{shape.Sphere{Radius: 2}}})
	got, err := WorkingInstances(world, 0)
	if err != nil || len(got.Batches) != 1 || got.UnsupportedSurfaces["globe.Globe"] != 1 || got.UnsupportedSurfaces["placement.unclassifiedSphere"] != 1 {
		t.Fatalf("unsupported surfaces hid supported geometry or went unreported: %+v, %v", got, err)
	}
	world.Add(thing.Thing{Shape: (*unclassifiedSphere)(nil)})
	if _, err := WorkingInstances(world, 0); err == nil || !strings.Contains(err.Error(), "invalid geometry") {
		t.Fatalf("missing geometry did not fail staging: %v", err)
	}
}

func TestGlobeOutlineLimitCountsLaterVisibleCandidates(t *testing.T) {
	ball, err := globe.New(12)
	if err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	add := func(centre maths.Vec) {
		place := maths.Local()
		place.Origin = centre
		world.Add(thing.Thing{Shape: ball, Motion: thing.Still(place)})
	}
	for range GlobeOutlineLimit {
		add(maths.V(0, 0, DisplayFar*2))
	}
	visible := maths.V(0, 0, -DisplayFar*2)
	add(visible)
	staged, err := WorkingInstances(world, 0)
	if err != nil {
		t.Fatal(err)
	}
	scene := DisplayScene{Eye: maths.Local(), Width: 100, Height: 100}
	scene.Lens.Wide = 45 * maths.Degree
	if len(scene.ProjectDistantSphere(DisplaySphereOutline{Centre: visible, Radius: ball.Radius()})) == 0 {
		t.Fatal("fixture's final globe should project")
	}
	if len(staged.GlobeOutlines) != GlobeOutlineLimit || staged.GlobeOutlinesOmitted != 1 {
		t.Fatalf("outline limit was not enforced and accounted for: %d shown candidates, %d omitted", len(staged.GlobeOutlines), staged.GlobeOutlinesOmitted)
	}
	for _, outline := range staged.GlobeOutlines {
		if len(scene.ProjectDistantSphere(outline)) != 0 {
			t.Fatal("the retained behind-camera candidates should not project")
		}
	}
}
