package manufacture

import (
	"maps"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// the scaffold's contracts: the tower stands its lifts and bays, every
// tube of a length is one shared mesh, the seed picks which outer bays
// are braced and the same seed picks the same, and a tower that cannot
// stand is refused.

// tubesOf is a made scaffold's steel and deck members.
func tubesOf(t *testing.T, made good) (steel, decks []shape.MeshMember) {
	t.Helper()
	if made.groups[scaffoldSteel] == nil || made.groups[scaffoldDeck] == nil {
		t.Fatal("the scaffold's parts are not shared")
	}
	return made.groups[scaffoldSteel].Members(),
		made.groups[scaffoldDeck].Members()
}

// meshesIn is how many distinct meshes the members share.
func meshesIn(members []shape.MeshMember) int {
	distinct := map[shape.Shape]bool{}
	for _, member := range members {
		distinct[member.Mesh] = true
	}
	return len(distinct)
}

// the standing tubes: 7 by 4 standards, 45 ledgers a lift for 20
// lifts, and 18 in the guard rail; the rest of the steel is braces.
const towerStanding = 7*4 + 45*20 + 18

func TestAScaffoldTowerStandsItsLiftsAndBays(t *testing.T) {
	made, err := scaffoldRow().made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	box := *made.extent
	tall := box.Max.Y - box.Min.Y
	wide := box.Max.X - box.Min.X
	deep := box.Max.Z - box.Min.Z
	for _, measured := range []struct {
		name      string
		got, want float64
	}{
		{"tall", tall, 20*2 + 1 + 2*0.024},
		{"wide", wide, 6*2.57 + 2*0.024},
		{"deep", deep, 3*2.57 + 2*0.024},
	} {
		if !(math.Abs(measured.got-measured.want) <= 1e-9) {
			t.Fatalf("%s: %.4f m, want %.4f", measured.name,
				measured.got, measured.want)
		}
	}
	for of, group := range made.groups {
		if !within(group.Bounds(), box) {
			t.Fatalf("matter %d reaches %v, outside the scaffold's %v", of,
				group.Bounds(), box)
		}
	}
	steel, decks := tubesOf(t, made)
	if len(steel) <= towerStanding {
		t.Fatalf("%d steel tubes: no braces past the %d standing",
			len(steel), towerStanding)
	}
	// a plank in each of 18 bays, at lifts 5, 10, 15 and 20
	if len(decks) != 18*4 {
		t.Fatalf("%d planks, want %d", len(decks), 18*4)
	}
}

func TestEveryScaffoldTubeOfALengthIsOneMesh(t *testing.T) {
	made, err := scaffoldRow().made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	steel, decks := tubesOf(t, made)
	// a standard, a ledger (the rail's are ledgers too), and a brace
	if got := meshesIn(steel); got != 3 {
		t.Fatalf("the steel is %d meshes, want 3", got)
	}
	if got := meshesIn(decks); got != 1 {
		t.Fatalf("the planks are %d meshes, want 1", got)
	}
	if stored, all := made.faces(), made.groups[scaffoldSteel].FaceCount(); stored*10 > all {
		t.Fatalf("%d faces stored for %d drawn: the tubes are not shared",
			stored, all)
	}
}

// brace is where a brace runs, from its foot along its way.
type brace struct{ foot, way maths.Vec }

// bracesOf is a made scaffold's braces: the tubes that run neither
// upright nor level. the group may hold its members in any order, so
// they are a set.
func bracesOf(t *testing.T, number uint64) map[brace]bool {
	t.Helper()
	made, err := scaffoldRow().made(sow(number))
	if err != nil {
		t.Fatal(err)
	}
	steel, _ := tubesOf(t, made)
	braces := map[brace]bool{}
	for _, member := range steel {
		way := member.Space.Orientation.Y
		if rise := math.Abs(way.Y); rise > 0.1 && rise < 0.9 {
			braces[brace{member.Space.Origin, way}] = true
		}
	}
	if len(braces) != len(steel)-towerStanding {
		t.Fatalf("%d braces among %d steel tubes, want %d", len(braces),
			len(steel), len(steel)-towerStanding)
	}
	return braces
}

func TestTheSeedPicksTheBracedBays(t *testing.T) {
	first, again, other := bracesOf(t, 7), bracesOf(t, 7), bracesOf(t, 8)
	if !maps.Equal(first, again) {
		t.Fatal("one seed braced different bays on two makings")
	}
	// 360 outer bays over the twenty lifts, about half braced
	if share := float64(len(first)) / 360; share < 0.35 || share > 0.65 {
		t.Fatalf("%.2f of the outer bays braced, want about half", share)
	}
	if maps.Equal(first, other) {
		t.Fatal("two seeds braced the same bays")
	}
}

// within is whether a box lies inside another, to a hair.
func within(inner, outer maths.Box) bool {
	const hair = 1e-9
	return inner.Min.X >= outer.Min.X-hair && inner.Min.Y >= outer.Min.Y-hair &&
		inner.Min.Z >= outer.Min.Z-hair && inner.Max.X <= outer.Max.X+hair &&
		inner.Max.Y <= outer.Max.Y+hair && inner.Max.Z <= outer.Max.Z+hair
}

func TestAStageTowerCarriesItsGearWithinItsBox(t *testing.T) {
	made, err := stageTowerRow().made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	gear := made.groups[scaffoldGear]
	if gear == nil {
		t.Fatal("the stage tower carries no gear")
	}
	// two laser heads, two flame units and their two nozzles, six
	// moving heads, two hangers and twenty cabinets, three road cases
	if got := len(gear.Members()); got != 2+2+2+6+2+20+3 {
		t.Fatalf("%d pieces of gear, want 37", got)
	}
	box := *made.extent
	if tall := box.Max.Y - box.Min.Y; !(math.Abs(tall-(10*2+1+2*0.024)) <= 1e-9) {
		t.Fatalf("the tower's box is %.3f m tall, want 21.048", tall)
	}
	for of, group := range made.groups {
		if !within(group.Bounds(), box) {
			t.Fatalf("matter %d reaches %v, outside the tower's %v", of,
				group.Bounds(), box)
		}
	}
	bare, err := scaffoldRow().made(sow(7))
	if err != nil {
		t.Fatal(err)
	}
	if bare.groups[scaffoldGear] != nil {
		t.Fatal("the bare scaffold carries gear")
	}
}

func TestAScaffoldThatCannotStandIsRefused(t *testing.T) {
	if _, err := scaffoldRow().made(seed{}); err != errGood {
		t.Fatalf("an unsown seed: got %v, want %v", err, errGood)
	}
	thin := scaffoldRow()
	thin.around = 2
	if _, err := thin.made(sow(7)); err != errCut {
		t.Fatalf("two sides: got %v, want %v", err, errCut)
	}
	hollow := scaffoldRow()
	hollow.tube = measure.Metres(0)
	if _, err := hollow.made(sow(7)); err == nil {
		t.Fatal("a tube of no radius was made")
	}
}
