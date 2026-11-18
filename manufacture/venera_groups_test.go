package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// veneraDetails is every detail the catalogue's menu gives the lander.
var veneraDetails = []string{"hull", "clumps", "quick", "demo"}

// veneraBoxNear is how near a shared group's box must be to its parts'.
const veneraBoxNear = 1e-9 // picked: sums of a few lengths, to rounding

// veneraBoxLoose is how much longer than its parts' box a group's box may
// run across, as a share of it.
const veneraBoxLoose = 0.25 // picked: a turned member box is a little bigger

// veneraStood is the lander's parts at a detail as drawn, none shared.
func veneraStood(detail string) []part {
	return veneraRead.stood(detail).object.parts
}

// veneraMadeAt is the lander made at a detail, or the test fails.
func veneraMadeAt(t *testing.T, detail string) good {
	t.Helper()
	made, err := veneraLander{}.madeAt(Sow(5), detail)
	if err != nil {
		t.Fatalf("the lander at %s: %v", detail, err)
	}
	return made
}

// veneraDrawn is how many faces a made lander draws, each instance of a
// shared part counted, matter by matter.
func veneraDrawn(made good) map[slot]int {
	drawn := map[slot]int{}
	for of, skin := range made.skins() {
		drawn[of] += len(skin)
	}
	for of, group := range made.groups {
		drawn[of] += group.FaceCount()
	}
	return drawn
}

// The lander makes at every detail its menu gives, and at each the faces
// it draws, counting every instance, are the faces of its parts: nothing
// was dropped to fit the limit. The demo flattens under the limit, and
// the faces stored, before and after, are written to the log.
func TestTheVeneraLanderKeepsEveryFaceAtEveryDetail(t *testing.T) {
	for _, detail := range veneraDetails {
		before := map[slot]int{}
		total := 0
		for _, piece := range veneraStood(detail) {
			faces := len(piece.shape.skin(piece.fine))
			before[piece.of] += faces
			total += faces
		}
		made := veneraMadeAt(t, detail)
		if _, err := made.Things(maths.Local()); err != nil {
			t.Fatalf("the lander at %s does not flatten: %v", detail, err)
		}
		drawn := 0
		for of, faces := range veneraDrawn(made) {
			drawn += faces
			if faces != before[of] {
				t.Errorf("%s: %s draws %d faces, its parts have %d",
					detail, veneraNames[of], faces, before[of])
			}
		}
		if drawn != total {
			t.Errorf("%s draws %d faces, its parts have %d", detail, drawn,
				total)
		}
		t.Logf("%s: %d faces before, %d stored after, %d drawn", detail,
			total, made.faces(), drawn)
	}
}

// The demo is the detail that was too fine to flatten: its parts hold
// more than the limit, and shared, its flattened faces are under it and
// its stored faces are fewer than its drawn ones.
func TestTheVeneraDemoFlattensOnlyBecauseItsRepeatsAreShared(t *testing.T) {
	total := 0
	for _, piece := range veneraStood("demo") {
		total += len(piece.shape.skin(piece.fine))
	}
	if total <= mostFaces {
		t.Fatalf("the demo's parts hold %d faces, no more than %d: "+
			"nothing needed sharing", total, mostFaces)
	}
	made := veneraMadeAt(t, "demo")
	flattened := 0
	for _, skin := range made.skins() {
		flattened += len(skin)
	}
	if flattened > mostFaces {
		t.Errorf("the demo flattens %d faces, over %d", flattened, mostFaces)
	}
	if made.faces() >= total {
		t.Errorf("the demo stores %d faces, no fewer than %d drawn",
			made.faces(), total)
	}
}

// A group holds flat meshes alone, so a rounded look stays whole parts
// and shaded smooth, and a look of one part has nothing to share: the
// ring, the module and the aerial are never groups, and every group is a
// look of more than one part.
func TestTheVeneraLanderSharesOnlyFlatRepeats(t *testing.T) {
	for _, detail := range veneraDetails {
		made := veneraMadeAt(t, detail)
		counts := map[slot]int{}
		for _, piece := range veneraStood(detail) {
			counts[piece.of]++
		}
		for _, rounded := range []slot{veneraRing, veneraModule,
			veneraAerial} {
			if made.groups[rounded] != nil {
				t.Errorf("%s: the %s is shared", detail, veneraNames[rounded])
			}
		}
		for of := range made.groups {
			if counts[of] < 2 {
				t.Errorf("%s: the %s is shared, of %d part", detail,
					veneraNames[of], counts[of])
			}
		}
		for _, of := range []slot{veneraBolt, veneraHole, veneraLeg} {
			if made.groups[of] == nil {
				t.Errorf("%s: the %s repeats and is not shared", detail,
					veneraNames[of])
			}
		}
	}
}

// Sharing moves nothing: each shared look's group has its parts inside
// its box, and the box is no bigger than the instance boxes around them
// need. A group's box is the box of each member's turned box, so it may be
// looser than the parts', never tighter.
func TestTheVeneraSharedLooksHoldWhereTheirPartsDid(t *testing.T) {
	made := veneraMadeAt(t, "quick")
	for of, group := range made.groups {
		var own []part
		for _, piece := range veneraStood("quick") {
			if piece.of == of {
				own = append(own, piece)
			}
		}
		want, got := partBounds(own), group.Bounds()
		inside := got.Min.X <= want.Min.X+veneraBoxNear &&
			got.Min.Y <= want.Min.Y+veneraBoxNear &&
			got.Min.Z <= want.Min.Z+veneraBoxNear &&
			got.Max.X >= want.Max.X-veneraBoxNear &&
			got.Max.Y >= want.Max.Y-veneraBoxNear &&
			got.Max.Z >= want.Max.Z-veneraBoxNear
		slack := veneraBoxLoose * (want.Max.Sub(want.Min).Len())
		loose := got.Max.Sub(got.Min).Len() - want.Max.Sub(want.Min).Len()
		if !inside || loose > slack {
			t.Errorf("the %s's group reaches %+v, its parts %+v",
				veneraNames[of], got, want)
		}
	}
}
