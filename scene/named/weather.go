package named

import (
	"fmt"
	"image"
	"math"
	"time"

	"github.com/janearc/slayer/weather"
)

// The weather: a planet's air on a turning sphere, stirred by storms
// from still air, drawn as three maps a moment, latitude and longitude
// straight across. Package weather carries it; this is its parameters,
// its time and its pictures, as cmd/weather has them.
//
// Time. A job's times stop at a day, and the weather's run for weeks, so
// a second of scene time is pace of the weather's: an hour by default.
// The weather command asks for a pace of a second, so its moments are
// the weather's own seconds.

// the weather's defaults, cmd/weather's flags: picked with the solver by
// its author, and kept as the command has had them
const (
	weatherAcross   = 512     // cells round each circle
	weatherRows     = 256     // rows of latitude
	weatherLimit    = 80.0    // the walls, degrees from the equator
	weatherRadius   = 6e7     // the planet's radius, metres
	weatherDay      = 18000   // seconds the planet turns in
	weatherDrag     = 1e6     // friction's time, seconds
	weatherRelax    = 2e6     // the bands' time, seconds
	weatherRate     = 3.8e-17 // storms a square metre a second
	weatherWidth    = 1.3e6   // a storm's width, metres
	weatherStrength = 1.5e-5  // a storm's spin, radians a second
	weatherBands    = 15.0    // a stripe's width, degrees
	weatherSeed     = 7       // the storms' seed
)

// weatherPace is a second of scene time as the weather's time: an hour,
// picked by the author of this file, 2026-11-12, so a job's day of scene
// time holds ten years of weather and its default run a quarter hour.
const weatherPace = time.Hour

// cooking is how long air takes to cook, for the age picture, in
// seconds: melancholia's, as cmd/weather has it.
const cooking = 1e6

// rightAngle is defined: degrees from the equator to a pole.
const rightAngle = 90

// the weather's pictures, in the order it makes them, and the one a
// film shows by default
const (
	spunMap  = "spun"
	bandsMap = "bands"
	agedMap  = "aged"
)

// clockName is the format of the clock in a map's name, as cmd/weather
// has always named them: whole seconds, eight wide.
const clockName = "%08.0f"

// weatherScene is the weather drawn as maps.
func weatherScene() Maps {
	return Maps{name: "weather",
		about: "a planet's air stirred by storms from still air, " +
			"drawn as maps of its spin, its bands and its air's age",
		renderers:  []Renderer{GPU, CPU},
		pictures:   []string{spunMap, bandsMap, agedMap},
		parameters: weatherParameters(),
		build:      buildWeather,
		cache:      weather.CacheDir}
}

// weatherParameters are what the weather takes, as cmd/weather's flags
// were, with the bounds the solver holds them to.
func weatherParameters() []Parameter {
	return []Parameter{
		whole("across", weatherAcross, atLeast(minimumCells),
			"cells round each circle of latitude, a power of two"),
		whole("rows", weatherRows, atLeast(minimumCells),
			"rows of latitude from wall to wall"),
		number("limit", weatherLimit, above(nothing),
			"the walls, degrees north and south, below 90"),
		number("radius", weatherRadius, above(nothing),
			"the planet's radius, metres"),
		number("day", weatherDay, above(nothing),
			"seconds the planet turns in"),
		number("drag", weatherDrag, above(nothing),
			"friction's time, seconds"),
		number("relax", weatherRelax, above(nothing),
			"the bands' time to be drawn back, seconds"),
		number("rate", weatherRate, above(nothing),
			"storms a square metre a second"),
		number("storm-width", weatherWidth, above(nothing),
			"a storm's width, metres"),
		number("strength", weatherStrength, above(nothing),
			"a storm's spin at its middle, radians a second"),
		number("bands", weatherBands, above(nothing),
			"a stripe's width on the bands map, degrees"),
		whole("seed", weatherSeed, atLeast(nothing), "the storms' seed"),
		oneOf("show", bandsMap, []string{spunMap, bandsMap, agedMap},
			"which map a film shows"),
		span("pace", weatherPace,
			"how much of the weather's time a second of scene time is"),
	}
}

// minimumCells is the fewest cells round a circle and rows the grid
// holds: defined by weather.NewGrid.
const minimumCells = 4

// weatherMaps is the weather built: its air, how to make it again from
// still air, and what was asked of it.
type weatherMaps struct {
	sky     *weather.Weather
	fresh   func() (*weather.Weather, error)
	held    values
	onGPU   bool
	ask     MapsAsk
	reached float64
}

// buildWeather is the weather from still air as asked, on the renderer
// said, or on the gpu where there is one.
func buildWeather(held values, ask MapsAsk) (Mapper, error) {
	if err := weatherRefusals(held); err != nil {
		return nil, err
	}
	fresh := func() (*weather.Weather, error) { return stillAir(held) }
	sky, err := fresh()
	if err != nil {
		return nil, err
	}
	onGPU := ask.renderer == GPU
	if !ask.said {
		onGPU = weather.HasGPU()
		if _, err := fmt.Fprintln(ask.out, carriedOn(onGPU)); err != nil {
			return nil, err
		}
	}
	return &weatherMaps{sky: sky, fresh: fresh, held: held, onGPU: onGPU,
		ask: ask}, nil
}

// carriedOn says which renderer carries the weather when none was said.
func carriedOn(onGPU bool) string {
	if onGPU {
		return "weather: on the gpu, in float32"
	}
	return "weather: on the cpu, in float64; no metal device here"
}

// weatherRefusals refuses by name what the bounds cannot say: cells
// round a circle that are not a power of two, walls at a pole or past
// it, and a pace of nothing.
func weatherRefusals(held values) error {
	across := held.whole("across")
	if across&(across-single) != nothing {
		return fmt.Errorf("weather's across is a power of two, not %d: %w",
			across, ErrBounds)
	}
	if limit := held.number("limit"); limit >= rightAngle {
		return fmt.Errorf("weather's limit is below %d degrees, not %g: %w",
			rightAngle, limit, ErrBounds)
	}
	if pace := held.span("pace"); pace <= nothing {
		return fmt.Errorf("weather's pace is longer than nothing, not "+
			"%s: %w", pace, ErrBounds)
	}
	return nil
}

// stillAir is the weather as asked, before it has run.
func stillAir(held values) (*weather.Weather, error) {
	grid, err := weather.NewGrid(held.whole("across"), held.whole("rows"),
		held.number("limit")*degree, held.number("radius"))
	if err != nil {
		return nil, err
	}
	stirring, err := weather.NewStirring(held.number("rate"),
		held.number("storm-width"), held.number("strength"))
	if err != nil {
		return nil, err
	}
	return weather.New(grid, held.number("day"), held.number("drag"),
		held.number("relax"), stirring, uint64(held.whole("seed")))
}

// degree is defined: a degree in radians.
const degree = math.Pi / halfTurn

// Shows is the map a film shows.
func (made *weatherMaps) Shows() string { return made.held.choice("show") }

// Renderer is what carries the weather.
func (made *weatherMaps) Renderer() Renderer {
	if made.onGPU {
		return GPU
	}
	return CPU
}

// At is the weather's three maps at a moment of scene time: the weather
// carried to the first step at or past it, through the cache where one
// was asked for, and from still air again where the moment is before
// the last one asked.
func (made *weatherMaps) At(moment time.Duration) ([]Map, error) {
	seconds := moment.Seconds() * made.held.span("pace").Seconds()
	if seconds < made.reached {
		sky, err := made.fresh()
		if err != nil {
			return nil, err
		}
		made.sky = sky
	}
	steps, err := made.reach(seconds)
	if err != nil {
		return nil, err
	}
	made.reached = seconds
	if err := made.said(steps); err != nil {
		return nil, err
	}
	return made.maps(), nil
}

// reach carries the weather to so many of its seconds: through the
// cache, or run straight, on the gpu or the cpu.
func (made *weatherMaps) reach(seconds float64) (int, error) {
	switch {
	case made.ask.cache != "":
		every := made.ask.every.Seconds() * made.held.span("pace").Seconds()
		return made.sky.Reach(made.ask.cache, seconds, every, made.onGPU)
	case made.onGPU:
		return made.sky.RunOnGPU(seconds)
	}
	return made.sky.Run(seconds), nil
}

// zonalLines is about how many of the rows' mean winds are said.
const zonalLines = 32

// said says where the weather is, as cmd/weather always has: its clock,
// the steps it took, its fastest wind, the share of its energy its jets
// hold, and its mean east wind from south to north.
func (made *weatherMaps) said(steps int) error {
	sky, out := made.sky, made.ask.out
	if _, err := fmt.Fprintf(out, "%.0f s: %d steps, fastest %.1f m/s, "+
		"jets hold %.2f of the energy (jupiter 0.93)\n", sky.Clock(), steps,
		sky.Fastest(), sky.Zonality()); err != nil {
		return err
	}
	zonal := sky.Zonal()
	for j := 0; j < len(zonal); j += max(single, len(zonal)/zonalLines) {
		if _, err := fmt.Fprintf(out, " %+.0f", zonal[j]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, " (mean east wind, m/s, south to north)")
	return err
}

// maps are the weather's three pictures as it is, named by its clock.
func (made *weatherMaps) maps() []Map {
	sky := made.sky
	clock := fmt.Sprintf(clockName, sky.Clock())
	kinds := []struct {
		kind    string
		picture *image.RGBA
	}{
		{spunMap, sky.Spun()},
		{bandsMap, sky.Bands(made.held.number("bands") * degree)},
		{agedMap, sky.Aged(cooking)},
	}
	pictures := make([]Map, len(kinds))
	for i, each := range kinds {
		pictures[i] = Map{kind: each.kind, name: each.kind + "-" + clock,
			picture: each.picture}
	}
	return pictures
}
