// Package primates constructs a continuous adult torso study in metres.
package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// Review sampling resolves the areolar relief on the same rings as the trunk.
const Around = 416
const Rows = 286

// Side controls the tissue envelope independently of the chest-wall sections.
// Width, projection and vertical landmark offset are metres.
type Side struct{ Width, Projection, Height, AreolaRadius, PapillaRadius, PapillaProjection float64 }
type Torso struct {
	LegacyContour                               bool
	ProximalArms                                bool
	LateralAttachment                           bool
	AgeYears                                    float64
	Undertone                                   Chroma
	Left, Right                                 Side
	Width, Depth, Compliance, Melanin, Vascular float64
	// Arm elevations, forward pitch and axial twist are radians. Gravity is
	// the downward share of load in a supported upright reference posture.
	LeftArm, RightArm, Pitch, Twist, Gravity float64

	paint       *pigment.Paint
	areolarTone AreolarTone
}

// WithPaint supplies an authored mark stack in the unposed torso's metres.
// The same stack follows every pose made from the returned torso.
func (s Torso) WithPaint(paint pigment.Paint) Torso {
	s.paint = &paint
	return s
}

func Adult() Torso {
	return Torso{AgeYears: 38, Left: Side{.084, .061, .0015, .0185, .0040, .0045}, Right: Side{.082, .058, -.001, .018, .0044, .0041}, Width: 1, Depth: 1, Compliance: .47, Melanin: .36, Vascular: .50, Gravity: 1}
}
func between(v, a, b float64) bool { return v >= a && v <= b }
func (s Torso) Check() error       { return s.check(.018) }

func (s Torso) check(minProjection float64) error {
	if s.ProximalArms && (s.LegacyContour || s.LeftArm != 0 || s.RightArm != 0 || s.Pitch != 0 || s.Twist != 0) {
		return fmt.Errorf("proximal arms currently support the new neutral torso only")
	}
	for _, k := range []Side{s.Left, s.Right} {
		if !between(k.Width, .055, .110) || !between(k.Projection, minProjection, .103) || !between(k.Height, -.020, .020) || !between(k.AreolaRadius, .010, .030) || !between(k.PapillaRadius, .002, .008) || !between(k.PapillaProjection, .001, .009) {
			return fmt.Errorf("adult torso: side dimensions outside study range")
		}
	}
	for _, x := range []float64{s.Compliance, s.Melanin, s.Vascular, s.Gravity} {
		if !between(x, 0, 1) {
			return fmt.Errorf("adult torso: invalid fraction")
		}
	}
	if !s.Undertone.valid() {
		return fmt.Errorf("adult torso: chroma outside range")
	}
	if !s.areolarTone.valid() {
		return fmt.Errorf("adult torso: areolar pigment control outside -1 to 1")
	}
	if !between(s.AgeYears, 20, 80) || !between(s.Width, .80, 1.22) || !between(s.Depth, .78, 1.30) || !between(s.Pitch, -.40, .60) || !between(s.Twist, -.55, .55) || !between(s.LeftArm, 0, 2.6) || !between(s.RightArm, 0, 2.6) {
		return fmt.Errorf("adult torso: invalid frame or posture")
	}
	return nil
}

type section struct{ y, w, front, back float64 }

var cage = []section{
	{.80, .176, .093, .121}, {.85, .184, .101, .126}, {.91, .165, .102, .112}, {.99, .138, .087, .099},
	{1.05, .137, .091, .101}, {1.10, .143, .101, .108}, {1.15, .151, .112, .116}, {1.22, .157, .117, .120},
	{1.28, .160, .108, .117}, {1.32, .181, .095, .113}, {1.345, .193, .077, .108}, {1.365, .177, .061, .096},
	{1.385, .127, .049, .081}, {1.405, .086, .045, .064}, {1.43, .058, .048, .055}, {1.455, .053, .049, .053},
}

func sectionAt(y float64) section {
	i := 0
	for i < len(cage)-2 && y > cage[i+1].y {
		i++
	}
	a, b := cage[i], cage[i+1]
	prev, next := cage[max(0, i-1)], cage[min(len(cage)-1, i+2)]
	h := b.y - a.y
	t := (y - a.y) / h
	cubic := func(p, a, b, n float64) float64 {
		return (2*t*t*t-3*t*t+1)*a + (t*t*t-2*t*t+t)*h*(b-p)/(cage[i+1].y-prev.y) + (-2*t*t*t+3*t*t)*b + (t*t*t-t*t)*h*(n-a)/(next.y-cage[i].y)
	}
	return section{y, cubic(prev.w, a.w, b.w, next.w), cubic(prev.front, a.front, b.front, next.front), cubic(prev.back, a.back, b.back, next.back)}
}
func bell(x float64) float64 { return math.Exp(-x * x) }
func (s Torso) side(x float64) (Side, float64, float64) {
	if x >= 0 {
		return s.Left, s.LeftArm, 1
	}
	return s.Right, s.RightArm, -1
}
func (s Torso) landmarks(side Side) (float64, float64) {
	return .081*s.Width + .11*(side.Width-.083), 1.184 + side.Height
}
func (s Torso) legacySample(a, y float64, seed uint64) (maths.Vec, maths.Vec) {
	ring := sectionAt(y)
	sa, ca := math.Sin(a), math.Cos(a)
	x := ring.w * s.Width * sa
	// Front and back depth blend continuously through the lateral wall.
	z := (.5*(ring.front+ring.back)*ca + .5*(ring.front-ring.back)) * s.Depth
	p := maths.V(x, y, z)
	rest := p
	side, arm, sign := s.side(x)
	sideX, sideY := s.landmarks(side)
	ax := math.Abs(x)
	front := maths.Smooth(-.03, .68, ca)
	dx := (ax - sideX) / side.Width
	dy := y - sideY
	// A broader superior slope fades into pectoral fascia. The distributed
	// footprint vanishes across the midline and turns around the lateral wall.
	// The upper footprint is wide and shallow. Its medial attachment changes
	// direction as it ascends, then crosses the sternum as chest wall.
	upper := maths.Smooth(-.032, .06, dy)
	axis := sideX + (.016*s.Width)*maths.Smooth(.006, .1, dy)
	du := (ax - axis) / (side.Width * (1 + .16*upper))
	medial := 1 + .20*(1-maths.Smooth(-.83, -.24, du))
	lowerRadius := .054 + .035*upper + .18*(side.Width-.083)
	if s.LateralAttachment {
		lowerRadius -= .011 * maths.Smooth(.22, 1.15, math.Abs(du)) * (1 - upper)
		du = (ax - axis - .020*maths.Smooth(-.035, .07, dy)) / (side.Width * (.89 + .23*upper))
	}
	v := (dy + .012) / lowerRadius
	radius := du*du*medial*medial + v*v
	root := maths.Smooth(0, .052, ax) * front
	// A broad pectoral transition carries the upper sweep. Below the papilla
	// a rounded lobe retains projection until it turns back toward the wall.
	apron := .29 * math.Exp(-.67*du*du-.65*math.Pow((dy-.002)/.080, 2))
	pole := .71 * math.Exp(-.32*radius-.62*radius*radius)
	envelope := (apron + pole) * root
	elevate := math.Sin(.62 * arm)
	suspend := side.Projection * (.15 + .50*s.Compliance) * s.Gravity * (1 + .22*(s.AgeYears-38)/42)
	lower := math.Exp(-1.12*du*du) * bell((dy+.035)/.031)
	p.Z += side.Projection * (envelope*(.985-.12*elevate) + .12*lower*root)
	sag := suspend * root * (.40*math.Exp(-.68*radius) + .37*math.Exp(-.5*du*du)*bell((dy+.025)/.057))
	p.X += sign * side.Projection * .13 * envelope * dx
	if s.LateralAttachment {
		// Soft tissue projects anterolaterally from the curved thorax. Narrowing the
		// lower footprint away from the papilla lets its underside turn locally.
		p.X += sign * .22 * side.Projection * envelope
		p.Z += front * .011 * bell(x/.051) * bell((y-1.238)/.072)
	}
	p.Y += elevate * (.011*envelope + .032*bell((ax-.147*s.Width)/.045)*bell((y-1.274)/.065))
	// Relief under the lower pole is strongest centrally and fades before
	// the midline and axillary tail. No independent circular seam is built.
	foldY := sideY - .064 - .018*(side.Projection/.061-1) + .023*dx*dx
	p.Z -= front * (.0004 + .0023*s.Compliance*s.Gravity) * bell((y-foldY)/.011) * math.Exp(-2.8*dx*dx) * maths.Smooth(0, .034, ax)
	// Axillary wall narrows below the lateral shoulder.
	p.X -= sign * .011 * bell((y-1.292)/.026) * math.Pow(math.Abs(sa), 4) * (1 - .32*elevate)
	// Axillary tail, anterior axillary pillar, sternum and clavicular arch.
	p.Z += front * (.009 + .008*elevate) * bell((ax-.132*s.Width)/.029) * bell((y-1.258-.16*(ax-.132))/.061)
	p.Z -= front * .0024 * bell(x/.018) * bell((y-1.228)/.065)
	p.Z += front * .0055 * bell((y-(1.344+.016*math.Sin(math.Pi*ax/(.175*s.Width))))/.009) * maths.Smooth(.005, .024, ax)
	p.Z -= front * .006 * bell((y-1.345)/.013) * bell(x/.016)
	p.Z -= front * .006 * bell(x/.010) * bell((y-.975)/.011)
	// An areolar plateau and papilla belong to the same vertex lattice.
	uAreola, vAreola := (ax-sideX)/side.AreolaRadius, (y-sideY+.008)/(.93*side.AreolaRadius)
	local := math.Hypot(uAreola, vAreola)
	papilla, tubercle := areolarRelief(uAreola, vAreola, sign)
	relief := .0018*math.Exp(-1.28*local*local) + .0052*papilla + .00064*tubercle
	p.Z += front * relief
	p.X += front * sign * .28 * relief
	// Fine geometric relief is bounded below a tenth of a millimetre.
	p.Z += front * .00009 * noise.Gradient(seed, p.X*530, p.Y*530, p.Z*530)
	p.Z += .009 * bell((ax-.064)/.04) * bell((y-1.27)/.084) * maths.Smooth(.15, .9, -ca)
	travel := maths.Smooth(.86, 1.39, y)
	roll := maths.Rotation(maths.XAxis, s.Pitch*travel)
	yaw := maths.Rotation(maths.YAxis, s.Twist*travel)
	pivot := maths.V(0, .91, 0)
	posed := yaw.Out(roll.Out(p.Sub(pivot))).Add(pivot)
	posed.Y -= sag
	return posed, rest
}

// Surface builds one watertight skin with shared indices across the entire
// trunk. The two flat end closures label this as a torso section.
func (s Torso) Surface(seed uint64, around, rows int) (thing.Thing, error) {
	if err := s.Check(); err != nil {
		return thing.Thing{}, err
	}
	if around < 32 || rows < 24 {
		return thing.Thing{}, fmt.Errorf("torso resolution too low")
	}
	var pts, rest []maths.Vec
	var faces [][3]int
	sockets := shoulderSockets(around, rows)
	for j := 0; j <= rows; j++ {
		y := cage[0].y + (cage[len(cage)-1].y-cage[0].y)*float64(j)/float64(rows)
		for i := 0; i < around; i++ {
			p, r := s.sample(2*math.Pi*float64(i)/float64(around), y, seed)
			pts = append(pts, p)
			rest = append(rest, r)
			if j > 0 && !(s.ProximalArms && (sockets[0].covers(i, j-1) || sockets[1].covers(i, j-1))) {
				a := (j-1)*around + i
				b := (j-1)*around + (i+1)%around
				faces = append(faces, [3]int{a, b, b + around}, [3]int{a, b + around, a + around})
			}
		}
	}
	for _, j := range []int{0, rows} {
		k := len(pts)
		var p, r maths.Vec
		for i := 0; i < around; i++ {
			p = p.Add(pts[j*around+i])
			r = r.Add(rest[j*around+i])
		}
		pts = append(pts, p.Scale(1/float64(around)))
		rest = append(rest, r.Scale(1/float64(around)))
		for i := 0; i < around; i++ {
			a, b := j*around+i, j*around+(i+1)%around
			if j == 0 {
				a, b = b, a
			}
			faces = append(faces, [3]int{k, a, b})
		}
	}
	if s.ProximalArms {
		for _, socket := range sockets {
			socket.grow(&pts, &rest, &faces)
		}
		pts, rest = referenced(pts, rest, faces)
	}
	return s.skinMesh(seed, pts, rest, faces)
}
func (s Torso) skinMesh(seed uint64, pts, rest []maths.Vec, faces [][3]int) (thing.Thing, error) {
	mesh, err := mesh.NewWithRestPoints(pts, rest, faces, geometry.AreaNormals(pts, faces), true)
	if err != nil {
		return thing.Thing{}, err
	}
	return thing.Thing{Shape: mesh, Material: &thing.Material{Name: "adult torso skin", Texture: skin{s, seed}, Gloss: .025, Index: 1.40, Satin: .72}}, nil
}

// Landmark is a named, reproducible surface location in a constructed specimen.
type Landmark struct {
	Name     string
	Position maths.Vec
}

func (s Torso) Landmarks(seed uint64) []Landmark {
	notch, _ := s.sample(0, 1.345, seed)
	result := []Landmark{{"sternal notch", notch}}
	for _, item := range []struct {
		name string
		sign float64
		side Side
	}{{"left", 1, s.Left}, {"right", -1, s.Right}} {
		x, y := s.landmarks(item.side)
		lowerY := y - .064 - .018*(item.side.Projection/.061-1)
		lowerName := "lower attachment"
		if !s.LegacyContour {
			lowerY = s.lowerTurn(item.side, item.sign)
			lowerName = "lower-pole turn"
		}
		for _, loc := range []struct {
			name string
			x, y float64
		}{{"papilla", x, y - .008}, {lowerName, x, lowerY}, {"anterior axillary point", .143 * s.Width, 1.28}} {
			a := math.Asin(loc.x/(sectionAt(loc.y).w*s.Width)) * item.sign
			q, _ := s.sample(a, loc.y, seed)
			result = append(result, Landmark{item.name + " " + loc.name, q})
		}
	}
	return result
}
