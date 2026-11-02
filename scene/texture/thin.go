package texture

import (
	"math"
	"sync"
	"time"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Thin is a thin disc's light by its temperature, round the y axis:
// shakura and sunyaev's (1973, A&A 24, 337), hottest a little out from
// the inner edge and cooling outward as the three quarter power of the
// radius.
//
// Each temperature's colour is a black body's through libtheme, so no
// colour is typed here. its brightness is the fourth power of its
// temperature, which is the whole light's; the visible's falls far less,
// at 550 nm a hundredth from 25000 K to 5000 K, not 1/625.
//
// Hot is the hottest temperature in kelvin and Bright the light there.
// below 500 kelvin it is dark, a cut chosen, not measured.
//
// Streaks is a dressing, not physics: a real disc is turbulent but
// smooth at this scale, and a film wants streaks that follow the
// rotation. Nought is honest; one is miami's. Shear winds them as the
// orbit does, and Seed makes them a disc's own.
type Thin struct {
	Inner, Outer float64
	Hot, Bright  float64
	Streaks      float64
	Shear        float64
	Seed         uint64
	// Omega is how fast the disc goes round at radius one, in radians a
	// second, falling with radius to the three halves as orbits do; Time is
	// when the disc is seen, in seconds. The streaks wind by the two: zero
	// for a disc that stands still.
	Omega, Time float64
}

// peak is the profile's largest value, at one and a third of the inner
// radius and a bit.
var peak = profile(49.0 / 36)

// profile is the temperature at a radius in inner radii, as a share of a scale:
// the three quarter power with the inner edge's factor, zero at the edge
// itself.
func profile(r float64) float64 {
	if r <= 1 {
		return 0
	}
	return math.Pow(1/r, 0.75) * math.Pow(1-math.Sqrt(1/r), 0.25)
}

// Temperature is the disc's temperature at a radius, in kelvin.
func (t Thin) Temperature(r float64) float64 {
	if t.Inner <= 0 || r < t.Inner || r > t.Outer {
		return 0
	}
	return t.Hot * profile(r/t.Inner) / peak
}

// At is the light at p: the black body's colour at the temperature
// there, as bright as the fourth power says, streaked if asked.
func (t Thin) At(p maths.Vec) maths.Light {
	r := math.Hypot(p.X, p.Z)
	k := t.Temperature(r)
	if math.IsNaN(k) {
		// a radius that is no number, the same on every machine
		return maths.Grey(math.NaN())
	}
	if k < 500 {
		return maths.Light{}
	}
	l := thermal(k)
	share := k / t.Hot
	bright := t.Bright * share * share * share * share
	if t.Streaks > 0 {
		// streaks along the orbit, wound by the shear as the inner
		// disc runs ahead of the outer, continuous round the circle,
		// and carried round with time at the orbit's own rate, from
		// +X toward -Z as the orbit goes
		phi := math.Atan2(p.Z, p.X) + t.Shear*math.Log(r/t.Inner) +
			t.Omega*math.Pow(r, -1.5)*t.Time
		across := 6 * (r - t.Inner) / (t.Outer - t.Inner)
		cos, sin := math.Cos(phi), math.Sin(phi)
		n := noise.Fractal(t.Seed, across, 2*cos, 2*sin, 3)
		bright *= math.Max(0, 1+t.Streaks*2*n)
	}
	return l.Scale(bright)
}

// thermals is the black body's colour by temperature, worked out once
// a step of fifty kelvin and kept, since the spectrum behind it is
// costly and a disc asks for it a thousand times a pixel.
var (
	thermalsOnce sync.Once
	thermals     []maths.Light
)

// thermalStep is the table's step in kelvin, and thermalTop its last
// temperature; above it the colour is the top's.
const (
	thermalStep = 50.0
	thermalTop  = 60000.0
)

// thermal is a black body's colour at a temperature, at unit
// luminance, from the table, blended between its steps.
func thermal(k float64) maths.Light {
	thermalsOnce.Do(func() {
		n := int(thermalTop/thermalStep) + 1
		thermals = make([]maths.Light, n)
		for i := range thermals {
			kelvin := float64(i) * thermalStep
			cr, cg, cb := srgb.Light(swatch.Planckian(kelvin))
			l := maths.RGB(math.Max(cr, 0), math.Max(cg, 0),
				math.Max(cb, 0))
			if y := l.Luminance(); y > 0 {
				l = l.Scale(1 / y)
			}
			thermals[i] = l
		}
	})
	if k >= thermalTop {
		return thermals[len(thermals)-1]
	}
	if k <= 0 {
		return thermals[0]
	}
	at := k / thermalStep
	i := int(at)
	f := at - float64(i)
	return thermals[i].Scale(1 - f).Add(thermals[i+1].Scale(f))
}

// Tick is the disc as it is at a time: the same disc, its streaks
// carried round.
func (t Thin) Tick(at time.Duration) Texture {
	t.Time = at.Seconds()
	return t
}
