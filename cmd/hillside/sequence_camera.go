package main

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/camera"
	"github.com/janearc/slayer/maths"
)

// sequenceAbsent is defined as no selected numbered smoke.
const sequenceAbsent = 0

// sequenceMinimumFrames is defined as one actual frame for a nonempty clip.
const sequenceMinimumFrames = 1

// sequenceCamera is decoded only after Puzzlemouth checks its schema.
// Coordinates are local scene metres: x east, y up, z south.
type sequenceCamera struct {
	LookAt maths.Vec     `json:"look-at" yaml:"look-at"`
	Up     *maths.Vec    `json:"up"`
	Zoom   *sequenceZoom `json:"zoom"`
	Moves  []struct {
		From     maths.Vec         `json:"from"`
		To       maths.Vec         `json:"to"`
		Speed    *sequenceQuantity `json:"speed"`
		Duration *sequenceQuantity `json:"duration"`
		Ease     string            `json:"ease"`
		Look     *sequenceLook     `json:"look"`
	} `json:"moves"`
}

// sequenceZoom carries explicit angle and time units checked by the schema.
type sequenceZoom struct {
	From, To sequenceQuantity
	Duration sequenceQuantity
	Ease     string
}

// zoomPath is optional; an absent zoom preserves the scene's existing lens.
func (input *sequenceCamera) zoomPath() (*camera.Zoom, error) {
	if input == nil || input.Zoom == nil {
		return nil, nil
	}
	zoom := input.Zoom
	duration := time.Duration(zoom.Duration.Value * float64(time.Second))
	return camera.NewZoom(zoom.From.Value, zoom.To.Value,
		duration, zoom.Ease)
}

// sequenceLook is a move's own look: the target it looks at as it
// starts and as it ends, in the same scene metres as its points.
type sequenceLook struct {
	From maths.Vec `json:"from"`
	To   maths.Vec `json:"to"`
}

// path converts checked YAML quantities to the reusable camera description.
func (input *sequenceCamera) path() (*camera.Path, error) {
	if input == nil {
		return nil, nil
	}
	moves := make([]camera.Move, len(input.Moves))
	for index, in := range input.Moves {
		var err error
		if in.Speed != nil {
			if in.Ease == "smooth" {
				return nil, fmt.Errorf("camera: smooth " +
					"easing needs duration; " +
					"speed means constant speed")
			}
			moves[index], err = camera.Travel(in.From, in.To,
				in.Speed.Value)
		} else if in.Duration != nil {
			nanoseconds := in.Duration.Value * float64(time.Second)
			duration := time.Duration(nanoseconds)
			moves[index], err = camera.Timed(in.From, in.To,
				duration, in.Ease)
		}
		if err == nil && in.Look != nil {
			moves[index], err = moves[index].Looking(in.Look.From,
				in.Look.To)
		}
		if err != nil {
			return nil, err
		}
	}
	up := maths.Up
	if input.Up != nil {
		up = *input.Up
	}
	return camera.New(moves, input.LookAt, up)
}

// withCamera samples time relative to the sequence start. Scene time can start
// partway through a fire without starting the camera partway along its path.
func (plan sequencePlan) withCamera(given *asked,
	at time.Duration) error {
	elapsed := at - plan.start
	if plan.camera != nil {
		eye, err := plan.camera.At(elapsed)
		if err != nil {
			return fmt.Errorf("sequence: %w", err)
		}
		given.eye = eye
	}
	if plan.zoom != nil {
		wide, err := plan.zoom.At(elapsed)
		if err != nil {
			return fmt.Errorf("sequence: %w", err)
		}
		given.wide = wide
	}
	return nil
}
