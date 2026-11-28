package primates

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func TestOneClosedNativeSurfaceAndRangeRefusals(t *testing.T) {
	s := Adult()
	if _, err := s.skinMesh(1, nil, nil, nil); err == nil {
		t.Fatal("empty native skin accepted")
	}
	if len(s.Landmarks(1)) != 7 {
		t.Fatal("missing landmarks")
	}
	s.LateralAttachment = true
	out, e := s.Surface(8, 128, 100)
	if e != nil {
		t.Fatal(e)
	}
	shell := out.Shape.(*mesh.Mesh)
	if !shell.Closed() || shell.FaceCount() != 2*128*101 {
		t.Fatal("open or disconnected torso section", shell.FaceCount())
	}
	if _, e = s.Surface(8, 8, 8); e == nil {
		t.Fatal("accepted missing sampling")
	}
	for _, edit := range []func(*Torso){func(s *Torso) { s.Left.Width = 0 }, func(s *Torso) { s.Compliance = 1.4 }, func(s *Torso) { s.AgeYears = 13 }, func(s *Torso) { s.Width = math.NaN() }} {
		bad := s
		edit(&bad)
		if _, e = bad.Surface(8, 32, 24); e == nil {
			t.Fatal(bad)
		}
	}
}
func TestLandmarksAndMovementOnSameRestSurface(t *testing.T) {
	s := Adult()
	x, y := s.landmarks(s.Left)
	angle := math.Asin(x / (sectionAt(y).w * s.Width))
	upright, rest := s.sample(angle, y, 12)
	noLoad := s
	noLoad.Gravity = 0
	unloaded, r := noLoad.sample(angle, y, 12)
	if r != rest || unloaded.Y-upright.Y < .010 {
		t.Fatal("load failed to lower anterior landmark", unloaded, upright)
	}
	lifted := s
	lifted.LeftArm = 1.7
	up, _ := lifted.sample(1.04, 1.28, 12)
	relax, _ := s.sample(1.04, 1.28, 12)
	if up.Y-relax.Y < .012 {
		t.Fatal("axilla failed to lift")
	}
	rightA, _ := s.sample(-1.04, 1.28, 12)
	rightB, _ := lifted.sample(-1.04, 1.28, 12)
	if rightA != rightB {
		t.Fatal("contralateral wall moved")
	}
	deeper := s
	deeper.Left.Projection += .024
	a, _ := deeper.sample(angle, y, 12)
	if a.Z-upright.Z < .015 {
		t.Fatal("independent projection")
	}
	sternum, _ := s.sample(0, y, 12)
	mid, _ := deeper.sample(0, y, 12)
	if mid != sternum {
		t.Fatal("lobe crossed sternum")
	}
	turn := s
	turn.Pitch = .2
	turn.Twist = .15
	q, r := turn.sample(angle, y, 12)
	if r != rest || q.Sub(upright).Len() < .025 {
		t.Fatal("torso failed to carry material")
	}
}
func TestSkinSeparatesPigmentAndRelief(t *testing.T) {
	s := Adult()
	p := maths.V(.081, 1.176, .091)
	q := maths.V(.081, 1.225, .091)
	o := s.Optics(p, 3)
	next := s.Optics(q, 3)
	if o.Colour.Luminance() >= next.Colour.Luminance() || o.Roughness >= next.Roughness {
		t.Fatal("missing areolar transition", o, next)
	}
	dark := s
	dark.Melanin = .86
	if dark.Optics(q, 3).Colour.Luminance() >= next.Colour.Luminance() {
		t.Fatal("melanin failed")
	}
	if !(o.Radius[0] > o.Radius[1] && o.Radius[1] > o.Radius[2]) {
		t.Fatal(o.Radius)
	}
	if (skin{s, 3}).At(q) != next.Colour {
		t.Fatal("native material disconnected")
	}
}

func TestUndertonesPreserveLocalLuminanceAcrossPigmentRange(t *testing.T) {
	s := Adult()
	points := []maths.Vec{maths.V(.08, 1.177, .09), maths.V(.16, 1.337, .03), maths.V(.15, 1.27, .03), maths.V(.03, 1.03, .08)}
	for _, p := range points {
		previous := math.Inf(1)
		for _, melanin := range []float64{.12, .50, .95} {
			s.Melanin = melanin
			s.Undertone = Chroma{}
			base := s.Optics(p, 431).Colour
			if base.Luminance() >= previous {
				t.Fatal("skin lightness lost ordering", p, melanin)
			}
			previous = base.Luminance()
			for _, shift := range []Chroma{{Red: .90}, {Gold: .90}, {Olive: .90}, {Red: .26, Gold: -.70}} {
				s.Undertone = shift
				c := s.Optics(p, 431).Colour
				if c == base || math.Abs(c.Luminance()-base.Luminance()) > 1e-12 {
					t.Fatal("undertone changed lightness or lost chroma", shift, p, melanin, c, base)
				}
			}
		}
	}
	s.Undertone = Chroma{Olive: 1.1}
	if s.Check() == nil {
		t.Fatal("out of gamut control accepted")
	}
}

func TestAreolarReliefUsesLocalGeometryAndFadesOffPatch(t *testing.T) {
	centre, _ := areolarRelief(0, 0, 1)
	edge, _ := areolarRelief(.38, 0, 1)
	if centre < .99 || edge > .08 {
		t.Fatal("missing papilla relief", centre, edge)
	}
	s := Adult()
	s.LateralAttachment = true
	left := s.Left
	x, y := s.landmarks(left)
	y -= .008
	point := func(x, y float64) maths.Vec {
		a := math.Asin(x / (s.Width * sectionAt(y).w))
		p, _ := s.sample(a, y, 431)
		return p
	}
	p := point(x, y)
	ring := (point(x-.008, y).Z + point(x+.008, y).Z + point(x, y+.008).Z + point(x, y-.008).Z) / 4
	if p.Z-ring < .003 {
		t.Fatal("papilla lost geometric height over areolar plateau", p.Z-ring)
	}
}
