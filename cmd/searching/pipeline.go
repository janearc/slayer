package main

import (
	"context"
	"errors"
	"time"
)

// pipelineTiming separates occupied stage time from elapsed time. Draw and
// output overlap, so their sum is not the film's elapsed rendering time.
type pipelineTiming struct {
	EncodeSeconds float64 `json:"encode_seconds,omitempty"`
	Mode          string  `json:"mode"`
	DrawSeconds   float64 `json:"draw_seconds"`
	OutputSeconds float64 `json:"output_seconds"`
}

// frameWork traces one frame and returns its owned CPU finishing work.
// That work must not read or change the mutable scene used by the next draw.
type frameWork func(context.Context, int) (func() error, error)

// pipeline overlaps a sequential drawer with one sequential output worker.
// The unbuffered handoff admits at most two frames: one in each stage. A
// slow writer holds the producer before another trace can allocate a frame.
// Every return joins the worker, including cancellation and output failure.
func pipeline(parent context.Context, count int, draw frameWork, serial bool) (pipelineTiming, error) {
	if serial {
		return serialFrames(parent, count, draw)
	}
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	timing := pipelineTiming{Mode: "overlapped"}
	jobs := make(chan func() error)
	done := make(chan error, 1)
	go func() {
		err := finishFrames(ctx, jobs, &timing.OutputSeconds)
		if err != nil {
			cancel(err)
		}
		done <- err
	}()
	err := drawFrames(ctx, count, draw, jobs, &timing.DrawSeconds)
	if err != nil {
		cancel(err)
	}
	close(jobs)
	finished := <-done
	return timing, errors.Join(err, finished, context.Cause(ctx))
}

// drawFrames hands off each finished trace before preparing another one.
func drawFrames(ctx context.Context, count int, draw frameWork,
	jobs chan<- func() error, seconds *float64) error {
	for index := range count {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		begun := time.Now()
		finish, err := draw(ctx, index)
		*seconds += time.Since(begun).Seconds()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case jobs <- finish:
		}
	}
	return nil
}

// finishFrames preserves frame order and stops before consuming cancelled
// work. It owns development, PNG output and the movie writer's Frame calls.
func finishFrames(ctx context.Context, jobs <-chan func() error, seconds *float64) error {
	for finish := range jobs {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		begun := time.Now()
		err := finish()
		*seconds += time.Since(begun).Seconds()
		if err != nil {
			return err
		}
	}
	return context.Cause(ctx)
}

// serialFrames supplies a matched control using exactly the same frame work.
func serialFrames(ctx context.Context, count int, draw frameWork) (pipelineTiming, error) {
	timing := pipelineTiming{Mode: "serial"}
	for index := range count {
		if err := context.Cause(ctx); err != nil {
			return timing, err
		}
		begun := time.Now()
		finish, err := draw(ctx, index)
		timing.DrawSeconds += time.Since(begun).Seconds()
		if err != nil {
			return timing, err
		}
		if err := context.Cause(ctx); err != nil {
			return timing, err
		}
		begun = time.Now()
		err = finish()
		timing.OutputSeconds += time.Since(begun).Seconds()
		if err != nil {
			return timing, err
		}
	}
	return timing, context.Cause(ctx)
}
