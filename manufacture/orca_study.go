package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// OrcaStudy builds an original, approximately seven-metre orca study.
// Stroke is a dimensionless, authored dorso-ventral tail pose in [-1,1].
// The skin is a single deforming body mesh; fins are separate closed lofts.
func OrcaStudy(stroke float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(stroke) || math.Abs(stroke) > 1 {
		return nil, fmt.Errorf("orca: stroke must be between -1 and 1")
	}
	body, err := orcaBody(stroke)
	if err != nil {
		return nil, err
	}
	coat := &thing.Material{Name: "orca skin with rest-coordinate markings", Texture: orcaCoat{}, Gloss: .20, Satin: .55}
	fin := &thing.Material{Name: "orca fins", Colour: maths.RGB(.025, .032, .038), Gloss: .20, Satin: .55}
	pieces := []thing.Thing{{Shape: body, Material: coat, Motion: thing.Still(at)}}
	for _, rings := range orcaFins(stroke) {
		piece, err := sealPiece(rings, fin, at)
		if err != nil {
			return nil, err
		}
		pieces = append(pieces, piece)
	}
	return pieces, nil
}

type orcaSection struct{ x, y, w, h float64 }

// Dimensions and intermediate profiles are authored, not measurements.
var orcaProfile = []orcaSection{
	{-3.30, 0, .055, .065}, {-3.10, 0, .12, .13}, {-2.70, .01, .20, .23},
	{-2.20, .02, .33, .36}, {-1.60, .025, .50, .52}, {-.9, .035, .70, .70},
	{-.1, .04, .80, .78}, {.65, .045, .80, .76}, {1.35, .04, .75, .70},
	{1.95, .025, .66, .61}, {2.45, 0, .55, .50}, {2.85, -.015, .41, .36},
	{3.20, -.02, .31, .29}, {3.40, -.015, .14, .14}, {3.46, -.015, .012, .014},
}

// orcaBend bends a point of the body by the stroke: the front half stays
// steady, and the displacement and turn build toward the tail.
func orcaBend(p maths.Vec, stroke float64) maths.Vec {
	// The front half is steady; displacement and tangent build aft.
	aft := math.Max(0, .5-p.X) / 3.8
	displace := stroke * .50 * aft * aft
	angle := math.Atan(-stroke * aft / 3.8)
	return maths.V(p.X-p.Y*math.Sin(angle), p.Y*math.Cos(angle)+displace, p.Z)
}

// orcaBody is the body's mesh, lofted through the profile in rest space and
// bent by the stroke.
func orcaBody(stroke float64) (*mesh.Mesh, error) {
	const sections = 32
	var rest, points []maths.Vec
	for i := 0; i < len(orcaProfile)-1; i++ {
		a, b := orcaProfile[i], orcaProfile[i+1]
		before, after := orcaProfile[max(0, i-1)], orcaProfile[min(len(orcaProfile)-1, i+2)]
		for j := 0; j < 4; j++ {
			t := float64(j) / 4
			s := orcaSection{a.x + (b.x-a.x)*t, alfaHermite(before.y, a.y, b.y, after.y, t), alfaHermite(before.w, a.w, b.w, after.w, t), alfaHermite(before.h, a.h, b.h, after.h, t)}
			for k := 0; k < sections; k++ {
				angle := 2 * math.Pi * float64(k) / sections
				p := maths.V(s.x, s.y+s.h*math.Cos(angle), s.w*math.Sin(angle))
				rest = append(rest, p)
				points = append(points, orcaBend(p, stroke))
			}
		}
	}
	s := orcaProfile[len(orcaProfile)-1]
	for k := 0; k < sections; k++ {
		angle := 2 * math.Pi * float64(k) / sections
		p := maths.V(s.x, s.y+s.h*math.Cos(angle), s.w*math.Sin(angle))
		rest = append(rest, p)
		points = append(points, orcaBend(p, stroke))
	}
	rows := len(rest) / sections
	faces := make([][3]int, 0, rows*sections*2)
	for row := 0; row < rows-1; row++ {
		for k := 0; k < sections; k++ {
			n := (k + 1) % sections
			a, b, c, d := row*sections+k, row*sections+n, (row+1)*sections+n, (row+1)*sections+k
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	// Caps have a single centre each and the same boundary winding as sides.
	for _, row := range []int{0, rows - 1} {
		p := orcaProfile[0]
		if row > 0 {
			p = orcaProfile[len(orcaProfile)-1]
		}
		centre := maths.V(p.x, p.y, 0)
		idx := len(points)
		rest = append(rest, centre)
		points = append(points, orcaBend(centre, stroke))
		for k := 0; k < sections; k++ {
			n := (k + 1) % sections
			if row == 0 {
				faces = append(faces, [3]int{idx, n, k})
			} else {
				faces = append(faces, [3]int{idx, row*sections + k, row*sections + n})
			}
		}
	}
	var volume float64
	for _, f := range faces {
		volume += points[f[0]].Dot(points[f[1]].Cross(points[f[2]]))
	}
	if volume < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	return mesh.NewWithRestPoints(points, rest, faces, sealSmoothNormals(points, faces), true)
}

// orcaCoat follows the undeformed body, so patches do not slide while posing.
type orcaCoat struct{}

// At is the coat's colour at a point: black above and white below, with a
// white patch above and behind the eye.
func (orcaCoat) At(p maths.Vec) maths.Light {
	black := maths.RGB(.024, .031, .038)
	white := maths.RGB(.76, .79, .78)
	// The post-ocular patch is above and behind the tiny eye, not the eye.
	eyePatch := 1 - maths.Smooth(.75, 1.0, math.Hypot((p.X-2.02)/.46, (p.Y-.29)/.21))
	eyePatch *= maths.Smooth(.30, .48, math.Abs(p.Z))
	belly := 1 - maths.Smooth(-.52, -.30, p.Y)
	belly *= maths.Smooth(-3.0, -2.25, p.X) * (1 - maths.Smooth(2.85, 3.20, p.X))
	flank := (1 - maths.Smooth(.72, 1, math.Hypot((p.X+1.30)/.75, (p.Y+.44)/.32))) * maths.Smooth(.25, .48, math.Abs(p.Z))
	belly = math.Max(belly, flank)
	saddle := (1 - maths.Smooth(.75, 1, math.Hypot((p.X+.82)/.54, (p.Y-.57)/.26))) * maths.Smooth(.22, .40, math.Abs(p.Z))
	colour := black.Scale(1 - belly).Add(white.Scale(belly))
	colour = colour.Scale(1 - eyePatch).Add(white.Scale(eyePatch))
	return colour.Scale(1 - saddle).Add(maths.RGB(.18, .22, .23).Scale(saddle))
}

// orcaFins is the rings of the fins, each fin a row of rings, bent by the
// stroke.
func orcaFins(stroke float64) [][][]maths.Vec {
	var fins [][][]maths.Vec
	// Swept dorsal fin, rounded by elliptical cross-sections.
	var dorsal [][]maths.Vec
	for _, s := range []struct{ x, y, chord, thick float64 }{{-.15, .70, .64, .13}, {-.26, 1.03, .51, .105}, {-.39, 1.48, .28, .06}, {-.51, 1.78, .055, .016}, {-.52, 1.80, .010, .005}} {
		dorsal = append(dorsal, orcaFinRing(maths.V(s.x, s.y, 0), maths.V(s.chord, 0, 0), maths.V(0, 0, s.thick), stroke))
	}
	fins = append(fins, dorsal)
	for _, side := range []float64{-1, 1} {
		var flipper, fluke [][]maths.Vec
		for _, s := range []struct{ x, y, z, chord, thick float64 }{{1.0, -.35, .60, .42, .10}, {.80, -.48, .88, .46, .09}, {.45, -.60, 1.20, .39, .07}, {.12, -.67, 1.48, .24, .04}, {-.04, -.69, 1.56, .025, .009}} {
			flipper = append(flipper, orcaFinRing(maths.V(s.x, s.y, side*s.z), maths.V(s.chord, 0, 0), maths.V(0, s.thick, 0), stroke))
		}
		for _, s := range []struct{ x, z, chord, thick float64 }{{-3.19, .02, .23, .070}, {-3.23, .35, .34, .060}, {-3.27, .80, .30, .045}, {-3.31, 1.25, .15, .025}, {-3.32, 1.42, .015, .006}} {
			fluke = append(fluke, orcaFinRing(maths.V(s.x, 0, side*s.z), maths.V(s.chord, 0, 0), maths.V(0, s.thick, 0), stroke))
		}
		fins = append(fins, orcaSmoothFin(flipper), orcaSmoothFin(fluke))
	}
	return fins
}

// orcaSmoothFin is a fin's rings with more rings between them, so that the
// fin's outline is smooth.
func orcaSmoothFin(rings [][]maths.Vec) [][]maths.Vec {
	var out [][]maths.Vec
	for i := 0; i < len(rings)-1; i++ {
		before, low, high, after := rings[max(0, i-1)], rings[i], rings[i+1], rings[min(len(rings)-1, i+2)]
		for j := 0; j < 4; j++ {
			t := float64(j) / 4
			ring := make([]maths.Vec, len(low))
			for k := range low {
				ring[k] = maths.V(alfaHermite(before[k].X, low[k].X, high[k].X, after[k].X, t), alfaHermite(before[k].Y, low[k].Y, high[k].Y, after[k].Y, t), alfaHermite(before[k].Z, low[k].Z, high[k].Z, after[k].Z, t))
			}
			out = append(out, ring)
		}
	}
	return append(out, rings[len(rings)-1])
}

// orcaFinRing is an elliptical ring of a fin at a centre with two
// semi-axes, bent by the stroke.
func orcaFinRing(center, a, b maths.Vec, stroke float64) []maths.Vec {
	return sealRing(center, a, b, func(p maths.Vec) maths.Vec { return orcaBend(p, stroke) })
}
