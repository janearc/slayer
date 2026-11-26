package snakes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// Head rounding must retain every authored cross section and stay within the
// two neighbouring radii. Its slope must not jump at the head's peak.
func TestHeadInterpolationIsRoundedAndPreservesStations(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	for i, s := range p.Stations[:len(p.Stations)-1] {
		w, h := section(p.Stations, s.At)
		if math.Abs(w-s.Width) > 1e-12 || math.Abs(h-s.Height) > 1e-12 {
			t.Fatalf("station %d moved", i)
		}
	}
	for i := len(p.Stations) - 3; i < len(p.Stations)-1; i++ {
		a, b := p.Stations[i], p.Stations[i+1]
		for k := 1; k < 20; k++ {
			w, h := section(p.Stations, a.At+(b.At-a.At)*float64(k)/20)
			minW, minH := math.Min(a.Width, b.Width), math.Min(a.Height, b.Height)
			if i == len(p.Stations)-2 {
				minW, minH = 0, 0
			}
			if w < minW || w > math.Max(a.Width, b.Width) || h < minH || h > math.Max(a.Height, b.Height) {
				t.Fatalf("head interpolation overshot span %d", i)
			}
		}
	}
	peak := p.Stations[len(p.Stations)-3].At
	const d = 1e-6
	before, _ := section(p.Stations, peak-d)
	mid, _ := section(p.Stations, peak)
	after, _ := section(p.Stations, peak+d)
	if math.Abs((mid-before)/d-(after-mid)/d) > .001 {
		t.Fatal("head keeps a sharp edge at its authored widest station")
	}
}

// Each eye is sunk in the body, but some of its surface remains outside.
func TestSnakeEyesAreEmbeddedAndVisible(t *testing.T) {
	for _, name := range []string{"bitis-gabonica", "boa-constrictor-occidentalis", "python-bivittatus"} {
		p := testProfile(t, name)
		row, err := Build(p)
		if err != nil {
			t.Fatal(err)
		}
		skin := row.Parts[0].Shape.(*mesh.Mesh)
		for _, part := range row.Parts[1:] {
			sphere, ok := part.Shape.(shape.Sphere)
			if !ok {
				t.Fatalf("%s eye is %T", name, part.Shape)
			}
			centre := part.At(0).Origin
			inside := skin.Distance(centre)
			if !(inside < 0 && inside > -sphere.Radius) {
				t.Fatalf("%s eye center is not partly embedded: distance=%g radius=%g", name, inside, sphere.Radius)
			}
			outward := skin.Normal(centre)
			if skin.Distance(centre.Add(outward.Scale(1.5*sphere.Radius))) <= 0 {
				t.Fatalf("%s eye has no visible surface", name)
			}
		}
	}
}

// Paired nostrils and a closed mouth should darken localized facial regions
// without painting an unrelated part of the body or the space above the lip.
func TestSnakeFacePaintHasPairedNostrilsAndAClosedMouth(t *testing.T) {
	for _, name := range []string{"bitis-gabonica", "boa-constrictor-occidentalis", "python-bivittatus"} {
		p := testProfile(t, name)
		base := maths.RGB(.65, .65, .65)
		nostril := p.Head.EyeAt + (1-p.Head.EyeAt)*.65
		noseAngle := math.Acos(p.Head.EyeHeight) + .15
		lipAngle := math.Pi/2 + .16
		for _, side := range []float64{-1, 1} {
			nose := facePaint(p, nostril, side*noseAngle, 0, base)
			mouth := facePaint(p, p.Head.EyeAt, side*lipAngle, 0, base)
			if nose == base || mouth == base {
				t.Fatalf("%s side %g is missing a nostril or mouth", name, side)
			}
			if facePaint(p, p.Head.EyeAt-.15, side*lipAngle, 0, base) != base {
				t.Fatalf("%s face leaked onto its body", name)
			}
			if facePaint(p, p.Head.EyeAt, side*(lipAngle-.20), 0, base) != base {
				t.Fatalf("%s mouth is not a narrow line", name)
			}
		}
	}
}

// The nose is a single pole at the authored end, and paint there is finite.
func TestRoundedSnoutEndsAtAuthoredLength(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	row, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	skin := row.Parts[0].Shape.(*mesh.Mesh)
	end := p.Length / 2
	found := 0
	for _, point := range skin.Points() {
		if point.X > end {
			t.Fatalf("snout exceeds authored length: %g > %g", point.X, end)
		}
		if point.X == end {
			found++
			if point.Y != 0 || point.Z != 0 {
				t.Fatal("snout endpoint is not a pole")
			}
		}
	}
	if found != 1 {
		t.Fatalf("nose poles = %d", found)
	}
	c := row.Parts[0].Material.Texture.At(maths.V(end, 0, 0))
	if math.IsNaN(c.R) || math.IsNaN(c.G) || math.IsNaN(c.B) || math.IsInf(c.R, 0) || math.IsInf(c.G, 0) || math.IsInf(c.B, 0) {
		t.Fatal("nose pole paint is not finite")
	}
}

// Terminal radii are taper controls, not discarded data or the radius at the
// pole. Changing either one affects the nearby rounded nose but not its end.
func TestTerminalRadiiControlSnoutTaper(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	at := (p.Stations[len(p.Stations)-2].At + 1) / 2
	wide, high := section(p.Stations, at)
	q := p
	q.Stations = append([]Station(nil), p.Stations...)
	last := len(q.Stations) - 1
	q.Stations[last].Width *= 1.5
	q.Stations[last].Height *= 1.5
	changedWide, changedHigh := section(q.Stations, at)
	if wide == changedWide || high == changedHigh {
		t.Fatal("terminal radii did not affect the nose taper")
	}
	if w, h := section(q.Stations, 1); w != 0 || h != 0 {
		t.Fatalf("nose is not a pole: %g,%g", w, h)
	}
}

// A built skin, not only the face helper, must actually show both features.
func TestBuiltSnakeSkinShowsNostrilsAndMouth(t *testing.T) {
	p := testProfile(t, "bitis-gabonica")
	p.Pattern.Kind = "plain"
	row, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	paint := row.Parts[0].Material.Texture
	point := func(along, angle float64) maths.Vec {
		width, height := section(p.Stations, along)
		return maths.V((along-.5)*p.Length, height*p.Length*math.Cos(angle), width*p.Length*math.Sin(angle))
	}
	nostrilAt := p.Head.EyeAt + (1-p.Head.EyeAt)*.65
	nostrilAngle := math.Acos(p.Head.EyeHeight) + .15
	if paint.At(point(nostrilAt, nostrilAngle)) == paint.At(point(nostrilAt, nostrilAngle-.20)) {
		t.Fatal("built skin omits the nostril")
	}
	mouthAngle := math.Pi/2 + .16
	if paint.At(point(p.Head.EyeAt, mouthAngle)) == paint.At(point(p.Head.EyeAt, mouthAngle-.20)) {
		t.Fatal("built skin omits the closed mouth")
	}
}
