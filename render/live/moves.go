package live

import (
	"fmt"
	"slices"
	"time"
)

// still is whether the job shown is a still.
func (watching *Watcher) still() bool {
	return watching.wanted.Clock().Still()
}

// frames is how many frames the job has, as staged; one before it is.
func (watching *Watcher) frames() int {
	return max(one, len(watching.facts.times))
}

// rate is the film's frames a second, and one for a still.
func (watching *Watcher) rate() int {
	rate, said := watching.wanted.Clock().Rate()
	if !said || rate < one {
		return one
	}
	return rate
}

// step moves a film's frame by count, held to the film's ends. A still
// has one frame, and is told how its moment moves.
func (watching *Watcher) step(count int) Change {
	if watching.still() {
		watching.foot.note("a still has one frame; [ and ] move its " +
			"moment by a second")
		return Unchanged
	}
	return watching.goTo(watching.view.index + count)
}

// goTo shows frame index, held to the film's ends: a redraw if it moved.
func (watching *Watcher) goTo(index int) Change {
	index = min(max(index, none), watching.frames()-one)
	if index == watching.view.index {
		return Unchanged
	}
	watching.view.index = index
	return Redraw
}

// second moves a film's frame by count seconds of frames, or a still's
// moment by count seconds, never before nought. A still restages: the
// session fixes its moment when it opens.
func (watching *Watcher) second(count int) Change {
	if !watching.still() {
		return watching.goTo(watching.view.index +
			count*watching.rate())
	}
	moment := watching.view.moment
	if !watching.view.said {
		moment, _ = watching.base.Clock().At()
	}
	moment = max(moment+time.Duration(count)*time.Second, none)
	watching.view.moment, watching.view.said = moment, true
	return watching.relaid()
}

// play starts or stops a film playing at its rate. A still does not
// play.
func (watching *Watcher) play() Change {
	if watching.still() {
		watching.foot.note("a still does not play; it has one frame")
		return Unchanged
	}
	watching.view.playing = !watching.view.playing
	watching.view.playFrom = watching.view.index
	watching.view.playStart = watching.given.Now()
	return Redraw
}

// playedTo is the frame play has reached by now, round to the start
// past the film's end.
func (watching *Watcher) playedTo() int {
	elapsed := watching.given.Now().Sub(watching.view.playStart)
	passed := int(elapsed.Seconds() * float64(watching.rate()))
	return (watching.view.playFrom + passed) % watching.frames()
}

// nextCamera chooses the scene's camera after the one shown, by name,
// round to the first. A scene with no cameras by name says so.
func (watching *Watcher) nextCamera() Change {
	names := watching.facts.cameras
	if len(names) == none {
		watching.foot.note("this scene has no cameras by name to " +
			"choose from")
		return Unchanged
	}
	shown := watching.view.camera
	if shown == "" {
		if lens, said := watching.base.Camera(); said {
			shown, _ = lens.Name()
		}
	}
	next := (slices.Index(names, shown) + one) % len(names)
	watching.view.camera = names[next]
	watching.foot.note(fmt.Sprintf("camera %s", names[next]))
	return watching.relaid()
}

// expose moves exposure by count thirds of a stop: the kept light is
// developed again.
func (watching *Watcher) expose(count int) Change {
	watching.view.thirds += count
	return Redevelop
}

// fastest is the rung play draws at: the finest of the pane's ladder
// that was last drawn within a frame's time, or none yet, for the
// ladder's first.
func (watching *Watcher) fastest() rung {
	if !watching.view.playing {
		return rung{}
	}
	frame := time.Second / time.Duration(watching.rate())
	chosen := rung{}
	ladder := ladderFor(watching.facts, watching.given.Screen.Screen())
	for _, each := range ladder {
		if took, drawn := watching.timings[each]; drawn && took <= frame {
			chosen = each
		}
	}
	return chosen
}
