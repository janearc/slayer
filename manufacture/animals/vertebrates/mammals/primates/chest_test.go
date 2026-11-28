package primates

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"math"
	"testing"
)

func sectionDepth(t *testing.T, m *mesh.Mesh, x, y float64) float64 {
	t.Helper()
	r := maths.Ray{At: maths.V(x, y, .5), Dir: maths.ZAxis.Neg()}
	h, ok := m.Meet(r, 0, .8)
	if !ok {
		t.Fatal("missing transverse surface", x, y)
	}
	return r.Point(h.Along()).Z
}
func TestTransverseMedialWall(t *testing.T) {
	s := Adult()
	s.LateralAttachment = true
	var baseline [3]float64
	for _, version := range []bool{true, false} {
		s.LegacyContour = version
		if len(s.Landmarks(431)) != 7 {
			t.Fatal("mapped landmarks")
		}
		part, err := s.Surface(431, Around, Rows)
		if err != nil {
			t.Fatal(err)
		}
		m := part.Shape.(*mesh.Mesh)
		for lane, y := range []float64{1.18, 1.22, 1.26} {
			z := func(x float64) float64 { return sectionDepth(t, m, x, y) }
			mid := z(0)
			prominence := .5*(z(.040)+z(-.040)) - mid
			if version {
				baseline[lane] = prominence
			} else if baseline[lane]-prominence < .003 || (lane == 2 && prominence > 0) {
				t.Fatal("medial groove failed to terminate", y, baseline[lane], prominence)
			}
			t.Logf("legacy=%t y=%.3f mid-to-40=%.3fmm central second difference=%.3fmm", version, y, 1000*(.5*(z(.040)+z(-.040))-mid), 1000*(z(.012)+z(-.012)-2*mid))
		}
	}
}
func TestPapillaryProjectionTracksPosedNormal(t *testing.T) {
	s := Adult()
	s.Twist = .34
	s.Pitch = .28
	cx, cy := s.landmarks(s.Left)
	y := cy - .008
	a := math.Asin(cx / (sectionAt(y).w * s.Width))
	base, _ := s.chestPoint(a, y)
	u, _ := s.chestPoint(a+.0002, y)
	v, _ := s.chestPoint(a, y+.00003)
	n := u.Sub(base).Cross(v.Sub(base)).Unit()
	first, _ := s.sample(a, y, 9)
	s.Left.PapillaProjection += .002
	second, _ := s.sample(a, y, 9)
	shift := second.Sub(first)
	if shift.Dot(n) < .0019 || shift.Cross(n).Len() > .00012 {
		t.Fatal("independent projection departed posed normal", shift, n)
	}
	radius := s.Left.PapillaRadius
	side := s.Left
	smaller, _ := s.relief(side, radius/side.AreolaRadius, 0, 1)
	side.PapillaRadius *= 1.5
	larger, _ := s.relief(side, radius/side.AreolaRadius, 0, 1)
	if larger <= smaller {
		t.Fatal("papilla width control")
	}
}

func TestShoulderContinuationIsOneManifoldWithAxillaryAir(t *testing.T) {
	s := Adult()
	s.ProximalArms = true
	made, e := s.Surface(431, 208, 143)
	if e != nil {
		t.Fatal(e)
	}
	m := made.Shape.(*mesh.Mesh)
	pts, faces := m.Points(), m.Faces()
	type counts struct{ number, direction int }
	edges := map[[2]int]counts{}
	adj := make([][]int, len(pts))
	for _, f := range faces {
		for i, x := range f {
			y := f[(i+1)%3]
			a, b := min(x, y), max(x, y)
			v := edges[[2]int{a, b}]
			v.number++
			if x == a {
				v.direction++
			} else {
				v.direction--
			}
			edges[[2]int{a, b}] = v
			adj[x] = append(adj[x], y)
		}
	}
	for k, v := range edges {
		if v.number != 2 || v.direction != 0 {
			t.Fatal("non-manifold or mismatched wound edge", k, v)
		}
	}
	visited := make([]bool, len(pts))
	queue := []int{0}
	visited[0] = true
	for h := 0; h < len(queue); h++ {
		for _, j := range adj[queue[h]] {
			if !visited[j] {
				visited[j] = true
				queue = append(queue, j)
			}
		}
	}
	if len(queue) != len(pts) || len(pts)-len(edges)+len(faces) != 2 || !m.Closed() {
		t.Fatal("disconnected or non-spherical shell", len(queue), len(pts), len(edges), len(faces))
	}
	for _, sign := range []float64{-1, 1} {
		ray := maths.Ray{At: maths.V(sign*.179, 1.15, .48), Dir: maths.ZAxis.Neg()}
		if _, hit := m.Meet(ray, 0, 1); hit {
			t.Fatal("closed axillary space")
		}
		ray.At.X = sign * .234
		if _, hit := m.Meet(ray, 0, 1); !hit {
			t.Fatal("missing arm column")
		}
	}
	s.LeftArm = .3
	if s.Check() == nil {
		t.Fatal("neutral shoulder study accepted raised arm")
	}
}
func TestLoadFollowsGravityOnBentTrunk(t *testing.T) {
	s := Adult()
	s.Twist = .3
	s.Pitch = .34
	x, y := s.landmarks(s.Left)
	a := math.Asin(x / (s.Width * sectionAt(y).w))
	loaded, _ := s.chestPoint(a, y)
	s.Gravity = 0
	free, _ := s.chestPoint(a, y)
	shift := loaded.Sub(free)
	if shift.Y >= -.01 || math.Hypot(shift.X, shift.Z) > -.22*shift.Y {
		t.Fatal("mobile tissue did not load downwards", shift)
	}
}

func TestUpperShoulderRayCrossings(t *testing.T) {
	s := Adult()
	s.ProximalArms = true
	for _, res := range [][2]int{{208, 143}, {Around, Rows}} {
		o, e := s.Surface(431, res[0], res[1])
		if e != nil {
			t.Fatal(e)
		}
		m := o.Shape.(*mesh.Mesh)
		for _, angle := range []float64{0, -15, 15} {
			dir := maths.V(math.Sin(angle*maths.Degree), 0, math.Cos(angle*maths.Degree))
			crossed, extra := 0, 0
			for yi := 0; yi < 25; yi++ {
				for xi := 0; xi < 32; xi++ {
					at := maths.V(.1207+.003*float64(xi), 1.3073+.0024*float64(yi), 0).Add(dir.Scale(.45))
					ray := maths.Ray{At: at, Dir: dir.Neg()}
					near := 0.
					n := 0
					for {
						h, hit := m.Meet(ray, near, .9)
						if !hit {
							break
						}
						near = h.Along() + 1e-7
						n++
					}
					if n > 0 {
						crossed++
					}
					if n > 2 {
						extra++
					}
				}
			}
			t.Logf("%dx%d, azimuth %.0f, %d ray hits, %d with extra layers", res[0], res[1], angle, crossed, extra)
			if angle == 0 && (extra != 0 || crossed < 700) {
				t.Fatal("anterior shoulder chart crossed itself")
			}
		}
	}
}

func TestPapillaryOpticsUseIndependentRegionalWidth(t *testing.T) {
	s := Adult()
	cx, cy := s.landmarks(s.Left)
	p := maths.V(cx+.004, cy-.008, .10)
	before := s.Optics(p, 431)
	s.Left.PapillaRadius = .007
	after := s.Optics(p, 431)
	if after.Colour.Luminance() >= before.Colour.Luminance() {
		t.Fatal("papilla color did not follow wider geometric papilla")
	}
	s.LegacyContour = true
	if s.Optics(p, 431).Colour == after.Colour {
		t.Fatal("legacy areolar control lost independently reproducible value")
	}
}

func TestAnteriorWallHasRegionalDepthPlanes(t *testing.T) {
	s := Adult()
	// Relative to the rest chart, the upper rib wall advances anteriorly as
	// it approaches the flank. Its central projection stays independently set.
	delta := func(a, y float64) float64 {
		p, rest := s.wall(a, y)
		return p.Z - rest.Z
	}
	if delta(.85, 1.095) < .009 || delta(0, .916) < .009 {
		t.Fatal("missing transverse costal plane or lower abdominal convexity")
	}
}

func TestLoadedInferiorMeridianHasLocalReturn(t *testing.T) {
	s := Adult()
	s.Left.Projection, s.Left.Width, s.Compliance = .086, .097, .76
	x, center := s.landmarks(s.Left)
	var backtrack [2]float64
	for i, load := range []float64{0, 1} {
		s.Gravity = load
		high := math.Inf(-1)
		for j := 0; j <= 150; j++ {
			y := center - .10 + float64(j)*.001
			a := math.Asin(x / (sectionAt(y).w * s.Width))
			p, _ := s.chestPoint(a, y)
			backtrack[i] = math.Max(backtrack[i], high-p.Y)
			high = math.Max(high, p.Y)
			q, _ := s.chestPoint(a-2*math.Pi, y)
			if p.Sub(q).Len() > 1e-9 {
				t.Fatal("circumferential seam", p, q)
			}
		}
	}
	if backtrack[0] != 0 || backtrack[1] < .002 {
		t.Fatal("expected a loaded, localized underturn", backtrack)
	}
	t.Logf("inferior chart return, unloaded %.2f mm and loaded %.2f mm", 1000*backtrack[0], 1000*backtrack[1])
}
