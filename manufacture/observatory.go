package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Optical describes an authored optical observatory, in metres and angles.
// Its surfaces suggest an instrument; they are not a calibrated optical model.
type Optical struct {
	Diameter, Aperture         measure.Length
	Opening, Azimuth, Altitude measure.Angle
	Segmented                  bool
}

// Observatory makes a recipe with a checked enclosure and pointing.
func Observatory(spec Optical) (kind, error) {
	diameter, aperture := spec.Diameter.InMetres(), spec.Aperture.InMetres()
	opening := spec.Opening.InDegrees()
	if !positive(diameter) || !positive(aperture) || aperture >= diameter*0.6 ||
		diameter > 200 || opening < 5 || opening > 90 || !finite(opening) ||
		!finite(spec.Azimuth.InDegrees()) || spec.Altitude.InDegrees() < 0 ||
		spec.Altitude.InDegrees() > 90 || !finite(spec.Altitude.InDegrees()) {
		return kind{}, fmt.Errorf("manufacture: invalid optical observatory dimensions or pointing")
	}
	return kind{name: "optical-observatory", how: optical{spec}}, nil
}

type optical struct{ spec Optical }

// made retains the default preview representation.
func (row optical) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt constructs the same enclosure and pointing at each detail.
func (row optical) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	sides := map[string]int{"hull": 12, "clumps": 20, "quick": 40, "demo": 80}[detail]
	if sides == 0 {
		return good{}, fmt.Errorf("manufacture: unknown complexity %q", detail)
	}
	radius, aperture := row.spec.Diameter.InMetres()/2, row.spec.Aperture.InMetres()
	base, thick := radius*0.38, radius*0.025
	made := good{dressed: looks{
		body: plain("observatory enclosure", 0.78, 0.79, 0.77),
		leaf: plain("telescope structure", 0.09, 0.11, 0.13),
		2:    &thing.Material{Name: "primary mirror", Colour: maths.Grey(0.75), Gloss: 1, Index: 8},
	}}
	add := func(shape form, at maths.Subspace, of slot) {
		made.parts = append(made.parts, part{shape: shape, fine: cut{around: sides}, space: at, of: of})
	}
	// A hollow drum supports the dome; wall thickness is authored.
	add(turnedProfile{[][2]float64{{radius, 0}, {radius, base},
		{radius - thick, base}, {radius - thick, 0}}}, maths.Subspace{}, body)
	add(domeShell{radius: radius, thick: thick, opening: row.spec.Opening.InRadians()},
		maths.Subspace{Origin: maths.V(0, base, 0),
			Orientation: maths.About(maths.YAxis, -row.spec.Azimuth.InRadians())}, body)
	// The pedestal, fork and truss share the telescope's azimuth.
	mount := maths.Subspace{Orientation: maths.About(maths.YAxis, -row.spec.Azimuth.InRadians())}
	box := func(w, h, d float64, at maths.Vec) {
		add(boxShape(w, h, d), maths.Subspace{Origin: mount.Out(at), Orientation: mount.Orientation}, leaf)
	}
	box(aperture*0.35, base*0.7, aperture*0.35, maths.V(0, 0, 0))
	for _, sign := range []float64{-1, 1} {
		box(aperture*0.10, radius*0.60, aperture*0.18, maths.V(sign*aperture*0.65, base*0.6, 0))
	}
	alt := row.spec.Altitude.InRadians()
	axis := mount.Orientation.Out(maths.V(0, math.Sin(alt), -math.Cos(alt)))
	centre := maths.V(0, base+radius*0.38, 0)
	length := aperture * 1.2
	start := centre.Sub(axis.Scale(length * 0.35))
	space, err := spaceAlong(start, axis)
	if err != nil {
		return good{}, err
	}
	// A shallow paraboloid closes behind the primary.
	mirror := mirrorDish{radius: aperture / 2, depth: aperture * 0.045,
		segmented: row.spec.Segmented && detail != "hull"}
	add(mirror, space, 2)
	for i := range 6 {
		angle := fullTurn * float64(i) / 6
		local := maths.V(math.Cos(angle)*aperture*0.54, 0, -math.Sin(angle)*aperture*0.54)
		at := space.Out(local)
		strut, _ := newTube(measure.Metres(length), measure.Metres(aperture*0.012), measure.Metres(aperture*0.009))
		placed := space
		placed.Origin = at
		add(strut, placed, leaf)
	}
	// Two rings make the open truss readable at every complexity.
	for _, height := range []float64{0, length} {
		ring := turnedProfile{[][2]float64{{aperture * 0.55, height},
			{aperture * 0.55, height + aperture*0.04},
			{aperture * 0.50, height + aperture*0.04}, {aperture * 0.50, height}}}
		add(ring, space, leaf)
	}
	boxExtent := maths.Box{Min: maths.V(-radius, 0, -radius), Max: maths.V(radius, base+radius, radius)}
	made.extent = &boxExtent
	return made, nil
}

// boxShape uses the already checked box primitive inside an assembly.
func boxShape(w, h, d float64) form {
	return box{measure.Metres(w), measure.Metres(h), measure.Metres(d)}
}

// turnedProfile revolves a closed radius/height profile about y.
type turnedProfile struct{ points [][2]float64 }

// skin revolves the closed profile without duplicating seam vertices.
func (shape turnedProfile) skin(fine cut) surface {
	var skin surface
	point := func(p [2]float64, side int) maths.Vec {
		angle := fullTurn * float64(side%fine.around) / float64(fine.around)
		return maths.V(p[0]*math.Cos(angle), p[1], -p[0]*math.Sin(angle))
	}
	for i, p := range shape.points {
		q := shape.points[(i+1)%len(shape.points)]
		for side := range fine.around {
			skin = quad(skin, point(p, side), point(p, side+1), point(q, side+1), point(q, side))
		}
	}
	return skin
}

// These are the defined corner indices in a face's stored winding order.
const (
	quadFirstCorner = iota
	quadSecondCorner
	quadThirdCorner
)

// quad retains only non-degenerate triangles, including at an axial pole.
func quad(skin surface, a, b, c, d maths.Vec) surface {
	for _, triangle := range []face{{a, b, c}, {a, c, d}} {
		// Coincident edges can leave a cross-product roundoff residue.
		first := triangle[quadFirstCorner]
		second := triangle[quadSecondCorner]
		third := triangle[quadThirdCorner]
		if first == second || second == third || third == first {
			continue
		}
		if triangle[1].Sub(triangle[0]).Cross(triangle[2].Sub(triangle[0])).Len() > 1e-14 {
			skin = append(skin, triangle)
		}
	}
	return skin
}

// domeShell is a hollow hemisphere with a real observing slit.
type domeShell struct{ radius, thick, opening float64 }

// skin joins inner and outer walls around the observing slit.
func (shape domeShell) skin(fine cut) surface {
	var skin surface
	levels := max(3, fine.around/4)
	point := func(level, side int, inside bool) maths.Vec {
		angle := shape.opening/2 + (fullTurn-shape.opening)*float64(side)/float64(fine.around)
		up := math.Pi / 2 * float64(level) / float64(levels)
		r := shape.radius
		if inside {
			r -= shape.thick
		}
		if level == levels {
			return maths.V(0, r, 0)
		}
		return maths.V(r*math.Cos(up)*math.Sin(angle), r*math.Sin(up), -r*math.Cos(up)*math.Cos(angle))
	}
	for side := range fine.around {
		for level := range levels {
			skin = quad(skin, point(level, side, false), point(level, side+1, false), point(level+1, side+1, false), point(level+1, side, false))
			skin = quad(skin, point(level, side, true), point(level+1, side, true), point(level+1, side+1, true), point(level, side+1, true))
		}
		skin = quad(skin, point(0, side, false), point(0, side, true), point(0, side+1, true), point(0, side+1, false))
	}
	for level := range levels {
		skin = quad(skin, point(level, 0, false), point(level+1, 0, false), point(level+1, 0, true), point(level, 0, true))
		skin = quad(skin, point(level, fine.around, false), point(level, fine.around, true), point(level+1, fine.around, true), point(level+1, fine.around, false))
	}
	for i := range skin {
		skin[i][1], skin[i][2] = skin[i][2], skin[i][1]
	}
	return skin
}

// mirrorDish is a backed shallow reflector. Segments are separate hexagonal
// panels with authored gaps; their figure is a visual approximation.
type mirrorDish struct {
	radius, depth float64
	segmented     bool
}

// skin makes a backed dish or separated hexagonal panels.
func (shape mirrorDish) skin(fine cut) surface {
	if shape.segmented {
		return shape.panels()
	}
	steps := max(2, fine.around/8)
	profile := [][2]float64{{0, shape.backPlane()}, {shape.radius, shape.backPlane()}}
	profile = append(profile, [2]float64{shape.radius, shape.depth})
	for i := steps - 1; i >= 0; i-- {
		r := shape.radius * float64(i) / float64(steps)
		profile = append(profile, [2]float64{r, shape.depth * r * r / (shape.radius * shape.radius)})
	}
	return (turnedProfile{profile}).skin(fine)
}

// backPlane is the authored backing surface, shared by supports and the dish.
func (shape mirrorDish) backPlane() float64 { return -shape.depth * 0.2 }

// panels leaves authored gaps between closed mirror tiles.
func (shape mirrorDish) panels() surface {
	var skin surface
	span := shape.radius / 5
	for q := -4; q <= 4; q++ {
		for r := -5; r <= 5; r++ {
			x, z := 1.5*span*float64(q), math.Sqrt(3)*span*(float64(r)+float64(q)/2)
			if math.Hypot(x, z)+span > shape.radius {
				continue
			}
			height := func(x, z float64) float64 {
				return shape.depth * (x*x + z*z) / (shape.radius * shape.radius)
			}
			centre := maths.V(x, height(x, z), z)
			bottom := maths.V(x, -shape.depth*0.2, z)
			for i := range 6 {
				corner := func(i int) maths.Vec {
					angle := fullTurn * float64(i%6) / 6
					x, z := x+span*0.96*math.Cos(angle), z-span*0.96*math.Sin(angle)
					return maths.V(x, height(x, z), z)
				}
				a, b := corner(i), corner(i+1)
				lowA, lowB := maths.V(a.X, bottom.Y, a.Z), maths.V(b.X, bottom.Y, b.Z)
				skin = append(skin, face{centre, a, b}, face{bottom, lowB, lowA})
				skin = quad(skin, a, lowA, lowB, b)
			}
		}
	}
	return skin
}
