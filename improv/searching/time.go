package searching

import "math"

// State is one exposure's authored action, evaluated without a mutable playhead.
type State struct {
	At, Beat, Phase                              float64
	Site, Chapter, Spectrum                      string
	Aim, Camera, Flash, Saturation, SceneSeconds float64
}

// Frames is the actual frame count, without duplicating a final endpoint.
func (edit *Edit) Frames() int {
	return int(math.Ceil(edit.Grid.Score.Duration * float64(edit.Plan.FPS)))
}

// motion integrates the four beat velocities. It is continuous at every
// beat and bar boundary; the derivative, deliberately, changes on the beat.
func (edit *Edit) motion(beat float64) float64 {
	beat = math.Max(0, beat)
	bar := math.Floor(beat / 4)
	sum := 0.0
	for _, weight := range edit.Plan.BeatMotion {
		sum += weight
	}
	value := bar * sum
	within := beat - bar*4
	for index, weight := range edit.Plan.BeatMotion {
		value += weight * math.Max(0, math.Min(1, within-float64(index)))
	}
	return value
}

// At maps the song clock to a shot and its independent scenery clock.
func (edit *Edit) At(seconds float64) State {
	beat := edit.Grid.BeatAt(seconds)
	state := State{At: seconds, Beat: beat, Phase: beat - math.Floor(beat),
		Site: "hawaii", Chapter: "search", Saturation: 1}
	bar := math.Max(0, math.Floor(beat/4))
	if int(bar)/edit.Plan.SearchCutBars%2 == 1 {
		state.Site = "atacama"
	}
	phase := int(math.Floor(math.Max(0, beat))) % 4
	if phase == 3 {
		state.Saturation = 1 - 0.35*state.Phase
	}
	// Quantize accents to their nearest preview frame, so six-fps sampling
	// cannot miss a beat that falls between two exposure instants.
	frame := int(math.Round(seconds * float64(edit.Plan.FPS)))
	for index := 0; index < len(edit.Grid.Score.Beats.Times); index += 4 {
		at := edit.Grid.Score.Beats.Times[index]
		if int(math.Round(at*float64(edit.Plan.FPS))) == frame {
			state.Flash = 1
		}
	}
	for _, at := range edit.Accents {
		if int(math.Round(at*float64(edit.Plan.FPS))) == frame {
			state.Flash = 0.5
		}
	}
	state.Camera = edit.motion(beat)
	state.SceneSeconds = math.Min(1, seconds/edit.Lock) * edit.Plan.SearchDays * 86400
	offset := 0.0
	if state.Site == "atacama" {
		offset = 1.7
	}
	state.Aim = edit.Plan.Azimuth + 65*math.Sin(state.Camera*0.19+offset)
	settle := edit.Grid.SecondAt(edit.Grid.BeatAt(edit.Lock) - 4)
	if seconds >= settle {
		u := smooth((seconds - settle) / (edit.Lock - settle))
		start := edit.Plan.Azimuth + 65*math.Sin(edit.motion(edit.Grid.BeatAt(settle))*0.19+offset)
		state.Aim = start*(1-u) + edit.Plan.Azimuth*u
	}
	if seconds >= edit.Lock {
		state.Chapter, state.Aim, state.Flash = "lock", edit.Plan.Azimuth, 0
		state.Camera = edit.motion(edit.Grid.BeatAt(edit.Lock)) + (seconds-edit.Lock)*0.03
		if int(bar)%2 == 1 {
			state.Site = "atacama"
		} else {
			state.Site = "hawaii"
		}
	}
	if seconds >= edit.RadioOne {
		state.Site, state.Chapter = "radio-hawaii", "radio response"
		state.Aim = edit.Plan.Azimuth - 90*(1-smooth((seconds-edit.RadioOne)/6))
	}
	if seconds >= edit.RadioTwo {
		state.Site = "radio-atacama"
		state.Aim = edit.Plan.Azimuth + 100*(1-smooth((seconds-edit.RadioTwo)/6))
	}
	if seconds >= edit.Reveal {
		state.Site, state.Chapter = "blazar", "reveal"
		spectra := []string{"radio", "infrared", "visible", "x-ray", "gamma"}
		start := int(math.Floor(edit.Grid.BeatAt(edit.Reveal) / 4))
		state.Spectrum = spectra[(int(bar)-start)%len(spectra)]
	}
	if seconds >= edit.Coda {
		state.Chapter, state.Site, state.Aim = "coda", "hawaii", edit.Plan.Azimuth
		if int(bar)%2 == 1 {
			state.Site = "atacama"
		}
		state.Flash = 0
	}
	return state
}

// smooth is an authored cubic ease, clamped to its shot's endpoints.
func smooth(value float64) float64 {
	value = math.Max(0, math.Min(1, value))
	return value * value * (3 - 2*value)
}
