package named

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// What the magnetar's picture is made of: the star, the drawings of its
// field and its danger, the things for scale, and zappy beside it.

// the star's quiet glow: pale and bluish, a rendering choice, since the
// star glows in X-rays and not in light an eye sees; picked by the
// author of this scene, 2026-11-12, to read as small and very hot
const glowRed, glowGreen, glowBlue = 2.6, 2.9, 3.6

// the field's lines: where each crosses the equator, in radii of the
// star, how many planes of them round the axis, how thick a line is as
// a share of the star's radius, and their glow. Picked by the author of
// this scene, 2026-11-12, by looking: the lines fill the near view and
// stay apart where they leave the star.
var lineApexes = []float64{1.6, 2.5, 4, 6.5}

const (
	linePlanes                   = 8
	lineThick                    = 0.012
	lineRed, lineGreen, lineBlue = 0.45, 0.65, 1.6
)

// a hoop's thickness as a share of its radius; picked by the author of
// this scene, 2026-11-12, as thin and still a pixel or two in its view
const hoopThick = 0.006

// the hoops' glows, strongest field first, then the light cylinder's
// and the Moon's; picked by the author of this scene, 2026-11-12, as
// hues apart from each other and from the lines, dim enough that the
// tone keeps their hue rather than washing them to cream
var hoopGlows = map[string]maths.Light{
	"quantum": maths.RGB(0.95, 0.25, 0.8),
	"atoms":   maths.RGB(1.0, 0.45, 0.15),
	"lab":     maths.RGB(0.9, 0.9, 0.2),
	"edge":    maths.RGB(0.9, 0.9, 0.9),
	"moon":    maths.RGB(0.35, 0.35, 0.4),
}

// the Earth drawn for scale: its faint glow, so it shows with nothing
// lighting it; picked by the author of this scene, 2026-11-12
const earthRed, earthGreen, earthBlue = 0.1, 0.18, 0.35

// placeStar puts the star at the middle, glowing, and is its glow's
// material, which the flare brightens.
func placeStar(world *thing.World, body star) *thing.Material {
	glowing := &thing.Material{Name: "the magnetar",
		Glow: maths.RGB(glowRed, glowGreen, glowBlue)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: body.radius},
		Material: glowing})
	return glowing
}

// placeLines puts the field's lines in, one mesh, turning with the star.
func placeLines(world *thing.World, body star,
	turning thing.Motion) error {
	var made tube
	for _, apex := range lineApexes {
		for plane := range linePlanes {
			round := 2 * math.Pi * float64(plane) / linePlanes
			points, err := fieldLine(body.radius, apex*body.radius, round)
			if err != nil {
				return err
			}
			err = made.add(points, lineThick*body.radius, false)
			if err != nil {
				return err
			}
		}
	}
	lines, err := made.mesh()
	if err != nil {
		return err
	}
	world.Add(thing.Thing{Shape: lines, Motion: turning,
		Material: &thing.Material{Name: "the field, drawn",
			Glow: maths.RGB(lineRed, lineGreen, lineBlue)}})
	return nil
}

// placeHoops puts a hoop in the magnetic equator where the field falls
// to each named strength, turning with the star, and one in the spin's
// equator at the light cylinder, still. A field stronger than the
// surface's has no hoop, and is left out.
func placeHoops(world *thing.World, body star,
	turning thing.Motion) error {
	for _, each := range dangers() {
		out, err := body.reach(each.field)
		if err != nil {
			continue
		}
		err = placeHoop(world, each.name, out, turning)
		if err != nil {
			return err
		}
	}
	return placeHoop(world, "edge", body.lightCylinder(), nil)
}

// placeHoop is one hoop out, glowing as its name says, moving so.
func placeHoop(world *thing.World, name string, out float64,
	moving thing.Motion) error {
	var made tube
	if err := made.add(circle(out), hoopThick*out, true); err != nil {
		return err
	}
	ring, err := made.mesh()
	if err != nil {
		return err
	}
	world.Add(thing.Thing{Shape: ring, Motion: moving,
		Material: &thing.Material{Name: "the " + name + " hoop, drawn",
			Glow: hoopGlows[name]}})
	return nil
}

// placeScale puts an Earth, its own size, in the spin's equator at the
// strongest magnet's hoop, and the Moon's distance as a hoop.
func placeScale(world *thing.World, body star) error {
	lab, err := body.reach(dangers()[labDanger].field)
	if err != nil {
		return err
	}
	at := maths.Subspace{Origin: maths.V(lab, 0, 0)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: earthKm() /
		diameter}, Motion: thing.Still(at),
		Material: &thing.Material{Name: "an Earth, for scale",
			Glow: maths.RGB(earthRed, earthGreen, earthBlue)}})
	return placeHoop(world, "moon", moonKm(), nil)
}

// labDanger is where the strongest laboratory magnet is in dangers,
// defined by its order there.
const labDanger = 2
