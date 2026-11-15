package job

import "encoding/json"

// Scene is which scene the job renders, in exactly one of its forms: a
// set file, a numbered smoke, a scene built in Go by name, or frames
// already rendered.
type Scene struct {
	wire sceneWire
}

// Scene is the job's scene.
func (job Job) Scene() Scene { return Scene{job.wire.Scene} }

// Form is the scene's form: set, smoke, named or frames.
func (scene Scene) Form() string { return scene.wire.forms()[nothing] }

// Set is the set file, absolute, and whether the scene is one.
func (scene Scene) Set() (string, bool) {
	return scene.wire.Set, scene.wire.Set != ""
}

// Smoke is the smoke's number, and whether the scene is one.
func (scene Scene) Smoke() (int, bool) {
	return scene.wire.Smoke, scene.wire.Smoke != nothing
}

// Named is the Go scene's name, and whether the scene is one.
func (scene Scene) Named() (string, bool) {
	return scene.wire.Named, scene.wire.Named != ""
}

// Parameters is what a named scene's with says, for the scene to check,
// as JSON decodes it, so every number is a float64. It is a fresh copy
// each call, and nil where the scene says none.
func (scene Scene) Parameters() map[string]any {
	if scene.wire.With == nil {
		return nil
	}
	raw, _ := json.Marshal(scene.wire.With)
	var copied map[string]any
	_ = json.Unmarshal(raw, &copied)
	return copied
}

// Play is the cue sheet a named scene is played by, absolute, and whether
// the scene says one. Its clock is the job's sound, a tune.
func (scene Scene) Play() (string, bool) {
	return scene.wire.Play, scene.wire.Play != ""
}

// Frames is the directory of frames already rendered, absolute, and
// whether the scene is one.
func (scene Scene) Frames() (string, bool) {
	return scene.wire.Frames, scene.wire.Frames != ""
}
