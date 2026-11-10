// Package mesh holds slayer's own mesh (mesh) to scene/shape's Mesh, which
// was first written, by the pictures they make of the same faces. every kind
// of the manufactory, made from a seed on slayer's mesh, is drawn again
// with its faces copied into a shape.Mesh, from three sides, and the two
// pictures are compared.
// Shared groups are expanded into separate world objects, each using the
// reference mesh. Their placement and traversal do not use MeshGroup.
//
// a pixel's light in just-noticeable differences, its cover and its
// depth are all compared. the faces are the same, so a pixel that
// differs is a ray that met them differently.
//
// the pictures see only the faces an eye sees, so every face is also
// met by a ray of its own, from a hair outside its middle, on both
// meshes; a face either mesh dropped lets that ray go on past it. a
// face with another within a millionth of the diagonal along its
// normal, doubled or in a shell that thin, is met by the other, so a
// drop of it passes; no kind made today has one.
//
// what this cannot see: the shape.Mesh copy is made from slayer's own faces, so
// a face slayer's mesh lost in the making is lost from both. slayer's
// own tests hold the making, by a cube, a ball and rays from inside,
// and the manufactory's skin reader here holds every kind's skin.
// Group members also come from the manufactured object: a missing placement
// in construction is missing on both sides. Construction needs its own tests.
//
// a shape.Mesh is no evidence that slayer's mesh is right; it is the other
// side of a change of mesh under the same scenes.
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

// inRefMesh is the same things with each of slayer's meshes copied face
// for face into a shape.Mesh.
func inRefMesh(t *testing.T, things []thing.Thing) []thing.Thing {
	t.Helper()
	var copied []thing.Thing
	cache := map[shape.Shape]shape.Shape{}
	converted := func(solid shape.Shape) shape.Shape {
		if known, ok := cache[solid]; ok {
			return known
		}
		var other shape.Shape = solid
		switch held := solid.(type) {
		case *mesh.Mesh:
			other = refMesh(t, held.Points(), held.Faces())
		}
		cache[solid] = other
		return other
	}
	for _, each := range things {
		if group, ok := each.Shape.(*shape.MeshGroup); ok {
			for _, member := range group.Members() {
				if _, ok := member.Mesh.(*mesh.Mesh); !ok {
					t.Fatalf("group member is not a production mesh: %T", member.Mesh)
				}
				copy := each
				copy.Shape = converted(member.Mesh)
				// These pictures are taken at time zero, for both renderers.
				copy.Motion = thing.Still(member.Space.Within(each.At(0)))
				copied = append(copied, copy)
			}
			continue
		}
		each.Shape = converted(each.Shape)
		copied = append(copied, each)
	}
	return copied
}

// Group comparisons must not route both pictures through MeshGroup.Hit.
func TestReferenceGroupUsesSeparatePlacedMeshes(t *testing.T) {
	prototype, err := mesh.New([]maths.Vec{
		maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0),
	}, [][3]int{{0, 1, 2}}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	group, err := shape.NewMeshGroup([]shape.MeshMember{
		{Mesh: prototype, Space: maths.Subspace{Origin: maths.V(2, 0, 0), Scale: 2}},
		{Mesh: prototype, Space: maths.Subspace{Origin: maths.V(-3, 0, 0)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	copies := inRefMesh(t, []thing.Thing{{Shape: group}})
	if len(copies) != 2 {
		t.Fatalf("group stayed one reference object: %d", len(copies))
	}
	for i, copy := range copies {
		if _, ok := copy.Shape.(*shape.Mesh); !ok {
			t.Fatalf("reference member %d shares group traversal: %T", i, copy.Shape)
		}
	}
	if copies[0].At(0).Out(maths.XAxis) != maths.V(4, 0, 0) ||
		copies[1].At(0).Out(maths.XAxis) != maths.V(-2, 0, 0) {
		t.Fatal("reference members lost their independent placements")
	}
}

// refMesh is a shape.Mesh of some faces, flat.
func refMesh(t *testing.T, corners []maths.Vec, faces [][3]int) *shape.Mesh {
	t.Helper()
	theirs, err := shape.NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return theirs
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

// A face slayer's mesh was not given is found by its own ray, so the
// test above would see a face either mesh dropped.
func TestADroppedFaceIsFound(t *testing.T) {
	corners, faces := ball(1)
	last := len(faces) - 1
	ours, err := mesh.New(corners, faces[:last], nil, false)
	if err != nil {
		t.Fatal(err)
	}
	missed := unmet(ours, refMesh(t, corners, faces), corners, faces)
	if !slices.Equal(missed, []int{last}) {
		t.Errorf("one face left out: %v not met, want [%d]",
			missed, last)
	}
}

// meeter is a mesh a ray can meet: mesh.Mesh or shape.Mesh.
type meeter interface {
	Hit(ray maths.Ray, near, far float64) (float64, bool)
}

// unmet is the faces, of those given, that a ray from a hair outside a
// face's middle, along its normal inward, does not meet within that hair
// on both meshes. a face of no area has no normal and is passed over. a
// face met nearer than the hair, by another lying close over it, is met.
func unmet(ours, theirs meeter, corners []maths.Vec, faces [][3]int) []int {
	box := maths.Box{Min: corners[0], Max: corners[0]}
	for _, corner := range corners {
		box = joined(box, maths.Box{Min: corner, Max: corner})
	}
	hair := 1e-6 * box.Max.Sub(box.Min).Len()
	var missed []int
	for i, at := range faces {
		one, two := corners[at[0]], corners[at[1]]
		three := corners[at[2]]
		across := two.Sub(one).Cross(three.Sub(one))
		if across.Len() == 0 {
			continue
		}
		out := across.Unit()
		middle := one.Add(two).Add(three).Scale(1.0 / 3)
		ray := maths.Ray{
			At:  middle.Add(out.Scale(hair)),
			Dir: out.Scale(-1),
		}
		_, oursMet := ours.Hit(ray, 0, 1.001*hair)
		_, theirsMet := theirs.Hit(ray, 0, 1.001*hair)
		if !oursMet || !theirsMet {
			missed = append(missed, i)
		}
	}
	return missed
}

// around is the box around some things standing at the origin.
func around(things []thing.Thing) maths.Box {
	box := things[0].Shape.Bounds()
	for _, each := range things[1:] {
		box = joined(box, each.Shape.Bounds())
	}
	return box
}

// joined is the box around two boxes.
func joined(one, other maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(math.Min(one.Min.X, other.Min.X),
			math.Min(one.Min.Y, other.Min.Y), math.Min(one.Min.Z, other.Min.Z)),
		Max: maths.V(math.Max(one.Max.X, other.Max.X),
			math.Max(one.Max.Y, other.Max.Y), math.Max(one.Max.Z, other.Max.Z)),
	}
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
// missed, or met somewhere else.
func misplaced(one, other *frame.Frame) int {
	count := 0
	for i := range one.Pix {
		near, far := one.Pix[i], other.Pix[i]
		cover := math.Abs(float64(near.Cover - far.Cover))
		depth := math.Abs(float64(near.Depth - far.Depth))
		if cover > 1e-6 || depth > 1e-6*math.Abs(float64(near.Depth)) {
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

// ball is an icosahedron's faces cut in four so many times, each cut's
// new corners pushed out to the unit sphere, every face turning out.
func ball(times int) ([]maths.Vec, [][3]int) {
	gold := (1 + math.Sqrt(5)) / 2
	corners := []maths.Vec{
		maths.V(-1, gold, 0), maths.V(1, gold, 0),
		maths.V(-1, -gold, 0), maths.V(1, -gold, 0),
		maths.V(0, -1, gold), maths.V(0, 1, gold), maths.V(0, -1, -gold),
		maths.V(0, 1, -gold), maths.V(gold, 0, -1), maths.V(gold, 0, 1),
		maths.V(-gold, 0, -1), maths.V(-gold, 0, 1),
	}
	for i, corner := range corners {
		corners[i] = corner.Unit()
	}
	faces := [][3]int{
		{0, 11, 5},
		{0, 5, 1},
		{0, 1, 7},
		{0, 7, 10},
		{0, 10, 11},
		{1, 5, 9},
		{5, 11, 4},
		{11, 10, 2},
		{10, 7, 6},
		{7, 1, 8},
		{3, 9, 4},
		{3, 4, 2},
		{3, 2, 6},
		{3, 6, 8},
		{3, 8, 9},
		{4, 9, 5},
		{2, 4, 11},
		{6, 2, 10},
		{8, 6, 7},
		{9, 8, 1},
	}
	for range times {
		corners, faces = quartered(corners, faces)
	}
	return corners, faces
}

// quartered is each face cut in four at the middles of its edges, the
// middles pushed out to the unit sphere and shared by the faces either
// side.
func quartered(corners []maths.Vec, faces [][3]int) ([]maths.Vec,
	[][3]int,
) {
	middles := map[[2]int]int{}
	middle := func(one, other int) int {
		key := [2]int{min(one, other), max(one, other)}
		if at, seen := middles[key]; seen {
			return at
		}
		corners = append(corners,
			corners[one].Add(corners[other]).Scale(0.5).Unit())
		middles[key] = len(corners) - 1
		return len(corners) - 1
	}
	var cut [][3]int
	for _, at := range faces {
		ab, bc, ca := middle(at[0], at[1]), middle(at[1], at[2]),
			middle(at[2], at[0])
		cut = append(cut, [3]int{at[0], ab, ca}, [3]int{at[1], bc, ab},
			[3]int{at[2], ca, bc}, [3]int{ab, bc, ca})
	}
	return corners, cut
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
