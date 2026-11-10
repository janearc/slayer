package flame

import (
	"time"

	"github.com/janearc/slayer/maths"
)

// Samples describes this flame to another evaluator: its inputs, time, puff
// period and a copy of its temperature-to-light table. The table is uniformly
// spaced from Edge to Hottest; changing the returned slice leaves Body alone.
func (body *Body) Samples() (Fire, time.Duration, time.Duration,
	[]maths.Light) {
	return body.fire, body.at, body.period, append([]maths.Light(nil),
		body.glows.glows...)
}

// Flame identifies the physical field behind the standard emission texture.
func (glow shine) Flame() *Body {
	return glow.body
}
