package manufacture

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// overlapTolerance is how far two parts' boxes may sink into each other,
// in metres, before they are said to overlap.
const overlapTolerance = 1e-6

// theAsk is a set's cluster, mostly tanks: six parts in ten tanks, two
// boats and two motorhomes.
var theAsk = ClusterAsk{Parts: map[string]float64{"sls-core-tank": 6,
	"alfa-hero": 2, "winnebago": 2}}

func TestTheScavengedClusterIsOneGoodOfSeveralKindsAndBands(t *testing.T) {
	made, err := (scavengedRow{ask: theAsk}).madeAt(sow(1), "hull")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(made.slots()); got < 5 {
		t.Errorf("%d materials, want those of three kinds and the bands",
			got)
	}
	banded := false
	for _, look := range made.dressed {
		banded = banded || look.Name == "cities lashing"
	}
	if !banded {
		t.Error("no material of the cluster is the bands'")
	}
	if size := made.Bounds().Max.Sub(made.Bounds().Min); size.Len() < 20 {
		t.Errorf("the cluster spans only %v", size)
	}
}

func TestTheClustersPartsAreTheSetsAndDoNotOverlap(t *testing.T) {
	laid, err := laidOut(sow(1), "hull", theAsk)
	if err != nil {
		t.Fatal(err)
	}
	if len(laid.parts) != laid.plan.Count {
		t.Fatalf("%d parts, want %d", len(laid.parts), laid.plan.Count)
	}
	for _, part := range laid.parts {
		if _, named := theAsk.Parts[part.kind]; !named {
			t.Errorf("a part is a %s, which the set does not name", part.kind)
		}
	}
	for first, one := range laid.cluster.Joins {
		for _, other := range laid.cluster.Joins[first+1:] {
			if cities.Overlap(laid.parts[one.Piece].piece.At(one.Offset),
				laid.parts[other.Piece].piece.At(other.Offset),
				overlapTolerance) {
				t.Fatalf("parts %d and %d overlap", one.Piece, other.Piece)
			}
		}
	}
}

func TestTheSetsCountAndSpacingAreTheClustersNotTheDatas(t *testing.T) {
	ask := ClusterAsk{Parts: map[string]float64{"yurt": 1}, Count: 5,
		BandSpacing: 2}
	laid, err := laidOut(sow(3), "hull", ask)
	if err != nil {
		t.Fatal(err)
	}
	if len(laid.parts) != 5 || laid.plan.Band.Spacing != 2 {
		t.Errorf("%d parts at a spacing of %v, want 5 at 2", len(laid.parts),
			laid.plan.Band.Spacing)
	}
	for _, part := range laid.parts {
		if part.kind != "yurt" {
			t.Errorf("a part is a %s, and only gers were named", part.kind)
		}
	}
}

func TestAHeavierWeightDrawsMoreOfThatKind(t *testing.T) {
	laid, err := laidOut(sow(9), "hull", ClusterAsk{Count: 40,
		Parts: map[string]float64{"yurt": 9, "winnebago": 1}})
	if err != nil {
		t.Fatal(err)
	}
	gers := 0
	for _, part := range laid.parts {
		if part.kind == "yurt" {
			gers++
		}
	}
	if gers < 28 || gers > 40 {
		t.Errorf("%d of 40 are gers, want most", gers)
	}
}

func TestTheSameSeedMakesTheSameScavengedCluster(t *testing.T) {
	first, err := laidOut(sow(4), "hull", theAsk)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := laidOut(sow(4), "hull", theAsk)
	other, _ := laidOut(sow(5), "hull", theAsk)
	if first.report() != again.report() {
		t.Error("seed 4 made two clusters")
	}
	for index := range first.cluster.Joins {
		if first.cluster.Joins[index].Offset !=
			again.cluster.Joins[index].Offset {
			t.Fatalf("part %d docked in two places", index)
		}
	}
	if first.report() == other.report() &&
		first.cluster.Centre == other.cluster.Centre {
		t.Error("two seeds made one cluster")
	}
}

func TestEachPartIsTurnedByItsSeedInThreeDimensions(t *testing.T) {
	laid, err := laidOut(sow(2), "hull", ClusterAsk{Count: 12,
		Parts: map[string]float64{"sls-core-tank": 1}})
	if err != nil {
		t.Fatal(err)
	}
	across := map[string]bool{}
	for _, part := range laid.parts {
		axis := part.piece.Axis
		if math.Abs(axis.Len()-1) > 1e-9 {
			t.Fatalf("an axis of %v is not a unit direction", axis)
		}
		across[fmtAxis(axis)] = true
	}
	if len(across) < 10 {
		t.Errorf("only %d of 12 tanks point their own way", len(across))
	}
}

// fmtAxis is an axis rounded to a tenth, as a key, so that two tanks that
// point the same way to a tenth are counted as one.
func fmtAxis(axis maths.Vec) string {
	return fmt.Sprint(math.Round(axis.X*10), math.Round(axis.Y*10),
		math.Round(axis.Z*10))
}

func TestATurnedPartsAxisTurnsWithIt(t *testing.T) {
	turn := maths.Rotation(maths.ZAxis, math.Pi/2)
	part, err := partOf("sls-core-tank", 1, turn, "hull")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(part.piece.Axis.X) < 0.999999 {
		t.Errorf("a tank turned a quarter turn about z has the axis %v",
			part.piece.Axis)
	}
	box := part.piece.Box
	if box.Max.X-box.Min.X < 60 || box.Max.Y-box.Min.Y > 10 {
		t.Errorf("its box is %v, want it long along x", box)
	}
}

func TestEveryPartIsCappedAtHullDetail(t *testing.T) {
	for _, name := range []string{"sls-core-tank", "winnebago", "yurt",
		"alfa-hero"} {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeWithComplexity(row, sow(1), "hull")
		if err != nil {
			t.Fatal(err)
		}
		skins := made.skins()
		for _, of := range made.slots() {
			look := made.dressed[of].Name
			if group := made.groups[of]; group != nil {
				for _, member := range group.Members() {
					if shut, ok := member.Mesh.(*mesh.Mesh); !ok || !shut.Closed() {
						t.Errorf("%s: a member of %q is not a closed mesh",
							name, look)
					}
				}
				continue
			}
			shut, err := skins[of].meshed()
			if err != nil || !shut.Closed() {
				t.Errorf("%s: %q is not a closed mesh: %v", name, look, err)
			}
		}
	}
}

func TestTheCoreTankIsTheColourOfTheSLSCoreStage(t *testing.T) {
	var core *thing.Material
	for _, look := range mustMake(t, "sls-block-1", 1).dressed {
		if look.Name == slsCoreSection {
			core = look
		}
	}
	if core == nil {
		t.Fatal("the SLS has no core stage look")
	}
	for _, look := range mustMake(t, slsCoreTankKind, 1).dressed {
		t.Logf("the tank is %v, gloss %v; the SLS core stage is %v, %v",
			look.Colour, look.Gloss, core.Colour, core.Gloss)
		if look.Colour != core.Colour || look.Gloss != core.Gloss {
			t.Errorf("the tank's look differs from the SLS core stage's")
		}
	}
}

// newDraws is a stream of draws from a seed.
func newDraws(number uint64) *rand.Rand {
	return rand.New(rand.NewPCG(number, 1))
}

func TestAClusterTakesOnlyKindsThatAreMadeAndWeighted(t *testing.T) {
	row, err := kindNamed(ClusterKind)
	if err != nil {
		t.Fatal(err)
	}
	for name, ask := range map[string]ClusterAsk{
		"a kind nobody has": {Parts: map[string]float64{"no-such-kind": 1}},
		"a cluster in it":   {Parts: map[string]float64{ClusterKind: 1}},
		"no weight":         {Parts: map[string]float64{"yurt": 0}},
		"a count below":     {Count: -1},
		"a spacing below":   {BandSpacing: -1},
	} {
		if _, err := Clustered(row, ask); err == nil {
			t.Errorf("a cluster of %s was taken", name)
		}
	}
	other, _ := kindNamed("yurt")
	if _, err := Clustered(other, theAsk); err == nil {
		t.Error("a yurt was given parts")
	}
	if _, err := Clustered(row, theAsk); err != nil {
		t.Errorf("the set's own cluster was refused: %v", err)
	}
}

func TestAPartOfAPartIsRefusedAndSoIsAnUnknownKind(t *testing.T) {
	turn := maths.Identity
	if _, err := partOf(ClusterKind, 1, turn, "hull"); err == nil {
		t.Error("a cluster was made a part of a cluster")
	}
	if _, err := partOf("no-such-kind", 1, turn, "hull"); err == nil {
		t.Error("a kind nobody has was made a part")
	}
	if _, err := partOf("yurt", 1, turn, "no such level"); err == nil {
		t.Error("a part was made at a level nobody names")
	}
}

func TestAPartEnclosesTheVolumeOfItsSolids(t *testing.T) {
	for name, between := range map[string][2]float64{
		"yurt": {40, 80}, "winnebago": {60, 90}, "alfa-hero": {1, 1e6},
		"sls-core-tank": {3000, 3600}} {
		part, err := partOf(name, 1, maths.Identity, "quick")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if part.piece.Volume < between[0] || part.piece.Volume > between[1] {
			t.Errorf("%s encloses %v m^3, want %v to %v", name,
				part.piece.Volume, between[0], between[1])
		}
	}
}

func TestAScavengedClusterNeedsASownSeed(t *testing.T) {
	if _, err := (scavengedRow{}).made(seed{}); !errors.Is(err, errGood) {
		t.Errorf("an unsown seed made a cluster: %v", err)
	}
}

func TestWithNoPartsTheCatalogueIsDrawnFromLessTheClusterItself(t *testing.T) {
	rng := newDraws(1)
	for range 50 {
		name := drawKind(ClusterAsk{}, rng)
		if name == ClusterKind {
			t.Fatal("a cluster drew itself")
		}
		if _, err := kindNamed(name); err != nil {
			t.Fatalf("drew %q: %v", name, err)
		}
	}
}

func TestOnlyALengthInTheSizeRangeFits(t *testing.T) {
	plan := cities.ClusterPlan{SizeLeast: 3, SizeMost: 120}
	for length, want := range map[float64]bool{2: false, 3: true, 50: true,
		120: true, 121: false} {
		if got := fitsSize(plan, length); got != want {
			t.Errorf("a part %v m long fits: %v, want %v", length, got, want)
		}
	}
}

// a good's length, taken face by face, is the longest side of the box of
// the faces it would keep.
func TestALengthIsTheLongestSideOfTheBoxOfTheFaces(t *testing.T) {
	for _, name := range []string{"yurt", "winnebago", "sls-core-tank"} {
		made, err := madePart(name, 1, "quick")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		faces, err := trianglesOf(made)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		box := cities.BoxOf(faces)
		_, want := longest(box.Max.Sub(box.Min))
		if got, err := longestSide(made); err != nil || got != want {
			t.Errorf("%s is %v m long face by face, %v by its box: %v",
				name, got, want, err)
		}
	}
}

// the faces a good is said to come to, walking none, are the faces it
// gives when every one is held: a yurt of plain surfaces, and a hubble
// with meshes that repeat.
func TestTheFacesAGoodComesToAreTheFacesItHolds(t *testing.T) {
	for _, name := range []string{"yurt", "hubble"} {
		made, err := madePart(name, 1, "quick")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		faces, err := trianglesOf(made)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := heldFaces(made); got != len(faces) || got == 0 {
			t.Errorf("%s comes to %d faces and holds %d", name, got,
				len(faces))
		}
	}
}

// a kind too heavy for a part is not drawn for one from the catalogue,
// and is still made when a set names it: a stand of kelp, whose fronds
// repeat past what a made thing's budgets are drawn up within.
func TestAKindTooHeavyForAPartIsDrawnAgain(t *testing.T) {
	const kelp = "giant-kelp-moderate"
	made, err := madePart(kelp, 1, "quick")
	if err != nil {
		t.Fatal(err)
	}
	if heldFaces(made) <= mostFaces {
		t.Fatalf("the kelp comes to %d faces, within %d: no heavy kind "+
			"is tried", heldFaces(made), mostFaces)
	}
	plan := cities.ClusterPlan{SizeLeast: 3, SizeMost: 120}
	_, err = fittingPart(plan, kelp, 1, maths.Identity, "quick")
	if !errors.Is(err, errHeavy) {
		t.Errorf("a heavy kind drawn from the catalogue: %v", err)
	}
	if _, err := fittingPart(plan, "yurt", 1, maths.Identity,
		"quick"); errors.Is(err, errHeavy) {
		t.Error("a yurt was too heavy for a part")
	}
}

// a kind drawn from the catalogue is measured before its faces are kept:
// one in the size range is the part partOf makes of it.
func TestAPartDrawnFromTheCatalogueThatFitsIsMade(t *testing.T) {
	plan := cities.ClusterPlan{SizeLeast: 3, SizeMost: 120}
	drawn, err := fittingPart(plan, "yurt", 1, maths.Identity, "quick")
	if err != nil {
		t.Fatal(err)
	}
	whole, err := partOf("yurt", 1, maths.Identity, "quick")
	if err != nil {
		t.Fatal(err)
	}
	if drawn.piece.Length != whole.piece.Length ||
		drawn.piece.Volume != whole.piece.Volume {
		t.Errorf("drawn %v m and %v m^3, made %v m and %v m^3",
			drawn.piece.Length, drawn.piece.Volume, whole.piece.Length,
			whole.piece.Volume)
	}
}

// and one outside the size range is said not to fit, whichever side of
// the range it is on, and a kind that cannot be made says why itself.
func TestAPartDrawnFromTheCatalogueThatDoesNotFitIsNotMade(t *testing.T) {
	for name, plan := range map[string]cities.ClusterPlan{
		"too small": {SizeLeast: 1000, SizeMost: 2000},
		"too large": {SizeLeast: 0.001, SizeMost: 0.002}} {
		_, err := fittingPart(plan, "yurt", 1, maths.Identity, "quick")
		if !errors.Is(err, errNoFit) {
			t.Errorf("a yurt %s for the range: %v", name, err)
		}
	}
	_, err := fittingPart(cities.ClusterPlan{SizeLeast: 3, SizeMost: 120},
		"no-such-kind", 1, maths.Identity, "quick")
	if err == nil || errors.Is(err, errNoFit) {
		t.Errorf("a kind nobody makes: %v", err)
	}
}
