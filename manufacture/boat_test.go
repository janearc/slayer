package manufacture

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// boatNamesAll is every boat kind, under way and at anchor.
func boatNamesAll() []string {
	var names []string
	for _, design := range boatDesigns {
		names = append(names, design, design+boatAtAnchor)
	}
	return names
}

// lampNames is the names of the lights a made boat carries.
func lampNames(made good) []string {
	var names []string
	for _, one := range made.lights {
		names = append(names, one.name)
	}
	return names
}

func TestTheBoatsDataIsReadAndSound(t *testing.T) {
	t.Parallel()
	if boatError != nil {
		t.Fatal(boatError)
	}
	for _, name := range boatNamesAll() {
		if _, err := kindNamed(name); err != nil {
			t.Errorf("%s is not a kind: %v", name, err)
		}
	}
}

func TestEveryBoatIsMadeAtEveryDetailAndMeshes(t *testing.T) {
	t.Parallel()
	for _, name := range boatNamesAll() {
		row, err := kindNamed(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, detail := range Complexities() {
			for number := range uint64(4) {
				made, err := MakeWithComplexity(row, sow(number), detail)
				if err != nil {
					t.Fatalf("%s at %s from %d: %v", name, detail, number, err)
				}
				if _, err := made.Things(maths.Local()); err != nil {
					t.Errorf("%s at %s from %d does not mesh: %v", name,
						detail, number, err)
				}
			}
		}
	}
}

func TestMoreDetailIsMoreFaces(t *testing.T) {
	t.Parallel()
	for _, name := range boatNamesAll() {
		row, _ := kindNamed(name)
		last := 0
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, sow(3), detail)
			if err != nil {
				t.Fatal(err)
			}
			if made.faces() < last {
				t.Errorf("%s at %s has %d faces, fewer than %d at the "+
					"detail before it", name, detail, made.faces(), last)
			}
			last = made.faces()
		}
	}
}

func TestEveryFigureSaysWhereItCameFrom(t *testing.T) {
	t.Parallel()
	for name, design := range boatRead.Kinds {
		for key, row := range design.Ranges {
			if !based(row.Basis) {
				t.Errorf("%s's %s has the basis %q", name, key, row.Basis)
			}
		}
	}
	for _, name := range []string{boatSampan, boatFishing} {
		if boatRead.Kinds[name].Ranges["length"].Basis != boatRecalled {
			t.Errorf("the %s's length was recalled in the brief", name)
		}
	}
	if boatRead.Kinds[boatFerry].Ranges["length"].Basis != boatPicked {
		t.Errorf("the ferry's length was given no figure; it is picked")
	}
	for name, lamp := range boatRead.Lamps {
		if lamp.Basis != boatPicked || !based(lamp.ArcBasis) {
			t.Errorf("the lamp %s has a basis %q and an arc basis %q", name,
				lamp.Basis, lamp.ArcBasis)
		}
	}
}

func TestTheBoatsFloatOnTheirOrigin(t *testing.T) {
	t.Parallel()
	for _, name := range boatNamesAll() {
		for number := range uint64(6) {
			box := mustMake(t, name, number).Bounds()
			if !(box.Min.Y < 0 && box.Max.Y > 0) {
				t.Errorf("%s from %d spans %v to %v up, and the waterline "+
					"is at nought", name, number, box.Min.Y, box.Max.Y)
			}
			if !(box.Min.X < 0 && box.Max.X > 0 && box.Min.Z < 0 &&
				box.Max.Z > 0) {
				t.Errorf("%s from %d is not centred on its origin: %v",
					name, number, box)
			}
		}
	}
}

// hullLength is how far along z the lofts of a boat reach: its hull, decks
// and bulwarks, and not the gear, an outboard or a boom, that stands out of
// either end of them.
func hullLength(made good) float64 {
	low, high := math.Inf(1), math.Inf(-1)
	for _, piece := range made.parts {
		if _, lofted := piece.shape.(boatLoft); !lofted {
			continue
		}
		for _, face := range piece.skin() {
			for _, corner := range face {
				low, high = math.Min(low, corner.Z), math.Max(high, corner.Z)
			}
		}
	}
	return high - low
}

// stemReach is the most a design's stem can lean past the length drawn.
// The stem leans by the rake over the height of the bow's top above the keel
// as a share of the freeboard and draught, so the bow's top, which stands
// the freeboard, the spring, the camber and a rail over the waterline, leans
// the rake times that share: most with every figure at its largest over
// the freeboard and draught at their least.
func stemReach(design string) float64 {
	ranges := boatRead.Kinds[design].Ranges
	top := ranges["draught"].Most + ranges["freeboard"].Most +
		ranges["spring"].Most + ranges["camber"].Most + ranges["rail"].Most
	base := ranges["draught"].Least + ranges["freeboard"].Least
	return ranges["rake"].Most * top / base
}

func TestEachSeedMakesItsOwnBoatInsideTheDesignsRange(t *testing.T) {
	t.Parallel()
	for _, design := range boatDesigns {
		row := boatRead.Kinds[design].Ranges["length"]
		reach := stemReach(design)
		seen := map[float64]bool{}
		for number := range uint64(40) {
			long := hullLength(mustMake(t, design, number))
			seen[math.Round(long*1000)] = true
			if long < row.Least || long > row.Most+reach+1e-6 {
				t.Errorf("%s from %d is %v m long, outside %v to %v", design,
					number, long, row.Least, row.Most+reach)
			}
		}
		if len(seen) < 30 {
			t.Errorf("%s: forty seeds made only %d lengths", design, len(seen))
		}
	}
}

// A sampan of seed 5 has a fitting 1.6 m past the end of its hull, measured:
// the box of the whole boat is longer than its hull, and the hull is what
// the design's length is.
func TestTheBoxOfABoatWithGearIsLongerThanItsHull(t *testing.T) {
	made := mustMake(t, "sampan", 5)
	box := made.Bounds()
	if hull, whole := hullLength(made), box.Max.Z-box.Min.Z; !(hull <
		whole-1e-3) {
		t.Errorf("the hull is %v m long and the boat %v m", hull, whole)
	}
}

func TestABoatUnderWayShowsTheLightsItsRuleCalls(t *testing.T) {
	t.Parallel()
	for _, design := range boatDesigns {
		made := mustMake(t, design, 5)
		names := lampNames(made)
		for _, want := range []string{"port light", "starboard light",
			"stern light"} {
			if !slices.Contains(names, want) {
				t.Errorf("%s under way has no %s: %v", design, want, names)
			}
		}
		if slices.Contains(names, "anchor light") {
			t.Errorf("%s under way shows an anchor light", design)
		}
	}
	for _, design := range []string{boatFishing, boatFerry} {
		made := mustMake(t, design, 5)
		if !slices.Contains(lampNames(made),
			"masthead light") {
			t.Errorf("the %s is a power vessel and shows a masthead light",
				design)
		}
	}
}

func TestTheLightsShineWhereTheRulesSay(t *testing.T) {
	t.Parallel()
	made := mustMake(t, boatFishing, 2)
	arcs := map[string]float64{"port light": 112.5, "starboard light": 112.5,
		"stern light": 135, "masthead light": 225}
	for _, one := range made.lights {
		want, ruled := arcs[one.name]
		if !ruled {
			continue
		}
		half, _ := one.made.angles()
		if got := half.InDegrees() * 2; math.Abs(got-want) > 1e-6 {
			t.Errorf("%s shines across %v degrees, not %v", one.name, got, want)
		}
	}
	for _, one := range made.lights {
		switch one.name {
		case "port light":
			if !(one.toward.X < 0 && one.toward.Z < 0) {
				t.Errorf("the port light looks %v, not to port and ahead",
					one.toward)
			}
			red, green, _ := tintOf(t, one)
			if !(red > 10*green) {
				t.Errorf("the port light is not red: %v, %v", red, green)
			}
		case "starboard light":
			if !(one.toward.X > 0 && one.toward.Z < 0) {
				t.Errorf("the starboard light looks %v, not to starboard "+
					"and ahead", one.toward)
			}
			red, green, _ := tintOf(t, one)
			if !(green > 10*red) {
				t.Errorf("the starboard light is not green: %v, %v", red,
					green)
			}
		case "stern light":
			if one.toward.Z != 1 {
				t.Errorf("the stern light looks %v, not aft", one.toward)
			}
		case "masthead light":
			if one.toward.Z != -1 {
				t.Errorf("the masthead light looks %v, not ahead", one.toward)
			}
		}
	}
}

// tintOf is a lamp's colour, red, green and blue.
func tintOf(t *testing.T, one lit) (red, green, blue float64) {
	t.Helper()
	colour, _, tinted := one.made.Tint()
	if !tinted {
		t.Fatalf("%s has no colour of its own", one.name)
	}
	return colour.R, colour.G, colour.B
}

func TestABoatAtAnchorShowsOneWhiteLightAllRound(t *testing.T) {
	t.Parallel()
	for _, design := range boatDesigns {
		made := mustMake(t, design+boatAtAnchor, 5)
		names := lampNames(made)
		for _, banned := range []string{"port light", "starboard light",
			"stern light", "masthead light"} {
			if slices.Contains(names, banned) {
				t.Errorf("%s at anchor shows a %s", design, banned)
			}
		}
		found := false
		for _, one := range made.lights {
			if one.name != "anchor light" {
				continue
			}
			found = true
			half, _ := one.made.angles()
			if math.Abs(half.InRadians()-math.Pi) > 1e-9 {
				t.Errorf("the anchor light of the %s does not shine all "+
					"round: %v", design, half.InDegrees())
			}
			if _, _, tinted := one.made.Tint(); tinted {
				t.Errorf("the anchor light of the %s is coloured", design)
			}
		}
		if !found {
			t.Errorf("the %s at anchor shows no anchor light: %v", design,
				names)
		}
	}
}

func TestEachDesignCarriesItsWorkingLights(t *testing.T) {
	t.Parallel()
	want := map[string]string{boatSampan: "lantern", boatFishing: "flood",
		boatFerry: "cabin"}
	for design, lamp := range want {
		for _, name := range []string{design, design + boatAtAnchor} {
			found := false
			for _, one := range lampNames(mustMake(t, name, 7)) {
				found = found || strings.HasPrefix(one, lamp+" light")
			}
			if !found {
				t.Errorf("%s has no %s", name, lamp)
			}
		}
	}
}

func TestARepeatedPartIsStoredOnceAndPlacedMany(t *testing.T) {
	t.Parallel()
	row, _ := kindNamed(boatFerry)
	made, err := MakeWithComplexity(row, sow(1), "quick")
	if err != nil {
		t.Fatal(err)
	}
	windows := made.groups[boatWindow]
	if windows == nil || len(windows.Members()) < 4 {
		t.Fatalf("the ferry's lit windows are not shared as one mesh placed "+
			"many times: %v", windows)
	}
	if windows.StoredFaces() >= windows.FaceCount() {
		t.Errorf("the windows are stored %d faces for %d placed",
			windows.StoredFaces(), windows.FaceCount())
	}
}

func TestBuoyantIsTheBoatsAndTheAlfasAndNothingElse(t *testing.T) {
	t.Parallel()
	for _, name := range boatNamesAll() {
		afloat, floats := Buoyant(name)
		if !floats || afloat.Waterline.InMetres() != 0 {
			t.Errorf("%s: floats %v, waterline %v", name, floats,
				afloat.Waterline.InMetres())
		}
	}
	if afloat, floats := Buoyant("alfa"); !floats ||
		afloat.Waterline.InMetres() <= 0 {
		t.Errorf("the alfa floats at a waterline above its keel: %v, %v",
			floats, afloat.Waterline.InMetres())
	}
	for _, name := range []string{"spruce", "yurt", "sampans", ""} {
		if _, floats := Buoyant(name); floats {
			t.Errorf("%q does not float", name)
		}
	}
}

func TestAnUnsownSeedAndAnUnknownDetailMakeNoBoat(t *testing.T) {
	t.Parallel()
	row, _ := kindNamed(boatSampan)
	if _, err := makeOf(row, seed{}); err == nil {
		t.Error("an unsown seed made a boat")
	}
	if _, err := MakeWithComplexity(row, sow(1), "grand"); err == nil {
		t.Error("an unknown detail made a boat")
	}
}

func TestADataFileThatCannotMakeABoatIsRefused(t *testing.T) {
	t.Parallel()
	text := string(boatRaw)
	broken := map[string]string{
		"a basis nobody knows": strings.Replace(text, "basis: picked",
			"basis: guessed", 1),
		"a lamp missing": strings.Replace(text, "  anchor: {",
			"  anchorage: {", 1),
		"a range nobody reads": strings.Replace(text, "      tyres: {",
			"      wheels: {", 1),
		"a hull figure missing": strings.Replace(text, "      bilge: {",
			"      bilgy: {", 1),
		"no figures read": strings.Replace(text, "read: \"", "read: \"\" #\"",
			1),
		"an arc past a turn": strings.Replace(text, "arc: 360", "arc: 400", 1),
		"a Chinese face and no Chinese names": strings.Replace(text,
			"chinese: {face: \"\", names: []}",
			"chinese: {face: \"some-face\", names: []}", 1),
		"lettering with no names": strings.Replace(text, "  names: [LUCKY STAR",
			"  names: [] #[LUCKY STAR", 1),
	}
	for what, raw := range broken {
		if raw == text {
			t.Fatalf("%s: the test's edit changed nothing", what)
		}
		if _, err := readBoats([]byte(raw)); err == nil {
			t.Errorf("a file with %s was read", what)
		}
	}
}

// boatLeastParts is the fewest real parts a boat has at its fullest detail:
// several hundred, the bar for a made thing.
const boatLeastParts = 200

func TestEveryBoatHasHundredsOfRealPartsAtTheFullestDetail(t *testing.T) {
	t.Parallel()
	for _, name := range boatDesigns {
		row, _ := kindNamed(name)
		made, err := MakeWithComplexity(row, sow(2), "demo")
		if err != nil {
			t.Fatal(err)
		}
		if made.PartCount() < boatLeastParts {
			t.Errorf("%s has %d parts at demo, fewer than %d", name,
				made.PartCount(), boatLeastParts)
		}
	}
}

func TestFewerPartsAtEachLowerDetail(t *testing.T) {
	t.Parallel()
	for _, name := range boatDesigns {
		row, _ := kindNamed(name)
		last := 0
		for _, detail := range Complexities() {
			made, err := MakeWithComplexity(row, sow(2), detail)
			if err != nil {
				t.Fatal(err)
			}
			if made.PartCount() < last {
				t.Errorf("%s at %s has %d parts, fewer than %d at the "+
					"detail before it", name, detail, made.PartCount(), last)
			}
			last = made.PartCount()
		}
	}
}

func TestNoTwoBoatsOfAKindHaveTheSameClutter(t *testing.T) {
	t.Parallel()
	for _, name := range boatDesigns {
		row, _ := kindNamed(name)
		counts := map[int]bool{}
		for number := range uint64(6) {
			made, err := MakeWithComplexity(row, sow(number+1), "demo")
			if err != nil {
				t.Fatal(err)
			}
			counts[made.PartCount()] = true
		}
		if len(counts) < 3 {
			t.Errorf("six %s boats have %d different counts of parts", name,
				len(counts))
		}
	}
}

func TestThePaintedLettersAreRealPartsOnTheHull(t *testing.T) {
	t.Parallel()
	for _, name := range boatDesigns {
		row, _ := kindNamed(name)
		made, err := MakeWithComplexity(row, sow(2), "demo")
		if err != nil {
			t.Fatal(err)
		}
		letters := made.groups[boatLetter]
		if letters == nil || len(letters.Members()) < 8 {
			t.Errorf("%s carries no painted letters as parts: %v", name,
				letters)
		}
		silhouette, _ := MakeWithComplexity(row, sow(2), "hull")
		if silhouette.groups[boatLetter] != nil {
			t.Errorf("%s at its silhouette carries letters", name)
		}
	}
}

func TestTheBoatsNumbersAreFourDigitsAndTheNamesComeFromTheData(t *testing.T) {
	t.Parallel()
	craft := boatRow{design: boatFishing}.craft(sow(5), boatDemo)
	number := craft.number()
	if len(number) != 4 || strings.Trim(number, "0123456789") != "" {
		t.Errorf("the number %q is not four digits", number)
	}
	face, words := craft.name()
	if face != boatRead.Lettering.Face ||
		!slices.Contains(boatRead.Lettering.Names, words) {
		t.Errorf("the name %q in %q is not the data's", words, face)
	}
}

// A boat's draught is the most of its design's range, under way or at
// anchor, and it is no more than any hull the design draws.
func TestABoatDraughtIsTheMostItsDesignDraws(t *testing.T) {
	t.Parallel()
	for _, name := range boatNamesAll() {
		most, boat := BoatDraught(name)
		if !boat || !(most > 0) {
			t.Fatalf("%s: draught %v, boat %v", name, most, boat)
		}
		design := boatRead.Kinds[strings.TrimSuffix(name, boatAtAnchor)]
		for index := range uint64(16) {
			hull := newBoatHull(design, sow(index+1), 0)
			if hull.draught > most {
				t.Errorf("%s from seed %d draws %v, over the most %v", name,
					index+1, hull.draught, most)
			}
		}
	}
}

// Anti-test: a kind that is not a boat has no draught, and the Alfa, which
// floats by its own waterline, is not given one.
func TestAKindThatIsNotABoatHasNoDraught(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"spruce", "terrace-groningen", alfaKind, ""} {
		if most, boat := BoatDraught(name); boat || most != 0 {
			t.Errorf("%s: draught %v, boat %v", name, most, boat)
		}
	}
}
