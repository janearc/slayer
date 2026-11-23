package manufacture

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
)

func sealBonesForTest(t *testing.T) *sealBones {
	t.Helper()
	made, err := harborSealBones()
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// sealRegion is the region a bone's name begins with: its letters
// before any digit or dash.
func sealRegion(name string) string {
	return strings.TrimRight(strings.SplitN(name, "-", 2)[0], "0123456789")
}

func TestSealColumnHasItsRegionsCounts(t *testing.T) {
	counts := map[string]int{}
	for _, name := range sealBonesForTest(t).names() {
		counts[sealRegion(name)]++
	}
	want := map[string]int{"neck": sealCervicals,
		"thoracic": sealThoracics, "lumbar": sealLumbars,
		"caudal": sealCaudals, "sacral": 1, "skull": 1,
		"sternebra": sealSternebrae, "rib": 2 * sealThoracics,
		"tooth": 2 * (len(sealUpperTeeth) + len(sealLowerTeeth))}
	if want["tooth"] != 34 {
		t.Fatalf("the jaws hold %d teeth, not the formula's 34", want["tooth"])
	}
	for region, count := range want {
		if counts[region] != count {
			t.Fatalf("%s: %d bones, want %d", region, counts[region], count)
		}
	}
}

func TestEachFlipperHasFiveDigitsAndTheThumbTwoPhalanges(t *testing.T) {
	made := sealBonesForTest(t)
	for _, side := range []string{"left", "right"} {
		for digit := 1; digit <= 5; digit++ {
			last := 3
			if digit == 1 {
				last = 2
			}
			fore := fmt.Sprintf("fore-%s-phalanx%d-", side, digit)
			if _, _, found := made.ends(fmt.Sprint(fore, last)); !found {
				t.Fatalf("%s has no phalanx %d", fore, last)
			}
			if _, _, found := made.ends(fmt.Sprint(fore, last+1)); found {
				t.Fatalf("%s has a phalanx past %d", fore, last)
			}
			hind := fmt.Sprintf("hind-%s-digit%d-", side, digit)
			if _, _, found := made.ends(fmt.Sprint(hind, last+1)); !found {
				t.Fatalf("%s has no bone %d", hind, last+1)
			}
		}
	}
}

func TestSealForelimbBonesAreTheSpecimensLengths(t *testing.T) {
	made := sealBonesForTest(t)
	want := map[string]float64{"fore-right-scapula": sealScapulaLength,
		"fore-right-humerus": sealHumerusLength,
		"fore-left-radius":   sealRadiusLength}
	for name, length := range want {
		head, tail, found := made.ends(name)
		if !found || math.Abs(tail.Sub(head).Len()-length) > 1e-9 {
			t.Fatalf("%s is %v long, want %v", name, tail.Sub(head).Len(),
				length)
		}
	}
}

func TestSealColumnAndRibsSitInsideTheRestSkin(t *testing.T) {
	made := sealBonesForTest(t)
	for _, name := range made.names() {
		switch sealRegion(name) {
		case "thoracic", "lumbar", "rib", "neck":
		default:
			continue
		}
		head, _, _ := made.ends(name)
		section := sealProfileAt(head.X)
		up := (head.Y - section.y) / section.height
		across := head.Z / section.width
		if up*up+across*across >= 1 {
			t.Fatalf("%s starts outside the skin at %v", name, head)
		}
	}
}

// sealHindTip is the posed X of the tip of the right hind flipper's
// first digit and of the ankle it hangs from, or the pose's refusal.
func sealHindTip(made *sealBones, flex, bend, twist float64) (float64,
	float64, error,
) {
	pose, err := made.joints.Turned(made.bones, skeleton.Pose{},
		"hind-right-astragalus", flex, bend, twist)
	if err != nil {
		return 0, 0, err
	}
	frames := made.bones.Frames(pose)
	tipAt, _ := made.bones.Find("hind-right-digit1-3")
	ankleAt, _ := made.bones.Find("hind-right-astragalus")
	tip := frames[tipAt].Apply(made.bones.Bone(tipAt).Tail())
	ankle := frames[ankleAt].Apply(made.bones.Bone(ankleAt).Head())
	return tip.X, ankle.X, nil
}

func TestSealHindFlippersCannotTurnForward(t *testing.T) {
	made := sealBonesForTest(t)
	for _, flex := range [...]float64{-.35, 0, .35} {
		for _, bend := range [...]float64{-.5, 0, .5} {
			for _, twist := range [...]float64{-.8, 0, .8} {
				tip, ankle, err := sealHindTip(made, flex, bend, twist)
				if err != nil {
					t.Fatal(err)
				}
				if tip >= ankle {
					t.Fatalf("flex %v bend %v twist %v put the flipper "+
						"forward of the ankle", flex, bend, twist)
				}
			}
		}
	}
}

func TestSealAnkleRefusesATurnPastItsBones(t *testing.T) {
	made := sealBonesForTest(t)
	if _, _, err := sealHindTip(made, math.Pi/2, 0, 0); err == nil {
		t.Fatal("a hind flipper turned forward under the body")
	}
}

func TestSealBonesKindMakesOneClosedSolid(t *testing.T) {
	kind, err := KindNamed("harbor-seal-bones")
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(407), "quick")
	if err != nil {
		t.Fatal(err)
	}
	if made.PartCount() != 1 {
		t.Fatalf("%d parts, want the skeleton as one", made.PartCount())
	}
}

func TestSealBonesKindNeedsASeed(t *testing.T) {
	if _, err := (harborSealBonesKind{}).made(seed{}); err == nil {
		t.Fatal("an unsown seed made a skeleton")
	}
}
