package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// orcaTestPaintSlope is the most the paint's colour changes with a
// metre's move, per channel. The sharpest edge in the paint is the
// teeth's, a step from .002 to .006 of the tooth field, which a smooth
// step crosses at 1.5 / .004 = 375 a metre at its steepest; it mixes
// ivory against a colour no more than .78 away, so a channel changes by
// under 375 * .78 = 293 a metre. Four hundred is over it. Measured over
// the matched hits, the greatest was 108.
const orcaTestPaintSlope = 400.

// orcaTestColourGap is the largest difference between two colours in
// any channel.
func orcaTestColourGap(one, two maths.Light) float64 {
	greenBlue := math.Max(math.Abs(one.G-two.G), math.Abs(one.B-two.B))
	return math.Max(math.Abs(one.R-two.R),
		greenBlue)
}

func TestOrcaClean05SkinRestPaintAndMotion(t *testing.T) {
	at := maths.NewSubspace(maths.Vec{}, maths.Identity, 1)
	var skins []*mesh.Mesh
	var mats []interface{ At(maths.Vec) maths.Light }
	for _, phase := range []float64{0, math.Pi / 2, 2 * math.Pi} {
		parts, err := OrcaClean05(phase, at)
		if err != nil {
			t.Fatal(err)
		}
		skin, ok := parts[0].Shape.(*mesh.Mesh)
		if !ok || !skin.Closed() {
			t.Fatal("skin is not closed")
		}
		skins = append(skins, skin)
		mats = append(mats, parts[0].Material.Texture)
		pointCount := len(skin.Points())
		t.Logf("phase %.2f points %d faces %d bounds %+v", phase, pointCount, skin.FaceCount(), skin.Bounds())
	}
	p0, p1, p2 := skins[0].Points(), skins[1].Points(), skins[2].Points()
	links := make([][]int, len(p0))
	minRatio, maxRatio := math.Inf(1), 0.0
	for _, f := range skins[0].Faces() {
		for j := 0; j < 3; j++ {
			a, b := f[j], f[(j+1)%3]
			links[a] = append(links[a], b)
			links[b] = append(links[b], a)
			r := p1[a].Sub(p1[b]).Len() / p0[a].Sub(p0[b]).Len()
			minRatio = math.Min(minRatio, r)
			maxRatio = math.Max(maxRatio, r)
		}
	}
	seen := make([]bool, len(links))
	seen[0] = true
	stack := []int{0}
	count := 0
	for len(stack) > 0 {
		a := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		for _, b := range links[a] {
			if !seen[b] {
				seen[b] = true
				stack = append(stack, b)
			}
		}
	}
	if count != len(links) {
		t.Fatalf("only %d / %d points connected", count, len(links))
	}
	t.Logf("connected points %d, triangle edge-length ratios %.4f to %.4f at phase pi/2", count, minRatio, maxRatio)

	maximum, gap := 0., 0.
	for i, p := range p0 {
		maximum = math.Max(maximum, p.Sub(p1[i]).Len())
		gap = math.Max(gap, p.Sub(p2[i]).Len())
	}
	if maximum < .1 || gap > 1e-9 {
		t.Fatalf("motion %.6g loop gap %.6g", maximum, gap)
	}
	// Shoot at interior face centroids from the local normal. Compare the
	// two real intersections only when both hits identify the intended face.
	matched := 0
	for f := 48; f < skins[0].FaceCount(); f += 101 {
		var rest [2]maths.Vec
		valid := true
		for k := 0; k < 2; k++ {
			a, b, c := skins[k].Corners(f)
			center := a.Add(b).Add(c).Scale(1. / 3)
			n := b.Sub(a).Cross(c.Sub(a)).Unit()
			start := center.Add(n.Scale(.002))
			met, ok := skins[k].Meet(maths.Ray{At: start, Dir: n.Scale(-1)}, 0, .004)
			if !ok || met.Face() != f {
				valid = false
				break
			}
			rest[k], ok = skins[k].RestPointAt(met)
			if !ok {
				t.Fatal("missing rest coordinates")
			}
		}
		if !valid {
			continue
		}
		if rest[0].Sub(rest[1]).Len() > 1e-8 {
			t.Fatalf("rest paint moved: %v / %v", rest[0], rest[1])
		}
		// The two rest points agree to rounding, not to the bit, so the
		// colours may differ by what the paint changes over that gap.
		gap := rest[0].Sub(rest[1]).Len()
		if orcaTestColourGap(mats[0].At(rest[0]), mats[1].At(rest[1])) >
			orcaTestPaintSlope*gap {
			t.Fatal("rest-paint colour changed")
		}
		matched++
	}
	if matched < 20 {
		t.Fatalf("only %d matched ray hits", matched)
	}
	// Signed volumes report an observed global change, not a proof that the
	// local deformation avoids self-intersections.
	volume := func(s *mesh.Mesh) float64 {
		v := 0.
		for i := 0; i < s.FaceCount(); i++ {
			a, b, c := s.Corners(i)
			v += a.Dot(b.Cross(c)) / 6
		}
		return v
	}
	v0, v1 := volume(skins[0]), volume(skins[1])
	t.Logf("max displacement %.4fm, loop gap %.2g, %d matched rest-paint hits, signed volume %.6f→%.6fm³ (%.3f%%)", maximum, gap, matched, v0, v1, 100*(v1/v0-1))
}
