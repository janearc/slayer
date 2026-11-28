package world

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// Relief is a fixed, body-space reading of the same terrain height that
// Scrooged uses for paint. Its scale in metres and angular sampling width
// are authored choices. It is a field, not yet a displaced shape.
type Relief struct {
	world     *World
	tree      *tree
	radius    float64
	amplitude float64
	angle     float64
	ridges    float64
}

// WithRidges returns a field with an authored ridge height in metres.
// Zero retains the original field. The contribution uses the fixed-angle
// Scrooged crest and rock mask and fades in on land just above sea level;
// it does not generate an erosion model or create a separate water surface.
func (relief *Relief) WithRidges(metres float64) (*Relief, error) {
	if relief == nil || !(metres >= 0) || math.IsInf(metres, 0) || metres >= relief.radius {
		return nil, fmt.Errorf("world: ridge height needs a field and finite metres from zero to below its radius")
	}
	copy := *relief
	copy.ridges = metres
	return &copy, nil
}

// ridgeShoreFade is the dimensionless base-height interval above sea
// level over which the authored ridge contribution fades in. Picked for
// the control, not a physical shoreline width.
const ridgeShoreFade = 0.05

// NewRelief fixes a world's height field independently of the camera.
// radius and amplitude are metres; angle is the angular width in radians
// used to filter its tree detail. The amplitude is the metres per unit of
// the existing dimensionless height, not a promised maximum mountain.
func NewRelief(w *World, radius, amplitude, angle float64) (*Relief, error) {
	if w == nil || !(radius > 0) || math.IsInf(radius, 0) ||
		!(amplitude >= 0) || math.IsInf(amplitude, 0) ||
		!(angle > 0 && angle <= math.Pi) || amplitude >= radius {
		return nil, fmt.Errorf("world: relief needs a world, positive finite " +
			"radius, amplitude from zero to below the radius, and an angle " +
			"above zero and at most pi")
	}
	return &Relief{world: w, tree: w.tree(), radius: radius,
		amplitude: amplitude, angle: angle}, nil
}

// Height is signed metres above the model's sea level at direction p.
// It reads the same warped coarse ground and tree height as Scrooged, at
// the fixed filter width given to NewRelief. It does not clamp the sea bed.
func (relief *Relief) Height(p maths.Vec) (float64, error) {
	if relief == nil || !directionOK(p) {
		return 0, fmt.Errorf("world: relief needs a finite nonzero direction")
	}
	p = p.Unit()
	base := relief.world.lifted(p)
	detail := relief.tree.gather(p, relief.angle).detail
	height := base + detail[heightAttribute]
	if relief.ridges == 0 {
		return relief.amplitude * height, nil
	}
	plain := base + detail[reliefAttribute]
	crest := crestOf(detail[ridgeAttribute], relief.tree.ridgeSpread(relief.angle))
	rock := maths.Smooth(crestRock-crestSoft, crestRock+crestSoft,
		crest+(plain-rockHeight)/crestPlain)
	land := maths.Smooth(0, ridgeShoreFade, height)
	return relief.amplitude*height + relief.ridges*land*rock, nil
}

// Normal estimates the outward normal to radius plus the relief height,
// using centred differences one fixed sampling width apart. It is a
// sampling approximation; no ray intersection is provided by this field.
func (relief *Relief) Normal(p maths.Vec) (maths.Vec, error) {
	height, err := relief.Height(p)
	if err != nil {
		return maths.Vec{}, err
	}
	p = p.Unit()
	a, b := tangents(p)
	step := relief.angle
	if step > 0.01 {
		step = 0.01
	}
	// Rotate on the sphere, rather than sampling a tangent plane's length.
	cos, sin := math.Cos(step), math.Sin(step)
	ahead, _ := relief.Height(p.Scale(cos).Add(a.Scale(sin)))
	aback, _ := relief.Height(p.Scale(cos).Sub(a.Scale(sin)))
	bhead, _ := relief.Height(p.Scale(cos).Add(b.Scale(sin)))
	bback, _ := relief.Height(p.Scale(cos).Sub(b.Scale(sin)))
	radius := relief.radius + height
	if !(radius > 0) {
		return maths.Vec{}, fmt.Errorf("world: relief reaches the planet's centre")
	}
	return p.Sub(a.Scale((ahead - aback) / (2 * step * radius))).
		Sub(b.Scale((bhead - bback) / (2 * step * radius))).Unit(), nil
}

func directionOK(p maths.Vec) bool {
	return !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsNaN(p.Z) &&
		!math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0) && !math.IsInf(p.Z, 0) &&
		p.Len() > 0 && !math.IsInf(p.Len(), 0)
}
