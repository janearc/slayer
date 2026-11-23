package manufacture

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// orcaTestSkeleton is the rest skeleton, or the test stops.
func orcaTestSkeleton(test *testing.T) *skeleton.Skeleton {
	test.Helper()
	bones, err := orcaSkeleton()
	if err != nil {
		test.Fatal(err)
	}
	return bones
}

func TestTheOrcasColumnHasItsRegions(test *testing.T) {
	bones := orcaTestSkeleton(test)
	want := map[string]int{"C": 7, "T": 10, "L": 12, "Ca": 24}
	for region, count := range want {
		for n := 1; n <= count; n++ {
			if _, ok := bones.Find(fmt.Sprintf("%s%d", region, n)); !ok {
				test.Fatalf("no %s%d", region, n)
			}
		}
		if _, ok := bones.Find(fmt.Sprintf("%s%d", region,
			count+1)); ok {
			test.Fatalf("%s has more than %d", region, count)
		}
	}
	built := 0
	for i := 0; i < bones.Len(); i++ {
		if region, _ := orcaVertebraOf(bones.Bone(i).Name()); region != "" {
			built++
		}
	}
	if built != 53 {
		test.Fatalf("%d vertebrae built, not S2's 53", built)
	}
}

func TestTheOrcaHasTenRibsASideAndFiveReachTheSternum(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for _, side := range orcaSides {
		for n := 1; n <= 10; n++ {
			if _, ok := bones.Find(fmt.Sprintf("rib%d-%s", n,
				side.name)); !ok {
				test.Fatalf("no rib %d on the %s", n, side.name)
			}
			_, sternal := bones.Find(fmt.Sprintf("rib%d-%s-sternal", n,
				side.name))
			if sternal != (n <= 5) {
				test.Fatalf("rib %d on the %s: sternal %v", n, side.name,
					sternal)
			}
		}
	}
}

func TestTheFlipperIsAHandOfTheSpecimensDigits(test *testing.T) {
	bones := orcaTestSkeleton(test)
	phalanges := []int{1, 4, 3, 2, 1}
	for _, side := range []string{"right-", "left-"} {
		for _, name := range []string{"humerus", "radius", "ulna"} {
			if _, ok := bones.Find(side + name); !ok {
				test.Fatalf("no %s%s", side, name)
			}
		}
		for n, want := range phalanges {
			prefix := fmt.Sprintf("%sphalanx%d-", side, n+1)
			if got := bones.Count(prefix); got != want {
				test.Fatalf("%s: %d phalanges, want %d", prefix, got, want)
			}
		}
	}
}

// TestEveryBoneLiesInsideTheSkin checks each bone's head, middle and
// tail lie inside the skin by at least the bone's half thickness there.
// It reads the bone's own thickness, not the drawn form's sizes, which
// for the skull, jaws, ribs and blades differ.
func TestEveryBoneLiesInsideTheSkin(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		head, tail := bone.Thickness()
		for _, share := range []float64{0, .5, 1} {
			p := bone.Head().Add(bone.Tail().Sub(bone.Head()).Scale(share))
			thick := head + share*(tail-head)
			if d := orcaField(p) + thick; d >= 0 {
				test.Errorf("%s comes through the skin at %v by %.3f m",
					bone.Name(), p, d)
			}
		}
	}
}

func TestTheSkeletonIsTheSameEachSide(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		mirror := orcaMirrorName(bone.Name())
		at, ok := bones.Find(mirror)
		if !ok {
			test.Fatalf("%s has no mirror %s", bone.Name(), mirror)
		}
		mirrored := orcaMirror(bone.Head(), -1)
		if !bones.Bone(at).Head().Near(mirrored, 1e-9) {
			test.Fatalf("%s and %s are not mirrored", bone.Name(), mirror)
		}
	}
}

func TestEachHalfOfEachJawHasTwelveTeeth(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for _, side := range orcaSides {
		for _, row := range []string{"lower", "upper"} {
			prefix := side.name + "-" + row + "-tooth-"
			if got := bones.Count(prefix); got != 12 {
				test.Fatalf("%s: %d teeth", prefix, got)
			}
		}
	}
}

func TestTheSkullRibsAndJawsHaveForms(test *testing.T) {
	bones := orcaTestSkeleton(test)
	forms := orcaForms(bones)
	for _, name := range []string{"cranium", "rostrum", "mandible-right",
		"mandible-left", "rib1-right", "rib10-left", "scapula-right",
		"sternum", "T3-spine", "L2-left", "Ca5-chevron", "Ca1"} {
		if forms[name] == nil {
			test.Fatalf("%s has no form", name)
		}
	}
	if _, ok := forms["rib4-left"].(skeleton.Arc); !ok {
		test.Fatal("a rib is not an arc")
	}
}

func TestTheDrawnSkeletonIsAMesh(test *testing.T) {
	parts, err := OrcaClean05Bones(maths.Local())
	if err != nil {
		test.Fatal(err)
	}
	if len(parts) != 1 {
		test.Fatalf("%d parts", len(parts))
	}
}

// Anti-test: the teeth keep the spindle's cone and are given no form.

func TestTheTeethAreNotRods(test *testing.T) {
	forms := orcaForms(orcaTestSkeleton(test))
	if forms["right-upper-tooth-1"] != nil {
		test.Fatal("a tooth has a form")
	}
}

// orcaMirrorName is the name of the bone on the other side from name;
// a bone on the midline is its own mirror.
func orcaMirrorName(name string) string {
	switch {
	case strings.Contains(name, "right"):
		return strings.Replace(name, "right", "left", 1)
	case strings.Contains(name, "left"):
		return strings.Replace(name, "left", "right", 1)
	}
	return name
}

// Anti-tests: what the skeleton must not have.

func TestThereIsNoBoneInTheDorsalFinOrTheFlukes(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		for _, p := range []maths.Vec{bone.Head(), bone.Tail()} {
			s := orcaTrunkProfile().at(p.X)
			if p.Y > s.centre+s.high {
				test.Errorf("%s stands above the back, in the fin",
					bone.Name())
			}
			if p.X < orcaFlukeLead && math.Abs(p.Z) > s.wide+.02 {
				test.Errorf("%s lies out in a fluke", bone.Name())
			}
		}
	}
}

func TestThereIsNoHindLimb(test *testing.T) {
	bones := orcaTestSkeleton(test)
	for _, name := range []string{"femur", "tibia", "fibula", "tarsal"} {
		if bones.Count(name) != 0 || bones.Count("right-"+name) != 0 {
			test.Fatalf("a %s", name)
		}
	}
	for _, side := range orcaSides {
		at, ok := bones.Find("pelvis-" + side.name)
		if !ok {
			test.Fatalf("no pelvic vestige on the %s", side.name)
		}
		if bones.Bone(at).Parent() != -1 {
			test.Fatal("a pelvic vestige is jointed to something")
		}
	}
}

func TestTheSkullIsNotPartOfTheColumn(test *testing.T) {
	bones := orcaTestSkeleton(test)
	atlas, hasAtlas := bones.Find("C1")
	cranium, hasCranium := bones.Find("cranium")
	if !hasAtlas || !hasCranium {
		test.Fatal("no atlas or no cranium")
	}
	if bones.Bone(atlas).Parent() != cranium {
		test.Fatal("the atlas is not jointed to the skull")
	}
	if region, _ := orcaVertebraOf("cranium"); region != "" {
		test.Fatal("the cranium is read as a vertebra")
	}
}
