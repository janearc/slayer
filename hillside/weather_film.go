package hillside

// A planet's weather through a film.
//
// Design. A set's weather is reached at its moment when the set is
// staged, and a still shows it there. Where the set's weather block says
// a pace, a film carries the weather on: each frame is the moment plus
// the frame's scene time at that pace, reached by way of the weather's
// own checkpoints, so a film goes from one frame's weather to the next
// and never again from still air.
//
// Where it says a turning, the planet turns eastward that many degrees a
// second of scene time: the flow is read that far west of where it is
// seen. The globe is the one thing redrawn each frame: its body takes
// the frame's flow, and its face is worked and charted again. Unsaid,
// both are nothing, and every frame is the still.

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/weather"
)

// weatherOnFilm is a planet whose weather goes on through a film: the
// set's weather and how it is seen, the body it flows on, the weather as
// last reached, and the moment of it last asked for.
type weatherOnFilm struct {
	written set.Weather
	look    planetLook
	body    planet.Body
	sky     *weather.Weather
	reached float64
}

// filmWeather keeps the planet's globe as a thing of the film's own,
// redrawn each frame, where the set's weather goes on through a film; a
// still, and a set whose weather says no pace and no turning, keep it as
// it was staged.
func (stage *staging) filmWeather(made set.Planet, look planetLook,
	body planet.Body, sky *weather.Weather, placed *thing.Thing) error {
	written, has := made.Weather()
	if !has || sky == nil {
		return nil
	}
	pace, turning := written.Film()
	if pace == nothing && turning == nothing {
		return nil
	}
	if stage.dynamicThings == nil {
		stage.dynamicThings = map[*thing.Thing]bool{}
	}
	stage.dynamicThings[placed] = true
	moment, _ := written.Moment()
	stage.weathers = append(stage.weathers, &weatherOnFilm{
		written: written, look: look, body: body, sky: sky,
		reached: moment})
	return nil
}

// addAt adds the globe as it is at a moment of scene time: its weather
// carried there, the planet turned, its face worked again. The tally
// counted the globe when it was staged.
func (film *weatherOnFilm) addAt(world *thing.World, _ *tally,
	at time.Duration) error {
	pace, turning := film.written.Film()
	start, _ := film.written.Moment()
	moment := start + at.Seconds()*pace
	if err := film.carried(moment); err != nil {
		return err
	}
	turned := turnedFlow{flow: film.sky,
		by: at.Seconds() * turning * maths.Degree}
	ball, err := film.look.globeOf(film.body.Flowing(turned))
	if err != nil {
		return err
	}
	world.Add(ball)
	return nil
}

// carried carries the weather to a moment: on from where it is, or from
// still air again where the moment is before the last one asked for, as
// the weather never runs backward.
func (film *weatherOnFilm) carried(moment float64) error {
	if moment < film.reached {
		sky, err := stillAir(film.written, film.body.Radius())
		if err != nil {
			return err
		}
		film.sky = sky
	}
	if err := reachedBy(film.sky, film.written, moment); err != nil {
		return err
	}
	film.reached = moment
	return nil
}

// turnedFlow is a flow on a planet turned eastward by an angle in
// radians: what is seen at a longitude was that far west of it.
type turnedFlow struct {
	flow planet.Flow
	by   float64
}

// fullTurn is defined: a whole turn, in radians.
const fullTurn = 2 * math.Pi

// At is the flow where the air seen at a latitude and longitude was
// before the planet turned, its longitude kept in a turn.
func (turned turnedFlow) At(latitude, longitude float64) (origin, spin,
	age float64) {
	return turned.flow.At(latitude, math.Mod(math.Mod(longitude-turned.by,
		fullTurn)+fullTurn, fullTurn))
}
