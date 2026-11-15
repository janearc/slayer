package named

import (
	"io"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// The magnetar: a real one, SGR 1806-20, at its true size, to stand
// where zappy stands when the music stops. One unit is a kilometre. The
// measured numbers are read from the research data (magnetar_facts.go);
// what is picked here is how it is drawn, and each says so.

// the views the eye may stand at by default: close on the star and its
// field, back far enough for the inner hoops of danger, far for the
// outer ones and the light cylinder, and beside zappy
var magnetarViews = []string{"near", "shells", "far", "beside", "world"}

// view is where an eye stands by default: how far out from what it looks
// at, in km, and how far above the spin's equator, in degrees.
type view struct {
	distance, elevation float64
}

// the views' eyes, picked by the author of this scene, 2026-11-12, by
// looking: near sees the field's outer lines across the frame; shells
// the atoms' hoop across it; far the light cylinder across it
var magnetarEyes = map[string]view{
	"near":   {distance: 220, elevation: 12},
	"shells": {distance: 3000, elevation: 35},
	"far":    {distance: 1.15e6, elevation: 35},
}

// the magnetic axis's tilt off the spin axis, in degrees: not measured
// for this star, picked by the author of this scene, 2026-11-12, so the
// lines sweep where a camera can see them
const magnetarTilt = 30

// the most a tilt may be, in degrees: the axis lying in the equator,
// defined
const mostTilt = 90

// the picture: the lens's width in degrees and the moment a still is
// drawn at; the frame and the tone are zappy's.
// Picked by the author of this scene, 2026-11-12.
const (
	magnetarWideDegrees = 40
	magnetarStartsAt    = 0
)

// magnetarRays is the rays a pixel takes along each side by default:
// eight, so the thin lines and hoops stay whole at a full frame and do
// not crawl in a film. Picked by the author of this scene, 2026-11-12,
// by looking at full size.
const magnetarRays = 8

// the dark the stars are on, the far view's, and the seed of the
// stars, the star's own catalogue number
const (
	magnetarSkyRed, magnetarSkyGreen = 0.003, 0.003
	magnetarSkyBlue                  = 0.006
	magnetarSkySeed                  = 1806
)

// magnetarScene is the real magnetar at its true size.
func magnetarScene() Scene {
	return Scene{name: "magnetar",
		about: "a real magnetar, SGR 1806-20, at its true size: its " +
			"field drawn turning, hoops where its field falls to named " +
			"strengths, its 2004 giant flare, and zappy beside it",
		unit:       "a kilometre",
		renderers:  []Renderer{CPU, GPU},
		parameters: magnetarParameters(),
		defaults: Defaults{width: zappyWidth, height: zappyHeight,
			rays: magnetarRays, wide: magnetarWideDegrees * maths.Degree,
			at: magnetarStartsAt, tone: colour.Tone{
				Exposure: zappyExposure, Knee: zappyKnee,
				Top: zappyTop}},
		build: buildMagnetar}
}

// magnetarParameters are what the magnetar takes: the star, what is
// shown, the flare's clock, and where the eye stands.
func magnetarParameters() []Parameter {
	body := realStar()
	return []Parameter{
		number("radius", body.radius, above(nothing),
			"the star's radius, in km; measured as 10 to 13"),
		number("field", body.field/tenToThe14, above(nothing), "the "+
			"surface field at the magnetic equator, in 1e14 gauss"),
		span("period", seconds(body.period),
			"how long the star takes to turn once"),
		number("tilt", magnetarTilt, between(nothing, mostTilt), "how "+
			"far the magnetic axis leans off the spin axis, degrees"),
		toggle("lines", true, "draw the field's lines"),
		toggle("hoops", true, "draw the hoops of danger"),
		toggle("scale", false, "draw an Earth at the strongest "+
			"magnet's hoop and the Moon's distance as a hoop"),
		toggle("flare", true, "run the 2004 giant flare"),
		span("flare-at", flareStarts, "when the flare's spike begins"),
		number("flare-speed", flareSpeed, atLeast(single), "how many "+
			"times faster than true the flare's tail is shown"),
		toggle("zappy", false, "put zappy beside it, its beams off"),
		toggle("world", false, "put an Earth beside it, its aurora "+
			"pushed as smoke 33 pushes it"),
		number("world-at", worldAt, above(nothing), "how far the "+
			"world is from the star, in km; inside the light cylinder"),
		oneOf("view", magnetarViews[0], magnetarViews,
			"where the eye stands"),
		number("distance", nothing, atLeast(nothing), "how far out the "+
			"eye is, in km; nought is the view's own"),
	}
}

// seconds is a time of that many seconds.
func seconds(count float64) time.Duration {
	return time.Duration(count * float64(time.Second))
}

// buildMagnetar is the star at the middle, its lines and hoops turning
// with it, the scale things, the flare on its glow, zappy if asked, and
// the eye where the view puts it.
func buildMagnetar(held values, _ io.Writer) (*Built, error) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(magnetarSkyRed,
		magnetarSkyGreen, magnetarSkyBlue), Seed: magnetarSkySeed}
	body := star{radius: held.number("radius"),
		field:  held.number("field") * tenToThe14,
		period: held.span("period").Seconds()}
	glow := placeStar(world, body)
	turning := thing.Spin{In: maths.Subspace{Orientation: maths.Rotation(
		maths.V(1, 0, 0), held.number("tilt")*maths.Degree)},
		Axis: maths.Up, Period: held.span("period")}
	if err := placeDrawn(world, body, held, turning); err != nil {
		return nil, err
	}
	built := &Built{world: world}
	if err := placeBeside(world, built, held); err != nil {
		return nil, err
	}
	if held.toggle("world") {
		err := placeWorld(world, body, held.number("world-at"))
		if err != nil {
			return nil, err
		}
	}
	if held.toggle("flare") {
		lit, err := newFlare(held.span("flare-at"),
			held.number("flare-speed"))
		if err != nil {
			return nil, err
		}
		quiet := glow.Glow
		built.tick = func(at time.Duration) {
			glow.Glow = quiet.Scale(lit.Brightness(at))
		}
	}
	built.eye = magnetarEye(held)
	return built, nil
}

// placeDrawn puts the drawings into the world: the field's lines and
// the hoops of danger, turning with the star, and the scale things.
func placeDrawn(world *thing.World, body star, held values,
	turning thing.Motion) error {
	if held.toggle("lines") {
		if err := placeLines(world, body, turning); err != nil {
			return err
		}
	}
	if held.toggle("hoops") {
		if err := placeHoops(world, body, turning); err != nil {
			return err
		}
	}
	if held.toggle("scale") {
		return placeScale(world, body)
	}
	return nil
}

// magnetarEye is where the eye stands: the view's distance, or the one
// asked, at its elevation, looking at the middle with the spin axis up;
// beside zappy, its own.
func magnetarEye(held values) maths.Subspace {
	chosen := held.choice("view")
	if chosen == "beside" {
		return besideEye(held.number("distance"))
	}
	if chosen == "world" {
		return worldEye(held.number("world-at"))
	}
	stand := magnetarEyes[chosen]
	away := stand.distance
	if asked := held.number("distance"); asked > nothing {
		away = asked
	}
	up := stand.elevation * maths.Degree
	from := maths.V(0, away*math.Sin(up), away*math.Cos(up))
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0), maths.Up)}
}
