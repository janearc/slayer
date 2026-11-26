//go:build draw

package mesh_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/burning-trousers/specimens"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// the pictures' size, and the seed each kind is made from.
const (
	wide, high = 320, 240
	seed       = 3
)

// made is a kind from the seed, standing at the origin.
func made(t *testing.T, name string) []thing.Thing {
	t.Helper()
	kind, err := manufacture.KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	good, err := manufacture.Make(kind, manufacture.Sow(seed))
	if err != nil {
		t.Fatal(err)
	}
	things, err := good.Things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	return things
}

// Every registered kind is made once for the independent face and picture checks.
// Faces are met on both mesh types; six independently drawn views must agree
// within the existing pixel and placement tolerances.
func TestRegisteredMeshReferenceChecks(t *testing.T) {
	seen := map[shape.Shape]bool{}
	for _, name := range manufacture.KindNames() {
		t.Run(name, func(t *testing.T) {
			ours := made(t, name)
			t.Run("faces", func(t *testing.T) {
				checkFaces(t, name, ours, seen)
			})
			t.Run("drawing", func(t *testing.T) {
				alike(t, name, ours, inRefMesh(t, ours))
			})
		})
	}
}

// Each checked profile is made once, then independently checked for face
// intersections, smooth normals and, when the reference supports it, drawing.
func TestProfileMeshReferenceChecks(t *testing.T) {
	profiles, err := specimens.Profiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			ours := profileThings(t, profile)
			t.Run("faces", func(t *testing.T) {
				checkFaces(t, profile.Name, ours, map[shape.Shape]bool{})
			})
			t.Run("normals", func(t *testing.T) {
				checkProfileSmoothNormals(t, ours)
			})
			t.Run("drawing", func(t *testing.T) {
				if why := cannotCopyProfileShading(ours); why != "" {
					t.Skip(why)
				}
				alike(t, profile.Name, ours, inRefMesh(t, ours))
			})
		})
	}
}

func profileThings(t *testing.T, profile specimens.Specimen) []thing.Thing {
	t.Helper()
	good, err := manufacture.Make(profile.Kind, manufacture.Sow(seed))
	if err != nil {
		t.Fatal(err)
	}
	things, err := good.Things(maths.Subspace{})
	if err != nil {
		t.Fatal(err)
	}
	return things
}

// checkFaces reads the constructed points, then independently meets each
// face on the original and reference meshes.
func checkFaces(t *testing.T, name string, things []thing.Thing, seen map[shape.Shape]bool) {
	t.Helper()
	for _, each := range things {
		meshes, err := faceMeshes(each.Shape)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, held := range meshes {
			if seen[held] {
				continue
			}
			seen[held] = true
			var corners []maths.Vec
			var faces [][3]int
			var ours, theirs meeter
			switch made := held.(type) {
			case *mesh.Mesh:
				corners, faces = made.Points(), made.Faces()
				ours, theirs = made, refMesh(t, corners, faces)
			case *shape.Mesh:
				corners, faces = made.Vertices, made.Faces
				copy, err := mesh.New(corners, faces, nil, false)
				if err != nil {
					t.Fatalf("%s: copy of reference mesh: %v", name, err)
				}
				ours, theirs = copy, made
			}
			missed := unmet(ours, theirs, corners, faces)
			if len(missed) > 0 {
				t.Errorf("%s: %d of %d faces not met "+
					"where they are: %v",
					name, len(missed), len(faces), missed)
			}
		}
	}
}

// faceMeshes names both mesh types, including group prototypes. Spheres and
// tubes are analytic shapes with no triangle faces; unknown shapes are not
// silently treated as if their faces had been checked.
func faceMeshes(solid shape.Shape) ([]shape.Shape, error) {
	switch held := solid.(type) {
	case *mesh.Mesh:
		if held == nil {
			return nil, fmt.Errorf("nil production mesh")
		}
		return []shape.Shape{solid}, nil
	case *shape.Mesh:
		if held == nil {
			return nil, fmt.Errorf("nil reference mesh")
		}
		return []shape.Shape{solid}, nil
	case shape.Sphere, shape.Tube:
		return nil, nil
	case *shape.MeshGroup:
		if held == nil {
			return nil, fmt.Errorf("nil mesh group")
		}
		var meshes []shape.Shape
		for _, member := range held.Members() {
			switch member.Mesh.(type) {
			case *mesh.Mesh, *shape.Mesh:
				meshes = append(meshes, member.Mesh)
			default:
				return nil, fmt.Errorf("unknown group prototype %T", member.Mesh)
			}
		}
		return meshes, nil
	default:
		return nil, fmt.Errorf("unknown shape %T in face comparison", solid)
	}
}

// Mixed profile animals can include either mesh type. Both must be read;
// analytic surfaces have no triangle faces, and unknown shapes are refused.
func TestFaceComparisonSeesBothMeshTypes(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	production, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := shape.NewMesh(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{{Mesh: production}, {Mesh: reference}})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[shape.Shape]bool{}
	checkFaces(t, "mixed", []thing.Thing{{Shape: reference}, {Shape: group}}, seen)
	if len(seen) != 2 || !seen[production] || !seen[reference] {
		t.Fatalf("not every mixed mesh was checked: %v", seen)
	}
	for _, analytic := range []shape.Shape{shape.Sphere{Radius: 1}, shape.Tube{Radius: 1, Thick: 1, Height: 1}} {
		faces, err := faceMeshes(analytic)
		if err != nil || len(faces) != 0 {
			t.Fatalf("analytic surface %T: %v, %v", analytic, faces, err)
		}
	}
	if _, err := faceMeshes(shape.Plane{}); err == nil {
		t.Fatal("unknown shape was silently ignored")
	}
	var missing *shape.Mesh
	if _, err := faceMeshes(missing); err == nil {
		t.Fatal("nil mesh was accepted")
	}
}

// around encloses every thing's placed shape at the frame being drawn.
func around(things []thing.Thing) maths.Box {
	var box maths.Box
	first := true
	for _, each := range things {
		local, at := each.Shape.Bounds(), each.At(0)
		for _, x := range []float64{local.Min.X, local.Max.X} {
			for _, y := range []float64{local.Min.Y, local.Max.Y} {
				for _, z := range []float64{local.Min.Z, local.Max.Z} {
					point := at.Out(maths.V(x, y, z))
					one := maths.Box{Min: point, Max: point}
					if first {
						box, first = one, false
					} else {
						box = joined(box, one)
					}
				}
			}
		}
	}
	return box
}

// views is three eyes looking at a box's middle from as far away as
// it is across, so the thing fills much of the picture: in front, from
// the side, and from above at a slant.
func views(box maths.Box) []maths.Subspace {
	middle := box.Min.Add(box.Max).Scale(0.5)
	away := box.Max.Sub(box.Min).Len()
	var eyes []maths.Subspace
	// Oblique views give flat foliage area too; an exactly edge-on leaf
	// cannot supply the surface pixels this comparison requires.
	for _, way := range []maths.Vec{
		maths.V(0, 0.2, 1), maths.V(1, 0.1, 0.2),
		maths.V(0.3, 1, 0.4),
	} {
		from := middle.Add(way.Unit().Scale(away))
		eyes = append(eyes, maths.Subspace{
			Origin:      from,
			Orientation: maths.LookAt(from, middle),
		})
	}
	return eyes
}

// drawn is some things under a sun, and a grey sky when asked, seen by
// an eye.
func drawn(t *testing.T, things []thing.Thing, eye maths.Subspace,
	sky bool,
) *frame.Frame {
	t.Helper()
	world := thing.NewWorld()
	if sky {
		world.Sky = fake.NewSky(maths.Grey(0.3))
	}
	for _, each := range things {
		world.Add(each)
	}
	world.Lamp(thing.Sun(maths.V(-0.4, 1, 0.5).Unit(), maths.Grey(1)))
	picture, err := trace.Render(world, eye, film.Pinhole{}, 0, wide, high)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

func cannotCopyProfileShading(things []thing.Thing) string {
	for _, each := range things {
		var meshes []*mesh.Mesh
		switch held := each.Shape.(type) {
		case *mesh.Mesh:
			meshes = append(meshes, held)
		case *shape.MeshGroup:
			for _, member := range held.Members() {
				if part, ok := member.Mesh.(*mesh.Mesh); ok {
					meshes = append(meshes, part)
				}
			}
		}
		for _, held := range meshes {
			if held.Posed() && each.Material != nil && each.Material.Texture != nil {
				return "reference mesh cannot preserve posed texture coordinates"
			}
			if held.Smooth() {
				return "reference mesh lacks smooth shadow-terminator correction"
			}
		}
	}
	return ""
}

func TestReferenceDistinguishesUnsupportedShading(t *testing.T) {
	points := []maths.Vec{maths.V(0, 0, 0), maths.V(1, 0, 0), maths.V(0, 1, 0)}
	faces := [][3]int{{0, 1, 2}}
	rest := []maths.Vec{maths.V(0, 0, 1), maths.V(1, 0, 1), maths.V(0, 1, 1)}
	posed, err := mesh.NewWithRestPoints(points, rest, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	flat, err := mesh.New(points, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	paint := &thing.Material{Texture: texture.Checker(1, maths.Grey(0), maths.Grey(1))}
	if cannotCopyProfileShading([]thing.Thing{{Shape: posed, Material: paint}}) == "" {
		t.Fatal("posed paint was accepted by a reference with no rest points")
	}
	if why := cannotCopyProfileShading([]thing.Thing{{Shape: flat, Material: paint}, {Shape: posed}}); why != "" {
		t.Fatalf("ordinary paint or unpainted pose refused: %s", why)
	}
	smooth, err := mesh.New(points, faces, []maths.Vec{maths.ZAxis, maths.ZAxis, maths.ZAxis}, true)
	if err != nil {
		t.Fatal(err)
	}
	if why := cannotCopyProfileShading([]thing.Thing{{Shape: smooth}}); why != "reference mesh lacks smooth shadow-terminator correction" {
		t.Fatalf("smooth reference is not distinguished: %q", why)
	}
}

// shadowEdges is the most pixels of a picture lit by the sun alone whose
// light may be past one jnd while their cover and depth agree: a point
// met the same, whose shadow ray toward the lamp grazes a needle and
// goes one way or the other on the last bit of where the hit was worked
// out. measured on 2026-10-15, 2026-10-16 and 2026-11-01 at two a
// picture at most, on the norway spruce only, and held there.
const shadowEdges = 2

// skyEdges is the same with the sky as well, which a matte surface
// gathers from sixteen looks round its normal, each a shadow ray of its
// own that may graze a needle as the lamp's does. measured on 2026-11-01
// at seven a picture at most, on the norway spruce only, and held a
// pixel over.
const skyEdges = 8

// a lighting is how the pictures compared are lit, and the pixels past
// one jnd its grazing shadow rays may leave.
type lighting struct {
	name  string
	sky   bool
	edges int
}

// lightings are the sun alone, and the sun with the sky.
var lightings = []lighting{
	{"by the sun", false, shadowEdges},
	{"by the sun and sky", true, skyEdges},
}

// alike holds two sets of the same things to drawing alike from each
// side and in each lighting, with three hundred pixels of each picture
// the thing: not one pixel whose cover or depth differs, which a face
// lost or met somewhere else would make, and no more past one jnd than
// the lighting's grazing shadow rays may leave.
func alike(t *testing.T, name string, ours, theirs []thing.Thing) {
	t.Helper()
	for _, lit := range lightings {
		for side, eye := range views(around(ours)) {
			// Loose shoots are new horizontal specimens. Their second eye
			// is oblique to the spray; existing kinds keep their old eyes.
			if side == 1 && (name == "coast-redwood-shoot" ||
				name == "giant-sequoia-shoot") {
				box := around(ours)
				middle := box.Min.Add(box.Max).Scale(0.5)
				from := middle.Add(maths.V(1, 0.1, 0.2).Unit().Scale(
					box.Max.Sub(box.Min).Len()))
				eye = maths.Subspace{
					Origin:      from,
					Orientation: maths.LookAt(from, middle),
				}
			}
			alikeFrom(t, name+" "+lit.name, side, lit,
				drawn(t, ours, eye, lit.sky), drawn(t, theirs, eye, lit.sky))
		}
	}
}

// alikeFrom holds two pictures of the same things, from one side and in
// one lighting, to drawing alike.
func alikeFrom(t *testing.T, name string, side int, lit lighting, picture,
	other *frame.Frame,
) {
	t.Helper()
	if count := covered(picture); count < 300 {
		t.Fatalf("%s, side %d: only %d pixels are it", name, side, count)
	}
	seen, err := divergence.Compare(picture, other)
	if err != nil {
		t.Fatal(err)
	}
	placed := misplaced(picture, other)
	past := int(math.Round(float64(seen.Past(1)) *
		float64(len(picture.Pix))))
	t.Logf("%s, side %d: %d pixels past one jnd, worst %.2f; %d placed "+
		"apart", name, side, past, float64(seen.Worst()), placed)
	if placed > 0 || past > lit.edges {
		t.Errorf("%s, side %d: %d pixels past one jnd, %d with another "+
			"cover or depth", name, side, past, placed)
	}
}

// misplaced is how many pixels of two pictures differ in cover, or in
// depth by more than a part in a million: a face met that the other
// missed, or met somewhere else. A pixel that misses in both pictures has
// an infinite depth in each, and the two are equal.
func misplaced(one, other *frame.Frame) int {
	count := 0
	for i := range one.Pix {
		near, far := one.Pix[i], other.Pix[i]
		cover := math.Abs(float64(near.Cover - far.Cover))
		depth := math.Abs(float64(near.Depth - far.Depth))
		bothMissed := math.IsInf(float64(near.Depth), 1) && near.Depth == far.Depth
		if !(cover <= 1e-6) || !(bothMissed || depth <= 1e-6*math.Abs(float64(near.Depth))) {
			count++
		}
	}
	return count
}

// A ball of faces with a normal at every corner, out from its middle,
// blended smooth across each face, is drawn to be a sphere. slayer's
// mesh asks for the light at the curved surface those normals make, a
// shape.Mesh at the flat face, so the two are held to a true sphere and
// not to each other: where every picture is wholly ball, slayer's is
// no farther from the sphere than a shape.Mesh's, from each side.
func TestASmoothBallDrawsAsASphere(t *testing.T) {
	corners, faces := ball(2)
	normals := slices.Clone(corners)
	ours, err := mesh.New(corners, faces, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := shape.NewMesh(corners, faces, normals, true)
	if err != nil {
		t.Fatal(err)
	}
	paint := &thing.Material{Name: "ball", Colour: maths.Grey(0.6)}
	still := thing.Still(maths.Subspace{})
	as := func(round shape.Shape) []thing.Thing {
		return []thing.Thing{{Shape: round, Material: paint, Motion: still}}
	}
	for side, eye := range views(around(as(shape.Sphere{Radius: 1}))) {
		truth := drawn(t, as(shape.Sphere{Radius: 1}), eye, true)
		mine := drawn(t, as(ours), eye, true)
		other := drawn(t, as(theirs), eye, true)
		whole := wholly(truth, mine, other)
		near, far := pastOn(t, mine, truth, whole), pastOn(t, other, truth,
			whole)
		t.Logf("side %d: of %d pixels wholly ball, slayer's mesh has %d "+
			"past one jnd from the sphere, a shape.Mesh %d", side, whole.n,
			near, far)
		if near > far || float64(near) > nearSphere*float64(whole.n) {
			t.Errorf("side %d: slayer's mesh has %d of %d pixels past one "+
				"jnd from the sphere, a shape.Mesh %d", side, near,
				whole.n, far)
		}
	}
}

// nearSphere is the share of a smooth ball's wholly covered pixels that
// may be past one jnd from a true sphere: three hundred and twenty faces
// are not a sphere, though their normals are. measured 2026-11-01 at a
// quarter of a percent at most, from each side, and held at a half.
const nearSphere = 0.005

// mask is the pixels every picture of some covers wholly, and how many.
type mask struct {
	in []bool
	n  int
}

// wholly is the pixels every picture covers wholly.
func wholly(pictures ...*frame.Frame) mask {
	held := mask{in: make([]bool, len(pictures[0].Pix))}
	for i := range held.in {
		held.in[i] = true
		for _, picture := range pictures {
			held.in[i] = held.in[i] && picture.Pix[i].Cover == 1
		}
		if held.in[i] {
			held.n++
		}
	}
	return held
}

// pastOn is how many pixels a mask holds where one picture is past one
// jnd from another.
func pastOn(t *testing.T, one, other *frame.Frame, held mask) int {
	t.Helper()
	only := func(picture *frame.Frame) *frame.Frame {
		kept := frame.New(picture.W, picture.H)
		for i, in := range held.in {
			if in {
				kept.Pix[i] = picture.Pix[i]
			}
		}
		return kept
	}
	seen, err := divergence.Compare(only(one), only(other))
	if err != nil {
		t.Fatal(err)
	}
	return int(math.Round(float64(seen.Past(1)) * float64(len(one.Pix))))
}

// covered is how many of a picture's pixels something covers.
func covered(picture *frame.Frame) int {
	count := 0
	for _, pixel := range picture.Pix {
		if pixel.Cover > 0 {
			count++
		}
	}
	return count
}
