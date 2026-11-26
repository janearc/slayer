package sauropods

import (
	"math"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// A deliberately picked fixture, not a claim about a fossil animal.
func testProfile() Profile {
	return Profile{Kind: "fixture", Scientific: "illustrative specimen", Basis: "picked", Source: "test fixture", Notes: "illustrative",
		Stations: []Station{
			{X: -8, Y: 1.4, Width: .12, Height: .12}, {X: -5, Y: 1.7, Width: .3, Height: .3},
			{X: -3, Y: 2, Width: .8, Height: .7}, {X: -1.5, Y: 2.3, Width: 1.1, Height: 1},
			{X: 0, Y: 2.4, Width: 1.3, Height: 1.1}, {X: 1.5, Y: 2.5, Width: 1.1, Height: 1},
			{X: 3, Y: 3.2, Width: .5, Height: .55}, {X: 5, Y: 4.2, Width: .28, Height: .32},
			{X: 6, Y: 4.5, Width: .35, Height: .35}, {X: 6.6, Y: 4.5, Width: .12, Height: .15},
		},
		Hind: Limb{Station: 3, FootX: -1.5, FootZ: .9, FootLength: 1, FootWidth: .6},
		Fore: Limb{Station: 5, FootX: 1.5, FootZ: .9, FootLength: .9, FootWidth: .55},
		Head: Head{EyeStation: 8, EyeHeight: .3, EyeRadius: .065},
		Skin: maths.RGB(.32, .31, .25), Underside: maths.RGB(.46, .43, .34)}
}

func contains(box maths.Box, p maths.Vec) bool {
	return p.X >= box.Min.X && p.X <= box.Max.X && p.Y >= box.Min.Y && p.Y <= box.Max.Y && p.Z >= box.Min.Z && p.Z <= box.Max.Z
}

// Four skin openings are joined to the actual feet: no capped cylinders
// crossing a hidden torso. Every mesh edge belongs to two faces and the
// entire animal is a single connected, positively oriented component.
func TestSauropodBodyHasFourPlantedAttachedLimbs(t *testing.T) {
	p := testProfile()
	made, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(made.Parts) != 3 {
		t.Fatalf("parts = %d, want body and paired eyes", len(made.Parts))
	}
	skin, ok := made.Parts[0].Shape.(*mesh.Mesh)
	if !ok || !skin.Closed() || skin.Posed() {
		t.Fatal("body is not one closed unposed mesh")
	}
	points, faces := skin.Points(), skin.Faces()
	if geometry.SignedVolume6(points, faces) <= 0 {
		t.Fatal("body volume is not positive")
	}
	type edge struct{ a, b int }
	edges := map[edge]int{}
	links := make([][]int, len(points))
	for _, face := range faces {
		if points[face[1]].Sub(points[face[0]]).Cross(points[face[2]].Sub(points[face[0]])).Len() == 0 {
			t.Fatal("degenerate face")
		}
		for i := 0; i < 3; i++ {
			a, b := face[i], face[(i+1)%3]
			links[a] = append(links[a], b)
			links[b] = append(links[b], a)
			if a > b {
				a, b = b, a
			}
			edges[edge{a, b}]++
		}
	}
	for e, count := range edges {
		if count != 2 {
			t.Fatalf("edge %+v has %d faces", e, count)
		}
	}
	seen := make([]bool, len(points))
	stack := []int{0}
	seen[0] = true
	for len(stack) > 0 {
		a := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, b := range links[a] {
			if !seen[b] {
				seen[b] = true
				stack = append(stack, b)
			}
		}
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("vertex %d is disconnected", i)
		}
	}
	if len(points)-len(edges)+len(faces) != 2 {
		t.Fatal("body surface is not a closed genus-zero manifold")
	}
	for _, leg := range []Limb{p.Fore, p.Hind} {
		for _, side := range []float64{-1, 1} {
			foot := maths.V(leg.FootX, 0, side*leg.FootZ)
			if d := math.Abs(skin.Distance(foot)); d > 1e-8 {
				t.Fatalf("foot at %v is not grounded: %g", foot, d)
			}
		}
	}
}

func TestSauropodExtentAndEyesArePhysical(t *testing.T) {
	p := testProfile()
	box, err := Extent(p)
	if err != nil {
		t.Fatal(err)
	}
	made, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	if made.Bounds != box {
		t.Fatal("build and analytic extent disagree")
	}
	skin := made.Parts[0].Shape.(*mesh.Mesh)
	for _, part := range made.Parts {
		b, at := part.Shape.Bounds(), part.At(0)
		for _, x := range []float64{b.Min.X, b.Max.X} {
			for _, y := range []float64{b.Min.Y, b.Max.Y} {
				for _, z := range []float64{b.Min.Z, b.Max.Z} {
					if point := at.Out(maths.V(x, y, z)); !contains(box, point) {
						t.Fatalf("part corner %v outside %+v", point, box)
					}
				}
			}
		}
	}
	for _, part := range made.Parts[1:] {
		eye, ok := part.Shape.(shape.Sphere)
		if !ok {
			t.Fatalf("eye is %T", part.Shape)
		}
		distance := skin.Distance(part.At(0).Origin)
		if !(distance < 0 && distance > -eye.Radius) {
			t.Fatalf("eye not partly embedded: %g", distance)
		}
	}
}

func TestSauropodRejectsMutatedProfilesAndKeepsPaintOwned(t *testing.T) {
	p := testProfile()
	made, err := Build(p)
	if err != nil {
		t.Fatal(err)
	}
	paint := made.Parts[0].Material.Texture
	if paint == nil {
		t.Fatal("body lacks local-space paint")
	}
	point := maths.V(0, 2, 0)
	before := paint.At(point)
	p.Skin = maths.RGB(1, 0, 1)
	p.Stations[4].Height *= .7
	if paint.At(point) != before {
		t.Fatal("paint changed with the caller's profile")
	}
	p.Stations[2].X = math.NaN()
	if _, err := Build(p); err == nil {
		t.Fatal("accepted a nonfinite station")
	}
	if _, err := Extent(p); err == nil {
		t.Fatal("bounded a nonfinite station")
	}
}

func TestSauropodAuthoredSpeciesBuild(t *testing.T) {
	file, raw := Embedded()
	book, err := Read(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range book.Species() {
		t.Run(name, func(t *testing.T) {
			p, err := book.Profile(name)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Build(p)
			if err != nil {
				t.Fatal(err)
			}
			skin := got.Parts[0].Shape.(*mesh.Mesh)
			if !skin.Closed() || skin.Posed() || geometry.SignedVolume6(skin.Points(), skin.Faces()) <= 0 {
				t.Fatal("body is not a positively oriented static closed mesh")
			}
			points := skin.Points()
			for _, point := range points {
				if !contains(got.Bounds, point) {
					t.Fatalf("body point outside extent: %v", point)
				}
			}
			for _, face := range skin.Faces() {
				if points[face[1]].Sub(points[face[0]]).Cross(points[face[2]].Sub(points[face[0]])).Len() == 0 {
					t.Fatal("degenerate face")
				}
			}
			for _, part := range got.Parts[1:] {
				eye := part.Shape.(shape.Sphere)
				center := part.At(0).Origin
				if d := skin.Distance(center); !(d < 0 && d > -eye.Radius) {
					t.Fatalf("eye not partly embedded: %g", d)
				}
				for _, x := range []float64{-eye.Radius, eye.Radius} {
					for _, y := range []float64{-eye.Radius, eye.Radius} {
						for _, z := range []float64{-eye.Radius, eye.Radius} {
							if !contains(got.Bounds, center.Add(maths.V(x, y, z))) {
								t.Fatal("eye outside extent")
							}
						}
					}
				}
			}
			for _, leg := range []Limb{p.Fore, p.Hind} {
				for _, side := range []float64{-1, 1} {
					if d := math.Abs(skin.Distance(maths.V(leg.FootX, 0, side*leg.FootZ))); d > 1e-8 {
						t.Fatalf("ungrounded foot: %g", d)
					}
				}
			}
		})
	}
}

// Each planted foot broadens below a narrower ankle; this catches a straight
// column while leaving the ground sole and shared-index skin unchanged.
func TestSauropodFeetBroadenBelowAnkles(t *testing.T) {
	book := checkedBook(t)
	for _, name := range book.Species() {
		p, err := book.Profile(name)
		if err != nil {
			t.Fatal(err)
		}
		made, err := Build(p)
		if err != nil {
			t.Fatal(err)
		}
		points := made.Parts[0].Shape.(*mesh.Mesh).Points()
		for _, leg := range []Limb{p.Fore, p.Hind} {
			for _, side := range []float64{-1, 1} {
				type span struct{ low, high float64 }
				levels := map[int64]span{}
				for _, point := range points {
					if point.Y <= 0 || math.Abs(point.X-leg.FootX) > leg.FootLength*.6 || math.Abs(point.Z-side*leg.FootZ) > leg.FootWidth*.6 {
						continue
					}
					key := int64(math.Round(point.Y * 1e7))
					if current, ok := levels[key]; ok {
						levels[key] = span{math.Min(current.low, point.X), math.Max(current.high, point.X)}
					} else {
						levels[key] = span{point.X, point.X}
					}
				}
				keys := make([]int64, 0, len(levels))
				for key, v := range levels {
					if v.high > v.low {
						keys = append(keys, key)
					}
				}
				sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
				if len(keys) < 2 {
					t.Fatalf("%s: missing foot/ankle rings", name)
				}
				foot, ankle := levels[keys[0]], levels[keys[1]]
				if foot.high-foot.low <= 1.2*(ankle.high-ankle.low) {
					t.Fatalf("%s: foot does not broaden below ankle", name)
				}
			}
		}
	}
}

// The head marking cannot extend backward from the eye into the neck.
func TestSauropodMouthDoesNotReachTheNeck(t *testing.T) {
	p := testProfile()
	c := coat{profile: p}
	ss := sections(p.Stations)
	eye := p.Head.EyeStation * steps
	s := ss[eye-2]
	point := s.center.Add(s.up.Scale(-.28 * s.height)).Add(maths.V(0, 0, .95*s.width))
	base := maths.RGB(.6, .4, .2)
	if got := c.face(point, s.center.Y, s.height, s.width, 0, base); got != base {
		t.Fatalf("neck acquired a jaw line: %v", got)
	}
}

func TestSauropodFacePaintHasPairedFeatures(t *testing.T) {
	p := testProfile()
	c := coat{profile: p}
	stations := p.Stations
	start := stations[p.Head.EyeStation].X
	end := stations[len(stations)-1].X
	sample := func(at, level, side float64) maths.Light {
		x := start + (end-start)*at
		i := 0
		for i < len(stations)-2 && x > stations[i+1].X {
			i++
		}
		a, b := stations[i], stations[i+1]
		f := (x - a.X) / (b.X - a.X)
		y, _ := centreY(stations, i, f)
		h := a.Height + (b.Height-a.Height)*f
		w := a.Width + (b.Width-a.Width)*f
		return c.At(maths.V(x, y+level*h, side*w*.95))
	}
	for _, side := range []float64{-1, 1} {
		mouth, near := sample(.5, -.28, side), sample(.5, -.08, side)
		nose, adjacent := sample(.78, .35, side), sample(.78, .55, side)
		if mouth.R >= near.R*.5 {
			t.Fatalf("mouth missing on side %v: %v %v", side, mouth, near)
		}
		if nose.R >= adjacent.R*.5 {
			t.Fatalf("nostril missing on side %v: %v %v", side, nose, adjacent)
		}
	}
}

func TestSauropodRefusesCrossingNeckSilhouette(t *testing.T) {
	p := testProfile()
	ys := []float64{2.4416417077897252, 3.6463818167513384, 2, 2.3, 2.4, 2.5, 3.2, 1.7531181668724185, 7.6434088698678355, 13.865063817543204}
	hs := []float64{1.4430721756661666, 2.8234180668808957, .7700277006361805, 1.716542427624639, .2483966335652714, .7311981289920995, 2.1585097880096558, 1.2728947518154983, 3.226118394740731, .8816852687443753}
	for i := range p.Stations {
		p.Stations[i].Y, p.Stations[i].Height = ys[i], hs[i]
	}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "crossing axial silhouette") {
		t.Fatalf("expected sampled silhouette refusal, got %v", err)
	}
	raw, err := yaml.Marshal(struct {
		Profiles []Profile `yaml:"profiles"`
	}{[]Profile{p}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Read("folded.yaml", raw); err == nil || !strings.Contains(err.Error(), "crossing axial silhouette") {
		t.Fatalf("book accepted folded silhouette: %v", err)
	}
	if _, err := Extent(p); err == nil {
		t.Fatal("bounded a folded axial silhouette")
	}
	if _, err := Build(p); err == nil {
		t.Fatal("built a folded axial silhouette")
	}
}

func TestSauropodCenterlineIsSmoothAndBounded(t *testing.T) {
	p := testProfile()
	stations := p.Stations
	for i := 0; i < len(stations)-1; i++ {
		a, b := stations[i], stations[i+1]
		for j := 0; j <= 100; j++ {
			y, _ := centreY(stations, i, float64(j)/100)
			if y < math.Min(a.Y, b.Y)-1e-12 || y > math.Max(a.Y, b.Y)+1e-12 {
				t.Fatalf("span %d leaves authored range: %g", i, y)
			}
		}
		if i+1 < len(stations)-1 {
			y0, d0 := centreY(stations, i, 1)
			y1, d1 := centreY(stations, i+1, 0)
			if math.Abs(y0-y1) > 1e-12 || math.Abs(d0-d1) > 1e-12 {
				t.Fatalf("kink at station %d", i+1)
			}
		}
	}
	mid, _ := centreY(stations, 6, .5)
	if math.Abs(mid-(stations[6].Y+stations[7].Y)/2) < 1e-5 {
		t.Fatal("fixture does not exercise a curved neck span")
	}
}
