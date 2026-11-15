package facts

import (
	"math"
	"strings"
	"testing"
)

// The numbers the tests choose: the quasar slice, 2026-11-12, unless
// a comment says defined.
const (
	// ton618LighterAU is the lighter mass's horizon worked by hand in
	// HorizonAU's comment: 804 au.
	ton618LighterAU = 804
	// auTolerance is how near the hand's arithmetic, in au.
	auTolerance = 0.5
	// sunHorizonKm is defined arithmetic: two GM of the sun over c
	// squared, 2953.25 metres, in au.
	sunHorizonMetres = 2953.25
	metreTolerance   = 0.01
	// twoPlaces is a value written to two places.
	twoPlaces = 2
	// aCount is a count a sentence may be handed.
	aCount = 3
)

func TestASentenceSaysItsRegisteredNumber(t *testing.T) {
	percent := SubringPercent()
	sentence, err := NewSentence("Each ring is about "+percent.Shown()+
		"% as wide as the last.", percent)
	if err != nil {
		t.Fatal(err)
	}
	if len(sentence.Numbers()) != one || !strings.Contains(
		sentence.Text(), "4%") {
		t.Fatalf("the sentence is %q with %v", sentence.Text(),
			sentence.Numbers())
	}
}

func TestASentenceWithANumberFromNowhereIsRefused(t *testing.T) {
	if _, err := NewSentence("Each ring is about 5% as wide.",
		SubringPercent()); err == nil {
		t.Fatal("a sentence saying 5 with the registered 4 was made")
	}
	if _, err := NewSentence("It is 66 billion suns."); err == nil {
		t.Fatal("a sentence with a bare number was made")
	}
}

func TestANumberHandedAndNotSaidIsRefused(t *testing.T) {
	if _, err := NewSentence("Each ring is thinner.",
		SubringPercent()); err == nil {
		t.Fatal("a sentence that does not say its number was made")
	}
}

func TestTheRegisteredRingsAreTheirPapers(t *testing.T) {
	for _, each := range []struct {
		number Number
		shown  string
	}{{M87Ring(), "42"}, {SgrARing(), "51.8"}, {SubringPercent(), "4"}} {
		if each.number.Shown() != each.shown || !strings.HasPrefix(
			each.number.From(), "research ") {
			t.Errorf("%v is written %q from %q", each.number.Value(),
				each.number.Shown(), each.number.From())
		}
	}
}

func TestTon618IsARangeNotOneNumber(t *testing.T) {
	lighter, heavier := Ton618Billions(false), Ton618Billions(true)
	if lighter.Shown() != "41" || heavier.Shown() != "66" {
		t.Fatalf("TON 618 is %s to %s billion suns, want 41 to 66",
			lighter.Shown(), heavier.Shown())
	}
}

func TestTheSunsHorizonIsTwoGMOverCSquared(t *testing.T) {
	metres := HorizonAU(one) * astronomicalUnit
	if math.Abs(metres-sunHorizonMetres) > metreTolerance {
		t.Fatalf("the sun's horizon is %v metres, want %v", metres,
			sunHorizonMetres)
	}
}

func TestTon618sHorizonIsTheHandsArithmetic(t *testing.T) {
	horizon := Ton618Horizon(false)
	if math.Abs(horizon.Value()-ton618LighterAU) > auTolerance {
		t.Fatalf("the horizon is %v au, the hand says %v",
			horizon.Value(), ton618LighterAU)
	}
	if !(Ton618Horizon(true).Value() > horizon.Value()) {
		t.Fatal("the heavier mass's horizon is not the larger")
	}
}

func TestTon618sHorizonInNeptunes(t *testing.T) {
	lighter, heavier := Ton618Neptunes(false), Ton618Neptunes(true)
	if lighter.Shown() != "27" || heavier.Shown() != "43" {
		t.Fatalf("the horizon is %s to %s Neptunes, want 27 to 43",
			lighter.Shown(), heavier.Shown())
	}
}

func TestComputedWritesToItsPlacesAndNamesItsSource(t *testing.T) {
	number, err := Computed(math.Pi, twoPlaces, "a test")
	if err != nil {
		t.Fatal(err)
	}
	if number.Shown() != "3.14" || number.From() != "a test" {
		t.Fatalf("pi is written %q from %q", number.Shown(),
			number.From())
	}
}

func TestComputedRefusesNoNumberAndNoSource(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1)} {
		if _, err := Computed(value, none, "a test"); err == nil {
			t.Errorf("%v was taken", value)
		}
	}
	if _, err := Computed(aCount, -one, "a test"); err == nil {
		t.Error("negative places were taken")
	}
	if _, err := Computed(aCount, none, " "); err == nil {
		t.Error("a number with no source was taken")
	}
}

func TestAnObjectsNameIsNotANumber(t *testing.T) {
	if _, err := NewSentence("M87 and TON 618 are names."); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSentence("M88 is not one of ours."); err == nil {
		t.Fatal("a name the package does not know passed its digits")
	}
}
