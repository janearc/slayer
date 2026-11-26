package cities

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// helixPoint is a point on the top of the ribbon; radial is the signed
// distance across it. Turns and start-angle are in turns and radians.
func (p CompositionPart) helixPoint(t, radial float64) [3]float64 {
	angle := p.StartAngle + 2*math.Pi*p.Turns*t
	radius := p.Radius + radial
	return [3]float64{p.Centre[0] + radius*math.Cos(angle), p.Centre[1] + p.Rise*t, p.Centre[2] + radius*math.Sin(angle)}
}
func helixVec(p [3]float64) maths.Vec { return maths.V(p[0], p[1], p[2]) }
func (p CompositionPart) helixCheck() error {
	if p.Ground || !finitePoint(p.Centre) || !finiteNumber(p.Radius) || !finiteNumber(p.Width) || !finiteNumber(p.Turns) || !finiteNumber(p.Rise) || !finiteNumber(p.StartAngle) || !finiteNumber(p.Thickness) || !finiteNumber(p.Clearance) || p.Radius <= p.Width/2 || p.Width <= 0 || p.Thickness <= 0 || p.Clearance <= 0 || p.Turns == 0 || math.Abs(p.Turns) > 10 || p.Segments < 16 || p.Segments > 4096 || math.Abs(2*math.Pi*p.Turns/float64(p.Segments)) > math.Pi/4 {
		return fmt.Errorf("part %s: invalid helix dimensions", p.ID)
	}
	// Limit scale and angle so the one millimetre clearance reserve remains
	// conservative for ordinary float64 computations. This is not a formal
	// floating-point error proof.
	const coordinateLimit = 1_000_000.0
	if math.Abs(p.StartAngle) > 2*math.Pi || p.Radius > coordinateLimit || p.Width > coordinateLimit || math.Abs(p.Rise) > coordinateLimit || p.Thickness > coordinateLimit || p.Clearance > coordinateLimit || math.Abs(p.Centre[0])+p.Radius+p.Width/2 > coordinateLimit || math.Abs(p.Centre[2])+p.Radius+p.Width/2 > coordinateLimit || math.Abs(p.Centre[1]) > coordinateLimit || math.Abs(p.Centre[1]+p.Rise) > coordinateLimit {
		return fmt.Errorf("part %s: helix exceeds numerical domain", p.ID)
	}
	if math.Abs(p.Turns) >= 1 {
		perTurn := float64(p.Segments) / math.Abs(p.Turns)
		if perTurn != math.Trunc(perTurn) {
			return fmt.Errorf("part %s: helix tessellation must align between turns", p.ID)
		}
	}
	if math.Abs(p.Turns) >= 1 && math.Abs(p.Rise/p.Turns) < p.Thickness+p.Clearance+0.001 {
		return fmt.Errorf("part %s: helix turns lack clearance", p.ID)
	}
	// Check the actual tessellated driving surface, including the steeper
	// inner edge. The normal gives its maximum slope in every direction.
	for i := 0; i < p.Segments; i++ {
		for _, tri := range p.helixTop(i) {
			n := tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0]))
			horizontal := math.Hypot(n.X, n.Z)
			if !finiteNumber(horizontal) || !finiteNumber(n.Y) || math.Abs(n.Y) == 0 || horizontal/math.Abs(n.Y) > 0.04 {
				return fmt.Errorf("part %s: helix grade exceeds 4%%", p.ID)
			}
		}
	}
	return nil
}
func (p CompositionPart) helixTop(i int) [2]Triangle {
	a, b := float64(i)/float64(p.Segments), float64(i+1)/float64(p.Segments)
	inner0, outer0 := helixVec(p.helixPoint(a, -p.Width/2)), helixVec(p.helixPoint(a, p.Width/2))
	inner1, outer1 := helixVec(p.helixPoint(b, -p.Width/2)), helixVec(p.helixPoint(b, p.Width/2))
	return [2]Triangle{{inner0, inner1, outer1}, {inner0, outer1, outer0}}
}

type helixForm struct{ p CompositionPart }

func (h helixForm) Mesh(Detail) (*Mesh, error) {
	p := h.p
	if err := p.helixCheck(); err != nil {
		return nil, err
	}
	mesh := &Mesh{}
	down := maths.V(0, -p.Thickness, 0)
	add := func(a, b, c maths.Vec, away maths.Vec) {
		mesh.Triangles = append(mesh.Triangles, outward(Triangle{a, b, c}, away))
	}
	for i := 0; i < p.Segments; i++ {
		a, b := float64(i)/float64(p.Segments), float64(i+1)/float64(p.Segments)
		inner0, outer0 := helixVec(p.helixPoint(a, -p.Width/2)), helixVec(p.helixPoint(a, p.Width/2))
		inner1, outer1 := helixVec(p.helixPoint(b, -p.Width/2)), helixVec(p.helixPoint(b, p.Width/2))
		top := p.helixTop(i)
		for _, tri := range top {
			mesh.Triangles = append(mesh.Triangles, outward(tri, maths.YAxis))
			mesh.Triangles = append(mesh.Triangles, outward(Triangle{tri[0].Add(down), tri[1].Add(down), tri[2].Add(down)}, maths.YAxis.Neg()))
		}
		mid := p.StartAngle + 2*math.Pi*p.Turns*(a+b)/2
		radial := maths.V(math.Cos(mid), 0, math.Sin(mid))
		add(inner0, inner1, inner1.Add(down), radial.Neg())
		add(inner0, inner1.Add(down), inner0.Add(down), radial.Neg())
		add(outer0, outer1, outer1.Add(down), radial)
		add(outer0, outer1.Add(down), outer0.Add(down), radial)
	}
	for _, t := range []float64{0, 1} {
		inner, outer := helixVec(p.helixPoint(t, -p.Width/2)), helixVec(p.helixPoint(t, p.Width/2))
		angle := p.StartAngle + 2*math.Pi*p.Turns*t
		direction := maths.V(-math.Sin(angle), 0, math.Cos(angle)).Scale(math.Copysign(1, p.Turns))
		if t == 0 {
			direction = direction.Neg()
		}
		add(inner, outer, outer.Add(down), direction)
		add(inner, outer.Add(down), inner.Add(down), direction)
	}
	return mesh, nil
}

// roadHeading gives a horizontal unit tangent in the direction of travel.
func roadHeading(r CompositionRoad, p CompositionPart, atEnd bool) [2]float64 {
	if r.Type != "helix" {
		dx, dz := r.To[0]-r.From[0], r.To[2]-r.From[2]
		length := math.Hypot(dx, dz)
		return [2]float64{dx / length, dz / length}
	}
	start := p.helixPoint(0, 0)
	forward := math.Abs(r.From[0]-start[0]) <= contactTolerance && math.Abs(r.From[1]-start[1]) <= contactTolerance && math.Abs(r.From[2]-start[2]) <= contactTolerance
	t := 0.0
	if atEnd == forward {
		t = 1
	}
	angle := p.StartAngle + 2*math.Pi*p.Turns*t
	direction := math.Copysign(1, p.Turns)
	if !forward {
		direction = -direction
	}
	return [2]float64{-math.Sin(angle) * direction, math.Cos(angle) * direction}
}
