package flame

import (
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

// Fire is what a flame's body is asked for: how wide its base and how
// tall, how thick its soot is to green light at its hottest, how hot it
// is in the middle and at its edge, and a seed for its turbulence.
//
// widths and heights are in metres; the absorption is per metre at
// 550 nm, the wavelength soot's tilt is given from.
type Fire struct {
	Radius, Height float64
	Absorption     float64
	Hottest, Edge  measure.Temperature
	Seed           uint64
	// Intermittency is an authored occupancy border; zero keeps 0.4.
	Intermittency float64
	// HotTongues keeps occupied tips hot; it is not a measured mean law.
	HotTongues bool
	// Composition replaces the thermal source function when supplied.
	Composition *composition
}

// A Body is a flame in space: soot that a ray marches through, taking light by
// its density and giving it by the thermal source function by default, and
// puffing as a buoyant flame does.
//
// it is a medium, its own glow as a texture, and thicker to blue than red, as
// the tracer asks of a tinted volume. in its own space its base is a disc at y
// zero and one unit is a metre. it is the same at a time however often it is
// asked, so a replay meets the same flame.
//
// Constructed by NewBody or as part of NewField.
type Body struct {
	fire   Fire
	period time.Duration
	at     time.Duration
	glows  glowing
}

// NewBody is the body of a fire, its glow in a picture's light at an
// exposure: what a watt a square metre a steradian is there, as a
// lamp's exposure is what a watt a steradian is.
func NewBody(fire Fire, exposure float64) (*Body, error) {
	return bodyWithPalette(fire, exposure, nil)
}

type paletteKey struct {
	cold, hot, exposure float64
	composition         *composition
}

// bodyWithPalette shares immutable colour tables within one field.
func bodyWithPalette(fire Fire, exposure float64, palettes map[paletteKey]glowing) (*Body, error) {
	hot, edge := fire.Hottest.InKelvin(), fire.Edge.InKelvin()
	if !sane(fire.Radius) || !sane(fire.Height) ||
		!(fire.Absorption >= 0) || math.IsInf(fire.Absorption, 0) ||
		!sane(hot) || !sane(edge) || edge > hot || !sane(exposure) ||
		!(fire.Intermittency >= 0 && fire.Intermittency < 1) {
		return nil, errFlame
	}
	if fire.Composition != nil && fire.Composition.Reference() != fire.Hottest {
		return nil, errFlame
	}
	period, err := Period(measure.Metres(2 * fire.Radius))
	if err != nil {
		return nil, err
	}
	key := paletteKey{edge, hot, exposure, fire.Composition}
	table, found := palettes[key]
	if !found {
		if fire.Composition != nil {
			table = compositionGlowing(fire.Composition, edge, hot, exposure)
		} else {
			table = newGlowing(edge, hot, exposure)
		}
		if palettes != nil {
			palettes[key] = table
		}
	}

	return &Body{fire: fire, period: period, glows: table}, nil

}

// the flame's shape and motion, all picked by looking, none measured:
// how much of its radius its turbulence moves a point, how big its
// eddies are against its radius, how many octaves they have, how much
// of its height it breathes, and how far up the base its widest is.
//
// six octaves, where it had three, so its eddies run from most of its
// radius down to a few centimetres: with three the flame had no depth,
// and the broken small detail is what makes fire look real.
const (
	sway, eddies, octaves = 0.25, 0.8, 6
	breathes              = 0.2
	taper                 = 0.6
)

// Finest is the smallest eddy the flame draws, in metres: its eddies'
// size halved for each octave past the first, and halved again for its
// soot's rags, which are half the size. a grid that holds the flame
// wants cells no wider than half of this.
func (body *Body) Finest() float64 {
	size := eddies * body.fire.Radius / ragged
	return size / math.Pow(2, octaves-1)
}

// mccaffrey's centreline speeds, metres a second for a kilowatt of heat to
// the fifth: the gas rising as the square root of the height through the
// continuous flame, steady through the tongues, and slowing as the height
// to the minus a third in the plume.
//
// the regions part where the intermittent flame and the plume begin, as
// heights over the heat to the two fifths. all five are his table 1's
// weighted averages, nbsir 79-1910, as slayer's research data holds them.
var (
	continuousSpeed = research.Value("continuous-speed")
	tongueSpeed     = research.Value("tongue-speed")
	plumeSpeed      = research.Value("plume-speed")
	tonguesFrom     = research.Value("tongues-from")
	plumeFrom       = research.Value("plume-from")
)

// travel is how long the fire's gas takes to rise up its axis from the
// base to a height in metres, by mccaffrey's speeds: what carries its
// eddies up, slowly near the base and faster and drawn out above.
//
// the heat's two fifths is the height over 0.2, the tongues' tips; so a
// flame ten metres tall rises about 13 metres a second in its tongues.
func (body *Body) travel(height float64) float64 {
	scale := body.fire.Height / plumeFrom
	fifth := math.Sqrt(scale)
	low, high := tonguesFrom*scale, plumeFrom*scale
	switch {
	case height <= 0:
		return 0
	case height <= low:
		return 2 * math.Sqrt(height) / continuousSpeed
	}
	inTongues := 2*math.Sqrt(low)/continuousSpeed +
		(math.Min(height, high)-low)/(tongueSpeed*fifth)
	if height <= high {
		return inTongues
	}
	return inTongues + 3/(4*plumeSpeed*fifth)*math.Cbrt(1/scale)*
		(math.Pow(height, 4.0/3)-math.Pow(high, 4.0/3))
}

// lifted is how far up the noise a point at a height is read at the
// flame's time: its gas's travel to there, less the time, at the
// tongues' speed, so an eddy rises with the gas, slow at the base and
// stretched as it speeds up.
func (body *Body) lifted(height float64) float64 {
	fifth := math.Sqrt(body.fire.Height / plumeFrom)
	return (body.travel(height) - body.at.Seconds()) * tongueSpeed * fifth
}

// Tick is the flame at a time: its eddies risen and its height
// breathed by noise about the frequency its width puffs at.
func (body *Body) Tick(at time.Duration) medium.Medium {
	ticked := *body
	ticked.at = at
	return &ticked
}

// shine is the flame's glow as the texture a volume's material shines
// with: the same flame, read for its light.
type shine struct{ body *Body }

// Shine is the flame's glow, for its volume's material.
func (body *Body) Shine() texture.Timed { return shine{body} }

// At reads the thermal or prescribed spectral source at the point's heat.
func (glow shine) At(p maths.Vec) maths.Light {
	return glow.body.glows.at(glow.body.Heat(p))
}

// Tick is the glow at a time, the flame's own.
func (glow shine) Tick(at time.Duration) texture.Texture {
	return shine{glow.body.Tick(at).(*Body)}
}

// puffs is how many times the flame has puffed by its time.
func (body *Body) puffs() float64 {
	return body.at.Seconds() / body.period.Seconds()
}

// breath is how much taller or shorter the flame is at its time, as a
// share of its height: noise read at its puffs, so it rises and falls
// about once a period.
func (body *Body) breath() float64 {
	const aside = 0.5
	return breathes * noise.Gradient(body.fire.Seed, body.puffs(), aside,
		aside)
}

// inside is how far inside the flame a point is, from one on its axis at its
// base to zero at its edge, after its eddies have moved it, and how far up it
// is as a share of its height.
func (body *Body) inside(p maths.Vec) (deep, up float64) {
	fire := body.fire
	size := eddies * fire.Radius
	risen := body.lifted(p.Y)
	seed := fire.Seed + 1
	moved := func(axis uint64, x, _, z float64) float64 {
		return noise.Fractal(seed+axis, x/size, risen/size, z/size,
			octaves)
	}
	push := sway * fire.Radius
	x := p.X + push*moved(0, p.X, p.Y, p.Z)
	z := p.Z + push*moved(1, p.X, p.Y, p.Z)
	up = p.Y / (fire.Height * (1 + body.breath()))
	if !(up > 0) || up >= 1 {
		return 0, up
	}
	widest := fire.Radius * math.Pow(1-up, taper)
	across := math.Hypot(x, z) / widest
	return math.Max(0, 1-across*across), up
}

// continuous is how much of the flame's height its continuous flame
// fills, below its tongues: mccaffrey's regions (nbsir 79-1910, table
// 1) start the tongues at 0.08 and the plume at 0.2 of the heat in kw to
// the two fifths. both scale with the heat alike, so the continuous
// flame is 0.4 of the tongues' tips whatever the fire, and a flame's
// height is its tongues' tips.
const continuous = 0.08 / 0.2

// the tongues: how much taller than wide their eddies are, so they rise
// as sheets and not balls, and how soft their edges are, as a share of
// the noise's swing. both picked by looking; soft was 0.15, a glow, and
// is a third of that, so a tongue's edge is a sheet's.
const (
	tall, soft = 3.0, 0.05
)

// the soot's rags: patches cooler than the flame round them, still thick
// with soot, so they glow dull red and stand dark against the hotter
// flame behind.
//
// how much smaller than the eddies they are, how high their noise must
// stand to make a rag, how much of the way to the edge's heat they cool,
// and how sharp their edges are. all picked by looking.
const (
	ragged, ragBar, ragCools, ragEdge = 2.0, 0.4, 0.6, 0.08
)

// Density is the soot's absorption at a point, per metre at 550 nm:
// the fire's in its middle, thinning to nothing at its edge, all of it
// through the continuous flame, and in its tongues above, there or not
// by turbulence, fewer the higher, none at the tip.
func (body *Body) Density(p maths.Vec) float64 {
	deep, up := body.inside(p)
	if deep == 0 {
		return 0
	}
	return body.fire.Absorption * deep * body.tongued(p, up)
}

// tongued is how much of the flame is there at a point a share of its
// height up: all of it in the continuous flame.
//
// above, it is there where its rising turbulence clears a bar that climbs
// from the noise's foot to its top across the tongues, so they lick up in
// sheets and thin to nothing at the tip.
func (body *Body) tongued(p maths.Vec, up float64) float64 {
	border := body.tonguesStart()
	if up <= border {
		return 1
	}
	fire := body.fire
	size := eddies * fire.Radius
	risen := noise.Fractal(fire.Seed+3, p.X/size,
		body.lifted(p.Y)/(tall*size), p.Z/size, octaves)
	bar := -0.65 + 1.3*(up-border)/(1-border)
	return math.Max(0, math.Min(1, (risen-bar)/soft))
}

// Heat is the soot's temperature at a point: along its axis, the fire's
// hottest through the continuous flame, then falling as the height
// grows, as mccaffrey found the tongues' heat fall, to its edge's plus
// 0.4 of the way to its hottest at the tip.
//
// across, it falls from the axis's to the edge's at its edge.
//
// mccaffrey's heat is over the air's; the edge's stands in for the air
// here, since soot below it gives little light to see.
//
// in the tongues, where the soot's rags are, it is cooler again, part of
// the way to the edge's. the continuous flame keeps no rags: mccaffrey
// measured its heat steady, and rags there would pull it down.
func (body *Body) Heat(p maths.Vec) measure.Temperature {
	deep, up := body.inside(p)
	hot, edge := body.fire.Hottest.InKelvin(), body.fire.Edge.InKelvin()
	heat := edge + (hot-edge)*math.Sqrt(deep)
	if up > continuous || (body.fire.HotTongues && up > body.tonguesStart()) {
		axis := hot
		if !body.fire.HotTongues {
			axis = edge + (hot-edge)*continuous/up
		}
		heat = edge + (axis-edge)*math.Sqrt(deep)
		heat -= (heat - edge) * ragCools * body.ragged(p)
	}
	return measure.Kelvin(heat)
}

// ragged is how much of a soot rag a point is in, from zero to one: a finer
// noise risen with the gas, there where it clears the rags' bar, with a sharp
// edge.
func (body *Body) ragged(p maths.Vec) float64 {
	fire := body.fire
	size := eddies * fire.Radius / ragged
	rag := noise.Fractal(fire.Seed+5, p.X/size,
		body.lifted(p.Y)/(tall*size), p.Z/size, octaves)
	return math.Max(0, math.Min(1, (rag-ragBar)/ragEdge))
}

// Bounds is the box the flame and its eddies keep inside.
func (body *Body) Bounds() maths.Box {
	wide := body.fire.Radius * (1 + 2*sway)
	tall := body.fire.Height * (1 + breathes)
	return maths.Box{Min: maths.V(-wide, 0, -wide),
		Max: maths.V(wide, tall, wide)}
}

// eye is the three wavelengths the tracer reads light at, red, green
// and blue in nanometres: trace's gas.go names the same three.
var eye = [3]float64{610, 550, 465}

// Tint is how many times its density at 550 nm each of the eye's three
// wavelengths meets: soot's tilt.
func (body *Body) Tint() maths.Light { return Tilt() }

// Tilt is how many times its absorption at 550 nm soot meets at each of
// the eye's three wavelengths: one over the wavelength to Visible. the
// flame's soot and the smoke's are the same soot.
func Tilt() maths.Light {
	tilt := func(nm float64) float64 { return math.Pow(eye[1]/nm, Visible) }
	return maths.RGB(tilt(eye[0]), tilt(eye[1]), tilt(eye[2]))
}

// glowing stores source radiance in picture units, worked out once
// a few kelvin apart across the temperatures a flame has, since a march
// asks it at every step.
type glowing struct {
	coolest, apart float64
	glows          []maths.Light
}

// glowSteps is how many temperatures the glow is worked out at.
const glowSteps = 256

// newGlowing is the glow from the coolest temperature to the hottest, at an
// exposure.
//
// a red-hot body is redder than srgb's red, so a channel can come out less than
// zero: it is left at zero, the nearest srgb can show.
func newGlowing(coolest, hottest, exposure float64) glowing {
	apart := (hottest - coolest) / (glowSteps - 1)
	table := glowing{coolest: coolest, apart: apart}
	for i := range glowSteps {
		heat := measure.Kelvin(coolest + float64(i)*apart)
		glowing := Glowing(heat).Scale(exposure)
		red, green, blue := srgb.Light(observer.Swatch(glowing))
		table.glows = append(table.glows, maths.RGB(math.Max(0, red),
			math.Max(0, green), math.Max(0, blue)))
	}
	return table
}

// at is the glow at a temperature, between the two worked out nearest.
func (table glowing) at(heat measure.Temperature) maths.Light {
	if !(table.apart > 0) {
		return table.glows[0]
	}
	place := (heat.InKelvin() - table.coolest) / table.apart
	place = math.Max(0, math.Min(place, glowSteps-1))
	low := math.Min(math.Floor(place), glowSteps-2)
	share := place - low
	below, above := table.glows[int(low)], table.glows[int(low)+1]
	return below.Scale(1 - share).Add(above.Scale(share))
}

// tonguesStart is an authored occupancy border when supplied. Zero keeps the
// original mean-region ratio; it does not claim a new measured correlation.
func (body *Body) tonguesStart() float64 {
	if body.fire.Intermittency > 0 {
		return body.fire.Intermittency
	}
	return continuous
}

// compositionGlowing uses the same observer and RGB clipping as the thermal
// table. The GPU receives this table, so it has no separate chemistry model.
func compositionGlowing(profile *composition, cold, hot, exposure float64) glowing {
	apart := (hot - cold) / (glowSteps - 1)
	table := glowing{coolest: cold, apart: apart}
	for i := range glowSteps {
		spectrum := profile.Spectrum(measure.Kelvin(cold + float64(i)*apart)).Scale(exposure)
		red, green, blue := srgb.Light(observer.Swatch(spectrum))
		table.glows = append(table.glows, maths.RGB(math.Max(0, red), math.Max(0, green), math.Max(0, blue)))
	}
	return table
}
