package manufacture

import (
	"strings"
	"testing"
	"time"
)

// TestKelpFiguresAreSound checks that the file reads and that every
// figure has a basis the file allows.
func TestKelpFiguresAreSound(test *testing.T) {
	if kelpError != nil {
		test.Fatalf("kelp.yaml refused: %v", kelpError)
	}
	if bubblesError != nil {
		test.Fatalf("bubbles.yaml refused: %v", bubblesError)
	}
}

// TestKelpRefusesAFigureWithNoBasis is the anti-test: a figure with no
// basis, or a cited one naming no listed source, is refused.
func TestKelpRefusesAFigureWithNoBasis(test *testing.T) {
	damages := [][2]string{
		{`basis: "picked: a young plant`, `basis: "invented: a young plant`},
		{`basis: "cited: wikipedia-macrocystis, stipes`, `basis: "cited: nowhere, stipes`},
	}
	for _, pair := range damages {
		whole, damaged := pair[0], pair[1]
		raw := strings.Replace(string(kelpRaw), whole, damaged, 1)
		if raw == string(kelpRaw) {
			test.Fatalf("the file has no %q to damage", whole)
		}
		if _, err := readKelp([]byte(raw)); err == nil {
			test.Fatalf("a figure with the basis %q was accepted", damaged)
		}
	}
}

// TestKelpHonoursItsSeed checks that a seed makes the same plant each time
// and that another seed makes another.
func TestKelpHonoursItsSeed(test *testing.T) {
	row, err := kindNamed(KelpKind)
	if err != nil {
		test.Fatal(err)
	}
	first, err := makeOf(row, sow(1))
	again, err2 := makeOf(row, sow(1))
	other, err3 := makeOf(row, sow(2))
	if err != nil || err2 != nil || err3 != nil {
		test.Fatal(err, err2, err3)
	}
	if first.groupParts[body] != again.groupParts[body] ||
		first.groupParts[leaf] != again.groupParts[leaf] {
		test.Fatal("one seed made two plants")
	}
	if first.groupParts[leaf] == other.groupParts[leaf] &&
		first.groupParts[body] == other.groupParts[body] {
		test.Fatal("two seeds made plants of one count; no two are alike")
	}
}

// TestKelpRefusesAnUnsownSeed is the anti-test for the seed's guard.
func TestKelpRefusesAnUnsownSeed(test *testing.T) {
	row, _ := kindNamed(KelpKind)
	if _, err := makeOf(row, seed{}); err == nil {
		test.Fatal("a seed nobody sowed made a plant")
	}
}

// TestKelpForestStoresNoMoreFacesThanAPlant checks that a forest shares
// its meshes: the faces stored do not grow with the plants.
func TestKelpForestStoresNoMoreFacesThanAPlant(test *testing.T) {
	plant, _ := kindNamed(KelpKind)
	forest, _ := kindNamed(KelpForestKind)
	one, err := makeOf(plant, sow(3))
	many, err2 := makeOf(forest, sow(3))
	if err != nil || err2 != nil {
		test.Fatal(err, err2)
	}
	if many.faces() != one.faces() {
		test.Fatalf("a forest stores %d faces and a plant %d", many.faces(),
			one.faces())
	}
	if many.groupParts[leaf] <= one.groupParts[leaf] {
		test.Fatal("a forest has no more blades than a plant")
	}
}

// TestKelpLeansWithACurrent checks that the same seed in a current hangs
// a stipe's top farther downstream than in still water, and that the
// swell moves it with time.
func TestKelpLeansWithACurrent(test *testing.T) {
	still := kelpPlant{from: sow(4), water: kelpRead.Conditions["slack"]}
	moving := kelpPlant{from: sow(4), water: kelpRead.Conditions["surge"]}
	top := still.top()
	if still.drift(top).X != 0 {
		test.Fatal("still water moved a stipe")
	}
	if moving.drift(top).X <= 0 {
		test.Fatal("a current did not lean a stipe downstream")
	}
	later := moving
	later.clock = moving.water.SwellPeriod / 4
	if later.drift(top).X == moving.drift(top).X {
		test.Fatal("the swell did not move with time")
	}
	if moving.drift(0).Len() != 0 {
		test.Fatal("the foot moved")
	}
}

// TestKelpMovesWithTime checks that a deforming kind is one.
func TestKelpMovesWithTime(test *testing.T) {
	row, _ := kindNamed(KelpKind)
	if !Deforms(row) {
		test.Fatal("kelp is not a deforming kind")
	}
	if _, err := MakeAtTime(row, sow(5), "quick", 3*time.Second); err != nil {
		test.Fatal(err)
	}
}

// TestBubblesAreAirInWater checks the bubble's material: the index of air,
// clear, and a rank above water's.
func TestBubblesAreAirInWater(test *testing.T) {
	row, _ := kindNamed(BubblesKind)
	made, err := makeOf(row, sow(6))
	if err != nil {
		test.Fatal(err)
	}
	air := made.dressed[body]
	if air.Index != 1.0003 || air.Through != 1 || air.Rank < 1 {
		test.Fatalf("a bubble is not air in water: %+v", air)
	}
	if air.Colour.R != 1 {
		test.Fatal("a bubble was tinted")
	}
}

// TestBubblesRiseAndBeginAgain checks that a stream's bubbles stay
// within its height and that the stream moves with time.
func TestBubblesRiseAndBeginAgain(test *testing.T) {
	row, _ := kindNamed(BubblesKind)
	early, err := MakeAtTime(row, sow(7), "quick", 0)
	later, err2 := MakeAtTime(row, sow(7), "quick", 5*time.Second)
	if err != nil || err2 != nil {
		test.Fatal(err, err2)
	}
	if early.groupParts[body] != later.groupParts[body] {
		test.Fatal("a stream gained or lost bubbles with time")
	}
}

// TestBubblesRefuseADamagedFile is the anti-test: a constant that is
// not a number with a basis is refused.
func TestBubblesRefuseADamagedFile(test *testing.T) {
	raw := strings.Replace(string(bubblesRaw), `basis: "cited: wikipedia-bubble"`,
		`basis: "cited: nowhere"`, 1)
	if _, err := readBubbles([]byte(raw)); err == nil {
		test.Fatal("a cited figure with no listed source was accepted")
	}
}

// TestBubbleShapeFlattensWithSize checks the shape against the form in
// the file: a bigger bubble is flatter.
func TestBubbleShapeFlattensWithSize(test *testing.T) {
	small, large := bubblesRead.aspect(0.0005), bubblesRead.aspect(0.004)
	if !(large < small && small <= 1) {
		test.Fatalf("aspects %v and %v are not a bubble's", small, large)
	}
}
