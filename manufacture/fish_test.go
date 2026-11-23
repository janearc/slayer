package manufacture

import (
	"maps"
	"math"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/scene/shape"
)

// fishNames are the species the file gives.
func fishNames(t *testing.T) []string {
	t.Helper()
	if fishError != nil {
		t.Fatal(fishError)
	}
	var names []string
	for name := range fishRead.Species {
		names = append(names, name)
	}
	return names
}

// fishWith is the file with one figure of one species changed.
func fishWith(name, key string, figure fishFigure) fishFile {
	file := fishFile{Read: fishRead.Read,
		Species: maps.Clone(fishRead.Species)}
	species := file.Species[name]
	species.Figures = maps.Clone(species.Figures)
	species.Figures[key] = figure
	file.Species[name] = species
	return file
}

// TestFishKindsAreMade holds every fish kind to being made from a seed,
// as surfaces in looks.
func TestFishKindsAreMade(t *testing.T) {
	for name, row := range fishRows() {
		made, err := row.made(sow(7))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(made.native) == 0 && len(made.groups) == 0 {
			t.Fatalf("%s made nothing", name)
		}
		for _, one := range made.native {
			if one.Material == nil {
				t.Fatalf("%s has a surface in no look", name)
			}
		}
	}
}

// TestFishColumnIsTheSpecies holds each skeleton to its species' counts:
// precaudal and caudal vertebrae, caudal rays, the fins' rays and the
// finlets.
func TestFishColumnIsTheSpecies(t *testing.T) {
	for _, name := range fishNames(t) {
		plan := planFish(name, sow(3))
		body, err := buildFish(plan)
		if err != nil {
			t.Fatal(err)
		}
		bones := body.bones
		precaudal := plan.whole("precaudal")
		counts := map[string]int{"P": precaudal,
			"Ca":                 plan.whole("vertebrae") - precaudal,
			"caudal-ray-":        plan.whole("caudal-rays"),
			"anal-ray-":          plan.whole("anal-rays"),
			"dorsal-finlet-":     plan.whole("dorsal-finlets"),
			"pectoral-left-ray-": plan.whole("pectoral-rays")}
		for prefix, want := range counts {
			if got := bones.Count(prefix); got != want {
				t.Fatalf("%s has %d bones named %s, not %d", name, got,
					prefix, want)
			}
		}
	}
}

// TestFishSeedsMakeDifferentFish holds the same seed to the same fish and
// two seeds to two sizes.
func TestFishSeedsMakeDifferentFish(t *testing.T) {
	for _, name := range fishNames(t) {
		one, again := planFish(name, sow(1)), planFish(name, sow(1))
		other := planFish(name, sow(2))
		if one.length() != again.length() {
			t.Fatalf("%s: one seed made two lengths", name)
		}
		row := fishRead.Species[name].Figures["length"]
		if row.Most > row.Least && one.length() == other.length() {
			t.Fatalf("%s: two seeds made one length", name)
		}
	}
}

// fishSwing is how far the column's vertebra nearest share s stands to
// the side in the stroke, of the fish's length.
func fishSwing(t *testing.T, name string, s float64) float64 {
	t.Helper()
	plan := planFish(name, sow(5))
	body, err := buildFish(plan)
	if err != nil {
		t.Fatal(err)
	}
	amplitude := body.plan.figure("amplitude")
	frames := body.bones.Frames(body.stroke(amplitude))
	at, _ := body.bones.Find(body.nearest(s))
	moved := frames[at].Apply(body.bones.Bone(at).Head())
	return math.Abs(moved.Z) / body.outline.length
}

// TestFishTunaBendsOnlyItsTail holds the tuna's middle to swing less
// than the sardine's, and its tail to swing.
func TestFishTunaBendsOnlyItsTail(t *testing.T) {
	const middle, tail = .5, .85
	tuna := fishSwing(t, "pacific-bluefin-tuna", middle)
	sardine := fishSwing(t, "pacific-sardine", middle)
	if !(tuna < sardine) {
		t.Fatalf("the tuna's middle swings %v, the sardine's %v", tuna,
			sardine)
	}
	if fishSwing(t, "pacific-bluefin-tuna", tail) == 0 {
		t.Fatal("the tuna's tail does not swing")
	}
}

// fishStored is the faces a school stores and the meshes it shares.
func fishStored(t *testing.T, count int) (int, int) {
	t.Helper()
	made, err := fishSchoolRow{species: "pacific-sardine",
		count: count}.made(sow(9))
	if err != nil {
		t.Fatal(err)
	}
	meshes := map[shape.Shape]bool{}
	for _, group := range made.groups {
		for _, member := range group.Members() {
			meshes[member.Mesh] = true
		}
	}
	return made.faces(), len(meshes)
}

// TestFishSchoolDoesNotGrowItsMeshes holds a school of ten times the fish
// to the same stored faces and the same shared meshes.
func TestFishSchoolDoesNotGrowItsMeshes(t *testing.T) {
	few, fewMeshes := fishStored(t, 500)
	many, manyMeshes := fishStored(t, 5000)
	if few != many || fewMeshes != manyMeshes {
		t.Fatalf("500 fish store %d faces in %d meshes, 5000 store %d in "+
			"%d", few, fewMeshes, many, manyMeshes)
	}
}

// TestFishSchoolPacksByBodyLength holds the school's nearest fish to lie
// within a few body lengths of one another, not on top of one another.
func TestFishSchoolPacksByBodyLength(t *testing.T) {
	plan := planFish("pacific-sardine", sow(4))
	places := plan.schoolPlaces(200)
	row := plan.species.Figures["length"]
	length := (row.Least + row.Most) / 2
	for i, place := range places {
		nearest := math.Inf(1)
		for j, other := range places {
			if i != j {
				nearest = math.Min(nearest,
					place.Origin.Sub(other.Origin).Len())
			}
		}
		if nearest < .2*length || nearest > 2*length {
			t.Fatalf("fish %d's nearest is %v away, %v lengths", i,
				nearest, nearest/length)
		}
	}
}

// TestFishRefusesAFigureWithNoBasis is an anti-test: a figure with no
// basis, a read one with no source, a name nobody reads, and a column
// with no caudal vertebrae are each refused.
func TestFishRefusesAFigureWithNoBasis(t *testing.T) {
	name := fishNames(t)[0]
	cases := map[string]fishFile{
		"no basis": fishWith(name, "depth", fishFigure{Least: .2,
			Most: .3}),
		"read with no source": fishWith(name, "depth", fishFigure{
			Least: .2, Most: .3, Basis: fishFromRead}),
		"a name nobody reads": fishWith(name, "girth", fishFigure{
			Least: .2, Most: .3, Basis: fishFromPicked}),
		"no caudal vertebrae": fishWith(name, "precaudal", fishFigure{
			Least: 60, Most: 60, Basis: fishFromPicked}),
		"backward range": fishWith(name, "depth", fishFigure{Least: .3,
			Most: .2, Basis: fishFromPicked}),
	}
	for why, file := range cases {
		if err := file.checked(); err == nil {
			t.Fatalf("a file with %s was taken", why)
		}
	}
}

// TestFishRefusesAnUnsownSeed is an anti-test: a fish and a school from
// a seed nobody sowed are refused.
func TestFishRefusesAnUnsownSeed(t *testing.T) {
	name := fishNames(t)[0]
	if _, err := (fishRow{species: name}).made(seed{}); err == nil {
		t.Fatal("a fish grew from an unsown seed")
	}
	if _, err := (fishSchoolRow{species: name, count: 3}).made(
		seed{}); err == nil {
		t.Fatal("a school grew from an unsown seed")
	}
}

// TestFishSkinFollowsTheBones holds the skin at rest to be where the
// loft put it, and in the stroke to move with the tail.
func TestFishSkinFollowsTheBones(t *testing.T) {
	plan := planFish("pacific-sardine", sow(6))
	body, err := buildFish(plan)
	if err != nil {
		t.Fatal(err)
	}
	rest := body.skin(body.bones.Frames(skeleton.Pose{}), fishSchoolRings,
		fishSchoolAround)
	for i, p := range rest.points {
		if !p.Near(rest.rest[i], 1e-9) {
			t.Fatalf("point %d moved at rest: %v from %v", i, p, rest.rest[i])
		}
	}
	frames := body.bones.Frames(body.stroke(.1))
	posed := body.skin(frames, fishSchoolRings,
		fishSchoolAround)
	last := len(posed.points) - 1
	if posed.points[last].Near(posed.rest[last], 1e-6) {
		t.Fatal("the tail's last point did not move in the stroke")
	}
}
