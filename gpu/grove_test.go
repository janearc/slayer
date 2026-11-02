//go:build darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// treeParams is manufacture's conifer, as the tests spell it
type treeParams = manufacture.ConiferParams

// upright turns a conifer's own up, z, to the world's up, y
var upright = maths.About(maths.V(1, 0, 0), -math.Pi/2)

// stands are three conifers of a few levels, a little apart, in the
// world's own space
func stands() []maths.Subspace {
	return []maths.Subspace{
		{Origin: maths.V(-5, 0, 2), Orientation: upright},
		{Origin: maths.V(0, 0, 0), Orientation: upright},
		{Origin: maths.V(4.5, 0, 3), Orientation: upright, Scale: 1.5},
	}
}

// grownParams is a conifer small enough to flatten and draw both ways
func grownParams() manufacture.ConiferParams {
	params := manufacture.DefaultConiferParams()
	params.Depth = 3
	return params
}

// flattened is the same conifers as one mesh in the world: every face
// manufacture makes, put where its conifer stands
func flattened(t *testing.T, material *thing.Material) thing.Thing {
	t.Helper()
	var points []maths.Vec
	var corners [][3]int
	for _, place := range stands() {
		faces := manufacture.Conifer(geom.Vec3{}, grownParams())
		for _, face := range faces {
			first := len(points)
			corner := [3]int{first, first + 1, first + 2}
			corners = append(corners, corner)
			for _, corner := range face.V {
				points = append(points, place.Out(vec3(corner)))
			}
		}
	}
	mesh, err := shape.NewMesh(points, corners, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return thing.Thing{Shape: mesh, Material: material}
}

// inFront is a small ball between the eye and the scaled conifer, over
// part of it: the kernel looks for a scaled tree only nearer than what
// it has already found, in the tree's own units, and a wrong limit would
// draw the tree over the ball.
func inFront() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 2},
		Material: &thing.Material{Colour: maths.RGB(0.2, 0.3, 0.9)},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(4.5, 3, -4)})}
}

// sceneWithoutTrees is the backdrop, the ball and the lamp alone, to say
// which pixels are the trees'
func sceneWithoutTrees() *thing.World {
	world := thing.NewWorld()
	world.Lamp(emit.Lit{At: maths.V(6, 14, -12), Strength: maths.Grey(2e3)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 30},
		Material: &thing.Material{Colour: maths.RGB(0.4, 0.3, 0.2)}})
	world.Add(inFront())
	return world
}

// sceneOf is a lamp, a sphere behind the trees, a ball in front of the
// scaled one, and the trees: grown in the kernel if asked, or flattened
// into a mesh
func sceneOf(t *testing.T, inKernel bool) (*thing.World, []grown) {
	t.Helper()
	bark := &thing.Material{Colour: maths.RGB(0.1, 0.8, 0.2)}
	wall := &thing.Material{Colour: maths.RGB(0.4, 0.3, 0.2)}
	world := thing.NewWorld()
	world.Lamp(emit.Lit{At: maths.V(6, 14, -12), Strength: maths.Grey(2e3)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 30}, Material: wall})
	world.Add(inFront())
	if !inKernel {
		world.Add(flattened(t, bark))
		return world, nil
	}
	var trees []grown
	for _, place := range stands() {
		tree, err := newGrown(place, grownParams(), bark)
		if err != nil {
			t.Fatal(err)
		}
		trees = append(trees, tree)
	}
	return world, trees
}

// differ is whether two pixels are not the same: by more than a small
// share of the light, or in how much of the pixel is covered
func differ(left, right frame.Pixel) bool {
	gap := math.Abs(float64(left.R-right.R)) +
		math.Abs(float64(left.G-right.G)) +
		math.Abs(float64(left.B-right.B))
	share := math.Abs(float64(left.R+left.G+left.B)) + 1e-3
	return gap > 0.03*share ||
		math.Abs(float64(left.Cover-right.Cover)) > 0.2
}

// apartBy is how many pixels of two pictures are not the same
func apartBy(one, other *frame.Frame) int {
	apart := 0
	for row := range one.H {
		for column := range one.W {
			if differ(one.At(column, row), other.At(column, row)) {
				apart++
			}
		}
	}
	return apart
}

// offTheFaces is the share of pixels the grown grove may differ from the
// flattened one by, one in this many: a ray along a face's edge may fall
// either side. measured on an M3 Max, none differ; this allows nine of
// 9600, and a far limit for the scaled tree left unscaled moves 195.
const offTheFaces = 1000

// the trees the kernel grows are the trees of the flattened mesh: the
// same picture, to within the edges of faces. this is also the smoke
// test of the feature: it draws trees.
func TestAGroveGrownInTheKernelIsTheMeshOfItsFaces(t *testing.T) {
	from := maths.V(0, 5, -26)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 4, 0))}
	width, height := 120, 80
	grownWorld, grove := sceneOf(t, true)
	got, err := sampledGrown(grownWorld, grove, eye, film.Pinhole{}, 0,
		width, height, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	meshWorld, _ := sceneOf(t, false)
	want, err := Sampled(meshWorld, eye, film.Pinhole{}, 0, width,
		height, 1)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := Sampled(sceneWithoutTrees(), eye, film.Pinhole{}, 0, width,
		height, 1)
	if err != nil {
		t.Fatal(err)
	}
	if trees := apartBy(got, bare); trees < width*height/10 ||
		trees > width*height*9/10 {
		t.Errorf("trees change %d of %d pixels: the picture needs both",
			trees, width*height)
	}
	if apart := apartBy(got, want); apart > width*height/offTheFaces {
		t.Errorf("%d of %d pixels differ from the flat trees", apart,
			width*height)
	}
}

// the door refuses what is not a tree, or what the kernel cannot grow
func TestAConiferThatCannotBeIsRefused(t *testing.T) {
	cases := map[string]func(*treeParams){
		"too deep":     func(p *treeParams) { p.Depth = 7 },
		"below nought": func(p *treeParams) { p.Depth = -1 },
		"two sides":    func(p *treeParams) { p.Sides = 2 },
		"no height":    func(p *treeParams) { p.Height = 0 },
		"no radius":    func(p *treeParams) { p.BaseRadius = 0 },
		"no taper":     func(p *treeParams) { p.TaperRate = 0 },
		"no scale": func(p *treeParams) {
			p.DepthScale = 0
		},
		"a height of nan": func(p *treeParams) {
			p.Height = math.NaN()
		},
		"too many sides": func(p *treeParams) {
			p.Sides = mostConiferSides + 1
		},
		"a lean of nan": func(p *treeParams) {
			p.BranchStart = math.NaN()
		},
		"an endless lean": func(p *treeParams) {
			p.BranchStart = math.Inf(-1)
		},
		"a height past a float32": func(p *treeParams) {
			p.Height = 1e300
		},
	}
	for name, spoil := range cases {
		params := manufacture.DefaultConiferParams()
		spoil(&params)
		_, err := newGrown(maths.Subspace{}, params, nil)
		if err == nil {
			t.Errorf("a conifer %s was not refused", name)
		}
	}
	places := map[string]float64{"below nought": -1, "nan": math.NaN(),
		"endless": math.Inf(1), "past a float32": 1e300}
	for name, scale := range places {
		place := maths.Subspace{Scale: scale}
		params := manufacture.DefaultConiferParams()
		_, err := newGrown(place, params, nil)
		if err == nil {
			t.Errorf("a conifer scaled %s was not refused", name)
		}
	}
	deepest := manufacture.DefaultConiferParams()
	deepest.Depth = mostConiferDepth
	deepest.Sides = mostConiferSides
	if _, err := newGrown(maths.Subspace{}, deepest, nil); err != nil {
		t.Errorf("the deepest, most sided conifer was refused: %v", err)
	}
}
