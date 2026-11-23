package cities

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// plainLook is a material for a test's parts.
func plainLook() *thing.Material {
	return &thing.Material{Name: "plain", Colour: maths.Grey(0.5)}
}

// testRoof is a roof of a form over a footprint of five by eight metres
// with a rise of two, a slab of a tenth and no eaves, the form's own
// other needs met.
func testRoof(form roofForm) roof {
	return roof{Form: form, Width: 5, Depth: 8, Rise: 2, Slab: 0.1,
		Parapet: 0.5, ParapetWall: 0.2, Inset: 1, Cover: plainLook(),
		Gable: plainLook()}
}

// roofReach is the lowest and highest corner of all of a roof's parts.
func roofReach(t *testing.T, roof roof) (low, high maths.Vec) {
	t.Helper()
	parts, err := roof.parts()
	if err != nil {
		t.Fatal(err)
	}
	low = maths.V(math.Inf(1), math.Inf(1), math.Inf(1))
	high = maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, part := range parts {
		partLow, partHigh := reach(rendered(t, part))
		low = maths.V(math.Min(low.X, partLow.X),
			math.Min(low.Y, partLow.Y), math.Min(low.Z, partLow.Z))
		high = maths.V(math.Max(high.X, partHigh.X),
			math.Max(high.Y, partHigh.Y), math.Max(high.Z, partHigh.Z))
	}
	return low, high
}

func TestEveryFormOfRoofIsDrawn(t *testing.T) {
	for _, form := range roofForms() {
		parts, err := testRoof(form).parts()
		if err != nil || len(parts) == 0 {
			t.Errorf("%s: %d parts, %v", form, len(parts), err)
		}
	}
}

func TestAGableRisesToItsRidgeAndNoFurther(t *testing.T) {
	roof := testRoof(gableAcross)
	roof.Eave = 0.3
	_, high := roofReach(t, roof)
	if high.Y < roof.Rise || high.Y > roof.Rise+2*roof.Slab {
		t.Errorf("a gable of rise %v tops out at %v", roof.Rise, high.Y)
	}
}

func TestAGableAcrossRunsItsRidgeFrontToBack(t *testing.T) {
	low, high := roofReach(t, testRoof(gableAcross))
	if low.X > 0 || high.X < 5 || low.Z > 0 || high.Z < 8 {
		t.Errorf("a gable over 5 by 8 spans %v to %v", low, high)
	}
	if high.X > 5+slabAllowance || high.Z > 8+tolerance ||
		low.X < -slabAllowance || low.Z < -tolerance {
		t.Errorf("a gable with no eave overhangs to %v and %v", low, high)
	}
}

// slabAllowance is how far past the wall a slope's slab may stand where
// it has no eave: its own thickness, since it is laid on the slope.
const slabAllowance = 0.1

func TestAGableAlongStandsOnTheSameFootprintTurned(t *testing.T) {
	across, _ := roofReach(t, testRoof(gableAcross))
	along, high := roofReach(t, testRoof(gableAlong))
	if along.X < -tolerance || high.X > 5+tolerance ||
		along.Z < -slabAllowance || high.Z > 8+slabAllowance {
		t.Errorf("a gable along over 5 by 8 spans %v to %v", along, high)
	}
	if math.Abs(high.Y-2) > 0.2 || across.Y < -1e-6 {
		t.Errorf("a gable along tops out at %v", high.Y)
	}
}

func TestAMonoPitchRoofRisesByItsRise(t *testing.T) {
	for _, form := range []roofForm{monoUp, monoDown} {
		_, high := roofReach(t, testRoof(form))
		if math.Abs(high.Y-2) > tolerance {
			t.Errorf("%s tops out at %v, want 2", form, high.Y)
		}
	}
}

func TestAMonoUpIsHighAtTheFrontAndAMonoDownAtTheBack(t *testing.T) {
	for form, front := range map[roofForm]float64{monoUp: 2, monoDown: 0} {
		parts, _ := testRoof(form).parts()
		one := rendered(t, parts[0])
		var top float64
		for _, triangle := range one.Mesh.Triangles {
			for _, corner := range triangle {
				if corner.Z > 7.9 {
					top = math.Max(top, corner.Y)
				}
			}
		}
		if math.Abs(top-front) > tolerance {
			t.Errorf("%s is %v high at the front, want %v", form, top, front)
		}
	}
}

func TestAHipsTopOutlineIsSmallerThanItsBottom(t *testing.T) {
	parts, err := testRoof(hip).parts()
	if err != nil {
		t.Fatal(err)
	}
	one := rendered(t, parts[0])
	var bottom, top float64
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			if corner.Y < 1 {
				bottom = math.Max(bottom, corner.X)
			} else {
				top = math.Max(top, corner.X)
			}
		}
	}
	if !(top < bottom) || math.Abs(top-4) > tolerance {
		t.Errorf("a hip is %v wide at the top and %v at the bottom", top,
			bottom)
	}
}

func TestAFlatRoofHasAParapetAndNoEaves(t *testing.T) {
	parts, err := testRoof(roofFlat).parts()
	if err != nil || len(parts) != 5 {
		t.Fatalf("a flat roof with a parapet is %d parts, %v", len(parts),
			err)
	}
	low, high := roofReach(t, testRoof(roofFlat))
	if low.X != 0 || high.X != 5 || high.Y != 0.5 {
		t.Errorf("a flat roof spans %v to %v", low, high)
	}
}

func TestAFlatRoofWithEavesIsRefused(t *testing.T) {
	roof := testRoof(roofFlat)
	roof.Eave = 0.2
	if _, err := roof.parts(); err == nil {
		t.Error("a flat roof with eaves was drawn")
	}
}

func TestARoofThatCannotStandIsRefused(t *testing.T) {
	flawed := map[string]func(*roof){
		"unknown form":      func(roof *roof) { roof.Form = "dome" },
		"no rise":           func(roof *roof) { roof.Rise = 0 },
		"hip past middle":   func(roof *roof) { roof.Form, roof.Inset = hip, 3 },
		"hip with no inset": func(roof *roof) { roof.Form, roof.Inset = hip, 0 },
		"no cover":          func(roof *roof) { roof.Cover = nil },
		"gable no end":      func(roof *roof) { roof.Gable = nil },
		"no footprint":      func(roof *roof) { roof.Width = 0 },
	}
	for name, spoil := range flawed {
		roof := testRoof(gableAcross)
		spoil(&roof)
		if _, err := roof.parts(); err == nil {
			t.Errorf("%s: a roof was drawn", name)
		}
	}
}
