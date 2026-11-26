package pinnipeds

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// sealFormsForTest is every species the lofted maker draws, by name.
var sealFormsForTest = map[string]sealForm{
	"harbor seal":        harborSealForm,
	"hawaiian monk seal": monkSealForm,
}

// sealNearTenth is whether got lies within a tenth of want.
func sealNearTenth(got, want float64) bool {
	return math.Abs(got-want) <= want/10
}

// sealLargestDip is the most a radius falls, walking from the tail to
// its greatest value, or rises, walking on from there to the nose: none
// for a spindle with no waist, bead or hump.
func sealLargestDip(radius func(x float64) float64) float64 {
	const steps = 2000
	at := func(step int) float64 {
		return radius(sealTailTipX + sealLength*float64(step)/steps)
	}
	widest := 0
	for step := range steps + 1 {
		if at(step) > at(widest) {
			widest = step
		}
	}
	dip := 0.
	for step := 1; step <= steps; step++ {
		change := at(step) - at(step-1)
		if step > widest {
			change = -change
		}
		dip = math.Max(dip, -change)
	}
	return dip
}

// sealDipAllowed is the dip an outline may show, in metres: less than
// a hair's width, for the arithmetic.
const sealDipAllowed = 1e-5

func TestSealOutlineMeetsItsSourceAtEverySite(t *testing.T) {
	for name, form := range sealFormsForTest {
		for _, site := range form.source.Sites {
			got := form.outline.meanRadius(form.source.place(site))
			want := form.source.radius(site, form.axillary)
			if !sealNearTenth(got, want) {
				t.Errorf("%s, %s: mean radius %.4f m, the source gives "+
					"%.4f m", name, site.Name, got, want)
			}
		}
	}
}

func TestSealOutlineCheckRefusesAnOutlineAFifthOff(t *testing.T) {
	form := harborSealForm
	for _, site := range form.source.Sites {
		want := form.source.radius(site, form.axillary)
		if sealNearTenth(1.2*want, want) || sealNearTenth(.8*want, want) {
			t.Fatalf("%s: a radius a fifth off passed for the source's",
				site.Name)
		}
	}
}

func TestSealOutlineIsOneSpindle(t *testing.T) {
	for name, form := range sealFormsForTest {
		dip := sealLargestDip(form.outline.meanRadius)
		if dip > sealDipAllowed {
			t.Errorf("%s: the outline dips by %.5f m between nose and "+
				"tail", name, dip)
		}
	}
}

func TestSealSpindleCheckSeesAPinchedNeck(t *testing.T) {
	pinched := func(x float64) float64 {
		// The pinch stands on the run from the shoulder to the neck,
		// where the outline falls more gently than a pinch of this
		// width recovers. On the head's steeper taper, from .55, the
		// radius still falls past such a pinch and no spindle is broken.
		const neck, wide, deep = .45, .05, .15
		return harborSealOutline.meanRadius(x) *
			(1 - deep*math.Exp(-(x-neck)*(x-neck)/(wide*wide)))
	}
	if dip := sealLargestDip(pinched); dip <= sealDipAllowed {
		t.Fatal("a neck pinched by a seventh passed for a spindle")
	}
}

func TestSealTrunkSurfaceIsClosed(t *testing.T) {
	for name, form := range sealFormsForTest {
		surface := form.outline.surface()
		if open := surface.open(); open != 0 {
			t.Errorf("%s: the trunk has %d edges not run once each way",
				name, open)
		}
		surface.faces = surface.faces[1:]
		if surface.open() == 0 {
			t.Errorf("%s: a trunk with a face missing passed for closed",
				name)
		}
	}
}

func TestSealFlippersAreClosedCushions(t *testing.T) {
	flippers, err := sealHeldFlippers(sealBonesForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(flippers) != len(sealOpenings) {
		t.Fatalf("%d flippers, want four", len(flippers))
	}
	for index, flipper := range flippers {
		if open := flipper.surface.open(); open != 0 {
			t.Errorf("flipper %d has %d edges not run once each way",
				index, open)
		}
	}
	turned := flippers[0].surface
	face := turned.faces[0]
	turned.faces[0] = [3]int{face[0], face[2], face[1]}
	if turned.open() == 0 {
		t.Fatal("a cushion with a face turned over passed for closed")
	}
}

// The fore flipper leaves the body between a quarter and a third of the
// length behind the nose, and the hind flipper is between an eighth and
// a fifth of the length; the bands are picked round what a reader of
// photographs gave, 25 to 30 and 15 to 20 percent.
func TestSealFlippersSitAndMeasureAsASeals(t *testing.T) {
	made := sealBonesForTest(t)
	wrist, _, found := made.ends("fore-right-carpal3")
	if !found {
		t.Fatal("no right wrist")
	}
	behind := (sealNoseX - wrist.X) / sealLength
	if behind < .25 || behind > .33 {
		t.Errorf("the wrist lies %.3f of the length behind the nose", behind)
	}
	ankle, _, _ := made.ends("hind-right-astragalus")
	_, tip, found := made.ends("hind-right-digit1-3")
	if !found {
		t.Fatal("no first hind digit")
	}
	if long := (ankle.X - tip.X) / sealLength; long < .125 || long > .2 {
		t.Errorf("the hind flipper is %.3f of the length", long)
	}
}

func TestSealAxialHoldCarriesEveryPointOfTheTrunk(t *testing.T) {
	trunk := sealHeldTrunk(sealBonesForTest(t), harborSealOutline)
	for index, hold := range trunk.holds {
		total := 0.
		for _, each := range hold {
			total += each.Weight()
		}
		if !(total > 0) {
			t.Fatalf("trunk point %d at %v is carried by no bone", index,
				trunk.surface.points[index])
		}
	}
}

func TestSealPressedKeepsTheBellyOverTheGround(t *testing.T) {
	before := 0.
	for height := -.3; height < .3; height += .001 {
		pressed := sealPressed(height)
		if pressed <= before {
			t.Fatalf("pressed height %v at %v is not over %v", pressed,
				height, before)
		}
		if height >= sealSlumpBand && pressed != height {
			t.Fatalf("a point %v over the ground was moved", height)
		}
		before = pressed
	}
}

// sealLowest is the least height of any point of a made seal's meshes.
func sealLowest(t *testing.T, made Result) float64 {
	t.Helper()
	lowest := math.Inf(1)
	for _, part := range made.Parts {
		skin, isMesh := part.Shape.(*mesh.Mesh)
		if !isMesh {
			continue
		}
		for _, point := range skin.Points() {
			lowest = math.Min(lowest, point.Y)
		}
	}
	return lowest
}

func TestSealHauledOutBearsOnTheGround(t *testing.T) {
	made, err := sealBanana.Build(407)
	if err != nil {
		t.Fatal(err)
	}
	if lowest := sealLowest(t, made); math.Abs(lowest) > 1e-9 {
		t.Fatalf("the hauled-out seal's lowest point is at %v, not on "+
			"the ground", lowest)
	}
	swimming, err := sealStroke.Build(407)
	if err != nil {
		t.Fatal(err)
	}
	if lowest := sealLowest(t, swimming); lowest > -.1 {
		t.Fatalf("the swimming seal was pressed onto a ground: %v", lowest)
	}
}

func TestSealHideIsDarkOverAndPaleUnder(t *testing.T) {
	for name, form := range sealFormsForTest {
		hide := sealHide{pelt: form.pelt, outline: form.outline, seed: 407}
		var over, under float64
		for x := -.6; x <= .5; x += .01 {
			top, _ := form.outline.radial(x, 0)
			bottom, _ := form.outline.radial(x, math.Pi)
			over += hide.At(top).G
			under += hide.At(bottom).G
		}
		if over >= under {
			t.Errorf("%s: the back's green %v is not under the belly's %v",
				name, over, under)
		}
		if hide.At(maths.V(.1, .2, 0)) != hide.At(maths.V(.1, .2, 0)) {
			t.Errorf("%s: one point of the coat gave two colours", name)
		}
	}
}

// sealLong is a made seal's extent from its foremost to its hindmost
// point, in metres.
func sealLong(t *testing.T, kind Loft) float64 {
	t.Helper()
	made, err := kind.Build(407)
	if err != nil {
		t.Fatal(err)
	}
	return made.Bounds.Max.X - made.Bounds.Min.X
}

func TestMonkSealIsTheLongerSealAndPlain(t *testing.T) {
	monk, harbor := sealLong(t, sealBasking), sealLong(t, sealBanana)
	if monk < 1.1*harbor || monk > 1.35*harbor {
		t.Fatalf("the monk seal is %.2f m long by the harbor seal's "+
			"%.2f m; it is a fifth longer", monk, harbor)
	}
	if monkSealPelt.depth != 0 || harborSealPelt.depth == 0 {
		t.Fatal("the monk seal's coat is plain and the harbor seal's is not")
	}
}

func TestSealPlainPeltHasNoMarks(t *testing.T) {
	plain := harborSealPelt
	plain.depth = 0
	hide := sealHide{pelt: plain, outline: harborSealOutline, seed: 407}
	marked := sealHide{pelt: harborSealPelt, outline: harborSealOutline,
		seed: 407}
	differ := false
	for x := -.6; x <= .5; x += .01 {
		point, _ := harborSealOutline.radial(x, .3)
		if mark, ring := hide.marks(point, 1, 0); mark != 0 || ring != 0 {
			t.Fatalf("a plain coat is marked at %v", point)
		}
		if mark, _ := marked.marks(point, 1, 0); mark > 0 {
			differ = true
		}
	}
	if !differ {
		t.Fatal("the harbor seal's back has no marks along it")
	}
}
