package cities

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// signSeeds is how many seeds the sign tests look at, and makeSeeds how
// many the tests that want every make of a sign to turn up look at.
const (
	signSeeds = 24
	makeSeeds = 120
)

// signKinds are the four ways a sign hangs.
var signKinds = []MountKind{MountRoof, MountScreen, MountBlade, MountAwning}

// letterName is the name of the solid of one letter of a sign, and tubeName
// that of a tube bent to the edge of one ring of a letter.
const (
	letterName = "sign letter"
	tubeName   = "sign tube"
)

// lettersOf is how many of the drawn parts are letters of a sign, solid or
// tube.
func lettersOf(drawn []Rendered) int {
	return meshesNamed(drawn, letterName) + meshesNamed(drawn, tubeName)
}

// lampTolerance is how far the luminance of a sign's lamps, added up, may
// be from the sign's: only rounding.
const lampTolerance = 1e-9

// drawnSign is a sign of a kind on its own, drawn at a detail, or the
// test's end.
func drawnSign(t *testing.T, kind MountKind, seed Seed,
	level Detail) []Rendered {
	t.Helper()
	design, err := FreeSign(kind, seed, level)
	if err != nil {
		t.Fatalf("%s from seed %d: %v", kind, seed, err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatalf("%s from seed %d: %v", kind, seed, err)
	}
	return drawn
}

// meshesNamed is how many of the drawn parts have a mesh of the name.
func meshesNamed(drawn []Rendered, name string) int {
	var count int
	for _, one := range drawn {
		if one.Mesh.Name == name {
			count++
		}
	}
	return count
}

// lamps are the drawn parts that are lights.
func lamps(drawn []Rendered) []Rendered {
	var found []Rendered
	for _, one := range drawn {
		if one.Lamp != nil {
			found = append(found, one)
		}
	}
	return found
}

// boxOf is a design of one box of masses, a building of a size standing on
// the ground at the origin.
func boxOf(wide, high, deep float64) Design {
	return Design{Name: "block", Massing: []mass{{Name: "block",
		Form:  Box{wide, high, deep},
		Place: standingAt(maths.V(0, 0, 0)), Role: roleBody}}}
}

func TestAFreeSignOfEachKindHasLettersAndLampsOfItsColour(t *testing.T) {
	for _, kind := range signKinds {
		for seed := range Seed(signSeeds) {
			drawn := drawnSign(t, kind, seed, "quick")
			if lettersOf(drawn) == 0 {
				t.Errorf("%s from seed %d has no letters", kind, seed)
			}
			lit := lamps(drawn)
			if len(lit) == 0 {
				t.Errorf("%s from seed %d has no lamp", kind, seed)
			}
			for _, one := range lit {
				if one.Lamp.Tint == (maths.Light{}) ||
					!(one.Lamp.Luminance > 0) {
					t.Errorf("%s from seed %d: a lamp of %+v", kind, seed,
						*one.Lamp)
				}
				glow, tint := one.Material.Glow, one.Lamp.Tint
				if math.Abs(glow.R*tint.G-glow.G*tint.R) > lampTolerance ||
					math.Abs(glow.G*tint.B-glow.B*tint.G) > lampTolerance {
					t.Errorf("%s from seed %d: a lamp of %v in glass that "+
						"glows %v", kind, seed, tint, glow)
				}
			}
		}
	}
}

func TestAFarSignIsAGlowingPanelWithNoLettersAndStillLightsTheStreet(t *testing.T) {
	for _, level := range []Detail{hullLevel, clumpsLevel} {
		for _, kind := range signKinds {
			drawn := drawnSign(t, kind, 3, level)
			if got := lettersOf(drawn); got != 0 {
				t.Errorf("%s at %s has %d letters", kind, level, got)
			}
			if len(lamps(drawn)) == 0 {
				t.Errorf("%s at %s throws no light", kind, level)
			}
		}
	}
}

func TestASignOfAKindThatIsNoMountIsRefusedAndSoIsNoLevel(t *testing.T) {
	if _, err := FreeSign("banner", 1, "quick"); err == nil {
		t.Error("a sign of kind banner was made")
	}
	if _, err := signs(boxOf(60, 80, 40), 1, nil); err == nil {
		t.Error("signs were drawn at no detail")
	}
}

func TestTheLampsOfASignShareItsLuminanceAndTakeItsColour(t *testing.T) {
	drawing := drawerOf(t, signCulture, 1)
	for seed := range Seed(signSeeds) {
		for _, kind := range signKinds {
			design, err := FreeSign(kind, seed, "quick")
			if err != nil {
				t.Fatal(err)
			}
			name := "sign-" + string(kind) + "-luminance"
			if strings.HasSuffix(design.Name, lookLamped) {
				name = "sign-lamped-luminance"
			}
			drawn, err := design.Rendered("quick")
			if err != nil {
				t.Fatal(err)
			}
			var total float64
			burning := lamps(drawn)
			for _, one := range burning {
				total += one.Lamp.Luminance
			}
			// the sign's luminance is the sign's whatever lamps are out
			if want := drawing.fixed(name); len(burning) == 0 ||
				math.Abs(total-want) > lampTolerance {
				t.Errorf("%s from seed %d: %d lamps add to %v, want %v",
					design.Name, seed, len(burning), total, want)
			}
		}
	}
}

// An awning of one lamp keeps it lit, at every seed; one of several may
// lose one lamp and no more. The seeds must give both sorts, or the test
// asks nothing.
func TestAnAwningOfOneLampKeepsItAndOneOfSeveralLosesAtMostOne(t *testing.T) {
	var alone, several int
	for seed := range Seed(signSeeds) {
		drawn := drawnSign(t, MountAwning, seed, "quick")
		heads, burning := meshesNamed(drawn, "sign lamp"), len(lamps(drawn))
		if burning == 0 {
			t.Errorf("awning from seed %d has no lamp burning", seed)
		}
		switch {
		case heads == 1:
			alone++
			if burning != 1 {
				t.Errorf("awning from seed %d: its one lamp is out", seed)
			}
		case heads >= leastLampsToSpare:
			several++
			if heads-burning > 1 {
				t.Errorf("awning from seed %d: %d of %d lamps are out",
					seed, heads-burning, heads)
			}
		}
	}
	if alone == 0 || several == 0 {
		t.Errorf("the seeds give %d awnings of one lamp and %d of several",
			alone, several)
	}
}

func TestABladeSignHasOneLetterForEachLetterOfItsWordOnBothFaces(t *testing.T) {
	for seed := range Seed(signSeeds) {
		signing, err := newSigner(seed, ConstantLevel("quick"))
		if err != nil {
			t.Fatal(err)
		}
		mount := signMount{Kind: MountBlade, Out: maths.ZAxis,
			Along: maths.XAxis, High: signing.drawing.fixed("sign-free-high"),
			Reach: signing.drawing.fixed("sign-blade-reach")}
		job := signing.job(mount, 0)
		layers := len(job.layers(1, option{Name: "x",
			Colour: []float64{1, 1, 1}}))
		parts := signing.hang(mount, 0)
		word := strings.ReplaceAll(signing.used[0], " ", "")
		got, tubes := 0, 0
		for _, part := range parts {
			if solid, ok := part.Element.(solid); ok && solid.Name == letterName {
				got++
			} else if ok && solid.Name == tubeName {
				tubes++
			}
		}
		if job.look == lookTube {
			if tubes < 2*len([]rune(word)) {
				t.Errorf("seed %d: %q has %d tubes, want at least two a "+
					"letter", seed, word, tubes)
			}
			continue
		}
		if want := 2 * layers * len([]rune(word)); got != want {
			t.Errorf("seed %d: %q has %d letter solids, want %d", seed, word,
				got, want)
		}
	}
}

func TestNoTwoSignsOfABuildingSayTheSameWords(t *testing.T) {
	design := boxOf(80, 120, 60)
	mounts, err := design.SignMounts()
	if err != nil {
		t.Fatal(err)
	}
	for seed := range Seed(signSeeds) {
		signing, err := newSigner(seed, ConstantLevel("quick"))
		if err != nil {
			t.Fatal(err)
		}
		for index, mount := range mounts {
			signing.hang(mount, index)
		}
		seen := map[string]bool{}
		for _, words := range signing.used {
			if seen[words] {
				t.Errorf("seed %d: %q is said twice", seed, words)
			}
			seen[words] = true
		}
		if signing.drawing.trouble != nil {
			t.Fatal(signing.drawing.trouble)
		}
	}
}

func TestTubeLettersAndChannelLettersAreMadeDifferently(t *testing.T) {
	tube, channel := false, false
	for seed := range Seed(makeSeeds) {
		design, err := FreeSign(MountRoof, seed, "quick")
		if err != nil {
			t.Fatal(err)
		}
		drawn, err := design.Rendered("quick")
		if err != nil {
			t.Fatal(err)
		}
		tubes, solids := meshesNamed(drawn, tubeName),
			meshesNamed(drawn, letterName)
		switch {
		case strings.HasSuffix(design.Name, lookTube):
			tube = tube || tubes > 0
			if solids > 0 {
				t.Errorf("seed %d: tube letters with %d solid letters", seed,
					solids)
			}
		case strings.HasSuffix(design.Name, lookChannel):
			channel = channel || solids > 0
			if tubes > 0 {
				t.Errorf("seed %d: channel letters with %d tubes", seed, tubes)
			}
		}
	}
	if !tube || !channel {
		t.Errorf("in %d seeds, tube %v and channel %v were drawn", makeSeeds,
			tube, channel)
	}
}

func TestALetterTheBevelCannotTakeIsSetFlatAndAGlyphTheFontLacksIsRefused(t *testing.T) {
	signing, err := newSigner(1, ConstantLevel("quick"))
	if err != nil {
		t.Fatal(err)
	}
	mount := signMount{Kind: MountRoof, Wide: 40, High: 14}
	job := signing.job(mount, 0).tubed()
	made, err := job.ink("KARAOKE", 2, maths.V(0, 0, 0), job.neon, -1)
	if err != nil || len(made) != len("KARAOKE") {
		t.Errorf("KARAOKE as tube neon: %d letters, %v", len(made), err)
	}
	if _, err := job.ink("拉面", 2, maths.V(0, 0, 0), job.neon,
		-1); err == nil {
		t.Error("letters the font does not have were set")
	}
}

func TestABoxOffersAllThreeKindsOfPlaceAndAThinOneOffersNone(t *testing.T) {
	mounts, err := boxOf(60, 80, 40).SignMounts()
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[MountKind]int{}
	for _, mount := range mounts {
		kinds[mount.Kind]++
		if math.Abs(mount.Out.Len()-1) > heightTol || mount.Out.Y != 0 {
			t.Errorf("a %s mount faces %v, not a horizontal unit", mount.Kind,
				mount.Out)
		}
	}
	for _, kind := range signKinds {
		if kinds[kind] == 0 {
			t.Errorf("a box offers no %s place", kind)
		}
	}
	thin, err := boxOf(4, 80, 4).SignMounts()
	if err != nil || len(thin) != 0 {
		t.Errorf("a box 4 m across offers %d places, %v", len(thin), err)
	}
}

func TestAPlaceAnotherBoxCoversIsNotOffered(t *testing.T) {
	alone := boxOf(40, 60, 40)
	joined := alone
	nextPlace := standingAt(maths.V(40, 0, 0))
	joined.Massing = append(append([]mass{}, alone.Massing...), mass{
		Name: "next", Form: Box{40, 60, 40}, Role: roleBody,
		Place: nextPlace})
	one, err := alone.SignMounts()
	if err != nil {
		t.Fatal(err)
	}
	two, err := joined.SignMounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(two) >= 2*len(one) {
		t.Errorf("two boxes side by side offer %d places, one offers %d: "+
			"the wall they share is offered", len(two), len(one))
	}
}

func TestATowerOffersPlacesAndDrawsSignsOnThem(t *testing.T) {
	var offered, drawn int
	for seed := range Seed(signSeeds) {
		tower, err := NewTower(seed, Irregularity{Value: 0.3}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		mounts, err := tower.SignMounts()
		if err != nil {
			t.Fatal(err)
		}
		parts, err := signs(tower, seed, ConstantLevel("quick"))
		if err != nil {
			t.Fatal(err)
		}
		offered, drawn = offered+len(mounts), drawn+len(parts)
	}
	if offered == 0 || drawn == 0 {
		t.Errorf("%d towers offered %d places and drew %d parts", signSeeds,
			offered, drawn)
	}
}
