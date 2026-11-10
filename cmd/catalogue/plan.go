package main

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/sky"
)

//go:embed plan.yaml
var planRaw []byte

//go:embed plan.schema.json
var planSchema []byte

type source struct {
	Toward   maths.Vec   `json:"toward" yaml:"toward"`
	Strength maths.Light `json:"strength" yaml:"strength"`
}

type plan struct {
	Width         int          `json:"width" yaml:"width"`
	Height        int          `json:"height" yaml:"height"`
	Rays          int          `json:"rays" yaml:"rays"`
	Seed          uint64       `json:"seed" yaml:"seed"`
	Details       []string     `json:"details" yaml:"details"`
	Wide          float64      `json:"wide" yaml:"wide"`
	Camera        maths.Vec    `json:"camera" yaml:"camera"`
	Distance      float64      `json:"distance" yaml:"distance"`
	Sky           sky.Gradient `json:"sky" yaml:"sky"`
	Sources       []source     `json:"sources" yaml:"sources"`
	Tone          colour.Tone  `json:"tone" yaml:"tone"`
	SampleLimit   int64        `json:"sample-limit" yaml:"sample-limit"`
	FaceLimit     int          `json:"face-limit" yaml:"face-limit"`
	PixelAspect   float64      `json:"pixel-aspect" yaml:"pixel-aspect"`
	LightExposure float64      `json:"light-exposure" yaml:"light-exposure"`
	MinCoverage   float64      `json:"minimum-coverage" yaml:"minimum-coverage"`
	Cutaway       bool         `json:"cutaway" yaml:"cutaway"`
	raw           []byte
}

// zeroLength is defined as a direction with no length.
const zeroLength = 0

// bothInfinitySigns is defined by math.IsInf's sign argument.
const bothInfinitySigns = 0

// loadPlan reads a caller's plan, or the embedded authored defaults.
func loadPlan(call pzm.Call) (*plan, error) {
	raw, name := planRaw, "plan.yaml"
	if call.Given("plan") {
		name = call.Value("plan")
		var err error
		raw, err = os.ReadFile(name)
		if err != nil {
			return nil, err
		}
	}
	return readPlan(name, raw)
}

// readPlan is shared by file input and the embedded defaults.
func readPlan(name string, raw []byte) (*plan, error) {
	var made plan
	if err := readDocument(name, raw, planSchema, &made); err != nil {
		return nil, err
	}
	if err := made.check(); err != nil {
		return nil, err
	}
	made.raw = raw
	return &made, nil
}

// check holds allocation and camera invariants after structural validation.
func (plan plan) check() error {
	samples := int64(plan.Width) * int64(plan.Height) *
		int64(plan.Rays) * int64(plan.Rays)
	if samples > plan.SampleLimit {
		return fmt.Errorf("catalogue asks for %d samples; limit %d",
			samples, plan.SampleLimit)
	}
	horizontal := plan.Camera.X*plan.Camera.X + plan.Camera.Z*plan.Camera.Z
	if !(plan.Camera.Len() > zeroLength) ||
		math.IsInf(plan.Camera.Len(), bothInfinitySigns) ||
		!(horizontal > zeroLength) {
		return fmt.Errorf("catalogue camera needs a finite heading")
	}
	for _, source := range plan.Sources {
		_, err := emit.NewDistant(source.Toward, source.Strength)
		if err != nil {
			return err
		}
	}
	for _, detail := range plan.Details {
		if !slices.Contains(manufacture.Complexities(), detail) {
			return fmt.Errorf("unknown catalogue detail %q", detail)
		}
	}
	return nil
}
