package snakes

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// facePaint adds illustrative paired nostrils and a fine closed mouth to the
// checked profile's head. Along is a tail-to-snout share and angle is the
// cross-section angle from the upper midline; both sides use the same rule.
func facePaint(p Profile, along, angle, footprint float64, colour maths.Light) maths.Light {
	ink := p.Pattern.Mark.Scale(.42)
	eyeAngle := math.Acos(p.Head.EyeHeight)
	noseAt := p.Head.EyeAt + (1-p.Head.EyeAt)*.65
	noseWidth := math.Max(.0015, p.Head.EyeSize*.6)
	nose := math.Hypot((along-noseAt)/noseWidth, (math.Abs(angle)-eyeAngle-.15)/.075)
	colour = blend(colour, ink, 1-smooth(.65, 1, nose))
	start := math.Max(0, p.Head.EyeAt-.04)
	ends := smooth(start, start+.003, along) * (1 - smooth(.998, 1, along))
	feather := math.Min(.06, math.Max(.01, math.Max(0, footprint)/(p.Length*.02)))
	lip := math.Abs(math.Abs(angle) - (math.Pi/2 + .16))
	return blend(colour, ink, ends*(1-smooth(.012, .012+feather, lip)))
}
