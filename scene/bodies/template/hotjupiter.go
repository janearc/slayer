package template

import (
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// HotJupiter is a giant too near its star: jupiter's template with its
// day an orbit of eighteen hours, tidally locked, and its air streaming
// away from the star in a tail thirty radii long, bent back along its
// orbit, the gas's own cream.
//
// The shot aims it.
func HotJupiter() bodies.Body {
	b := Jupiter()
	b.Name = "a hot jupiter"
	b.Moons, b.Aurora = nil, nil
	b.Day = hours(18)
	b.Tails = []bodies.Tail{{Length: 30 * b.Radius, Base: 2 * b.Radius, Spread: 0.06,
		Curve: 0.25, Dense: 0.4 / b.Radius,
		Colour: swatches.Light(swatches.Cream),
		Glow:   swatches.Light(swatches.TailGlow), Wisp: 0.6, Seed: 21}}
	return b
}
