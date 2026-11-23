package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// Distances, in metres straight ahead of the eye of lookingDown, at which
// a gargoyle spans well under a pixel, a few pixels and many. A span in
// pixels falls as the distance grows, and a gargoyle of the cornice spans
// between 4700 and 6300 pixels at a metre, measured, so that from 1600 to
// 4700 m it spans more than a pixel and under four, as mountMid says, and
// at 600 m it spans eight to ten.
const (
	mountFar    = 60000.0
	mountMid    = 2000.0
	mountNear   = 30.0
	leastMounts = 3
)

// sightAt is the eye of lookingDown with a tower standing so far ahead.
func sightAt(distance float64) *mountSight {
	place := maths.Local()
	place.Origin = maths.V(0, 0, -distance)
	return &mountSight{eye: lookingDown(), place: place}
}

// corniceWall is a cornice's wall, its mounts, and its row, in granite.
func corniceWall(t *testing.T) (good, []cities.Mount, gargoyleCorniceRow) {
	t.Helper()
	row := gargoyleCorniceRow{stone: gargoyleStones()["granite"]}
	design, err := cities.GargoyleCornice(cities.Seed(1), "granite")
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(cities.Detail("quick"))
	if err != nil {
		t.Fatal(err)
	}
	wall, err := goodOfRendered(drawn)
	if err != nil {
		t.Fatal(err)
	}
	if len(design.Mounts) < leastMounts {
		t.Fatalf("the cornice has %d mounts", len(design.Mounts))
	}
	return wall, design.Mounts, row
}

func TestAMountUnderAPixelIsNotMadeAndTheWallStandsForIt(t *testing.T) {
	wall, mounts, row := corniceWall(t)
	made, err := row.withGargoyles(wall, mounts, sow(1), "quick",
		sightAt(mountFar))
	if err != nil {
		t.Fatal(err)
	}
	if len(made.parts) != len(wall.parts) ||
		len(made.native) != len(wall.native) {
		t.Errorf("%d parts and %d native pieces, the wall has %d and %d",
			len(made.parts), len(made.native), len(wall.parts),
			len(wall.native))
	}
}

func TestAMountOfAFewPixelsIsAPlainBlockEach(t *testing.T) {
	wall, mounts, row := corniceWall(t)
	made, err := row.withGargoyles(wall, mounts, sow(1), "quick",
		sightAt(mountMid))
	if err != nil {
		t.Fatal(err)
	}
	if want := len(wall.parts) + len(mounts); len(made.parts) != want ||
		len(made.native) != len(wall.native) {
		t.Errorf("%d parts and %d native pieces, want %d parts of the wall "+
			"and a block a mount and no carving", len(made.parts),
			len(made.native), want)
	}
}

func TestAMountOfManyPixelsIsTheGargoyleWholeAsWithNoSight(t *testing.T) {
	wall, mounts, row := corniceWall(t)
	near, err := row.withGargoyles(wall, mounts, sow(1), "quick",
		sightAt(mountNear))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := row.withGargoyles(wall, mounts, sow(1), "quick", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(near.parts) != len(plain.parts) ||
		len(near.native) != len(plain.native) || len(near.native) == 0 {
		t.Errorf("near: %d parts, %d native; as today: %d parts, %d native",
			len(near.parts), len(near.native), len(plain.parts),
			len(plain.native))
	}
}

func TestABlockKeepsTheBoxOfItsMountSoTheSilhouetteStays(t *testing.T) {
	wall, mounts, row := corniceWall(t)
	made, err := row.withGargoyles(wall, mounts[:1], sow(1), "quick",
		sightAt(mountMid))
	if err != nil {
		t.Fatal(err)
	}
	kind, err := kindNamed(mounts[0].Kind)
	if err != nil {
		t.Fatal(err)
	}
	box, err := mountExtent(kind, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	want := transformed(box, mounts[0].Place)
	got := made.Bounds()
	if got.Min.Y > want.Min.Y+1e-6 || got.Max.Y < want.Max.Y-1e-6 {
		t.Errorf("the cornice spans %v in y, the mount's box %v", got, want)
	}
}

func TestAGargoyleKindsBoxIsMeasuredOnceAndKept(t *testing.T) {
	_, mounts, _ := corniceWall(t)
	kind, err := kindNamed(mounts[0].Kind)
	if err != nil {
		t.Fatal(err)
	}
	first, err := mountExtent(kind, sow(1))
	if err != nil {
		t.Fatal(err)
	}
	again, err := mountExtent(kind, sow(99))
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Errorf("a seed changed the kind's box: %v then %v", first, again)
	}
}

func TestATowerOutOfViewMakesNoMountsAndOneInViewMakesThem(t *testing.T) {
	t.Setenv(scrooge.Variable, "1")
	yes := true
	row := towerRow{spec: TowerSpec{Gargoyles: &yes}}
	tower, err := row.drawn(sow(3))
	if err != nil {
		t.Fatal(err)
	}
	made, err := towerRow{}.madeAt(sow(3), "quick")
	if err != nil {
		t.Fatal(err)
	}
	eye := lookingDown()
	pass := cities.Pass{Leveller: cities.Constant("quick"), Finest: "quick",
		Eye: &eye}
	behind := maths.Local()
	behind.Origin = maths.V(0, 0, 500)
	none, err := row.gargoyled(made, tower, sow(3), pass, behind)
	if err != nil {
		t.Fatal(err)
	}
	if len(none.parts) != len(made.parts) ||
		len(none.native) != len(made.native) {
		t.Errorf("a tower behind the eye was given mounts")
	}
	ahead := maths.Local()
	ahead.Origin = maths.V(0, 0, -mountNear)
	some, err := row.gargoyled(made, tower, sow(3), pass, ahead)
	if err != nil {
		t.Fatal(err)
	}
	if len(some.native) <= len(made.native) {
		t.Errorf("a tower close ahead was given no gargoyles")
	}
}

func TestWithScroogeNotAskedEveryMountIsWholeWhateverTheEye(t *testing.T) {
	t.Setenv(scrooge.Variable, "")
	yes := true
	row := towerRow{spec: TowerSpec{Gargoyles: &yes}}
	tower, err := row.drawn(sow(3))
	if err != nil {
		t.Fatal(err)
	}
	made, err := towerRow{}.madeAt(sow(3), "quick")
	if err != nil {
		t.Fatal(err)
	}
	eye := lookingDown()
	pass := cities.Pass{Leveller: cities.Constant("quick"), Finest: "quick",
		Eye: &eye}
	far := maths.Local()
	far.Origin = maths.V(0, 0, -mountFar)
	some, err := row.gargoyled(made, tower, sow(3), pass, far)
	if err != nil {
		t.Fatal(err)
	}
	if len(some.native) <= len(made.native) {
		t.Errorf("a far tower lost its gargoyles with scrooge not asked")
	}
}
