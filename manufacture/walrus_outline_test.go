package manufacture

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// testWalrusOutline is the walrus's outline, failing the test on a fault.
func testWalrusOutline(t *testing.T) *walrusOutline {
	t.Helper()
	outline, err := walrusTheOutline()
	if err != nil {
		t.Fatal(err)
	}
	return outline
}

func TestWalrusOutlineIsABullsLength(t *testing.T) {
	// "about 315 cm", read; the outline is held to within a tenth of it.
	const read = 3.15
	outline := testWalrusOutline(t)
	length := outline.last - outline.first
	if !(math.Abs(length-read) <= .1*read) {
		t.Fatalf("the outline is %.3f m long, not within a tenth of %.2f",
			length, read)
	}
	rows := walrusStations
	if rows[0].behind != 0 ||
		!(math.Abs(rows[len(rows)-1].behind-length) <= 1e-9) {
		t.Fatalf("stations run %.3f to %.3f, the outline %.3f",
			rows[0].behind, rows[len(rows)-1].behind, length)
	}
}

func TestWalrusOutlineEnclosesABullsMass(t *testing.T) {
	// "weighing up to 1200 to 1500 kg", read; the density is picked.
	const density, least, most = 1030., 1200., 1500.
	mass := density * testWalrusOutline(t).volume(walrusFloor)
	if !(mass >= least && mass <= most) {
		t.Fatalf("the outline holds %.0f kg, outside %.0f to %.0f",
			mass, least, most)
	}
}

func TestWalrusOutlinePassesThroughItsStations(t *testing.T) {
	const near = .002
	outline := testWalrusOutline(t)
	for _, row := range walrusStations {
		section := outline.at(outline.last - row.behind)
		top := section.centre + section.high - walrusFloor
		bottom := section.centre - section.high - walrusFloor
		if !(math.Abs(section.wide-row.wide) <= near) ||
			!(math.Abs(top-row.top) <= near) ||
			!(math.Abs(bottom-row.bottom) <= near) {
			t.Fatalf("%.2f m behind the nose: wide %.3f, top %.3f, bottom %.3f; "+
				"the table has %.3f, %.3f, %.3f", row.behind, section.wide, top,
				bottom, row.wide, row.top, row.bottom)
		}
	}
}

func TestWalrusOutlineHasNoRingBeadOrWaist(t *testing.T) {
	// From the nose the width only grows to the greatest girth and only
	// falls from there to the tail: one swelling, in the front half.
	const slack = .002
	outline := testWalrusOutline(t)
	widest, at := 0., 0
	for i, section := range outline.sections {
		if section.wide > widest {
			widest, at = section.wide, i
		}
	}
	for i := 1; i < len(outline.sections); i++ {
		grew := outline.sections[i].wide - outline.sections[i-1].wide
		if i <= at && grew < -slack || i > at && grew > slack {
			t.Fatalf("section %d of %d changes width by %.4f on the wrong "+
				"side of the widest, %d", i, len(outline.sections), grew, at)
		}
	}
	behind := (outline.last - outline.sections[at].x) /
		(outline.last - outline.first)
	if !(behind > .3 && behind < .5) {
		t.Fatalf("the greatest width lies %.2f of the way back", behind)
	}
}

func TestWalrusHeadIsSmallOnItsNeck(t *testing.T) {
	outline := testWalrusOutline(t)
	eyes := outline.at(walrusEyeSocket(1).X).wide
	shoulders := outline.at(outline.last - .95).wide
	if !(eyes < .5*shoulders) {
		t.Fatalf("the head is %.3f wide at the eyes, the shoulders %.3f",
			2*eyes, 2*shoulders)
	}
	face := walrusStations[0]
	if !(2*face.wide > face.top-face.bottom) {
		t.Fatalf("the pads' face is %.3f wide and %.3f tall", 2*face.wide,
			face.top-face.bottom)
	}
}

// testWalrusInsideBones is the bones that lie wholly inside the body wall.
var testWalrusInsideBones = []string{"cervical", "thoracic", "lumbar",
	"sacral", "caudal", "skull", "mandible", "sternebra", "scapula", "humerus",
	"innominate", "femur"}

func TestWalrusOutlineHoldsTheBones(t *testing.T) {
	outline := testWalrusOutline(t)
	for name, form := range walrusRest.forms {
		inside := false
		for _, prefix := range testWalrusInsideBones {
			inside = inside || strings.HasPrefix(name, prefix)
		}
		if !inside {
			continue
		}
		for _, p := range form.Middle() {
			if gap := outline.distance(p); !(gap < 0) {
				t.Errorf("%s at %v lies %.3f m outside the outline", name, p, gap)
			}
		}
	}
}

func TestWalrusOutlineDistanceIsSigned(t *testing.T) {
	outline := testWalrusOutline(t)
	middle := outline.at(0)
	for _, row := range []struct {
		p      maths.Vec
		inside bool
	}{
		{maths.V(0, middle.centre, 0), true},
		{maths.V(0, middle.centre, middle.wide-.02), true},
		{maths.V(0, middle.centre, middle.wide+.02), false},
		{maths.V(0, middle.centre+middle.high+.02, 0), false},
		{maths.V(outline.last+.05, outline.at(outline.last).centre, 0), false},
		{maths.V(outline.first-.05, outline.at(outline.first).centre, 0), false},
	} {
		if gap := outline.distance(row.p); (gap < 0) != row.inside {
			t.Fatalf("%v: distance %.4f, inside wanted %v", row.p, gap, row.inside)
		}
	}
}

func TestWalrusOutlineFlankAndCrownLieOnIt(t *testing.T) {
	const near = .003
	outline := testWalrusOutline(t)
	for _, x := range []float64{-.9, 0, .8, 1.5} {
		section := outline.at(x)
		y := section.centre + .4*section.high
		z, ok := outline.flank(x, y)
		onFlank := maths.V(x, y, z)
		if !ok || !(math.Abs(outline.distance(onFlank)) <= near) {
			t.Fatalf("flank at %.1f: %.3f, found %v", x, z, ok)
		}
		top, ok := outline.crown(x, .4*section.wide)
		at := maths.V(x, top, .4*section.wide)
		if !ok || !(math.Abs(outline.distance(at)) <= near) {
			t.Fatalf("crown at %.1f: %.3f, found %v", x, top, ok)
		}
	}
}

func TestWalrusOutlineHasNoFlankAboveItsBack(t *testing.T) {
	outline := testWalrusOutline(t)
	section := outline.at(0)
	if _, ok := outline.flank(0, section.centre+section.high+.01); ok {
		t.Fatal("a flank above the back")
	}
	if _, ok := outline.flank(outline.last+.01, section.centre); ok {
		t.Fatal("a flank in front of the nose")
	}
	if _, ok := outline.crown(0, section.wide+.01); ok {
		t.Fatal("a back wider than the body")
	}
	if _, ok := outline.crown(outline.first-.01, 0); ok {
		t.Fatal("a back behind the tail")
	}
}

func TestWalrusOutlineBackLiesOnTheRoundedEnd(t *testing.T) {
	const exact, across, tenfold = 1e-9, .04, 10
	outline := testWalrusOutline(t)
	drops := map[string]float64{}
	for name, x := range map[string]float64{"mid-body": 0,
		"a rim from the nose": outline.last - walrusOutlineRim} {
		top, found := outline.crown(x, across)
		back, ok := outline.back(x, across)
		if !found || !ok || back > top {
			t.Fatalf("%s: back %.5f under crown %.5f, found %v", name,
				back, top, ok)
		}
		if on := outline.distance(maths.V(x, back, across)); math.Abs(on) > exact {
			t.Errorf("%s: the back reads %g from the outline", name, on)
		}
		drops[name] = top - back
	}
	if !(drops["a rim from the nose"] > tenfold*drops["mid-body"]) {
		t.Errorf("the rim lowers the back by %v at the nose, %v mid-body",
			drops["a rim from the nose"], drops["mid-body"])
	}
	if _, ok := outline.back(0, outline.at(0).wide+.01); ok {
		t.Error("a back wider than the body")
	}
}

func TestWalrusOutlineRefusesABadTable(t *testing.T) {
	good := walrusSections(walrusStations, 1)
	swapped := append([]walrusSection{}, good...)
	swapped[3], swapped[4] = swapped[4], swapped[3]
	flat := append([]walrusSection{}, good...)
	flat[5].high = 0
	thin := append([]walrusSection{}, good...)
	thin[5].wide = math.NaN()
	pinched := append([]walrusSection{}, good...)
	pinched[5].square = .5
	for name, sections := range map[string][]walrusSection{
		"out of order": swapped, "no height": flat, "no width": thin,
		"an exponent under one": pinched, "one section": good[:1],
		"none": nil,
	} {
		if _, err := walrusOutlineOf(sections); !errors.Is(err, errWalrusOutline) {
			t.Errorf("%s gave %v", name, err)
		}
	}
	if _, err := walrusOutlineOf(good); err != nil {
		t.Fatalf("the walrus's own table gave %v", err)
	}
}

func TestWalrusSectionAreaIsItsShapes(t *testing.T) {
	none := math.Inf(-1)
	round := walrusSection{wide: .5, high: .25, square: 2}
	if area := round.area(none); !(math.Abs(area-math.Pi*.5*.25) <= .002) {
		t.Fatalf("an ellipse's area came to %.4f", area)
	}
	if area := round.area(0); !(math.Abs(area-math.Pi*.5*.25/2) <= .002) {
		t.Fatalf("the half of an ellipse over its middle came to %.4f", area)
	}
	if area := round.area(round.high + .01); area != 0 {
		t.Fatalf("an ellipse wholly under the ground has the area %.4f", area)
	}
	square := walrusSection{wide: .5, high: .25, square: 40}
	if area := square.area(none); !(math.Abs(area-4*.5*.25) <= .02) {
		t.Fatalf("a nearly square section's area came to %.4f", area)
	}
	disc := walrusSection{wide: .5, high: .5, square: 2}
	circle := walrusOutline{first: 0, last: 1, step: 1,
		sections: []walrusSection{disc, disc}}
	if girth := circle.girth(.5); !(math.Abs(girth-math.Pi) <= .001) {
		t.Fatalf("a circle a metre across is %.4f round", girth)
	}
}

func TestWalrusForeDigitsShortenAfterTheFirst(t *testing.T) {
	previous := math.Inf(1)
	for ray, bones := range walrusForeDigits {
		length := walrusForePalm[ray]
		for _, bone := range bones {
			length += bone
		}
		if length >= previous {
			t.Fatalf("fore digit %d is %.3f, not shorter than the one before",
				ray+1, length)
		}
		previous = length
	}
}
