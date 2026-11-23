package manufacture

import (
	"testing"
)

// roofMix is a set's mix for the roofs of a superblock: motorhomes, which
// are small enough for a berth.
var roofMix = map[string]float64{"winnebago": 1}

// roofSeed is a seed whose roofs, on the test's parcel, keep berths: the
// cities tests find them for it.
const roofSeed = 6

// roofParcel is a parcel big enough, 120 by 80 m, for roofs that take a
// berth, grown over.
var roofParcel = SuperblockAsk{Width: 120, Depth: 80, Ceiling: 8,
	Accretion: 1}

// theParcel is the size of the parcel the manufactory tests build over,
// 60 by 40 m and eight storeys, grown over.
var theParcel = SuperblockAsk{Width: 60, Depth: 40, Ceiling: 8,
	Accretion: 1}

func TestTheSetsAccretionAndPeopleReachTheBlock(t *testing.T) {
	planned := theParcel
	planned.Accretion = 0
	fewer := planned
	fewer.Occupants = theParcel.Ceiling
	goods := map[string]SuperblockAsk{"grown": theParcel,
		"planned": planned, "fewer": fewer}
	sizes := map[string]float64{}
	faces := map[string]int{}
	for name, ask := range goods {
		made, err := MakeWithComplexity(asked(t, ask), sow(2), "hull")
		if err != nil {
			t.Fatal(err)
		}
		sizes[name] = made.Bounds().Max.Sub(made.Bounds().Min).Y
		faces[name] = made.faces()
	}
	if faces["grown"] == faces["planned"] {
		t.Error("a block grown over and one planned made the same good")
	}
	if sizes["fewer"] >= sizes["planned"] {
		t.Errorf("a block for %d people stands %v m, not under the %v m "+
			"of one for the data's", fewer.Occupants, sizes["fewer"],
			sizes["planned"])
	}
}

// mixOf is an ask whose roofs take one kind of part at a weight.
func mixOf(name string, weight float64) SuperblockAsk {
	return SuperblockAsk{Parts: map[string]float64{name: weight}}
}

// asked is the superblock row with an ask, or the test's end.
func asked(t *testing.T, ask SuperblockAsk) Kind {
	t.Helper()
	row, err := KindNamed(SuperblockKind)
	if err != nil {
		t.Fatal(err)
	}
	row, err = Superblocked(row, ask)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestASuperblockIsOneGoodOfManyWindowsAndBands(t *testing.T) {
	made, err := MakeWithComplexity(asked(t, theParcel), sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, look := range made.dressed {
		names[look.Name] = true
	}
	for _, want := range []string{"cities glazing", "cities lashing",
		"cities roof tar"} {
		if !names[want] {
			t.Errorf("no material %q among %v", want, names)
		}
	}
	// The block stands on a podium of the parcel less an apron on every
	// side, and what it carries over the apron stops at the parcel.
	size := made.Bounds().Max.Sub(made.Bounds().Min)
	if size.X < theParcel.Width-2*apronWidth || size.X > theParcel.Width ||
		size.Z < theParcel.Depth-2*apronWidth || size.Z > theParcel.Depth {
		t.Errorf("a superblock of %v on a parcel of %v by %v m with an apron "+
			"of %v", size, theParcel.Width, theParcel.Depth, apronWidth)
	}
}

// apronWidth is the pavement between a parcel's edge and the podium, in
// metres: the culture's apron-width, 3 m.
const apronWidth = 3.0

func TestASetThatSaysNothingGetsTheCulturesOwnParcel(t *testing.T) {
	row, err := KindNamed(SuperblockKind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MakeWithComplexity(row, sow(1), "hull"); err != nil {
		t.Fatal(err)
	}
}

func TestPartsFromTheSetStandInTheBerthsOfTheRoofs(t *testing.T) {
	bare := asked(t, roofParcel)
	withParts := roofParcel
	withParts.Parts = roofMix
	mixed := asked(t, withParts)
	without, err := MakeWithComplexity(bare, sow(roofSeed), "hull")
	if err != nil {
		t.Fatal(err)
	}
	with, err := MakeWithComplexity(mixed, sow(roofSeed), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if len(with.slots()) <= len(without.slots()) {
		withParts := len(with.slots())
		withoutParts := len(without.slots())
		t.Errorf("%d materials with parts and %d without: no part stands "+
			"on a roof", withParts, withoutParts)
	}
}

func TestASuperblockWithTheSameSeedIsTheSameGood(t *testing.T) {
	first, err := MakeWithComplexity(asked(t, theParcel), sow(4), "hull")
	if err != nil {
		t.Fatal(err)
	}
	second, err := MakeWithComplexity(asked(t, theParcel), sow(4), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if first.faces() != second.faces() || first.Bounds() != second.Bounds() {
		t.Error("one seed made two different superblocks")
	}
}

func TestASuperblockAskThatMakesNoSenseIsRefused(t *testing.T) {
	superblock, err := KindNamed(SuperblockKind)
	if err != nil {
		t.Fatal(err)
	}
	other, err := KindNamed("yurt")
	if err != nil {
		t.Fatal(err)
	}
	flawed := map[string]struct {
		row Kind
		ask SuperblockAsk
	}{
		"not a superblock": {other, theParcel},
		"another habit":    {superblock, SuperblockAsk{Habit: "lattice"}},
		"no such part":     {superblock, mixOf("atlantis", 1)},
		"a cluster":        {superblock, mixOf(ClusterKind, 1)},
		"itself":           {superblock, mixOf(SuperblockKind, 1)},
		"no weight":        {superblock, mixOf("yurt", 0)},
		"negative width":   {superblock, SuperblockAsk{Width: -1}},
		"negative depth":   {superblock, SuperblockAsk{Depth: -1}},
		"negative ceiling": {superblock, SuperblockAsk{Ceiling: -1}},
		"negative people":  {superblock, SuperblockAsk{Occupants: -1}},
		"negative crowd":   {superblock, SuperblockAsk{Crowding: -1}},
		"overgrown":        {superblock, SuperblockAsk{Accretion: 1 + 1}},
		"ungrown":          {superblock, SuperblockAsk{Accretion: -1}},
	}
	for name, flaw := range flawed {
		if _, err := Superblocked(flaw.row, flaw.ask); err == nil {
			t.Errorf("%s: %+v was taken", name, flaw.ask)
		}
	}
}

func TestAnUnsownSeedMakesNoSuperblock(t *testing.T) {
	if _, err := (superblockRow{}).madeAt(seed{}, "hull"); err == nil {
		t.Error("a superblock was made from a seed nobody sowed")
	}
}
