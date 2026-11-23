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
// rotation. Nought is honest; one is the streaks at their full strength.
// Shear winds them as the orbit does, and Seed makes them a disc's own.
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

// peakRadius is defined by the profile: the radius, in inner radii, where
// its derivative is nought and it is largest, 49 over 36.
const peakRadius = 49.0 / 36

// coolingPower is defined by Shakura and Sunyaev's disc: the temperature
// falls as the radius to minus this power, three quarters.
const coolingPower = 0.75

// edgePower is defined by Shakura and Sunyaev's disc: the power of the
// factor that takes the temperature to nought at the inner edge, a quarter.
const edgePower = 0.25

// peak is the profile's largest value, at one and a third of the inner
// radius and a bit.
var peak = profile(peakRadius)

// the numbers the disc is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// thinDarkBelow is the temperature, in kelvin, below which the disc is
	// dark.
	thinDarkBelow = pickedNumber("thin-dark-below")
	// streakBands is how many streaks lie across the disc from its inner
	// edge to its outer.
	streakBands = pickedNumber("thin-streak-bands")
	// streakRadius is the radius of the circle in the noise that a streak's
	// angle is read round.
	streakRadius = pickedNumber("thin-streak-radius")
	// streakOctaves is how many scales of noise the streaks have.
	streakOctaves = int(pickedNumber("thin-streak-octaves"))
	// streakGain is how far the noise moves a streak's brightness, for
	// each unit of Streaks.
	streakGain = pickedNumber("thin-streak-gain")
	// thermalStep is the table's step in kelvin.
	thermalStep = pickedNumber("thermal-step")
	// thermalTop is the table's last temperature, in kelvin; above it the
	// colour is the top's.
	thermalTop = pickedNumber("thermal-top")
)

// profile is the temperature at a radius in inner radii, as a share of a scale:
// the three quarter power with the inner edge's factor, zero at the edge
// itself.
func profile(radius float64) float64 {
	if radius <= unity {
		return nought
	}
	return math.Pow(unity/radius, coolingPower) *
		math.Pow(unity-math.Sqrt(unity/radius), edgePower)
}

// temperature is the disc's temperature at a radius, in kelvin.
func (disc Thin) temperature(radius float64) float64 {
	if disc.Inner <= nought || radius < disc.Inner || radius > disc.Outer {
		return nought
	}
	return disc.Hot * profile(radius/disc.Inner) / peak
}

// At is the light at the point: the black body's colour at the
// temperature there, as bright as the fourth power says, streaked if
// asked.
func (disc Thin) At(point maths.Vec) maths.Light {
	radius := math.Hypot(point.X, point.Z)
	kelvin := disc.temperature(radius)
	if math.IsNaN(kelvin) {
		return noNumberLight()
	}
	if kelvin < thinDarkBelow {
		return noPaint
	}
	share := kelvin / disc.Hot
	bright := disc.Bright * share * share * share * share
	if disc.Streaks > nought {
		bright *= disc.streaked(point, radius)
	}
	return thermal(kelvin).Scale(bright)
}

// streaked is how much a streak brightens the disc at the point, at a
// radius: streaks along the orbit, wound by the shear as the inner disc
// runs ahead of the outer, continuous round the circle, and carried round
// with time at the orbit's own rate, from +X toward -Z as the orbit goes.
func (disc Thin) streaked(point maths.Vec, radius float64) float64 {
	phi := math.Atan2(point.Z, point.X) +
		disc.Shear*math.Log(radius/disc.Inner) +
		disc.Omega*math.Pow(radius, orbitPower)*disc.Time
	across := streakBands * (radius - disc.Inner) / (disc.Outer - disc.Inner)
	cos, sin := math.Cos(phi), math.Sin(phi)
	noisy := noise.Fractal(disc.Seed, across, streakRadius*cos,
		streakRadius*sin, streakOctaves)
	return math.Max(nought, unity+disc.Streaks*streakGain*noisy)
}

// thermals is the black body's colour by temperature, worked out once
// a step of fifty kelvin and kept, since the spectrum behind it is
// costly and a disc asks for it a thousand times a pixel.
var (
	thermalsOnce sync.Once
	thermals     []maths.Light
)

// fillThermals works out the table of black body colours.
func fillThermals() {
	count := int(thermalTop/thermalStep) + unity
	thermals = make([]maths.Light, count)
	for step := range thermals {
		kelvin := float64(step) * thermalStep
		red, green, blue := srgb.Light(swatch.Planckian(kelvin))
		light := maths.RGB(math.Max(red, nought), math.Max(green, nought),
			math.Max(blue, nought))
		if luminance := light.Luminance(); luminance > nought {
			light = light.Scale(unity / luminance)
		}
		thermals[step] = light
	}
}

// thermal is a black body's colour at a temperature, at unit
// luminance, from the table, blended between its steps.
func thermal(kelvin float64) maths.Light {
	thermalsOnce.Do(fillThermals)
	if kelvin >= thermalTop {
		return thermals[len(thermals)-unity]
	}
	if kelvin <= nought {
		return thermals[firstPlace]
	}
	at := kelvin / thermalStep
	step := int(at)
	fraction := at - float64(step)
	return thermals[step].Scale(unity - fraction).
		Add(thermals[step+unity].Scale(fraction))
}

// Tick is the disc as it is at a time: the same disc, its streaks
// carried round.
func (disc Thin) Tick(elapsed time.Duration) Texture {
	disc.Time = elapsed.Seconds()
	return disc
}
