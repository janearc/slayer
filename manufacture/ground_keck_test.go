package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestKeckFrameLeavesThePrimaryClear checks actual pale-frame vertices
// around the primary, plus the depth that distinguishes the secondary cage.
func TestKeckFrameLeavesThePrimaryClear(t *testing.T) {
	row := groundRows()["keck-i"].(groundObservatory)
	assembly := &instrumentAssembly{fine: cut{around: 48}, object: good{dressed: groundLooks()}}
	row.keckTube(assembly, maths.Local(), 5.6, 15)
	if assembly.trouble != nil {
		t.Fatal(assembly.trouble)
	}
	lowCage, highCage := false, false
	diagonals := 0
	for _, part := range assembly.object.parts {
		direction := part.space.Orientation.Y
		if math.Abs(direction.Y) > 0.5 && math.Abs(direction.Y) < 0.99 {
			diagonals++
		}
		for _, face := range part.skin() {
			for _, p := range face {
				radius := math.Hypot(p.X, p.Z)
				if p.Y < 0.5 && radius < 5 {
					t.Fatal("frame covers primary aperture", p)
				}
				if radius < 1.2 {
					lowCage = lowCage || p.Y > 12.5 && p.Y < 13
					highCage = highCage || p.Y > 14.8
				}
			}
		}
	}
	if diagonals < 16 || !lowCage || !highCage {
		t.Fatal("missing triangulation or secondary depth", diagonals, lowCage, highCage)
	}
}

// TestKeckOpticsFitInsideTheDome checks the curved clearance, not just the
// rectangular envelope. An upper ring can pass a box check and pierce the dome.
func TestKeckOpticsFitInsideTheDome(t *testing.T) {
	row := groundRows()["keck-i"].(groundObservatory)
	for _, detail := range Complexities() {
		assembly := &instrumentAssembly{fine: cut{around: int(groundValue("sides-" + detail))},
			object: good{dressed: groundLooks()}}
		pose := row.opticalFrame()
		row.optics(assembly, pose)
		row.opticsDetails(assembly, pose)
		if assembly.trouble != nil {
			t.Fatal(assembly.trouble)
		}
		// Independently specified inner envelope of the current authored shell.
		centre := maths.V(0, 13, 0)
		const inside = 14.625
		farthest := 0.0
		for _, part := range assembly.object.parts {
			for _, face := range part.skin() {
				for _, point := range face {
					farthest = math.Max(farthest, point.Sub(centre).Len())
				}
			}
		}
		if farthest >= inside {
			t.Errorf("%s optical radius %.6f exceeds inner dome %.6f", detail, farthest, inside)
		}
	}
}

// TestKeckExteriorLeavesTheSlitClear checks the visible exterior vertices,
// including the tracks near the crown where angular clearance is narrowest.
func TestKeckExteriorLeavesTheSlitClear(t *testing.T) {
	for _, name := range []string{"keck-i", "keck-ii"} {
		row := groundRows()[name].(groundObservatory)
		assembly := &instrumentAssembly{fine: cut{around: 48},
			object: good{dressed: groundLooks()}}
		row.keckExterior(assembly)
		if assembly.trouble != nil {
			t.Fatal(assembly.trouble)
		}
		if len(assembly.object.parts) < 40 {
			t.Fatal("missing exterior members", name)
		}
		for _, part := range assembly.object.parts {
			for _, face := range part.skin() {
				for _, point := range face {
					if math.Abs(point.X) > 15 || math.Abs(point.Z) > 15 || point.Y < 0 || point.Y > 28 {
						t.Fatal("exterior escaped physical bounds", name, point)
					}
					if point.Y > 13 && math.Abs(point.X) < 4.5 && point.Z < 0 {
						t.Fatal("exterior crosses observing slit", name, point)
					}
				}
			}
		}
	}
}

// TestKeckPairRetainsMatchingGeometry keeps both named recipes identical
// before placement. Their shared pointing must not conceal a model mismatch.
func TestKeckPairRetainsMatchingGeometry(t *testing.T) {
	var digests []string
	for _, name := range []string{"keck-i", "keck-ii"} {
		made, err := groundRows()[name].(groundObservatory).madeAt(Sow(5), "quick")
		if err != nil {
			t.Fatal(err)
		}
		digests = append(digests, referenceDigest(t, made))
		if made.Bounds().Min != maths.V(-15, 0, -15) {
			t.Fatal("changed pair origin")
		}
	}
	if digests[0] != digests[1] {
		t.Fatal("the two Keck recipes differ before placement")
	}
}

// TestKeckOpeningStaysParallel checks the actual cut wall over its useful
// height, and distinguishes the curved lower body from a tall straight drum.
func TestKeckOpeningStaysParallel(t *testing.T) {
	shell := keckShell{radius: 15, thick: 0.375, centre: 13, bottom: 3, top: 28, halfWidth: 4.5}
	count := 0
	bulge := false
	for _, face := range shell.skin(cut{around: 48}) {
		for _, p := range face {
			if p.Y > 12 && p.Y < 25 && p.Z < 0 {
				if math.Abs(p.X) < 4.5-1e-9 {
					t.Fatal("shell intrudes into rectangular aperture", p)
				}
				if math.Abs(math.Abs(p.X)-4.5) < 1e-9 {
					count++
				}
			}
			if p.Y > 7 && p.Y < 12 && math.Abs(p.X) > 13 {
				bulge = true
			}
		}
	}
	if count < 20 || !bulge {
		t.Fatal("lost parallel edges or curved lower body", count, bulge)
	}
}

// TestKeckGeneratedEnvelope reports the complete vertex envelope at each
// representation, so an early offending face cannot hide the true maximum.
func TestKeckGeneratedEnvelope(t *testing.T) {
	row := groundRows()["keck-i"].(groundObservatory)
	for _, detail := range Complexities() {
		made, err := row.madeAt(Sow(5), detail)
		if err != nil {
			t.Fatal(err)
		}
		high := 0.0
		for _, part := range made.parts {
			for _, face := range part.skin() {
				for _, point := range face {
					high = math.Max(high, point.Y)
				}
			}
		}
		if high > 28+1e-9 {
			t.Errorf("%s highest vertex %g", detail, high)
		}
	}
}
