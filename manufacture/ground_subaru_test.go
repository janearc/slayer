package manufacture

import (
	"math"
	"testing"
)

// TestSubaruExteriorHasSteppedSilhouette checks the actual shells, rather
// than blessing an image digest or merely re-reading the authored settings.
func TestSubaruExteriorHasSteppedSilhouette(t *testing.T) {
	row := groundRows()["subaru"].(groundObservatory)
	assembly := &instrumentAssembly{fine: cut{around: 48},
		object: good{dressed: groundLooks()}}
	row.subaruEnclosure(assembly)
	if assembly.trouble != nil {
		t.Fatal(assembly.trouble)
	}
	radii := make([]float64, 3)
	for index, part := range assembly.object.parts[:3] {
		for _, face := range part.skin() {
			for _, point := range face {
				radii[index] = math.Max(radii[index], math.Hypot(point.X, point.Z))
			}
		}
	}
	if !(radii[2] < radii[0] && radii[0] < radii[1] && radii[1] < 20) {
		t.Fatal("lost upper drum, white base and projecting belt:", radii)
	}
	// All enclosure pieces above the service tower must leave the
	// 50-degree observing sector empty, including shutter-edge housings.
	for _, part := range assembly.object.parts {
		for _, face := range part.skin() {
			for _, point := range face {
				if point.Y > 25 && math.Abs(math.Atan2(point.X, -point.Z)) < 25*math.Pi/180-1e-9 {
					t.Fatal("exterior closes observing slit:", point)
				}
			}
		}
	}
	white, metal := assembly.object.dressed[body], assembly.object.dressed[spaceFoil]
	if white.Colour.R <= metal.Colour.R {
		t.Fatal("white base no longer distinct from cladding")
	}
}
