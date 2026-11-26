package theropods

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

func allosaurusAt(profile theropodProfile, seconds float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return nil, fmt.Errorf("allosaurus: non-finite time")
	}
	root := maths.NewSubspace(maths.V(seconds*(.56/1.2)*profile.Scale, 0, 0), maths.Identity, profile.Scale).Within(at)
	hide := &thing.Material{Name: "speculative neutral hide", Colour: maths.RGB(profile.Hide[0], profile.Hide[1], profile.Hide[2]), Gloss: .07, Satin: .48}
	keratin := &thing.Material{Name: "speculative claws and mouth", Colour: maths.RGB(profile.Keratin[0], profile.Keratin[1], profile.Keratin[2]), Gloss: .12}
	eye := &thing.Material{Name: "speculative eye", Colour: maths.RGB(profile.Eye[0], profile.Eye[1], profile.Eye[2]), Gloss: .3}
	main, e := allosaurusBodyWithStations(seconds, profile.Stations)
	if e != nil {
		return nil, e
	}
	pieces := []thing.Thing{{Shape: main, Material: hide, Motion: thing.Still(root)}}
	// A generic loft spans control joints. Attachments on the head are carried
	// by the same skin deformation; limb lofts are freshly posed at each time.
	loft := func(center []maths.Vec, rad [][2]float64, mat *thing.Material, head bool) error {
		piece, err := geometry.JointLoft(center, rad, mat, root, func(q maths.Vec) maths.Vec {
			if head {
				return allosaurusDeform(q, seconds)
			}
			return q
		})
		if err != nil {
			return err
		}
		pieces = append(pieces, piece)
		return nil
	}
	dot := func(p maths.Vec, r float64, mat *thing.Material) {
		pieces = append(pieces, thing.Thing{Shape: shape.Sphere{Radius: r}, Material: mat, Motion: thing.Still(maths.NewSubspace(allosaurusDeform(p, seconds), maths.Identity, 1).Within(root))})
	}
	// A shallow lower jaw is separate and intentionally does not claim teeth
	// or lips as known external soft tissue.
	if err := loft([]maths.Vec{maths.V(2.04, 2.09, 0), maths.V(2.37, 2.07, 0), maths.V(2.78, 2.08, 0), maths.V(3.14, 2.16, 0)}, [][2]float64{{.09, .15}, {.16, .19}, {.14, .17}, {.035, .055}}, hide, true); err != nil {
		return nil, err
	}
	for _, side := range []float64{-1, 1} {
		hip, knee, ankle, heel := allosaurusLeg(seconds, side)
		hipInside := hip.Add(maths.V(-.10, .14, -side*.09))
		thigh := hip.Add(knee.Sub(hip).Scale(.47))
		shin := knee.Add(ankle.Sub(knee).Scale(.52))
		if err := loft([]maths.Vec{hipInside, hip, thigh, knee, shin, ankle, heel}, [][2]float64{{.28, .25}, {.31, .24}, {.27, .22}, {.155, .14}, {.12, .105}, {.09, .085}, {.10, .09}}, hide, false); err != nil {
			return nil, err
		}
		for toe := -1; toe <= 1; toe++ {
			p := maths.V(heel.X, heel.Y, side*.36+float64(toe)*.13)
			q := p.Add(maths.V(.43, 0, float64(toe)*.06))
			tip := q.Add(maths.V(.16, -.025, float64(toe)*.02))
			if err := loft([]maths.Vec{p, p.Add(q.Sub(p).Scale(.5)), q}, [][2]float64{{.095, .085}, {.07, .06}, {.045, .04}}, hide, false); err != nil {
				return nil, err
			}
			if err := loft([]maths.Vec{q, tip}, [][2]float64{{.045, .04}, {.006, .006}}, keratin, false); err != nil {
				return nil, err
			}
		}
		sh := maths.V(1, 1.75, side*.38)
		el := maths.V(.93, 1.40, side*.48)
		wr := maths.V(1.16, 1.22, side*.44)
		if err := loft([]maths.Vec{sh, el, wr}, [][2]float64{{.15, .13}, {.10, .09}, {.065, .055}}, hide, false); err != nil {
			return nil, err
		}
		for finger := -1; finger <= 1; finger++ {
			p := wr.Add(maths.V(0, float64(finger)*.065, 0))
			q := p.Add(maths.V(.20, -.11, -side*.025))
			if err := loft([]maths.Vec{p, q, q.Add(maths.V(.07, -.07, 0))}, [][2]float64{{.04, .04}, {.022, .022}, {.006, .006}}, keratin, false); err != nil {
				return nil, err
			}
		}
		dot(maths.V(2.27, 2.49, side*.27), .04, eye)
		if err := loft([]maths.Vec{maths.V(2.17, 2.59, side*.23), maths.V(2.27, 2.70, side*.23)}, [][2]float64{{.075, .065}, {.012, .012}}, hide, true); err != nil {
			return nil, err
		}
		if err := loft([]maths.Vec{maths.V(2.08, 2.18, side*.21), maths.V(2.55, 2.15, side*.20), maths.V(3.10, 2.20, side*.11)}, [][2]float64{{.009, .009}, {.009, .009}, {.006, .006}}, keratin, true); err != nil {
			return nil, err
		}
	}
	return pieces, nil
}

// Cross sections rotate approximately with the tail's centerline. The
// differentially weighted rotation can compress tissue; it is not a solver.
func allosaurusDeform(p maths.Vec, seconds float64) maths.Vec {
	s := math.Sin(math.Pi * seconds)
	if p.X < -.4 {
		w := math.Min(1, (-.4-p.X)/3.8)
		angle := math.Atan(-.64 * s * w / 3.8)
		z := p.Z
		p.X -= z * math.Sin(angle)
		p.Z = .32*s*w*w + z*math.Cos(angle)
	}
	if p.X > 1 {
		w := math.Min(1, (p.X-1)/1.2)
		angle := .07 * s * w
		dx, dy := p.X-1, p.Y-1.95
		p.X = 1 + dx*math.Cos(angle) - dy*math.Sin(angle)
		p.Y = 1.95 + dx*math.Sin(angle) + dy*math.Cos(angle)
	}
	return p
}

func allosaurusBodyWithStations(seconds float64, stations [][5]float64) (*mesh.Mesh, error) {
	// x, center y, height radius, width radius, squareness (authored).
	const around, steps = 32, 4
	signedPow := func(x, p float64) float64 { return math.Copysign(math.Pow(math.Abs(x), p), x) }
	var rest, posed []maths.Vec
	row := func(s [5]float64) {
		for j := 0; j < around; j++ {
			a := 2 * math.Pi * float64(j) / around
			p := maths.V(s[0], s[1]+s[2]*signedPow(math.Cos(a), s[4]), s[3]*signedPow(math.Sin(a), s[4]))
			rest = append(rest, p)
			posed = append(posed, allosaurusDeform(p, seconds))
		}
	}
	for i := 0; i < len(stations)-1; i++ {
		for k := 0; k < steps; k++ {
			t := float64(k) / steps
			lo, hi := stations[i], stations[i+1]
			var q [5]float64
			q[0] = lo[0] + (hi[0]-lo[0])*t
			for d := 1; d < 5; d++ {
				q[d] = geometry.CatmullRom(stations[max(i-1, 0)][d], lo[d], hi[d], stations[min(i+2, len(stations)-1)][d], t)
			}
			row(q)
		}
	}
	row(stations[len(stations)-1])
	rows := len(rest) / around
	for _, i := range []int{0, rows - 1} {
		s := stations[0]
		if i > 0 {
			s = stations[len(stations)-1]
		}
		p := maths.V(s[0], s[1], 0)
		rest = append(rest, p)
		posed = append(posed, allosaurusDeform(p, seconds))

	}
	faces := geometry.IndexedLoftFaces(rows, around)
	if geometry.SignedVolume6(posed, faces) < 0 {
		geometry.ReverseFaces(faces)
	}
	return mesh.NewWithRestPoints(posed, rest, faces, geometry.SmoothNormals(posed, faces), true)
}

// allosaurusLeg is the four points of one leg's chain at the time in
// seconds, on a two-second stride; side is the sign of the leg's z.
func allosaurusLeg(seconds, side float64) (maths.Vec, maths.Vec, maths.Vec, maths.Vec) {
	phase := math.Mod(seconds/2, 1)
	if phase < 0 {
		phase++
	}
	if side > 0 {
		phase = math.Mod(phase+.5, 1)
	}
	x, lift := 0.0, 0.0
	if phase < .6 {
		x = .28 - .56*phase/.6
	} else {
		u := (phase - .6) / .4
		x = -.28 + .56*u
		lift = .20 * math.Sin(math.Pi*u)
	}
	hip := maths.V(0, 1.65, side*.36)
	ankle := maths.V(x, .40+lift, side*.36)
	heel := maths.V(x, .095+lift, side*.36)
	dx, dy := ankle.X-hip.X, ankle.Y-hip.Y
	distance := math.Hypot(dx, dy)
	upper, lower := .88, .73
	along := (upper*upper - lower*lower + distance*distance) / (2 * distance)
	height := math.Sqrt(math.Max(0, upper*upper-along*along))
	knee := maths.V(hip.X+along*dx/distance-height*dy/distance, hip.Y+along*dy/distance+height*dx/distance, hip.Z)
	return hip, knee, ankle, heel
}
