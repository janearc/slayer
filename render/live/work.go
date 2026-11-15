package live

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
)

// facts are what the loop keeps of a stage, read by the worker as it
// opened: the loop never asks a stage while a plan holds it.
type facts struct {
	times         []time.Duration
	tone          colour.Tone
	width, height int
	exposure      float64
	cameras       []string
}

// factsOf is a stage's facts.
func factsOf(stage Stage) facts {
	width, height := stage.Shape()
	return facts{times: stage.Times(), tone: stage.Tone(), width: width,
		height: height, exposure: stage.Exposure(),
		cameras: stage.Cameras()}
}

// plan is one run of the worker: the stage it is lent, or none to stage
// the job afresh, the frame to draw and how.
type plan struct {
	stage   Stage
	wanted  job.Job
	restage bool
	index   int
	screen  terminal.Screen
	mode    mode
	// fastest is, while playing, the rung to draw at; the zero rung
	// otherwise
	fastest rung
	playing bool
}

// message is what the worker tells the loop: a stage opened, a rung
// drawn, a refusal, or that it is done, with the stage it was lent.
type message struct {
	staged bool
	facts  facts
	drawn  Drawn
	rung   rung
	took   time.Duration
	index  int
	err    error
	done   bool
	stage  Stage
}

// worker draws plans, one at a time, telling the loop of each step.
type worker struct {
	stager Stager
	now    func() time.Time
	out    chan<- message
}

// run is one plan: the job staged if asked, then its rungs drawn in
// order until one would pass the ceiling or the context is done. It
// always ends with a done message carrying the stage.
func (working worker) run(ctx context.Context, asked plan) {
	// a scene that panics while staged or drawn ends this plan with an
	// error at the foot, so the loop lands and the terminal is given
	// back; the stage it held is let go with the panic
	defer func() {
		if caught := recover(); caught != nil {
			working.out <- message{err: fmt.Errorf("the frame panicked: %v",
				caught), done: true}
		}
	}()
	stage, err := working.staged(ctx, asked)
	if err != nil {
		working.out <- message{err: err, done: true}
		return
	}
	rungs := asked.rungs(factsOf(stage))
	var last rung
	var took time.Duration
	for place, each := range rungs {
		if place > none && !admitted(last, took, each) {
			break
		}
		if ctx.Err() != nil {
			break
		}
		start := working.now()
		drawn, err := stage.Frame(ctx, asked.index, each.size)
		took = working.now().Sub(start)
		if err != nil {
			working.refused(ctx, err)
			break
		}
		working.out <- message{drawn: drawn, rung: each, took: took,
			index: asked.index}
		last = each
	}
	working.out <- message{done: true, stage: stage}
}

// staged is the plan's stage: the one lent, or the job opened afresh,
// the one lent closed first. A fresh stage's facts go to the loop.
func (working worker) staged(ctx context.Context, asked plan) (Stage,
	error) {
	if !asked.restage && asked.stage != nil {
		return asked.stage, nil
	}
	var closing error
	if asked.stage != nil {
		closing = asked.stage.Close()
	}
	opened, err := working.stager.Stage(ctx, asked.wanted)
	if err = errors.Join(closing, err); err != nil {
		if opened != nil {
			err = errors.Join(err, opened.Close())
		}
		return nil, err
	}
	working.out <- message{staged: true, facts: factsOf(opened)}
	return opened, nil
}

// refused tells the loop of a frame that could not be drawn, unless the
// plan was cancelled, which is no refusal.
func (working worker) refused(ctx context.Context, err error) {
	if ctx.Err() != nil {
		return
	}
	working.out <- message{err: err}
}

// rungs is what the plan draws: the frame at the job's own size for a
// full frame, the one rung play draws at, or the ladder for the pane.
func (asked plan) rungs(made facts) []rung {
	if asked.mode == full {
		return []rung{{words: "full size"}}
	}
	if asked.playing && asked.fastest != (rung{}) {
		return []rung{asked.fastest}
	}
	ladder := ladderFor(made, asked.screen)
	if asked.playing {
		return ladder[:one]
	}
	return ladder
}

// ladderFor is the ladder for a stage's shape in a screen's pane.
func ladderFor(made facts, screen terminal.Screen) []rung {
	place := placed(made.width, made.height, paneOf(screen))
	return ladderOf(place.width, place.height)
}
