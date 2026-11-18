//go:build netcdf

package main

// A film of the earth turning: frames drawn offscreen into a directory in
// one process, the camera going round in longitude while its latitude
// swings from north to south, and the weather, if there is any, running
// once through the steps that hold data.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/internal/observability"
)

const (
	// fullTurn is the longitude's period in degrees.
	fullTurn = 360.0
	// directoryMode is the mode of the directory the frames go in: its
	// owner writes, others read.
	directoryMode = 0o755
)

// frameName is the name of a frame from its number, which counts from 1.
func frameName(number int) string {
	return fmt.Sprintf("frame-%04d.png", number)
}

// filmPlace is the place under the camera a share of the way through the
// film, from none to all. The longitude goes round the film's turns of a
// whole circle westward from where it started, as the earth is seen to
// turn from outside; the latitude runs evenly from the tilt north to the
// tilt south.
func filmPlace(share, start float64, settings options) (float64, float64) {
	lon := start - fullTurn*settings.turns*share
	lat := settings.tilt * (1 - 2*share)
	return lon, lat
}

// blendCover fills into with the cover at a position along the steps,
// which counts from 0 at the first to one less than their number at the
// last, and is clamped to that. Between two neighbouring steps a
// triangle's cover is a straight crossfade. It is a crossfade between
// fields a day apart and not a statement of where the cloud went.
func blendCover(steps []weatherStep, position float64, into []float64) {
	last := len(steps) - 1
	position = math.Max(0, math.Min(position, float64(last)))
	before := int(math.Floor(position))
	after := min(before+1, last)
	share := position - float64(before)
	for triangle := range into {
		from, to := steps[before].cover[triangle], steps[after].cover[triangle]
		switch {
		case share <= 0:
			into[triangle] = from
		case share >= 1:
			into[triangle] = to
		default:
			into[triangle] = from + (to-from)*share
		}
	}
}

// layFrame shows the weather at a share of the way through the film: the
// steps run once, evenly, and the earth is painted again with their
// crossfade.
func layFrame(share float64, shown []float64) {
	steps := view.earth.weather
	blendCover(steps, share*float64(len(steps)-1), shown)
	view.earth.cover = shown
	recolourEarth()
}

// drawFilm writes the film's frames into its directory, one process, by
// the writer it is given. It has the earth's view and no hud, and the
// observer's place is not marked.
func drawFilm(settings options, write func(path string)) error {
	if err := os.MkdirAll(settings.into, directoryMode); err != nil {
		return err
	}
	view.mode = earthView
	start := view.lon
	shown := make([]float64, len(view.earth.cover))
	for frame := range settings.frames {
		measured := observability.Enabled()
		var started time.Time
		if measured {
			started = time.Now()
		}
		share := float64(frame) / float64(settings.frames-1)
		view.lon, view.lat = filmPlace(share, start, settings)
		if len(view.earth.weather) > 0 {
			layFrame(share, shown)
		}
		write(filepath.Join(settings.into, frameName(frame+1)))
		if measured {
			observability.Observe("grissom_film_frame_seconds", time.Since(started).Seconds())
			observability.Add("grissom_film_frames_total", 1)
		}
	}
	return nil
}
