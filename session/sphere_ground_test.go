package session

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan/mocks"
	"go.uber.org/mock/gomock"
)

// the numbers the sphere ground's tests stand on
const (
	venusRadius  = 6051000.0 // defined: the radius Magellan's map is on, metres
	magellanGap  = 4641.0589 // defined: Magellan's samples' distance apart, metres
	veneraLat    = -7.5      // defined: Venera 13's latitude, degrees north
	veneraLon    = 303.0     // defined: Venera 13's longitude, degrees east
	smallSide    = 2.0       // picked: a window two metres on a side, three points
	smallPoints  = 3         // picked: the points on a side of that window at a metre
	oneMetre     = 1.0       // picked: a step
	reliefNoise  = 50.0      // picked: metres of test relief that varies faster than the samples
	fastTurns    = 20000.0   // picked: radians a degree of latitude turns the test relief
	slowTurns    = 17000.0   // picked: the same for a degree of longitude
	nearlyEqual  = 1e-9      // picked: how close two angles must be to be one
	sixtyDegrees = 60.0      // defined: a latitude where the cosine is a half
)

// pinOne and pinTwo are two sha256 sums, as a pin is. picked: any two that
// differ.
const (
	pinOne = "11bdc2ae93beba691276f94294d8e4c61c7953c53c6522a45d269127b85d9beb"
	pinTwo = "21bdc2ae93beba691276f94294d8e4c61c7953c53c6522a45d269127b85d9beb"
)

// somePlace is a place on a sphere to lay a window at.
func somePlace() groundPlace {
	return groundPlace{latitude: veneraLat, longitude: veneraLon,
		across: smallSide, down: smallSide, source: "a-map"}
}

// someCut is the small window's cut, laid on somePlace.
func someCut() groundCut {
	place := somePlace()
	return groundCut{west: 0, north: smallSide, step: oneMetre,
		across: smallPoints, down: smallPoints, place: &place}
}

// aSphere is a raster of a sphere whose heights vary faster than its samples.
func aSphere(t *testing.T) *mocks.MockSphere {
	t.Helper()
	raster := mocks.NewMockSphere(gomock.NewController(t))
	raster.EXPECT().EquirectangularSphere().Return(venusRadius, true).AnyTimes()
	raster.EXPECT().GroundSpacing(gomock.Any()).
		Return(magellanGap, magellanGap, nil).AnyTimes()
	raster.EXPECT().EquirectangularElevation(gomock.Any(), gomock.Any()).
		DoAndReturn(func(longitude, latitude float64) (float64, bool, error) {
			return reliefNoise*math.Sin(latitude*fastTurns) +
				reliefNoise*math.Cos(longitude*slowTurns), true, nil
		}).AnyTimes()
	return raster
}

// North of the middle is more latitude, by the distance over the radius.
func TestLaidOnNorthIsLatitude(t *testing.T) {
	place := groundPlace{latitude: veneraLat, longitude: veneraLon}
	latitude, longitude := place.laidOn(venusRadius, 0, magellanGap)
	want := veneraLat + magellanGap/venusRadius/maths.Degree
	if math.Abs(latitude-want) > nearlyEqual || longitude != veneraLon {
		t.Fatalf("got %v %v, want %v", latitude, longitude, want)
	}
}

// Anti: east is not latitude, and the middle is where the place is.
func TestLaidOnEastLeavesLatitudeAlone(t *testing.T) {
	place := groundPlace{latitude: veneraLat, longitude: veneraLon}
	latitude, longitude := place.laidOn(venusRadius, magellanGap, 0)
	if latitude != veneraLat || longitude <= veneraLon {
		t.Fatalf("got %v %v", latitude, longitude)
	}
	if there, along := place.laidOn(venusRadius, 0, 0); there != veneraLat ||
		along != veneraLon {
		t.Fatalf("the middle is %v %v", there, along)
	}
}

// East is more longitude the nearer a pole: at sixty degrees a metre east is
// twice the change in longitude it is on the equator.
func TestLaidOnEastIsWiderTowardAPole(t *testing.T) {
	_, flat := groundPlace{}.laidOn(venusRadius, magellanGap, 0)
	_, steep := groundPlace{latitude: sixtyDegrees}.laidOn(venusRadius,
		magellanGap, 0)
	if math.Abs(steep-2*flat) > nearlyEqual {
		t.Fatalf("equator %v, sixty %v", flat, steep)
	}
}

// Anti: at the equator a metre east is the same change as a metre north.
func TestLaidOnEquatorIsEvenHanded(t *testing.T) {
	north, _ := groundPlace{}.laidOn(venusRadius, 0, magellanGap)
	_, east := groundPlace{}.laidOn(venusRadius, magellanGap, 0)
	if math.Abs(north-east) > nearlyEqual {
		t.Fatalf("north %v east %v", north, east)
	}
}

// A window wider than a hundredth of the radius is refused, either way.
func TestRefuseWideRefusesWhatIsWide(t *testing.T) {
	wide := venusRadius*patchShare + 1
	for _, place := range []groundPlace{{across: wide, down: 1},
		{across: 1, down: wide}} {
		err := place.refuseWide(venusRadius)
		if err == nil || !strings.Contains(err.Error(), "hundredth") {
			t.Errorf("%+v: %v", place, err)
		}
	}
}

// Anti: a window at the share exactly, and a small one, are not refused.
func TestRefuseWideLetsTheNarrowBe(t *testing.T) {
	for _, side := range []float64{1, venusRadius * patchShare} {
		place := groundPlace{across: side, down: side}
		if err := place.refuseWide(venusRadius); err != nil {
			t.Errorf("%g: %v", side, err)
		}
	}
}

// A map alone gives the map's heights at the places the grid lays on the
// sphere, north rows above south, and keeps nothing of made ground.
func TestOnSphereReadsTheMapAlone(t *testing.T) {
	found, err := someCut().onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	across, down, step := found.window.Size()
	if across != smallPoints || down != smallPoints || step != oneMetre {
		t.Fatalf("size %d %d %g", across, down, step)
	}
	if len(found.made) != 0 || len(found.generated) != 0 {
		t.Fatalf("a map alone kept %d made", len(found.made))
	}
	if found.spacing != magellanGap {
		t.Fatalf("spacing %g", found.spacing)
	}
}

// Anti: with detail asked, every point of ground a metre apart is below a
// map whose samples are kilometres apart, so every one is generated, and a
// number is kept for each point.
func TestOnSphereWithDetailKeepsWhatWasMade(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	found, err := cut.onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	points := smallPoints * smallPoints
	if len(found.made) != points || len(found.generated) != points {
		t.Fatalf("kept %d and %d for %d", len(found.made),
			len(found.generated), points)
	}
	for point, generated := range found.generated {
		if !generated {
			t.Errorf("point %d is not generated", point)
		}
	}
}

// The same seed makes the same ground.
func TestOnSphereSameSeedSameGround(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	first, err := cut.onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := cut.onSphere(aSphere(t))
	if err != nil || string(first.bytes()) != string(second.bytes()) {
		t.Fatalf("ground differs between runs: %v", err)
	}
}

// Anti: another seed makes other ground.
func TestOnSphereOtherSeedOtherGround(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	first, err := cut.onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	cut.place.seed = 14
	second, err := cut.onSphere(aSphere(t))
	if err != nil || string(first.bytes()) == string(second.bytes()) {
		t.Fatalf("two seeds made one ground: %v", err)
	}
}

// A place where the map has no height is an error naming latitude and
// longitude, and no ground stands in for it.
func TestOnSphereMissingMapIsAnError(t *testing.T) {
	raster := mocks.NewMockSphere(gomock.NewController(t))
	raster.EXPECT().EquirectangularSphere().Return(venusRadius, true).AnyTimes()
	raster.EXPECT().GroundSpacing(gomock.Any()).
		Return(magellanGap, magellanGap, nil).AnyTimes()
	raster.EXPECT().EquirectangularElevation(gomock.Any(), gomock.Any()).
		Return(0.0, false, nil).AnyTimes()
	_, err := someCut().onSphere(raster)
	if err == nil || !strings.Contains(err.Error(), "latitude") ||
		!strings.Contains(err.Error(), "longitude") {
		t.Fatalf("got %v", err)
	}
}

// Anti: a map that has a height everywhere is not an error.
func TestOnSphereFullMapIsNoError(t *testing.T) {
	if _, err := someCut().onSphere(aSphere(t)); err != nil {
		t.Fatal(err)
	}
}

// A raster that is not an equirectangular sphere is refused, naming the
// source.
func TestOnSphereRefusesOtherRasters(t *testing.T) {
	raster := mocks.NewMockSphere(gomock.NewController(t))
	raster.EXPECT().EquirectangularSphere().Return(0.0, false)
	_, err := someCut().onSphere(raster)
	if err == nil || !strings.Contains(err.Error(), "a-map") {
		t.Fatalf("got %v", err)
	}
}

// Anti: a window too wide for the sphere is refused, and only then.
func TestOnSphereRefusesAWideWindow(t *testing.T) {
	cut := someCut()
	cut.place.across = venusRadius
	if _, err := cut.onSphere(aSphere(t)); err == nil {
		t.Fatal("a window as wide as the sphere was cut")
	}
}

// The sentence says the figures: points, step, source, spacing, that all
// are made, and how far made relief spans.
func TestSentenceSaysWhatWasMade(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	found, err := cut.onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	said := found.sentence("a-map")
	for _, want := range []string{"ground: 9 points at 1 m from a-map",
		"samples are 4,641 m apart", "every point is below the map and made",
		"made relief spans"} {
		if !strings.Contains(said, want) {
			t.Errorf("%q does not say %q", said, want)
		}
	}
}

// Anti: ground read from the map alone does not say any of it is made.
func TestSentenceOfTheMapAloneSaysNoneIsMade(t *testing.T) {
	found, err := someCut().onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	said := found.sentence("a-map")
	if strings.Contains(said, "every point") ||
		strings.Contains(said, "relief") ||
		!strings.Contains(said, "none of it is made") {
		t.Fatalf("got %q", said)
	}
}

// Some made and some measured is said as a count.
func TestSentenceOfSomeMade(t *testing.T) {
	found := placedGround{made: []float64{0, 0.5, -0.25},
		generated: []bool{false, true, true}, spacing: 1,
		window: mustWindow(t, smallPoints, 2)}
	said := found.sentence("a-map")
	if !strings.Contains(said, "2 of them are below the map and made") ||
		!strings.Contains(said, "spans 0.75 m") {
		t.Fatalf("got %q", said)
	}
}

// Anti: none made, with made numbers kept, says none, and not every point.
func TestSentenceOfNoneMade(t *testing.T) {
	found := placedGround{made: []float64{0, 0}, generated: []bool{false,
		false}, spacing: 1, window: mustWindow(t, 2, 2)}
	said := found.sentence("a-map")
	if strings.Contains(said, "every point") ||
		!strings.Contains(said, "none is made") {
		t.Fatalf("got %q", said)
	}
}

// mustWindow is a window of the points across and down, all at nought.
func mustWindow(t *testing.T, across, down int) terrain.Window {
	t.Helper()
	window, err := terrain.BuildWindow(0, 0, across, down, oneMetre,
		func(int, int) (float64, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	return window
}

// Counts get commas and lengths the places a person reads.
func TestWrittenFigures(t *testing.T) {
	for count, want := range map[int]string{0: "0", 999: "999", 1000: "1,000",
		641601: "641,601", 12345678: "12,345,678"} {
		if got := grouped(count); got != want {
			t.Errorf("grouped %d: %s, want %s", count, got, want)
		}
	}
	for metres, want := range map[float64]string{4641.0589: "4,641",
		0.25: "0.25", 0.4231: "0.42", 12.4: "12", 0.01234: "0.0123"} {
		if got := lengthText(metres); got != want {
			t.Errorf("lengthText %g: %s, want %s", metres, got, want)
		}
	}
}

// Anti: a length is not written in an exponent, however small.
func TestWrittenLengthsHaveNoExponent(t *testing.T) {
	if got := lengthText(0.00001); strings.ContainsAny(got, "e") {
		t.Fatalf("got %s", got)
	}
}

// Placed ground read back from its bytes is the same, made and not.
func TestPlacedGroundSurvivesItsBytes(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	for _, cut := range []groundCut{someCut(), cut} {
		found, err := cut.onSphere(aSphere(t))
		if err != nil {
			t.Fatal(err)
		}
		read, err := placedFromBytes(found.bytes())
		if err != nil || string(read.bytes()) != string(found.bytes()) ||
			read.spacing != found.spacing ||
			len(read.made) != len(found.made) {
			t.Fatalf("read back %v", err)
		}
	}
}

// Anti: bytes cut short, or with an odd amount of made ground, are refused.
func TestPlacedGroundRefusesBadBytes(t *testing.T) {
	cut := someCut()
	cut.place.detailed, cut.place.seed = true, 13
	found, err := cut.onSphere(aSphere(t))
	if err != nil {
		t.Fatal(err)
	}
	whole := found.bytes()
	for name, raw := range map[string][]byte{"empty": nil,
		"short": whole[:headBytes-1], "cut": whole[:len(whole)-1],
		"head only": whole[:headBytes]} {
		if _, err := placedFromBytes(raw); err == nil {
			t.Errorf("%s was read", name)
		}
	}
}

// The tally says the ground after the rest, and says nothing where it has
// none to say.
func TestTallySaysTheGround(t *testing.T) {
	bare := tally{things: 1, faces: 2, kept: 2, lamps: 3}
	said := tally{things: 1, faces: 2, kept: 2, lamps: 3,
		ground: "ground: nine points"}
	if bare.String() != "1 things of 2 faces, lit by 3 lamps" {
		t.Errorf("got %q", bare.String())
	}
	if said.String() != bare.String()+". ground: nine points" {
		t.Errorf("got %q", said.String())
	}
}

// Anti: a ground that is not on a sphere leaves the tally as it was.
func TestTallyWithoutGroundIsUnchanged(t *testing.T) {
	if strings.Contains(tally{}.String(), "ground") {
		t.Fatal("an empty tally speaks of ground")
	}
}

// Each thing that decides a height is in the ground's key: change it alone
// and the key changes.
func TestGroundKeyTakesInEveryInput(t *testing.T) {
	cachedHome(t)
	base := func() groundCut {
		cut := someCut()
		cut.place.detailed, cut.place.seed, cut.smooth = true, 13, 2
		cut.place.relieved, cut.place.slope, cut.place.longest = true,
			0.05, 60 // picked: any relief
		return cut
	}
	was := groundKey(pinOne, base()).Name()
	for name, change := range map[string]func(*groundCut){
		"relieved":  func(cut *groundCut) { cut.place.relieved = false },
		"slope":     func(cut *groundCut) { cut.place.slope++ },
		"longest":   func(cut *groundCut) { cut.place.longest++ },
		"latitude":  func(cut *groundCut) { cut.place.latitude++ },
		"longitude": func(cut *groundCut) { cut.place.longitude++ },
		"across":    func(cut *groundCut) { cut.place.across++ },
		"down":      func(cut *groundCut) { cut.place.down++ },
		"step":      func(cut *groundCut) { cut.step++ },
		"smooth":    func(cut *groundCut) { cut.smooth++ },
		"detail":    func(cut *groundCut) { cut.place.detailed = false },
		"seed":      func(cut *groundCut) { cut.place.seed++ },
		"columns":   func(cut *groundCut) { cut.across++ },
		"rows":      func(cut *groundCut) { cut.down++ },
		"west":      func(cut *groundCut) { cut.west++ },
		"north":     func(cut *groundCut) { cut.north++ },
		"place":     func(cut *groundCut) { cut.place = nil },
	} {
		cut := base()
		change(&cut)
		if groundKey(pinOne, cut).Name() == was {
			t.Errorf("changing %s alone left the key as it was", name)
		}
	}
	if groundKey(pinTwo, base()).Name() == was {
		t.Error("changing the pin alone left the key as it was")
	}
}

// Anti: the same inputs are the same key, so a run finds what it kept.
func TestGroundKeyIsTheSameForTheSameInputs(t *testing.T) {
	cachedHome(t)
	first, second := groundKey(pinOne, someCut()), groundKey(pinOne, someCut())
	if first.Name() != second.Name() {
		t.Fatal("one ground, two keys")
	}
}
