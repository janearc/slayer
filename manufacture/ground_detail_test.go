package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestGroundDetailKeepsTheShellAndSlitClear checks the generated vertices,
// including the tube thickness near the dome apex, not just rod endpoints.
func TestGroundDetailKeepsTheShellAndSlitClear(t *testing.T) {
	for _, name := range []string{"keck-i", "subaru", "elt"} {
		t.Run(name, func(t *testing.T) {
			row := groundRows()[name].(groundObservatory)
			assembly := &instrumentAssembly{fine: cut{around: 24}}
			row.enclosureDetails(assembly)
			if assembly.trouble != nil {
				t.Fatal(assembly.trouble)
			}
			radius := row.diameter / 2
			base := row.height - radius
			if row.cylindrical {
				base = row.height * groundValue("cylinder-base")
			}
			for _, part := range assembly.object.parts {
				for _, face := range part.skin() {
					for _, point := range face {
						if point.Y < 0 || point.Y > row.height {
							t.Fatalf("outside height: %v", point)
						}
						if point.Y < base {
							continue
						}
						distance := math.Hypot(point.X, point.Z)
						if !row.cylindrical {
							distance = point.Sub(maths.YAxis.Scale(base)).Len()
						}
						if distance > radius+1e-9 {
							t.Fatalf("outside shell: %v", point)
						}
						// The authored opening is 50 degrees wide about local -Z.
						if math.Abs(math.Atan2(point.X, -point.Z)) < 25*math.Pi/180 {
							t.Fatalf("member crosses observing slit: %v", point)
						}
					}
				}
			}
		})
	}
}

// TestGroundDetailRailingStaysHumanScale catches aperture-scaled posts.
func TestGroundDetailRailingStaysHumanScale(t *testing.T) {
	for _, name := range []string{"keck-i", "subaru", "elt"} {
		row := groundRows()[name].(groundObservatory)
		assembly := &instrumentAssembly{fine: cut{around: 12}}
		row.gallery(assembly, row.diameter/2, row.diameter/2*groundValue("wall-share"))
		for _, part := range assembly.object.parts[1:] {
			low, high := math.Inf(1), math.Inf(-1)
			for _, face := range part.skin() {
				for _, point := range face {
					low, high = math.Min(low, point.Y), math.Max(high, point.Y)
				}
			}
			if high-low < 1 || high-low > 1.2 {
				t.Fatalf("%s railing height %g", name, high-low)
			}
		}
	}
}

// TestGroundDetailStaysBehindThePrimary holds the new cell behind its mirror.
func TestGroundDetailStaysBehindThePrimary(t *testing.T) {
	for _, name := range []string{"keck-i", "subaru", "rubin", "elt"} {
		row := groundRows()[name].(groundObservatory)
		pose := row.opticalFrame()
		assembly := &instrumentAssembly{fine: cut{around: 12}}
		row.opticsDetails(assembly, pose)
		for _, part := range assembly.object.parts {
			for _, face := range part.skin() {
				for _, point := range face {
					if pose.In(point).Y >= 0 {
						t.Fatalf("%s support in front of primary: %v", name, point)
					}
				}
			}
		}
	}
}

// TestGroundDetailWingMembersStayOnTheirTier checks actual surface bounds.
func TestGroundDetailWingMembersStayOnTheirTier(t *testing.T) {
	assembly := &instrumentAssembly{fine: cut{around: 12}}
	survey := surveyOptics{}
	survey.facilityDetails(assembly)
	for _, part := range assembly.object.parts {
		if part.of != surveyCladding {
			t.Fatal("service-building support survives the enclosure cutaway")
		}
	}
	// Each tier has six roof beams followed by its two corner posts.
	for index, tier := range []string{"lower", "middle", "upper"} {
		low := facilityValue(tier + "-east")
		high := low + facilityValue(tier+"-width")
		for _, part := range assembly.object.parts[index*8 : (index+1)*8] {
			for _, face := range part.skin() {
				for _, point := range face {
					if point.X < low-1e-9 || point.X > high+1e-9 {
						t.Fatalf("%s member outside wing [%g,%g]: %v", tier, low, high, point)
					}
				}
			}
		}
	}
}
