package sauropods

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// coat is defined in the unposed body's local coordinates and owns its profile.
type coat struct{ profile Profile }

func (c coat) At(point maths.Vec) maths.Light { return c.Within(point, 0) }
func (c coat) Within(point maths.Vec, footprint float64) maths.Light {
	stations := c.profile.Stations
	i := 0
	for i < len(stations)-2 && point.X > stations[i+1].X {
		i++
	}
	a, b := stations[i], stations[i+1]
	t := math.Max(0, math.Min(1, (point.X-a.X)/(b.X-a.X)))
	y, _ := centreY(stations, i, t)
	h := a.Height + (b.Height-a.Height)*t
	// A soft boundary avoids a hard seam on broad torso and leg surfaces.
	width := math.Max(.12*h, math.Abs(footprint))
	mix := math.Max(0, math.Min(1, .5+(point.Y-y+.25*h)/(2*width)))
	base := c.profile.Underside.Scale(1 - mix).Add(c.profile.Skin.Scale(mix))
	return c.face(point, y, h, a.Width+(b.Width-a.Width)*t, footprint, base)
}

// face adds illustrative paired nostrils and a fine closed mouth in local
// coordinates. Both features follow the authored head span and radii.
func (c coat) face(point maths.Vec, centerY, height, width, footprint float64, base maths.Light) maths.Light {
	stations := c.profile.Stations
	start := stations[c.profile.Head.EyeStation].X
	end := stations[len(stations)-1].X
	if point.X < start || point.X > end || height <= 0 || width <= 0 {
		return base
	}
	at := (point.X - start) / (end - start)
	level, side := (point.Y-centerY)/height, math.Abs(point.Z)/width
	blur := math.Max(.01, math.Abs(footprint)/math.Min(height, width))
	line := func(distance, radius float64) float64 {
		return math.Max(0, math.Min(1, (radius+blur-distance)/(2*blur)))
	}
	// The line follows both sides of the head and fades before the snout.
	mouth := line(math.Abs(level+.28), .025) * line(math.Abs(at-.52), .43) * line(math.Abs(side-.95), .35)
	// Two small high-set spots sit ahead of the eyes, one on each side.
	nostril := line(math.Hypot((at-.78)*2.5, (level-.35)*1.1), .055) * line(math.Abs(side-.90), .22)
	return base.Scale(1 - .75*math.Max(mouth, nostril))
}
