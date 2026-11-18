package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestSurveyCameraSitsBetweenMirrors rejects the former camera-through-M2
// arrangement using actual vertices in the optical coordinate frame.
func TestSurveyCameraSitsBetweenMirrors(t *testing.T) {
	row := surveyGround()
	for _, detail := range Complexities() {
		made, err := row.madeAt(Sow(5), detail)
		if err != nil {
			t.Fatal(err)
		}
		pose := row.opticalFrame()
		cameraLow, cameraHigh, secondaryLow := math.Inf(1), math.Inf(-1), math.Inf(1)
		for _, part := range made.parts {
			if part.of != surveyCamera && part.of != spaceSecondary {
				continue
			}
			for _, face := range part.skin() {
				for _, p := range face {
					y := pose.In(p).Y
					if part.of == surveyCamera {
						cameraLow = min(cameraLow, y)
						cameraHigh = math.Max(cameraHigh, y)
					} else {
						secondaryLow = min(secondaryLow, y)
					}
				}
			}
		}
		if !(cameraLow > 1 && cameraHigh < secondaryLow-0.5 && secondaryLow < 6) {
			t.Fatalf("%s camera [%g,%g], secondary %g", detail, cameraLow, cameraHigh, secondaryLow)
		}
	}
}

// TestSurveyFrameHasDiagonalStaysAndClearPrimary catches a generic straight
// cage and braces placed across the primary aperture.
func TestSurveyFrameHasDiagonalStaysAndClearPrimary(t *testing.T) {
	row := surveyGround()
	assembly := &instrumentAssembly{fine: cut{around: 48}, object: good{dressed: groundLooks()}}
	row.survey.frame(row, assembly, maths.Local(), 4.704, 5.46)
	if assembly.trouble != nil {
		t.Fatal(assembly.trouble)
	}
	diagonal := 0
	for _, part := range assembly.object.parts {
		y := math.Abs(part.space.Orientation.Y.Y)
		if y > 0.4 && y < 0.98 {
			diagonal++
		}
		for _, face := range part.skin() {
			for _, p := range face {
				if p.Y < 0.1 && math.Hypot(p.X, p.Z) < 4.2 {
					t.Fatal("support obscures primary", p)
				}
			}
		}
	}
	if diagonal < 16 {
		t.Fatal("missing paired diagonals", diagonal)
	}
}

// TestSurveyWingSlopesAndEnclosureClearsOptics checks geometry, independently
// of catalogue appearance hashes and declared bounding boxes.
func TestSurveyWingSlopesAndEnclosureClearsOptics(t *testing.T) {
	tier := surveyTier{width: 50, height: 5, depth: 22, inset: 2}
	for _, face := range tier.skin(cut{around: 12}) {
		for _, p := range face {
			if p.Y == 5 && (p.X > 48 || p.Z < 2 || p.Z > 20) {
				t.Fatal("flat-sided tier", p)
			}
		}
	}
	row := surveyGround()
	assembly := &instrumentAssembly{fine: cut{around: 48}, object: good{dressed: groundLooks()}}
	row.optics(assembly, row.opticalFrame())
	for _, part := range assembly.object.parts {
		for _, face := range part.skin() {
			for _, p := range face {
				if math.Hypot(p.X, p.Z) >= 11.3 || p.Y >= 25 {
					t.Fatal("instrument leaves clear interior", p)
				}
			}
		}
	}
}
