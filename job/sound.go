package job

import "time"

// Sound is a film's soundtrack: music rendered here from a score and a
// voicing, or an audio file read from a point onward.
type Sound struct {
	wire soundWire
}

// Sound is the job's sound, and whether it says any.
func (job Job) Sound() (Sound, bool) {
	if job.wire.Sound == nil {
		return Sound{}, false
	}
	return Sound{*job.wire.Sound}, true
}

// Tune is the score and the voicing of music rendered here, each an
// absolute path or @name for a built-in, and whether the sound is one.
func (track Sound) Tune() (score, voicing string, has bool) {
	return track.wire.Tune, track.wire.Voicing, track.wire.Tune != ""
}

// File is the audio file, absolute, and whether the sound is one.
func (track Sound) File() (string, bool) {
	return track.wire.File, track.wire.File != ""
}

// From is where in the audio file the film's sound starts, and whether
// the job said; unsaid, it is the file's start.
func (track Sound) From() (time.Duration, bool) {
	return spanOf(track.wire.From)
}

// Beats is the score that gives the audio file's beats, absolute, and
// whether the job names one; without one the beats are estimated.
func (track Sound) Beats() (string, bool) {
	return track.wire.Score, track.wire.Score != ""
}
