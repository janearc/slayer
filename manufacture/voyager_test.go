package manufacture

import (
	_ "embed"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/research"
)

// voyagerTestNumbersRaw is the numbers the Voyager tests chose.
//
//go:embed testdata/voyager-test-numbers.yaml
var voyagerTestNumbersRaw []byte

// voyagerTestNumber is a number the Voyager tests chose, by its name.
func voyagerTestNumber(t *testing.T, name string) float64 {
	t.Helper()
	read, err := numbers.Read("voyager-test-numbers.yaml",
		voyagerTestNumbersRaw)
	if err != nil {
		t.Fatal(err)
	}
	number, err := read.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return number.Value
}

// voyagerSeed is the seed the Voyager tests sow.
func voyagerSeed(t *testing.T) seed {
	t.Helper()
	number := uint64(voyagerTestNumber(t, "seed"))
	return sow(number)
}

// voyagerTwins are both twins made from the seed of the tests.
func voyagerTwins(t *testing.T) (good, good) {
	t.Helper()
	number := uint64(voyagerTestNumber(t, "seed"))
	return mustMake(t, "voyager-1", number), mustMake(t, "voyager-2", number)
}

// firstFailingPart says which part of a material first makes its mesh
// invalid, by adding the parts of the material one at a time.
func firstFailingPart(one good, material slot) string {
	var prefix surface
	for index, piece := range one.parts {
		if piece.of != material {
			continue
		}
		prefix = append(prefix, piece.skin()...)
		if _, err := prefix.meshed(); err != nil {
			return reflect.TypeOf(piece.shape).String() +
				" at part " + strconv.Itoa(index)
		}
	}
	return "no part"
}

func TestVoyagerIsAMultiSubsystemModel(t *testing.T) {
	one, _ := voyagerTwins(t)
	least := int(voyagerTestNumber(t, "least-parts"))
	if len(one.parts) < least {
		t.Fatalf("Voyager has %d parts; want at least %d", len(one.parts),
			least)
	}
	most := int(voyagerTestNumber(t, "least-faces"))
	if faces := one.faces(); faces < most {
		t.Fatalf("Voyager has %d faces; want at least %d", faces, most)
	}
}

func TestVoyagerTwinsDoNotDiverge(t *testing.T) {
	one, two := voyagerTwins(t)
	if !reflect.DeepEqual(one.skins(), two.skins()) ||
		!reflect.DeepEqual(one.groups, two.groups) {
		t.Fatal("identical Voyager twins diverged in geometry or materials")
	}
}

func TestEveryVoyagerMaterialIsAValidSceneMesh(t *testing.T) {
	one, _ := voyagerTwins(t)
	for material, skin := range one.skins() {
		if _, err := skin.meshed(); err != nil {
			t.Errorf("Voyager material slot %d is not a valid scene mesh "+
				"(first fails at %s): %v", material,
				firstFailingPart(one, material), err)
		}
	}
}

func TestVoyagerDemoKeepsItsAssemblyUnderTheFaceLimit(t *testing.T) {
	one, _ := voyagerTwins(t)
	row, err := kindNamed("voyager-1")
	if err != nil {
		t.Fatal(err)
	}
	number := uint64(voyagerTestNumber(t, "seed"))
	demo, err := MakeWithComplexity(row, sow(number), "demo")
	if err != nil {
		t.Fatal(err)
	}
	limit := int(voyagerTestNumber(t, "face-limit"))
	t.Logf("Voyager per twin: %d pieces, %d standard faces, %d demo faces",
		len(one.parts), one.faces(), demo.faces())
	if len(demo.parts) != len(one.parts) || demo.faces() > limit {
		t.Fatalf("demo representation has %d parts and %d faces; want the "+
			"same assembly under the %d-face draw limit", len(demo.parts),
			demo.faces(), limit)
	}
}

func TestVoyagerSourceDimensionsArePresent(t *testing.T) {
	for _, name := range []string{
		"voyager-hga-diameter-metres",
		"voyager-magnetometer-boom-metres",
		"voyager-radio-antennas-metres",
		"voyager-rtg-boom-metres",
		"voyager-science-boom-metres",
		"voyager-bus-diameter-metres",
	} {
		if value := research.Value(name); value <= spaceZero {
			t.Errorf("source dimension %q is missing: %v", name, value)
		}
	}
}

func TestVoyagerBookAndLooksAreWhole(t *testing.T) {
	if voyagerBookError != nil {
		t.Errorf("the Voyager book is broken: %v", voyagerBookError)
	}
	if voyagerDressError != nil {
		t.Errorf("the Voyager looks are broken: %v", voyagerDressError)
	}
}

func TestVoyagerAsksForEveryNameItsBookHolds(t *testing.T) {
	seeded := voyagerSeed(t)
	if _, err := (voyagerMaker{}).madeAt(seeded, "quick"); err != nil {
		t.Fatalf("Voyager is refused by its own book: %v", err)
	}
}

func TestAVoyagerWithNoSeedIsRefused(t *testing.T) {
	_, err := (voyagerMaker{}).made(seed{})
	if !errors.Is(err, errGood) {
		t.Fatalf("a Voyager of no seed gave %v; want errGood", err)
	}
	seeded := voyagerSeed(t)
	if _, err := (voyagerMaker{}).made(seeded); err != nil {
		t.Fatalf("a Voyager of a seed was refused: %v", err)
	}
}
