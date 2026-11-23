package manufacture

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

func TestWalrusColumnHasItsCounts(t *testing.T) {
	body := walrusSkeleton()
	for prefix, want := range map[string]int{
		"cervical ": walrusCervicalCount, "thoracic ": walrusThoracicCount,
		"lumbar ": walrusLumbarCount, "sacral ": walrusSacralCount,
		"caudal ": walrusCaudalCount,
	} {
		if got := body.tree.Count(prefix); got != want {
			t.Errorf("%s: %d vertebrae, want %d", prefix, got, want)
		}
	}
	if walrusThoracicCount != 14 || walrusLumbarCount != 6 {
		t.Fatal("the sourced counts are fourteen thoracic and six lumbar")
	}
}

func TestWalrusHasARibPairToEachThoracicVertebra(t *testing.T) {
	body := walrusSkeleton()
	for number := 1; number <= walrusThoracicCount; number++ {
		for _, side := range []string{"left", "right"} {
			at, there := body.tree.Find(fmt.Sprintf("rib %d %s", number, side))
			if !there {
				t.Fatalf("no rib %d %s", number, side)
			}
			parent := body.tree.Bone(body.tree.Bone(at).Parent()).Name()
			if parent != fmt.Sprintf("thoracic %d", number) {
				t.Fatalf("rib %d %s hangs from %s", number, side, parent)
			}
		}
	}
	if got := body.tree.Count("rib "); got != 2*walrusThoracicCount {
		t.Fatalf("%d ribs", got)
	}
}

func TestWalrusFlippersAreHandsOfBones(t *testing.T) {
	body := walrusSkeleton()
	for _, side := range []string{"left", "right"} {
		for _, name := range []string{"humerus", "radius", "ulna", "scapholunar",
			"pisiform", "unciform", "femur", "tibia", "fibula", "astragalus",
			"calcaneus", "cuboid"} {
			if _, there := body.tree.Find(name + " " + side); !there {
				t.Errorf("no %s %s", name, side)
			}
		}
		for _, limb := range []string{"metacarpal", "metatarsal"} {
			for ray := 1; ray <= walrusDigitCount; ray++ {
				name := fmt.Sprintf("%s %d %s", limb, ray, side)
				if _, there := body.tree.Find(name); !there {
					t.Errorf("no %s", name)
				}
			}
		}
	}
	for _, kind := range []string{"fore", "hind"} {
		if got := body.tree.Count(kind + " phalanx "); got != 2*14 {
			t.Errorf("%s: %d phalanges on both sides, want 28", kind, got)
		}
	}
}

func TestWalrusFirstDigitHasTwoPhalanges(t *testing.T) {
	body := walrusSkeleton()
	if _, there := body.tree.Find("fore phalanx 1.3 right"); there {
		t.Fatal("the first digit has a third phalanx")
	}
	if _, there := body.tree.Find("fore phalanx 2.3 right"); !there {
		t.Fatal("the second digit has no third phalanx")
	}
}

func TestWalrusTuskIsItsLength(t *testing.T) {
	frame := walrusSkullFrameAt(maths.V(1.2, .35, 0))
	canine := walrusCanine{frame: frame, side: 1}
	length := 0.
	const steps = 1000
	for i := 0; i < steps; i++ {
		ahead, behind := float64(i+1)/steps, float64(i)/steps
		length += canine.at(ahead).Sub(canine.at(behind)).Len()
	}
	if length < walrusTuskLength*.98 || length > walrusTuskLength*1.02 {
		t.Fatalf("tusk %.3f m long, want %.3f", length, walrusTuskLength)
	}
	if walrusTuskExposed != .80 {
		t.Fatal("the sourced exposed share is eighty percent")
	}
}

func TestWalrusTusksClearTheGroundAtRest(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		if tip := walrusCanineOf(side).at(1); tip.Y <= walrusFloor {
			t.Fatalf("side %v: the tusk's tip is at %.3f, in the ground", side, tip.Y)
		}
	}
}

func TestWalrusSkeletonDrawsClosed(t *testing.T) {
	parts, err := WalrusFresh5Skeleton()
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("%d parts", len(parts))
	}
}

func TestWalrusRestAnkleIsTurnedForwardWithinItsReach(t *testing.T) {
	body := walrusSkeleton()
	frames := body.tree.Frames(skeleton.Pose{})
	for _, side := range []string{"left", "right"} {
		turn := walrusAnkleTurn(body, frames, side)
		if turn < 1.5 || turn > walrusAnkleForward {
			t.Fatalf("%s ankle turned %.2f rad", side, turn)
		}
	}
}

func TestWalrusSkeletonLaysOutWithoutAFault(t *testing.T) {
	if fault := walrusSkeleton().fault; fault != nil {
		t.Fatal(fault)
	}
}

func TestWalrusBoneOfOneNameTwiceIsAFault(t *testing.T) {
	body := walrusSkeleton()
	from, to := maths.V(0, 0, 0), maths.V(.1, 0, 0)
	body.add("skull", "", from, to, .01, skeleton.NewRod(from, to, .01, .6))
	if body.fault == nil {
		t.Fatal("a second skull was taken")
	}
}
