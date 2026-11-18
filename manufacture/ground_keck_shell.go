package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// keckShell is a truncated spherical shell with a parallel-sided front
// opening. The lower shutter uses the complementary front sector.
type keckShell struct {
	radius, thick, centre, bottom, top, halfWidth float64
	front                                         bool
}

// skin joins both skins and each cut edge into one closed shell piece.
func (shape keckShell) skin(fine cut) surface {
	levels := max(6, fine.around/2)
	point := func(level, side int, inside bool) maths.Vec {
		radius := shape.radius
		if inside {
			radius -= shape.thick
		}
		low := math.Asin((shape.bottom - shape.centre) / radius)
		high := math.Pi / 2
		if shape.top < shape.centre+radius {
			high = math.Asin((shape.top - shape.centre) / radius)
		}
		latitude := low + (high-low)*float64(level)/float64(levels)
		if level == levels && high == math.Pi/2 {
			return maths.V(0, shape.centre+radius, 0)
		}
		run := radius * math.Cos(latitude)
		opening := math.Asin(math.Min(1, shape.halfWidth/run))
		start, span := opening, fullTurn-2*opening
		if shape.front {
			start, span = -opening, 2*opening
		}
		angle := start + span*float64(side)/float64(fine.around)
		return maths.V(run*math.Sin(angle), shape.centre+radius*math.Sin(latitude),
			-run*math.Cos(angle))
	}
	var skin surface
	for side := range fine.around {
		for level := range levels {
			skin = quad(skin, point(level, side, false), point(level, side+1, false),
				point(level+1, side+1, false), point(level+1, side, false))
			skin = quad(skin, point(level, side, true), point(level+1, side, true),
				point(level+1, side+1, true), point(level, side+1, true))
		}
		skin = quad(skin, point(0, side, false), point(0, side, true),
			point(0, side+1, true), point(0, side+1, false))
		skin = quad(skin, point(levels, side, false), point(levels, side+1, false),
			point(levels, side+1, true), point(levels, side, true))
	}
	for level := range levels {
		skin = quad(skin, point(level, 0, false), point(level+1, 0, false),
			point(level+1, 0, true), point(level, 0, true))
		skin = quad(skin, point(level, fine.around, false), point(level, fine.around, true),
			point(level+1, fine.around, true), point(level+1, fine.around, false))
	}
	for i := range skin {
		skin[i][1], skin[i][2] = skin[i][2], skin[i][1]
	}

	return skin
}

// keckEnclosure gives the enclosure its full curved body above a low plinth.
func (row groundObservatory) keckEnclosure(assembly *instrumentAssembly) {
	radius := row.diameter * spaceHalf
	centre := row.height - radius
	base := groundValue("keck-plinth-height")
	wall := radius * groundValue("wall-share")
	gap := groundValue("rib-joint-metres")
	half := groundValue("keck-slit-width") * spaceHalf
	foot := math.Sqrt(radius*radius - (base-centre)*(base-centre))
	assembly.add(groundDrum(foot, foot-wall, base-gap), maths.Local(), body)
	defer assembly.turnSince(len(assembly.object.parts))
	shell := keckShell{radius: radius, thick: wall, centre: centre, bottom: base,
		top: row.height, halfWidth: half}
	assembly.add(shell, maths.Local(), body)
	shutter := shell
	shutter.front = true
	shutter.halfWidth -= gap
	shutter.bottom += gap
	shutter.top = groundValue("keck-lower-shutter-top")
	assembly.add(shutter, maths.Local(), body)
	row.keckExterior(assembly)
}
