package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// owlHousing is an authored two-axis retractable service enclosure. Its two
// quarter-barrel shells are parked behind the instrument and spread sideways:
// the longitudinal bogies bring it over the telescope before the transverse
// carriages close the roof. These are fictional site engineering choices.
func (s *heardStage) owlHousing() error {
	steel := &thing.Material{Name: "OWL housing steel frame", Colour: maths.RGB(.13, .18, .22), Gloss: .25}
	slab := &thing.Material{Name: "OWL housing foundation", Colour: maths.RGB(.28, .29, .3)}
	light := maths.RGB(.12, .065, .025)
	rod := func(a, b maths.Vec, r float64, name string, glow maths.Light) error {
		p, err := s.characterRod(a, b, r, name, glow)
		if err == nil {
			p.Material.Colour = steel.Colour
		}
		return err
	}
	box := func(at, size maths.Vec, mat *thing.Material) error { return s.housingBox(at, size, mat) }
	// Longitudinal tracks carry the closed shelter; cross-slide rails at the
	// parked station let both roof halves open without blocking the aperture.
	for _, x := range []float64{-188, -181, 181, 188} {
		if err := box(maths.V(x, .1, 92), maths.V(1.1, .5, 370), steel); err != nil {
			return err
		}
	}
	for _, z := range []float64{82, 88, 252, 258} {
		if err := box(maths.V(0, .55, z), maths.V(430, .45, 1.1), steel); err != nil {
			return err
		}
	}
	for _, sign := range []float64{-1, 1} {
		if err := box(maths.V(sign*185, -1.0, 90), maths.V(18, 2, 380), slab); err != nil {
			return err
		}
		// Double-walled insulated roof panels with visible expansion joints.
		point := func(angle, z, radial float64) maths.Vec {
			return maths.V(sign*(65+radial*math.Cos(angle)), 8+radial*math.Sin(angle), z)
		}
		for bay := 0; bay < 12; bay++ {
			z0, z1 := 85+float64(bay)*14.0+.22, 85+float64(bay+1)*14.0-.22
			for sector := 0; sector < 12; sector++ {
				a0, a1 := (float64(sector)+.018)*math.Pi/24, (float64(sector+1)-.018)*math.Pi/24
				vertices := []maths.Vec{}
				for _, r := range []float64{120., 120.65} {
					vertices = append(vertices, point(a0, z0, r), point(a1, z0, r), point(a1, z1, r), point(a0, z1, r))
				}
				faces := [][3]int{{0, 2, 1}, {0, 3, 2}, {4, 5, 6}, {4, 6, 7}, {0, 1, 5}, {0, 5, 4}, {1, 2, 6}, {1, 6, 5}, {2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7}}
				if sign < 0 {
					for i := range faces {
						faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
					}
				}
				mesh, err := shape.NewMesh(vertices, faces, nil, false)
				if err != nil {
					return err
				}
				shade := .38 + .06*math.Sin(float64(bay*7+sector)*2.3)
				mat := &thing.Material{Name: "OWL insulated shell panel", Colour: maths.RGB(shade, shade*1.03, shade*1.08), Gloss: .22}
				s.world.Add(thing.Thing{Shape: mesh, Material: mat})
			}
		}
		// Paired structural arches are tied together by visible diagonal braces.
		for bay := 0; bay <= 12; bay++ {
			z := 85 + float64(bay)*14
			for sector := 0; sector < 12; sector++ {
				a, b := float64(sector)*math.Pi/24, float64(sector+1)*math.Pi/24
				for _, r := range []float64{116., 119.} {
					if err := rod(point(a, z, r), point(b, z, r), .48, "OWL shell arch", maths.Light{}); err != nil {
						return err
					}
				}
				if err := rod(point(a, z, 116), point(b, z, 119), .2, "OWL arch web", maths.Light{}); err != nil {
					return err
				}
				if bay < 12 {
					if err := rod(point(a, z, 116), point(b, z+14, 116), .18, "OWL roof diagonal", maths.Light{}); err != nil {
						return err
					}
					if err := rod(point(a, z, 119), point(a, z+14, 119), .3, "OWL longitudinal purlin", maths.Light{}); err != nil {
						return err
					}
				}
			}
		}
		// Rail bogies have chassis, motor housings and wheel sets at both ends.
		for _, z := range []float64{86, 254} {
			if err := box(maths.V(sign*185, 3, z), maths.V(12, 4, 19), steel); err != nil {
				return err
			}
			for wheel := 0; wheel < 4; wheel++ {
				if err := rod(maths.V(sign*180, 1.8, z-6+float64(wheel)*4), maths.V(sign*190, 1.8, z-6+float64(wheel)*4), 1.7, "OWL drive wheel", maths.Light{}); err != nil {
					return err
				}
			}
			if err := box(maths.V(sign*175, 4, z), maths.V(7, 5, 7), &thing.Material{Name: "OWL transverse drive", Colour: maths.RGB(.45, .23, .055)}); err != nil {
				return err
			}
		}
		// Accessible service galleries follow the shell, with walkable decks,
		// handrails, uprights, and stairs. No floating decorative seam lights.
		for level := 0; level < 3; level++ {
			y := 24 + float64(level)*28
			angle := math.Asin((y - 8) / 116)
			x := sign * (65 + 116*math.Cos(angle))
			if err := box(maths.V(x, y, 169), maths.V(4, .45, 166), steel); err != nil {
				return err
			}
			for _, edge := range []float64{-1.8, 1.8} {
				if err := rod(maths.V(x+edge, y+1.1, 86), maths.V(x+edge, y+1.1, 252), .07, "OWL gallery handrail", maths.Light{}); err != nil {
					return err
				}
				for z := 86.; z <= 252; z += 4 {
					if err := rod(maths.V(x+edge, y, z), maths.V(x+edge, y+1.1, z), .055, "OWL gallery upright", maths.Light{}); err != nil {
						return err
					}
				}
			}
			if err := rod(maths.V(x, y+2, 90), maths.V(x, y+2, 249), .08, "OWL gallery practical lighting", light); err != nil {
				return err
			}
		}
		// Ground service bay, ribbed doors, ventilation and a real people-scale
		// entrance establish the scale of the much larger moving enclosure.
		hall := maths.V(sign*158, 7, 210)
		if err := box(hall, maths.V(38, 14, 58), &thing.Material{Name: "OWL maintenance hall", Colour: maths.RGB(.24, .29, .33)}); err != nil {
			return err
		}
		for door := 0; door < 3; door++ {
			x := hall.X + float64(door-1)*11
			if err := box(maths.V(x, 5, 180.8), maths.V(8, 10, .35), steel); err != nil {
				return err
			}
			for slat := 0; slat < 20; slat++ {
				if err := rod(maths.V(x-3.9, .3+float64(slat)*.5, 180.5), maths.V(x+3.9, .3+float64(slat)*.5, 180.5), .035, "OWL maintenance door slat", maths.Light{}); err != nil {
					return err
				}
			}
		}
		if err := box(maths.V(hall.X, 11.5, 180.5), maths.V(32, 1.8, .3), &thing.Material{Name: "OWL control glazing", Colour: maths.RGB(.025, .07, .09), Glow: maths.RGB(.025, .045, .055), Gloss: .3}); err != nil {
			return err
		}
		for i := 0; i < 24; i++ {
			z := 185 + float64(i)*2
			if err := box(maths.V(hall.X+sign*19.3, 10, z), maths.V(.7, 3, .35), steel); err != nil {
				return err
			}
		}
	}
	// An overhead servicing bridge lives at the parked rear of the roof.
	for _, y := range []float64{103, 108} {
		if err := rod(maths.V(-65, y, 244), maths.V(65, y, 244), .55, "OWL maintenance bridge chord", maths.Light{}); err != nil {
			return err
		}
	}
	for i := 0; i < 20; i++ {
		x := -65 + float64(i)*6.5
		if err := rod(maths.V(x, 103, 244), maths.V(x+6.5, 108, 244), .22, "OWL maintenance bridge web", maths.Light{}); err != nil {
			return err
		}
	}
	if err := box(maths.V(15, 102, 244), maths.V(8, 5, 9), &thing.Material{Name: "OWL bridge crane trolley", Colour: maths.RGB(.55, .27, .055)}); err != nil {
		return err
	}
	return rod(maths.V(15, 99, 244), maths.V(15, 70, 244), .08, "OWL crane hoist", maths.Light{})
}

// housingBox adds one box at a position as a still mesh.
func (s *heardStage) housingBox(at, size maths.Vec, material *thing.Material) error {
	x, y, z := size.X/2, size.Y/2, size.Z/2
	v := []maths.Vec{maths.V(-x, -y, -z), maths.V(x, -y, -z), maths.V(x, y, -z), maths.V(-x, y, -z), maths.V(-x, -y, z), maths.V(x, -y, z), maths.V(x, y, z), maths.V(-x, y, z)}
	f := [][3]int{{0, 3, 2}, {0, 2, 1}, {4, 5, 6}, {4, 6, 7}, {0, 4, 7}, {0, 7, 3}, {1, 2, 6}, {1, 6, 5}, {3, 7, 6}, {3, 6, 2}, {0, 1, 5}, {0, 5, 4}}
	mesh, err := shape.NewMesh(v, f, nil, false)
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: mesh, Material: material, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
	return nil
}
