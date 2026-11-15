//go:build draw

package mesh_test

import (
	"math"
	"slices"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
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

// Every face of every kind is met where it is, on slayer's mesh and on
// a shape.Mesh: a face the three eyes never see, behind or under the rest, is
// held as well as one they do.
func TestEveryFaceIsMetOnBothMeshes(t *testing.T) {
	seen := map[*mesh.Mesh]bool{}
	for _, name := range manufacture.KindNames() {
		for _, each := range made(t, name) {
			var meshes []*mesh.Mesh
			switch held := each.Shape.(type) {
			case *mesh.Mesh:
				meshes = append(meshes, held)
			case *shape.MeshGroup:
				for _, member := range held.Members() {
					if mesh, ok := member.Mesh.(*mesh.Mesh); ok {
						meshes = append(meshes, mesh)
					} else {
						t.Fatalf("%s group member is not a mesh: %T", name, member.Mesh)
					}
				}
			}
			for _, held := range meshes {
				if seen[held] {
					continue
				}
				seen[held] = true
				corners, faces := held.Points(), held.Faces()
				theirs := refMesh(t, corners, faces)
				missed := unmet(held, theirs, corners, faces)
				if len(missed) > 0 {
					t.Errorf("%s: %d of %d faces not met "+
						"where they are: %v",
						name, len(missed), len(faces), missed)
				}
			}
		}
	}
}

// around is the box around some things standing at the origin.
func around(things []thing.Thing) maths.Box {
	box := things[0].Shape.Bounds()
	for _, each := range things[1:] {
		box = joined(box, each.Shape.Bounds())
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

// Every kind drawn on slayer's mesh is the same faces drawn on a shape.Mesh,
// from each side and in each lighting: not one pixel placed apart, and
// none past one just-noticeable difference but where a shadow ray grazes.
// the still budget's two in a hundred is for shortcuts that trade
// light for time; this is no shortcut, so a pixel past one jnd is a
// face met differently, and is to be looked at. three hundred pixels
// of each picture at least are the thing, so the pixels compared are
// its; a log seen end on is few more.
func TestSlayersMeshDrawsWhatAShapeMeshDoes(t *testing.T) {
	for _, name := range manufacture.KindNames() {
		ours := made(t, name)
		alike(t, name, ours, inRefMesh(t, ours))
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
