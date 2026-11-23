package manufacture

import (
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// telescopeName is the kind a telescope on a roof is.
const telescopeName = "optical-observatory"

// telescopeScale is the share of its catalogue size a telescope is drawn
// at in these tests, as the condominium's culture gives it.
const telescopeScale = 0.4

// keepers are the condominiums of the seeds that keep a telescope, drawn
// at the demo level, each with the good made of what it draws before any
// telescope is stood on it.
type keeper struct {
	base   good
	design cities.Design
	from   seed
}

func keepers(t *testing.T) []keeper {
	t.Helper()
	const district = 256
	var found []keeper
	for index := range 24 {
		from := Sow(uint64(index * district))
		design, err := cities.CondominiumAt(cities.Seed(from.number), nil,
			"demo")
		if err != nil {
			t.Fatal(err)
		}
		if len(design.Mounts) == 0 {
			continue
		}
		drawn, err := design.Rendered("demo")
		if err != nil {
			t.Fatal(err)
		}
		base, err := goodOfRendered(drawn)
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, keeper{base, design, from})
	}
	if len(found) == 0 {
		t.Fatal("no condominium of the seeds keeps a telescope")
	}
	return found
}

func TestEveryMountIsTheCataloguesObservatory(t *testing.T) {
	for _, one := range keepers(t) {
		for _, mount := range one.design.Mounts {
			if _, err := kindNamed(mount.Kind); err != nil {
				t.Errorf("a mount of the kind %q: %v", mount.Kind, err)
			}
		}
	}
}

func TestACondominiumWithItsTelescopesIsAlwaysAMesh(t *testing.T) {
	drawn := 0
	for _, one := range keepers(t) {
		made, err := condominiumRow{}.madeAt(one.from, "demo")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := made.things(maths.Local()); err != nil {
			t.Errorf("seed %d: not a mesh: %v", one.from.number, err)
		}
		if made.PartCount() > one.base.PartCount() {
			drawn++
		}
	}
	if drawn == 0 {
		t.Error("no telescope was drawn on any condominium that keeps one")
	}
}

func TestSomeTelescopesAreDrawnAndSomeAreLeftOutWhereTheyWillNotMesh(t *testing.T) {
	base := keepers(t)[0]
	drawn, left := 0, 0
	for index := range 60 {
		at := maths.V(float64(index)*0.37-30, 18+float64(index%7)*3.1,
			float64(index%13)*1.73-20)
		mount := cities.Mount{Kind: telescopeName,
			Place: maths.NewSubspace(at, maths.Identity, telescopeScale)}
		with, err := withMounts(base.base, []cities.Mount{mount}, base.from,
			"demo")
		if err != nil {
			t.Fatalf("place %d: %v", index, err)
		}
		if with.PartCount() > base.base.PartCount() {
			drawn++
		} else {
			left++
		}
	}
	if drawn == 0 || left == 0 {
		t.Errorf("%d telescopes drawn and %d left out, of 60 places", drawn,
			left)
	}
}

func TestAMountOfAKindThereIsNoneOfIsRefused(t *testing.T) {
	one := keepers(t)[0]
	nothing := []cities.Mount{{Kind: "no-such-kind", Place: maths.Local()}}
	if _, err := withMounts(one.base, nothing, one.from, "demo"); err == nil {
		t.Error("a mount of a kind that does not exist was made")
	}
}

func TestAGoodWithNoMountsIsReturnedAsItWas(t *testing.T) {
	one := keepers(t)[0]
	same, err := withMounts(one.base, nil, one.from, "demo")
	if err != nil || same.PartCount() != one.base.PartCount() {
		t.Errorf("a good with no mounts came back changed: %v", err)
	}
}
