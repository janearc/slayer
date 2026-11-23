package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// a spruce three times taller is grown, not magnified
// (2026-11-10: "when you scale the trees up their branches must not
// increase in proportion to their trunks, and they must have a
// proportionate amount of needles"). from the same seed it is three
// times as tall on a trunk three times as thick; its thickest branch is
// thinner than the short spruce's, as thick as its own length asks; its
// needles are the same shoots, unscaled, and as many more as its branches
// carry: three times the whorls, each with more branches by whorlFill,
// each branch longer by branchGrowth.
func TestATallerSpruceIsGrownNotMagnified(t *testing.T) {
	short, err := KindNamed("spruce")
	if err != nil {
		t.Fatal(err)
	}
	tall, err := Taller(short, 3)
	if err != nil {
		t.Fatal(err)
	}
	small, err := MakeWithComplexity(short, sow(51), "quick")
	if err != nil {
		t.Fatal(err)
	}
	big, err := MakeWithComplexity(tall, sow(51), "quick")
	if err != nil {
		t.Fatal(err)
	}
	high := func(made good) float64 {
		box := made.Bounds()
		return box.Max.Y - box.Min.Y
	}
	if ratio := high(big) / high(small); !(math.Abs(ratio-3) <= 0.15) {
		t.Errorf("three times taller is %.2f times as tall", ratio)
	}
	trunk := func(made good) float64 {
		return made.parts[0].shape.(tube).foot.InMetres()
	}
	if ratio := trunk(big) / trunk(small); !(math.Abs(ratio-3) <= 1e-9) {
		t.Errorf("three times taller has a trunk %.2f times as thick",
			ratio)
	}
	thickest := func(made good) float64 {
		most := 0.0
		for _, piece := range made.parts[1:] {
			if limb, ok := piece.shape.(tube); ok {
				most = math.Max(most, limb.foot.InMetres())
			}
		}
		return most
	}
	if thickest(big) >= thickest(small) {
		t.Errorf("three times taller has a branch %.3f m through, the "+
			"short one %.3f m", thickest(big), thickest(small))
	}
	smallShoots := small.groups[leaf].Members()
	bigShoots := big.groups[leaf].Members()
	for _, shoot := range bigShoots {
		if scaleOf(shoot.Space) != 1 {
			t.Fatalf("a shoot is magnified %v times", scaleOf(shoot.Space))
		}
	}
	want := math.Pow(3, 1+branchGrowth+whorlFill)
	ratio := float64(len(bigShoots)) / float64(len(smallShoots))
	if !(math.Abs(ratio-want) <= want/4) {
		t.Errorf("%d shoots on the tall spruce, %d on the short, %.1f "+
			"times as many, not about %.1f", len(bigShoots),
			len(smallShoots), ratio, want)
	}
}

// a grown tree's branch is as thick as its own length asks, not as its
// trunk is: two stems as tall, one twice as thick, carry the same
// branches, each own times its length through, wherever the thinner
// trunk allows it. by ninja's girth the stout trunk's would be twice as
// thick.
func TestAGrownBranchDoesNotThickenWithItsTrunk(t *testing.T) {
	set := spruceHabit()
	set.own = grownGirth
	up := mustWhorls(t, 20, 7, set)
	thin := size{length: measure.Metres(20), girth: measure.Metres(0.4)}
	stout := size{length: measure.Metres(20), girth: measure.Metres(0.8)}
	same, all := 0, 0
	for row := range up.rows() {
		for index := range up.across(row) {
			var got [2]float64
			for which, stem := range []size{thin, stout} {
				at, err := up.at(stem, row, index, sow(5))
				if err != nil {
					t.Fatal(err)
				}
				length := inScene(at.size.length)
				stemHere := inScene(stem.girth) *
					(1 - at.space.Origin.Y/inScene(stem.length))
				want := math.Min(set.own*length, set.girth*stemHere)
				got[which] = inScene(at.size.girth)
				if !(math.Abs(got[which]-want) <= close) {
					t.Errorf("whorl %d branch %d is %v through, not %v",
						row, index, got[which], want)
				}
			}
			all++
			if math.Abs(got[0]-got[1]) < close {
				same++
			}
		}
	}
	if same < all*3/4 {
		t.Errorf("%d of %d branches are as thick on the stout trunk as "+
			"on the thin one", same, all)
	}
}

// a branched spruce's limbs carry branches of their own, each in
// foliage (2026-11-10: "there needs to be much more foliage on the
// branches... the branches themselves should also have branches"): grown
// three times taller and branched, it stands on the same trunk as
// unbranched and carries at least three times the shoots, all at their
// own size. A side branch is foliage on its own axis, its wood the
// shoots' twigs, as the norway's branchlets are.
func TestABranchedSpruceCarriesSideBranches(t *testing.T) {
	spruce, err := KindNamed("spruce")
	if err != nil {
		t.Fatal(err)
	}
	tall, err := Taller(spruce, 3)
	if err != nil {
		t.Fatal(err)
	}
	full, err := Branched(tall)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := MakeWithComplexity(tall, sow(51), "quick")
	if err != nil {
		t.Fatal(err)
	}
	fanned, err := MakeWithComplexity(full, sow(51), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if bare.parts[0].shape.(tube).length != fanned.parts[0].shape.(tube).length {
		t.Errorf("branched, the trunk is %v, not %v",
			fanned.parts[0].shape.(tube).length, bare.parts[0].shape.(tube).length)
	}
	shoots := fanned.groups[leaf].Members()
	for _, shoot := range shoots {
		if scaleOf(shoot.Space) != 1 {
			t.Fatalf("a shoot is magnified %v times", scaleOf(shoot.Space))
		}
	}
	if few := len(bare.groups[leaf].Members()); len(shoots) < 3*few {
		t.Errorf("branched, %d shoots; unbranched %d", len(shoots), few)
	}
	if _, err := Branched(mustKind(t, "log")); !errors.Is(err, errBranched) {
		t.Errorf("a branched log: %v, not %v", err, errBranched)
	}
}

// a grown tree's limb flows (2026-11-10: "they should be
// non-straight, but they should flow and twist just slightly"): in
// grownPieces pieces it turns at no joint as far as the three-piece limb
// kinks, and it wanders out of the plane it set out in, which the
// three-piece limb never leaves.
func TestAGrownLimbFlows(t *testing.T) {
	way := maths.V(1, 0.2, 0.3).Unit()
	space, err := spaceAlong(maths.V(2, 5, -1), way)
	if err != nil {
		t.Fatal(err)
	}
	on := place{space: space, size: size{length: measure.Metres(10),
		girth: measure.Metres(0.1)}}
	bent := func(pieces int, droop, wander float64) (sharpest, out float64) {
		widths := girths(1, 0.75, 0.5, 0.25)
		bark, err := sleeved(resampled(widths, pieces+1),
			5, body)
		if err != nil {
			t.Fatal(err)
		}
		made, err := newLimb(pieces, 1, droop, bark)
		if err != nil {
			t.Fatal(err)
		}
		made.wander = wander
		runs, err := made.runs(on)
		if err != nil {
			t.Fatal(err)
		}
		plane := way.Cross(maths.Up).Unit()
		for at, run := range runs {
			along := run.space.Orientation.Out(maths.YAxis)
			component := math.Abs(along.Dot(plane))
			out = math.Max(out, component)
			if at > 0 {
				before := runs[at-1].space.Orientation.Out(maths.YAxis)
				alignment := before.Dot(along)
				turn := math.Acos(math.Min(1, alignment))
				sharpest = math.Max(sharpest, turn)
			}
		}
		return sharpest, out
	}
	kinked, flat := bent(3, 0.22, 0)
	flowed, wandered := bent(grownPieces, 0.22*2/float64(grownPieces-1),
		grownWander)
	if flowed >= kinked {
		t.Errorf("the grown limb turns %.3f at a joint, the three-piece "+
			"one %.3f", flowed, kinked)
	}
	if flat > 1e-9 || wandered < 0.01 {
		t.Errorf("out of their plane: the three-piece limb %.4f, the "+
			"grown one %.4f", flat, wandered)
	}
}

// mustKind is the kind of a name, or the test stops.
func mustKind(t *testing.T, name string) kind {
	t.Helper()
	row, err := KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

// only a whorled tree grows taller, by a number of times above nought.
func TestWhatCannotGrowTallerIsRefused(t *testing.T) {
	spruce, err := KindNamed("spruce")
	if err != nil {
		t.Fatal(err)
	}
	for _, times := range []float64{0, -2, math.Inf(1), math.NaN()} {
		if _, err := Taller(spruce, times); !errors.Is(err, errTaller) {
			t.Errorf("a spruce %v times taller: %v, not %v", times, err,
				errTaller)
		}
	}
	log, err := KindNamed("log")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Taller(log, 2); !errors.Is(err, errTaller) {
		t.Errorf("a log twice as tall: %v, not %v", err, errTaller)
	}
}
