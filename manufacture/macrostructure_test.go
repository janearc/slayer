package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestMacrostructureFamiliesHaveMetreBoundsOpeningsAndBatchedGeometry(t *testing.T) {
	for _, style := range []MacrostructureStyle{TerracedArcology, TwistedSpires,
		InhabitedBridge} {
		spec := DefaultMacrostructure(style)
		row, err := Macrostructure(spec)
		if err != nil {
			t.Fatal(err)
		}
		var reference maths.Box
		for _, detail := range []string{"hull", "quick", "demo"} {
			made, err := MakeWithComplexity(row, Sow(31), detail)
			if err != nil {
				t.Fatal(style, detail, err)
			}
			things, err := made.Things(maths.Local())
			if err != nil {
				t.Fatal(style, detail, err)
			}
			if len(things) > 4 || made.faces() > 20_000 {
				t.Fatalf("unbounded geometry: %s %d %d", style, len(things), made.faces())
			}
			box := made.Bounds()
			actual := partBounds(made.parts)
			epsilon := 1e-8 * spec.Height.InMetres()
			if actual.Min.X < box.Min.X-epsilon || actual.Min.Y < -epsilon || actual.Min.Z < box.Min.Z-epsilon || actual.Max.X > box.Max.X+epsilon || actual.Max.Y > box.Max.Y+epsilon || actual.Max.Z > box.Max.Z+epsilon {
				t.Fatalf("%s geometry escapes its envelope: %+v vs %+v", style, actual, box)
			}
			if box.Min.Y != 0 || box.Max.Y != spec.Height.InMetres() || box.Max.X-box.Min.X != spec.Width.InMetres() || box.Max.Z-box.Min.Z != spec.Depth.InMetres() {
				t.Fatalf("%s bounds %+v", style, box)
			}
			if detail == "hull" {
				reference = box
			} else if box != reference {
				t.Fatal("detail changes envelope")
			}
			t.Logf("%s %s: %d parts, %d faces, %d material meshes", style, detail,
				made.PartCount(), made.faces(), len(things))
		}
		made, _ := Make(row, Sow(31))
		parts, _ := made.Things(maths.Local())
		ray := maths.Ray{At: maths.V(0, spec.Height.InMetres()*.3, -spec.Depth.InMetres()),
			Dir: maths.ZAxis}
		if style == TerracedArcology {
			ray = maths.Ray{At: maths.V(0, spec.Height.InMetres()+1, 0), Dir: maths.Up.Neg()}
		}
		if style == TwistedSpires {
			ray = maths.Ray{At: maths.V(-spec.Width.InMetres()*.23, spec.Height.InMetres()+1,
				-spec.Depth.InMetres()*.12), Dir: maths.Up.Neg()}
			for _, part := range parts {
				if _, hit := part.Shape.(shape.Hitter).Hit(ray, 0, spec.Height.InMetres()*.5); hit {
					t.Fatal("twisted spire closed its upper atrium")
				}
			}
		} else {
			for _, part := range parts {
				if _, hit := part.Shape.(shape.Hitter).Hit(ray, 0, 2*spec.Depth.InMetres()); hit {
					t.Fatalf("%s closed its central opening", style)
				}
			}
		}
	}
}

// anti-test of TestMacrostructureFamiliesHaveMetreBoundsOpeningsAndBatchedGeometry
func TestMacrostructureRefusesInvalidDimensionsAndSeeds(t *testing.T) {
	for _, change := range []func(*MacrostructureSpec){
		func(s *MacrostructureSpec) { s.Style = "unknown" },
		func(s *MacrostructureSpec) { s.Width = measure.Metres(0) },
		func(s *MacrostructureSpec) { s.Height = measure.Metres(math.Inf(1)) },
		func(s *MacrostructureSpec) { s.Depth = measure.Metres(math.NaN()) },
		func(s *MacrostructureSpec) { s.Twist = measure.Degrees(900) },
		func(s *MacrostructureSpec) { s.Height = measure.Metres(1e10) },
		func(s *MacrostructureSpec) { s.Height = measure.Metres(1); s.Width = measure.Metres(10000) },

		func(s *MacrostructureSpec) { s.Style = TwistedSpires; s.Depth = s.Width.Times(3) },
		func(s *MacrostructureSpec) { s.Style = TwistedSpires; s.Height = measure.Metres(10) },
		func(s *MacrostructureSpec) { s.Style = InhabitedBridge; s.Width = measure.Metres(1000) },
	} {
		spec := DefaultMacrostructure(TerracedArcology)
		change(&spec)
		if _, err := Macrostructure(spec); err == nil {
			t.Fatal("invalid structure admitted", spec)
		}
	}
	row, _ := Macrostructure(DefaultMacrostructure(TwistedSpires))
	if _, err := MakeWithComplexity(row, seed{}, "quick"); err == nil {
		t.Fatal("unsown macro seed")
	}
}

func TestMacrostructureSeedsChangeGeometryWithoutChangingPhysicalScale(t *testing.T) {
	for _, style := range []MacrostructureStyle{TerracedArcology, TwistedSpires,
		InhabitedBridge} {
		row, _ := Macrostructure(DefaultMacrostructure(style))
		a, _ := Make(row, Sow(2))
		same, _ := Make(row, Sow(2))
		other, _ := Make(row, Sow(3))
		if componentFingerprint(flatMacroSkin(a)) != componentFingerprint(flatMacroSkin(same)) {
			t.Fatal("seed is unstable", style)
		}
		if componentFingerprint(flatMacroSkin(a)) == componentFingerprint(flatMacroSkin(other)) {
			t.Fatal("no seeded variation", style)
		}
		if _,
			err := other.Things(maths.Local()); err != nil {
			t.Fatal("seed produced invalid geometry",
				style, err)
		}
		if a.Bounds() != other.Bounds() {
			t.Fatal("seed changed requested physical scale")
		}
	}
}

// flatMacroSkin preserves component order for an exact same-seed signature.
func flatMacroSkin(made good) surface {
	var skin surface
	for _, part := range made.parts {
		skin = append(skin, part.skin()...)
	}
	return skin
}

func TestLandformMountainAndVolcanoFollowTheirProfiles(t *testing.T) {
	for _, style := range []LandformStyle{MountainMassif, CalderaVolcano} {
		spec := DefaultLandform(style)
		row, err := Landform(spec)
		if err != nil {
			t.Fatal(err)
		}
		for _, detail := range []string{"hull", "quick", "demo"} {
			made, err := MakeWithComplexity(row, Sow(31), detail)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := made.Things(maths.Local())
			if err != nil {
				t.Fatal(style, detail, err)
			}
			if len(parts) > 3 || made.faces() > 20_000 {
				t.Fatal("landform geometry budget")
			}
			if made.Bounds().Max.Y != spec.Height.InMetres() {
				t.Fatal("requested summit height lost")
			}
			t.Logf("%s %s: %d faces", style, detail, made.faces())
		}
		profile := landformMaker{spec: spec}
		if style == MountainMassif {
			if profile.elevation(0, 0, Sow(31)) != spec.Height.InMetres() {
				t.Fatal("mountain peak")
			}
		} else {
			if profile.elevation(0, 0, Sow(31)) != spec.Height.InMetres()-spec.CraterDepth.InMetres() {
				t.Fatal("caldera floor")
			}
			if profile.elevation(spec.CraterRadius.InMetres(), 0, Sow(31)) != spec.Height.InMetres() {
				t.Fatal("caldera rim")
			}
		}
		if profile.elevation(spec.Radius.InMetres(), 1, Sow(31)) != 0 {
			t.Fatal("landform base is not grounded")
		}
	}
}

// anti-test of TestLandformMountainAndVolcanoFollowTheirProfiles
func TestLandformRefusesImpossibleCraters(t *testing.T) {
	spec := DefaultLandform(CalderaVolcano)
	spec.CraterRadius = spec.Radius
	if _, err := Landform(spec); err == nil {
		t.Fatal("crater consumed the mountain")
	}
	spec = DefaultLandform(CalderaVolcano)
	spec.CraterDepth = spec.Height
	if _, err := Landform(spec); err == nil {
		t.Fatal("crater reached below the base")
	}
}

// Independent annulus volume and arch stations expose hollow/supported rules.
func TestMacrostructureLoftVolumeAndArchFormula(t *testing.T) {
	loft := macroLoft{width: 10, depth: 8, innerWidth: 4, innerDepth: 2, height: 20, segments: 3}
	volume := 0.0
	for _, triangle := range loft.skin(cut{}) {
		volume += triangle[0].Dot(triangle[1].Cross(triangle[2])) / 6
	}
	if math.Abs(volume-1440) > 1e-9 {
		t.Fatalf("hollow volume %v, expected (80-8)*20", volume)
	}
	arch := macroArch{span: 600, rise: 100, radius: 4, segments: 12}
	left, _ := arch.station(0)
	middle, _ := arch.station(.5)
	right, _ := arch.station(1)
	if left != maths.V(-300, 0, 0) || middle != maths.V(0, 100,
		0) || right != maths.V(300, 0, 0) {
		t.Fatal("parabolic arch has moved its supports or crown")
	}
}

func TestLandformElevationUsesTypedMetresAndStableSeeds(t *testing.T) {
	spec := DefaultLandform(MountainMassif)
	height, err := LandformElevation(spec, Sow(2), measure.Metres(0), measure.Metres(0))
	if err != nil || height != spec.Height {
		t.Fatal("typed mountain summit", height, err)
	}
	outside, err := LandformElevation(spec, Sow(2), measure.Metres(6000), measure.Metres(0))
	if err != nil || outside.InMetres() != 0 {
		t.Fatal("outside landform")
	}
	if _, err := LandformElevation(spec, seed{}, measure.Metres(0),
		measure.Metres(0)); err == nil {
		t.Fatal("unsown landform sample")
	}
	unbounded := measure.Metres(math.Inf(1))
	if _, err := LandformElevation(spec, Sow(2), unbounded,
		measure.Metres(0)); err == nil {
		t.Fatal("infinite landform sample")
	}
	row, _ := Landform(spec)
	first, _ := Make(row, Sow(2))
	same, _ := Make(row, Sow(2))
	other, _ := Make(row, Sow(3))
	if componentFingerprint(flatMacroSkin(first)) != componentFingerprint(flatMacroSkin(same)) || componentFingerprint(flatMacroSkin(first)) == componentFingerprint(flatMacroSkin(other)) {
		t.Fatal("landform seeded variation")
	}
}

// anti-test of TestMacrostructureFamiliesHaveMetreBoundsOpeningsAndBatchedGeometry
func TestMacrostructureRecipeDataRefusesInvalidUnitsAndBudgets(t *testing.T) {
	source := string(macroChoices)
	for _, raw := range []string{
		"numbers: [",
		strings.Replace(source, "unit: m", "unit: km", 1),
		strings.Replace(source, "value: 1800", "value: -1800", 1),
		strings.Replace(source, "value: 6\n", "value: 6.5\n", 1),
		strings.Replace(source, "value: 20000", "value: 200000", 1),
		strings.Replace(source, "value: 0.18", "value: 1.8", 1),
		strings.Replace(source, "value: 0.55", "value: 0.95", 1),
	} {
		if _, err := readMacroChoices([]byte(raw)); err == nil {
			t.Fatal("invalid macro choices admitted")
		}
	}
}

func TestMacrostructureDimensionsConvertToOrbitalUnitsExplicitly(t *testing.T) {
	spec := DefaultMacrostructure(TwistedSpires)
	spec.Width, spec.Depth, spec.Height = measure.Metres(120000),
		measure.Metres(120000), measure.Metres(400000)
	row, err := Macrostructure(spec)
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(row, Sow(31))
	if err != nil {
		t.Fatal(err)
	}
	place := maths.NewSubspace(maths.V(1, 2, 3), maths.Identity, 1e-6)
	things, err := made.Things(place)
	if err != nil {
		t.Fatal(err)
	}
	box := transformed(made.Bounds(), things[0].At(0))
	if math.Abs((box.Max.Y-box.Min.Y)-.4) > 1e-12 || made.faces() != 4236 {
		t.Fatal("physical scale or dimension-independent geometry budget", box, made.faces())
	}
}
