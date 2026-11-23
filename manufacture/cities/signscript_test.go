package cities

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/typography"
)

// shelfFile is the ending of a font file on the source shelf.
const shelfFile = ".otf"

// inkTolerance is how far the ink of a character may stray from the line
// it is set by, as a share of the em: only rounding.
const inkTolerance = 1e-6

// traditionalCharacters are characters that only the traditional script
// writes, in the forms of the words the simplified lists use. None may be
// in a word of the simplified script.
const traditionalCharacters = "藥麵館網廳機髮發時營業歡臨優買費熱餃餛飩釀" +
	"雞賓膠診門遊戲將婦護線蘭電鋪點"

// herTrades are the two trades a set asks for, which no weight draws.
var herTrades = []string{"womens-shelter", "dog-boutique"}

// herWords are what her trades say in simplified Chinese, then in Latin.
var herWords = []string{"妇女庇护所", "狗狗精品店", "WOMENS SHELTER",
	"DOG BOUTIQUE"}

// shelfFaces are the files of the faces of the simplified script, as the
// shelf SLAYER_SHELF names keeps them, or the test's skip when the shelf
// lacks them.
func shelfFaces(t *testing.T) map[string]string {
	t.Helper()
	keys, err := ScriptFaces(scriptSimplifiedChinese)
	if err != nil {
		t.Fatal(err)
	}
	faces := map[string]string{}
	for _, key := range keys {
		path := filepath.Join(os.Getenv("SLAYER_SHELF"), key+shelfFile)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("the shelf has no %s: %v", key, err)
		}
		faces[key] = path
	}
	return faces
}

// wordsOfScript are every word the sign culture says in a script's
// suffix, by the name of the element that says it.
func wordsOfScript(t *testing.T, suffix string) map[string][]string {
	t.Helper()
	drawing := drawerOf(t, signCulture, 1)
	found := map[string][]string{}
	for name, each := range drawing.culture.elements {
		if !strings.HasSuffix(name, suffix) || (!strings.HasPrefix(name,
			"sign-word-") && !strings.HasPrefix(name, "sign-short-") &&
			name != "sign-screen-line"+suffix) {
			continue
		}
		for _, one := range each.Options {
			found[name] = append(found[name], one.Name)
		}
	}
	return found
}

func TestAZeroLetteringIsTheSignsSignsDraws(t *testing.T) {
	for seed := range Seed(signSeeds) {
		tower, err := NewTower(seed, Irregularity{Value: 0.3}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		plain, err := signs(tower, seed, ConstantLevel("quick"))
		if err != nil {
			t.Fatal(err)
		}
		for _, lettering := range []Lettering{{},
			{Script: scriptLatin}} {
			again, err := SignsIn(tower, seed, ConstantLevel("quick"),
				lettering)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plain, again) {
				t.Errorf("seed %d: lettering %+v changed the signs", seed,
					lettering)
			}
		}
	}
}

func TestNoWeightDrawsHerTradesInAnyScript(t *testing.T) {
	drawing := drawerOf(t, signCulture, 1)
	for _, name := range []string{"sign-trade", "sign-trade-hans"} {
		for _, one := range drawing.options(name) {
			if slices.Contains(herTrades, one.Name) {
				t.Errorf("%s draws %s by weight", name, one.Name)
			}
		}
	}
}

func TestEveryTradeOfEveryScriptHasLongAndShortWordsAndEveryTradeHasBoth(t *testing.T) {
	drawing := drawerOf(t, signCulture, 1)
	for _, suffix := range scriptSuffix {
		trades := drawing.options("sign-trade" + suffix)
		for _, one := range trades {
			for _, kind := range []string{"word", "short"} {
				name := "sign-" + kind + "-" + one.Name + suffix
				if len(drawing.options(name)) == 0 {
					t.Errorf("trade %s has no %s words in %q", one.Name,
						kind, suffix)
				}
			}
		}
		for _, trade := range herTrades {
			if got := carryable(drawing, suffix); !slices.Contains(got,
				trade) {
				t.Errorf("%q has no trade %s among %v", suffix, trade, got)
			}
		}
	}
	latin := drawing.options("sign-trade")
	chinese := drawing.options("sign-trade-hans")
	if len(latin) != len(chinese) {
		t.Fatalf("%d Latin trades and %d simplified ones", len(latin),
			len(chinese))
	}
	for index := range latin {
		if latin[index].Name != chinese[index].Name ||
			latin[index].Weight != chinese[index].Weight {
			t.Errorf("trade %d is %+v in Latin and %+v in simplified "+
				"Chinese", index, latin[index], chinese[index])
		}
	}
}

func TestNoWordOfTheSimplifiedScriptIsTraditionalOrNotAHanCharacter(t *testing.T) {
	for name, words := range wordsOfScript(t, "-hans") {
		for _, word := range words {
			for _, letter := range word {
				if strings.ContainsRune(traditionalCharacters, letter) {
					t.Errorf("%s: %q has the traditional %q", name, word,
						letter)
				}
				plain := letter < unicode.MaxASCII &&
					(unicode.IsLetter(letter) || unicode.IsDigit(letter))
				if !plain && !unicode.Is(unicode.Han, letter) {
					t.Errorf("%s: %q has %q, which is neither a character "+
						"nor a plain letter or digit", name, word, letter)
				}
			}
		}
	}
}

func TestEverySignOfTheSimplifiedScriptSetsWholeInOneFaceTheCultureNames(t *testing.T) {
	faces := shelfFaces(t)
	signing, err := newLetteredSigner(1, ConstantLevel("quick"),
		Lettering{Script: scriptSimplifiedChinese, Faces: faces})
	if err != nil {
		t.Fatal(err)
	}
	names, err := ScriptFaces(scriptSimplifiedChinese)
	if err != nil {
		t.Fatal(err)
	}
	words := wordsOfScript(t, "-hans")
	for name, said := range words {
		for _, word := range said {
			if !slices.ContainsFunc(names, func(face string) bool {
				return signing.sets(face, word)
			}) {
				t.Errorf("%s: no face draws every letter of %q", name, word)
			}
		}
	}
	// the words one sign says together: a trade's long word and short
	// word, and the lines of a screen, up to the most a screen carries
	drawing := signing.drawing
	for _, trade := range drawing.options("sign-trade-hans") {
		for _, long := range words["sign-word-"+trade.Name+"-hans"] {
			for _, short := range words["sign-short-"+trade.Name+"-hans"] {
				together := long + short
				if !slices.ContainsFunc(names, func(face string) bool {
					return signing.sets(face, together)
				}) {
					t.Errorf("trade %s: no face draws %q and %q together",
						trade.Name, long, short)
				}
			}
		}
	}
	lines := words["sign-screen-line-hans"]
	for _, first := range lines {
		for _, second := range lines {
			if !slices.ContainsFunc(names, func(face string) bool {
				return signing.sets(face, first+second)
			}) {
				t.Errorf("no face draws the screen lines %q and %q together",
					first, second)
			}
		}
	}
}

func TestACharacterIsSquareAndItsInkStandsOnTheLineItIsSetBy(t *testing.T) {
	faces := shelfFaces(t)
	drawing := drawerOf(t, signCulture, 1)
	rise, tall := drawing.fixed("sign-rise-hans"),
		drawing.fixed("sign-cap-share-hans")
	for key, path := range faces {
		face, err := typography.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		for name, words := range wordsOfScript(t, "-hans") {
			for _, word := range words {
				for _, letter := range word {
					if !unicode.Is(unicode.Han, letter) {
						continue
					}
					glyph, err := face.Index(letter)
					if err != nil {
						t.Errorf("%s: %q: %v", key, letter, err)
						continue
					}
					if advance, err := face.Advance(glyph); err != nil ||
						advance != face.Units() {
						t.Errorf("%s: %q advances %v of an em of %v: %v",
							key, letter, advance, face.Units(), err)
					}
					rings, err := typography.Outlines(face,
						typography.Text{Words: string(letter), Size: 1,
							Tolerance: 10})
					if err != nil {
						continue
					}
					for _, round := range rings[0].Rings {
						for _, point := range round {
							if point.Y+rise < -inkTolerance ||
								point.Y+rise > tall+inkTolerance {
								t.Errorf("%s %s: %q has ink at %v, set "+
									"%v up, outside 0 to %v", key, name,
									letter, point.Y, rise, tall)
							}
						}
					}
				}
			}
		}
	}
}

// The Latin half needs no file; the Chinese half skips by itself where the
// shelf lacks the faces, and does not take the Latin half with it.
func TestEveryBuildingCarriesEachTradeItIsAskedToOnce(t *testing.T) {
	for _, script := range []Script{scriptSimplifiedChinese, scriptLatin} {
		t.Run(string(script)+"script", func(t *testing.T) {
			lettering := Lettering{Script: script, Carries: herTrades}
			wants := herWords[2:]
			if script == scriptSimplifiedChinese {
				lettering.Faces, wants = shelfFaces(t), herWords[:2]
			}
			everyBuildingCarriesEachTradeOnce(t, lettering, wants)
		})
	}
}

// everyBuildingCarriesEachTradeOnce draws a tower at each seed that has
// places for the trades and holds that each word the trades say is said
// once in all of it.
func everyBuildingCarriesEachTradeOnce(t *testing.T, lettering Lettering,
	wants []string) {
	t.Helper()
	for seed := range Seed(signSeeds) {
		tower, err := NewTower(seed, Irregularity{Value: 0.3}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		mounts, err := tower.SignMounts()
		if err != nil {
			t.Fatal(err)
		}
		if len(owing(mounts, herTrades)) < len(herTrades) {
			continue
		}
		_, said, err := signsSaying(tower, seed, ConstantLevel("quick"),
			lettering)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			count := 0
			for _, one := range said {
				if one == want {
					count++
				}
			}
			if count != 1 {
				t.Errorf("seed %d, %q: %d signs say %q among %v", seed,
					lettering.Script, count, want, said)
			}
		}
	}
}

// A seed draws the same signs whatever was drawn before it: the words said
// at a seed, drawn again after other seeds and the other script, are the
// same. A draw that kept something between draws would differ.
func TestASeedSaysTheSameWordsInAnyCompany(t *testing.T) {
	draw := func(seed Seed, lettering Lettering) []string {
		tower, err := NewTower(seed, Irregularity{Value: 0.3}, Pins{})
		if err != nil {
			t.Fatal(err)
		}
		_, said, err := signsSaying(tower, seed, ConstantLevel("quick"),
			lettering)
		if err != nil {
			t.Fatal(err)
		}
		return said
	}
	latin := Lettering{Carries: herTrades}
	for seed := range Seed(signSeeds) {
		first := draw(seed, latin)
		for other := range Seed(signSeeds) {
			if other != seed {
				draw(other, latin)
			}
		}
		draw(seed, Lettering{})
		if again := draw(seed, latin); !slices.Equal(first, again) {
			t.Errorf("seed %d said %v, and after others %v", seed, first,
				again)
		}
	}
}

func TestCarriedTradesAreGivenBladesThenAwningsThenRoofs(t *testing.T) {
	mounts := []signMount{{Kind: MountRoof}, {Kind: MountScreen},
		{Kind: MountBlade}, {Kind: MountAwning}, {Kind: MountBlade},
		{Kind: MountRoof}}
	got := owing(mounts, []string{"a", "b", "c", "d", "e", "f"})
	want := map[int]string{2: "a", 4: "b", 3: "c", 0: "d", 5: "e"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("places %v, want %v: a screen reads no trade, and the "+
			"sixth trade has no place", got, want)
	}
	if got := owing(mounts, nil); len(got) != 0 {
		t.Errorf("trades %v were given places when none were carried", got)
	}
}

func TestALetteringIsRefusedForWhatTheCultureDoesNotHave(t *testing.T) {
	for _, lettering := range []Lettering{
		{Script: "klingon"},
		{Carries: []string{"bank"}},
		{Script: scriptSimplifiedChinese, Carries: []string{"pachinko"}},
		{Carries: []string{"ramen", "ramen"}},
		{Carries: []string{"womens-shelter-hans"}}} {
		if err := lettering.Check(); err == nil {
			t.Errorf("lettering %+v was kept", lettering)
		}
	}
	for _, lettering := range []Lettering{{}, {Script: scriptLatin},
		{Script: scriptSimplifiedChinese},
		{Script: scriptSimplifiedChinese, Carries: herTrades},
		{Carries: herTrades}} {
		if err := lettering.Check(); err != nil {
			t.Errorf("lettering %+v: %v", lettering, err)
		}
	}
	if _, err := ScriptFaces("klingon"); err == nil {
		t.Error("the faces of a script there is not were named")
	}
}

func TestASignOnItsOwnCarriesOneTradeAndNotOnAScreen(t *testing.T) {
	if _, err := FreeSignIn(MountScreen, 1, "quick",
		Lettering{Carries: herTrades[:1]}); err == nil {
		t.Error("a screen carried a trade")
	}
	if _, err := FreeSignIn(MountBlade, 1, "quick",
		Lettering{Carries: herTrades}); err == nil {
		t.Error("a blade sign carried two trades")
	}
	for _, kind := range tradeMounts {
		design, err := FreeSignIn(kind, 1, "quick",
			Lettering{Carries: herTrades[:1]})
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if _, err := design.Rendered("quick"); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
	}
}

func TestAScriptWithNoFileForItsFaceIsRefusedNamingTheFace(t *testing.T) {
	_, err := FreeSignIn(MountBlade, 1, "quick",
		Lettering{Script: scriptSimplifiedChinese})
	if err == nil || !strings.Contains(err.Error(), "noto-") {
		t.Errorf("a script with no file was drawn, or refused without "+
			"naming its face: %v", err)
	}
}

func TestAFreeSignOfEachKindIsDrawnInSimplifiedChineseWithLettersAndLamps(t *testing.T) {
	faces := shelfFaces(t)
	for _, kind := range signKinds {
		for seed := range Seed(signSeeds) {
			design, err := FreeSignIn(kind, seed, "quick", Lettering{
				Script: scriptSimplifiedChinese, Faces: faces})
			if err != nil {
				t.Fatalf("%s from seed %d: %v", kind, seed, err)
			}
			drawn, err := design.Rendered("quick")
			if err != nil {
				t.Fatalf("%s from seed %d: %v", kind, seed, err)
			}
			if lettersOf(drawn) == 0 || len(lamps(drawn)) == 0 {
				lampCount := len(lamps(drawn))
				t.Errorf("%s from seed %d has %d letters and %d lamps",
					kind, seed, lettersOf(drawn), lampCount)
			}
		}
	}
}

func TestASignKeepsItsTradeInEveryScriptThatHasTheTrade(t *testing.T) {
	faces := shelfFaces(t)
	for seed := range Seed(signSeeds) {
		latin, err := newLetteredSigner(seed, ConstantLevel("quick"),
			Lettering{})
		if err != nil {
			t.Fatal(err)
		}
		simplified, err := newLetteredSigner(seed, ConstantLevel("quick"),
			Lettering{Script: scriptSimplifiedChinese, Faces: faces})
		if err != nil {
			t.Fatal(err)
		}
		at := place{Opening: 0}
		if one, other := latin.trade(at, 0), simplified.trade(at,
			0); one.Name != other.Name {
			t.Errorf("seed %d: a sign is for %s in Latin and %s in "+
				"simplified Chinese", seed, one.Name, other.Name)
		}
	}
}

func TestAHansCharacterSignStandsWithTheRiseOfItsScriptAndALatinOneDoesNot(t *testing.T) {
	faces := shelfFaces(t)
	latin, err := newSigner(1, ConstantLevel("quick"))
	if err != nil {
		t.Fatal(err)
	}
	simplified, err := newLetteredSigner(1, ConstantLevel("quick"),
		Lettering{Script: scriptSimplifiedChinese, Faces: faces})
	if err != nil {
		t.Fatal(err)
	}
	mount := signMount{Kind: MountRoof, Wide: 40, High: 14,
		Out: maths.ZAxis, Along: maths.XAxis}
	if got := latin.job(mount, 0).rise; got != 0 {
		t.Errorf("a Latin sign is set %v up", got)
	}
	if got := simplified.job(mount, 0).rise; !(math.Abs(got-0.1) <
		inkTolerance) {
		t.Errorf("a simplified Chinese sign is set %v up, not a tenth", got)
	}
}
