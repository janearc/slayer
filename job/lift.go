package job

import (
	"encoding/json"
	"os"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// Plan is the values a sequence-v1 plan holds, as the session reads them,
// for Lift to make a job of. The session reads the plan; this package
// never does. Paths are absolute, or resolved against the working directory.
type Plan struct {
	// File is the plan's file, for messages.
	File string
	// Scene is the set file, or nothing for a numbered smoke.
	Scene string
	// Smoke is the smoke's number, or zero for a set.
	Smoke int
	// Width, Height and Rays are the plan's film, or zeros where it says
	// none.
	Width, Height, Rays int
	// FPS is frames a second.
	FPS int
	// Start, Duration and SceneDuration are seconds, as the plan writes
	// them; a SceneDuration of zero is one the plan does not say.
	Start, Duration, SceneDuration float64
	// Knee, Top and Bloom are the plan's print.
	Knee, Top, Bloom float64
	// Camera is the plan's camera block as YAML, or nil where it says
	// none.
	Camera []byte
	// Output is the plan's output directory, which becomes the frames.
	Output string
}

// Lift is the job with the same frames as a sequence plan: its scene or
// smoke, film, rate, span, print, camera, and output as the frames. It
// is checked as a job read from a file is.
func Lift(plan Plan) (Job, error) {
	name := plan.File + ", lifted to a job"
	wire := plan.document()
	if plan.Camera != nil {
		var lens cameraWire
		if err := yamlfile.Read(name, plan.Camera, &lens); err != nil {
			return Job{}, refusal{err}
		}
		wire.Camera = &lens
	}
	base, err := os.Getwd()
	if err != nil {
		return Job{}, err
	}
	// the marshalling cannot fail: every value is a plain number or a
	// string, or a camera just read from YAML
	raw, _ := json.Marshal(wire)
	return admitted(source{name: name, base: base}, raw)
}

// document is the job the plan's values say, but for its camera.
func (plan Plan) document() document {
	bloom := plan.Bloom
	wire := document{Schema: schemaName, FPS: plan.FPS,
		Scene:    sceneWire{Set: plan.Scene, Smoke: plan.Smoke},
		Start:    &quantity{Value: plan.Start, Unit: seconds},
		Duration: &quantity{Value: plan.Duration, Unit: seconds},
		Print: &printWire{Knee: plan.Knee, Top: plan.Top,
			Bloom: &bloom},
		Outputs: &outputsWire{Frames: plan.Output}}
	if plan.SceneDuration != nothing {
		wire.Stretch = &quantity{Value: plan.SceneDuration,
			Unit: seconds}
	}
	if plan.Width != nothing || plan.Height != nothing ||
		plan.Rays != nothing {
		wire.Film = &filmWire{Width: plan.Width, Height: plan.Height,
			Rays: plan.Rays}
	}
	return wire
}
