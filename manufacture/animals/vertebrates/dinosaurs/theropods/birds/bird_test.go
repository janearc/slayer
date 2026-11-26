package birds

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

var birdRead, birdError = readBirds(birdRaw)

func readBirds(raw []byte) (birdFile, error) {
	book, err := Read("birds.yaml", raw)
	if err != nil {
		return birdFile{}, err
	}
	return book.file, nil
}

func birdSpeciesNamed(key string) (birdSpecies, error) {
	if birdError != nil {
		return birdSpecies{}, birdError
	}
	species, ok := birdRead.Species[key]
	if !ok {
		return birdSpecies{}, fmt.Errorf("no such bird: %s", key)
	}
	return species, nil
}

func birdThings(key, stance string, from seed, at maths.Subspace) ([]thing.Thing, maths.Box, error) {
	species, err := birdSpeciesNamed(key)
	if err != nil {
		return nil, maths.Box{}, err
	}
	return (Specimen{profile: species, stance: stance}).Build(from, at)
}

// birdTestSeed is the seed the bird tests grow from.
const birdTestSeed = 7

// birdTestBody is the species named key laid out from the test seed.
func birdTestBody(t *testing.T, key string) *birdBody {
	t.Helper()
	species, err := birdSpeciesNamed(key)
	if err != nil {
		t.Fatal(err)
	}
	from := sow(birdTestSeed)
	measures, err := measured(species, from.draw(picking, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	body, err := newBirdBody(species, measures, from)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestTheBirdsFileReads(t *testing.T) {
	if birdError != nil {
		t.Fatal(birdError)
	}
	if len(birdRead.Species) == 0 {
		t.Fatal("no species")
	}
}

func TestABirdHasItsRowsCounts(t *testing.T) {
	for _, key := range birdRead.keys() {
		body := birdTestBody(t, key)
		m := body.measures
		for prefix, want := range map[string]int{"C": m.cervicals,
			"p": 2 * m.primaries, "s": 2 * m.secondaries,
			birdRectrix: m.rectrices, "toe": 2 * (2 + 3 + 4 + 5)} {
			got := 0
			for i := 0; i < body.bones.Len(); i++ {
				name := body.bones.Bone(i).Name()
				region, _ := birdVertebraOf(name)
				if prefix == "C" && region == "C" ||
					prefix != "C" && birdNumbered(name, prefix) {
					got++
				}
			}
			if got != want {
				t.Errorf("%s: %d bones of %s, want %d", key, got, prefix, want)
			}
		}
	}
}

// birdNumbered is whether name is prefix followed by a digit.
func birdNumbered(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	return ok && rest != "" && rest[0] >= '0' && rest[0] <= '9'
}

func TestABirdHasSeveralHundredParts(t *testing.T) {
	const least = 300 // the bar for a made thing here
	for _, key := range birdRead.keys() {
		body := birdTestBody(t, key)
		if got := len(body.skinPieces()) + 1; got < least {
			t.Errorf("%s: %d parts, fewer than %d", key, got, least)
		}
	}
}

func TestAStandingBirdStandsOnItsToes(t *testing.T) {
	for _, key := range birdRead.keys() {
		parts, box, err := birdThings(key, Standing, sow(birdTestSeed),
			maths.Local())
		if err != nil || len(parts) == 0 {
			t.Fatalf("%s: %v", key, err)
		}
		if math.Abs(box.Min.Y) > .01 {
			t.Errorf("%s: lowest point at %.3f m, not on the ground",
				key, box.Min.Y)
		}
	}
}

func TestAGlidingBirdSpansItsWingspan(t *testing.T) {
	const near = .2 // the share the span may miss by; the bend takes some
	for _, key := range birdRead.keys() {
		body := birdTestBody(t, key)
		pieces := body.skinPieces()
		frames := body.bones.Frames(body.gliding())
		_, box, err := body.outlined(pieces,
			frames, maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		span := box.Max.Z - box.Min.Z
		want := body.measures.wingspan
		if math.Abs(span-want) > near*want {
			t.Errorf("%s: spans %.2f m, wingspan %.2f m", key, span, want)
		}
	}
}

func TestAStandingBirdsWingsAreFolded(t *testing.T) {
	for _, key := range birdRead.keys() {
		body := birdTestBody(t, key)
		pieces := body.skinPieces()
		frames := body.bones.Frames(body.standing())
		_, box, err := body.outlined(pieces,
			frames, maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		if span := box.Max.Z - box.Min.Z; span > 2*body.measures.width {
			t.Errorf("%s: %.2f m across, wings not folded", key, span)
		}
	}
}

func TestBirdsOfTwoSeedsDiffer(t *testing.T) {
	species, err := birdSpeciesNamed("grey-heron")
	if err != nil {
		t.Fatal(err)
	}
	one, _ := measured(species, sow(1).draw(picking, 0, 0))
	two, _ := measured(species, sow(2).draw(picking, 0, 0))
	again, _ := measured(species, sow(1).draw(picking, 0, 0))
	if one.length == two.length || one.length != again.length {
		t.Error("the seed does not pick the size, or picks it twice apart")
	}
}

// Anti-tests: what the reader and the maker refuse.

func TestARowWithoutABasisIsRefused(t *testing.T) {
	if err := (birdFigure{Least: 1, Most: 1}).checked(); err == nil {
		t.Error("a figure with no basis was taken")
	}
}

func TestAReadFigureWithoutASourceIsRefused(t *testing.T) {
	figure := birdFigure{Least: 1, Most: 1, Basis: birdReadBasis}
	if err := figure.checked(); err == nil {
		t.Error("a read figure with no source was taken")
	}
}

func TestARangeTheWrongWayRoundIsRefused(t *testing.T) {
	figure := birdFigure{Least: 2, Most: 1, Basis: birdPickedBasis}
	if err := figure.checked(); err == nil {
		t.Error("a range whose least is above its most was taken")
	}
}

func TestASpeciesNotNativeAnywhereIsRefused(t *testing.T) {
	species := birdRead.Species["grey-heron"]
	species.Native = birdNative{}
	if err := species.checked(); err == nil {
		t.Error("a species with no source for where it is native was taken")
	}
}

func TestAFileWithNoSpeciesIsRefused(t *testing.T) {
	if _, err := readBirds([]byte("read: somewhere\n")); err == nil {
		t.Error("a file with no species was taken")
	}
}

func TestNoSuchBirdIsRefused(t *testing.T) {
	if _, err := birdSpeciesNamed("dodo"); err == nil {
		t.Error("a species not in the file was made")
	}
}

func TestSharesThatLeaveNoTrunkAreRefused(t *testing.T) {
	species := birdRead.Species["grey-heron"]
	figures := map[string]birdFigure{}
	for name, figure := range species.Figures {
		figures[name] = figure
	}
	figures["neck"] = birdFigure{Least: .9, Most: .9, Basis: birdPickedBasis}
	species.Figures = figures
	if _, err := measured(species, 0); err == nil {
		t.Error("a bird with no trunk was measured")
	}
}

func TestAForkedTailIsLongestOutside(t *testing.T) {
	body := birdTestBody(t, "great-frigatebird")
	length := func(name string) float64 {
		at, _ := body.bones.Find(name)
		return body.bones.Bone(at).Length()
	}
	last := body.measures.rectrices
	if length("rectrix1") <= length(fmt.Sprintf("rectrix%d", last/2)) {
		t.Error("the frigatebird's outer tail feather is not the longest")
	}
}

func TestStreamersAreTheMiddlePair(t *testing.T) {
	body := birdTestBody(t, "white-tailed-tropicbird")
	at, _ := body.bones.Find(fmt.Sprintf("rectrix%d",
		body.measures.rectrices/2))
	middle := body.bones.Bone(at).Length()
	at, _ = body.bones.Find("rectrix1")
	if outer := body.bones.Bone(at).Length(); middle < 2*outer {
		t.Errorf("streamer %.2f m, outer feather %.2f m", middle, outer)
	}
}

func TestOnlyAWebbedRowHasWebs(t *testing.T) {
	for key, want := range map[string]bool{"great-frigatebird": true,
		"grey-heron": false} {
		if got := len(birdTestBody(t, key).webPieces()) > 0; got != want {
			t.Errorf("%s: webs %v, want %v", key, got, want)
		}
	}
}

func TestATailOfANoughtFeatherIsRefused(t *testing.T) {
	tail := birdTail{Lengths: []float64{1, 0}, Basis: birdPickedBasis}
	if err := tail.checked(); err == nil {
		t.Error("a tail with a feather of no length was taken")
	}
}
