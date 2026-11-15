package fixture_test

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/trace"
)

// tinted is a medium thicker to some light than other: its density
// times tint, channel by channel, as trace.Tinted asks.
type tinted struct {
	medium.Medium
	tint maths.Light
}

// Tint is how many times its density each channel meets.
func (slab tinted) Tint() maths.Light { return slab.tint }

// a tinted medium is a door of the tracer's
var _ trace.Tinted = tinted{}
