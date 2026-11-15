package door

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
)

// track is a film's sound, ready for the movie, and the one line that
// says how it fits the film.
type track struct {
	sound movie.Sound
	said  string
}

// soundtrack is the sound the job asks for, for a film length long: a
// tune rendered here, looped to the film's end when its score loops, or
// an audio file read from its from onward.
func soundtrack(said job.Sound, length time.Duration) (track, error) {
	if score, voicing, isTune := said.Tune(); isTune {
		return tuned(score, voicing, length)
	}
	path, _ := said.File()
	from, _ := said.From()
	return filed(path, from, length)
}

// tuned is a tune's sound for the film. A tune that loops and is shorter
// than the film is played again until the film ends, and cut there; one
// that does not loop is followed by silence.
func tuned(score, voicing string, length time.Duration) (track, error) {
	piece, err := cheeptune.Open(score, voicing)
	if err != nil {
		return track{}, err
	}
	pairs := len(piece.Left)
	samples := make([]float32, nothing, pairs*stereo)
	for i := range pairs {
		samples = append(samples, piece.Left[i], piece.Right[i])
	}
	said := fmt.Sprintf("the tune is %s and the film %s", seconds(
		piece.Duration()), seconds(length))
	switch {
	case piece.Duration() >= length:
		said += ": the tune is cut at the film's end"
	case piece.LoopPeriod() > nothing:
		samples = looped(samples, sampleCount(piece.Rate, length))
		said += ": the tune loops to the film's end"
	default:
		said += ": silence follows the tune"
	}
	sound, err := movie.NewSound(piece.Rate, samples)
	return track{sound: sound, said: said}, err
}

// sampleCount is how many samples, both channels counted, a sound at
// rate holds for length.
func sampleCount(rate int, length time.Duration) int {
	return int(length.Seconds()*float64(rate)) * stereo
}

// looped is samples played again and again to count, and cut there.
func looped(samples []float32, count int) []float32 {
	made := make([]float32, nothing, count)
	for len(made) < count {
		made = append(made, samples[:min(len(samples),
			count-len(made))]...)
	}
	return made
}

// filed is an audio file's sound from a point onward. A point at or past
// the file's end is refused, by the file's name.
func filed(path string, from, length time.Duration) (track, error) {
	read, err := movie.ReadSound(path)
	if err != nil {
		return track{}, err
	}
	skip := sampleCount(read.Rate(), from)
	samples := read.Samples()
	if skip >= len(samples) {
		return track{}, fmt.Errorf("%s is %s long, and the sound is to "+
			"start at %s: nothing is left; say an earlier from", path,
			seconds(read.Duration()), seconds(from))
	}
	sound, err := movie.NewSound(read.Rate(), samples[skip:])
	if err != nil {
		return track{}, err
	}
	said := fmt.Sprintf("the sound file plays %s from %s, and the film "+
		"is %s", seconds(sound.Duration()), seconds(from),
		seconds(length))
	return track{sound: sound, said: said}, nil
}

// seconds is a duration as a person reads it in a line: 20.60 s.
func seconds(length time.Duration) string { return hundredths(length) }
