package job

import (
	"fmt"
	"strings"
)

// check is one thing the schema cannot see: where in the document it is
// wrong, and what is wrong, or no words where nothing is.
type check func(wire document) (where []string, why string)

// checks are every check, in the order a job is refused by them.
var checks = []check{
	knownPreset, sceneForm, sceneParameters, scenePlayed, stillOrFilm,
	filmDuration,
	filmRate, lensAngle, lensOpening, printOrStock, cameraForm, soundForm,
	stillOutputs, filmOutputs, stillShutter, stillSound, framesUntraced,
	titleSpans,
}

// film is whether the job is a film: it says a rate or a span.
func (wire document) film() bool {
	return wire.FPS != nothing || wire.Duration != nil ||
		wire.Start != nil || wire.Stretch != nil
}

// knownPreset refuses a preset the table does not have.
func knownPreset(wire document) ([]string, string) {
	if wire.Film == nil || wire.Film.Preset == "" {
		return nil, ""
	}
	if _, found := presetNamed(wire.Film.Preset); found {
		return nil, ""
	}
	return []string{"film", "preset"}, fmt.Sprintf("film preset %q is "+
		"not in the presets table; Presets lists them",
		wire.Film.Preset)
}

// sceneForm refuses a scene of no form or of two.
func sceneForm(wire document) ([]string, string) {
	said := wire.Scene.forms()
	const oneOf = "a scene is exactly one of set, smoke, named or frames"
	switch len(said) {
	case nothing:
		return []string{"scene"}, "scene says no scene; " + oneOf
	case one:
		return nil, ""
	}
	return []string{"scene", said[one]}, fmt.Sprintf("scene says both "+
		"%s and %s; %s", said[nothing], said[one], oneOf)
}

// forms is each form the scene says, in the order the schema lists them.
func (scene sceneWire) forms() []string {
	var said []string
	for _, form := range []struct {
		name string
		has  bool
	}{
		{formSet, scene.Set != ""},
		{formSmoke, scene.Smoke != nothing},
		{formNamed, scene.Named != ""},
		{formFrames, scene.Frames != ""},
	} {
		if form.has {
			said = append(said, form.name)
		}
	}
	return said
}

// sceneParameters refuses parameters with no named scene to take them.
func sceneParameters(wire document) ([]string, string) {
	if wire.Scene.With == nil || wire.Scene.Named != "" {
		return nil, ""
	}
	return []string{"scene", "with"}, "scene with gives a named " +
		"scene its parameters, and this scene says no named"
}

// scenePlayed refuses a cue sheet with no named scene to play, or with
// no soundtrack to keep its time.
func scenePlayed(wire document) ([]string, string) {
	if wire.Scene.Play == "" {
		return nil, ""
	}
	if wire.Scene.Named == "" {
		return []string{"scene", "play"}, "scene play gives a named " +
			"scene its cue sheet, and this scene says no named"
	}
	if wire.Sound == nil || (wire.Sound.Tune == "" && wire.Sound.File == "") {
		return []string{"scene", "play"}, "scene play is read against " +
			"the job's sound: a tune {tune, voicing} or audio {file, score}"
	}
	return nil, ""
}

// stillOrFilm refuses a still's time beside a film's rate or span.
func stillOrFilm(wire document) ([]string, string) {
	if wire.At == nil || !wire.film() {
		return nil, ""
	}
	return []string{"at"}, "at is a still's time, and fps, duration, " +
		"start or scene-duration make a film; a job says one or the other"
}

// filmDuration refuses a film that does not say how long it plays.
func filmDuration(wire document) ([]string, string) {
	if !wire.film() || wire.Duration != nil {
		return nil, ""
	}
	where := []string{"scene-duration"}
	if wire.Start != nil {
		where = []string{"start"}
	}
	if wire.FPS != nothing {
		where = []string{"fps"}
	}
	return where, "a film says its duration, as {value: 10, unit: s}"
}

// filmRate refuses a film with no rate from the job or its preset.
func filmRate(wire document) ([]string, string) {
	if !wire.film() || wire.FPS != nothing {
		return nil, ""
	}
	where := []string{"duration"}
	if wire.Film != nil && wire.Film.Preset != "" {
		where = []string{"film", "preset"}
	}
	return where, "a film needs a rate: say fps, or a film preset " +
		"that gives one"
}

// lensAngle refuses a lens said both across the picture and down it.
func lensAngle(wire document) ([]string, string) {
	if wire.Film == nil || wire.Film.Wide == nothing ||
		wire.Film.Tall == nothing {
		return nil, ""
	}
	return []string{"film", "tall"}, "film says wide and tall; say " +
		"one: wide is the angle across the picture, tall the angle down it"
}

// lensOpening refuses an aperture with no focus, a focus with no
// aperture, and an aperture as wide as its focus is far or wider, which
// no lens is.
func lensOpening(wire document) ([]string, string) {
	if wire.Film == nil {
		return nil, ""
	}
	aperture, focus := wire.Film.Aperture, wire.Film.Focus
	switch {
	case aperture == nothing && focus == nothing:
		return nil, ""
	case focus == nothing:
		return []string{"film", "aperture"}, "film says an aperture " +
			"and no focus; say how far off the lens is focused, in the " +
			"scene's unit, as focus: 12"
	case aperture == nothing:
		return []string{"film", "focus"}, "film says a focus and no " +
			"aperture; say how wide the lens is open, in the scene's " +
			"unit, as aperture: 0.1, or leave focus out for a pinhole"
	case aperture >= focus:
		return []string{"film", "aperture"}, "film's aperture is as " +
			"wide as its focus is far, or wider; open it less than the " +
			"focus distance"
	}
	return nil, ""
}

// printOrStock refuses two developments.
func printOrStock(wire document) ([]string, string) {
	if wire.Print == nil || wire.Stock == nil {
		return nil, ""
	}
	return []string{"stock"}, "print and stock both develop the " +
		"picture; say one, or neither for the scene's own print"
}

// cameraForm refuses a camera that is a name and a path, or neither, or
// a path that leaves out what a path must say.
func cameraForm(wire document) ([]string, string) {
	lens := wire.Camera
	if lens == nil {
		return nil, ""
	}
	path := lens.Unit != "" || lens.LookAt != nil || lens.Up != nil ||
		lens.Moves != nil
	whole := lens.Unit != "" && lens.LookAt != nil && lens.Moves != nil
	switch {
	case lens.Name != "" && path:
		return []string{"camera", "name"}, "camera says a name and a " +
			"path; a camera is one the scene defines or a path, not both"
	case lens.Name == "" && !path:
		return []string{"camera"}, "camera says neither a name nor a " +
			"path; a zoom stands beside one of them"
	case path && !whole:
		return []string{"camera"}, "a camera path says its unit, " +
			"look-at and moves"
	}
	return nil, ""
}

// soundForm refuses sound that is a tune and a file, or neither.
func soundForm(wire document) ([]string, string) {
	sound := wire.Sound
	switch {
	case sound == nil:
		return nil, ""
	case sound.Tune != "" && sound.File != "":
		return []string{"sound", "file"}, "sound is a tune or a file, " +
			"not both"
	case sound.Tune == "" && sound.File == "":
		return []string{"sound"}, "sound says a tune and its voicing, " +
			"or a file"
	}
	return nil, ""
}

// stillOutputs refuses a still asked for a film's outputs.
func stillOutputs(wire document) ([]string, string) {
	if wire.film() || wire.Outputs == nil {
		return nil, ""
	}
	return roleRefused(map[string]string{"frames": wire.Outputs.Frames,
		"movie": wire.Outputs.Movie}, "a still writes picture or raw, "+
		"and outputs.%s is a film's; say fps and duration to make a film")
}

// filmOutputs refuses a film asked for a still's outputs.
func filmOutputs(wire document) ([]string, string) {
	if !wire.film() || wire.Outputs == nil {
		return nil, ""
	}
	return roleRefused(map[string]string{"picture": wire.Outputs.Picture,
		"raw": wire.Outputs.Raw}, "a film writes frames or movie, and "+
		"outputs.%s is a still's")
}

// roleRefused is the first role, by name, that is said, refused with
// the words given.
func roleRefused(roles map[string]string, words string) ([]string,
	string) {
	for _, role := range []string{"picture", "raw", "frames", "movie"} {
		if roles[role] != "" {
			return []string{"outputs", role}, fmt.Sprintf(words, role)
		}
	}
	return nil, ""
}

// stillShutter refuses motion blur on a still.
func stillShutter(wire document) ([]string, string) {
	if wire.film() || wire.Shutter == nil {
		return nil, ""
	}
	return []string{"shutter"}, "shutter blurs a film's motion across " +
		"each frame's interval, and a still has none; say fps and duration"
}

// stillSound refuses sound laid under a still.
func stillSound(wire document) ([]string, string) {
	if wire.film() || wire.Sound == nil {
		return nil, ""
	}
	return []string{"sound"}, "sound is laid under a film, and a " +
		"still has none; say fps and duration"
}

// framesUntraced refuses what only tracing uses, for frames already
// rendered.
func framesUntraced(wire document) ([]string, string) {
	if wire.Scene.Frames == "" {
		return nil, ""
	}
	var traced []string
	if wire.Camera != nil {
		traced = append(traced, "camera")
	}
	if wire.Shutter != nil {
		traced = append(traced, "shutter")
	}
	if len(traced) == nothing {
		return nil, ""
	}
	return []string{traced[nothing]}, fmt.Sprintf("scene frames are "+
		"already rendered and nothing is traced, so %s does nothing; "+
		"take it out", strings.Join(traced, " and "))
}

// titleSpans refuses a title that goes before it comes.
func titleSpans(wire document) ([]string, string) {
	if wire.Titles == nil {
		return nil, ""
	}
	for _, each := range wire.Titles.Elements {
		from, until := each.title.From, each.title.Until
		if from == nil || until == nil || until.Value > from.Value {
			continue
		}
		return []string{"titles", "elements", each.name, "until"},
			fmt.Sprintf("title %s goes at %gs, which is not after it "+
				"comes at %gs; until is after from", each.name,
				until.Value, from.Value)
	}
	return nil, ""
}
