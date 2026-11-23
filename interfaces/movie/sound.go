package movie

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// Sound is a stereo soundtrack: float32 samples in -1..1, left then
// right, at a stated rate. The zero Sound is no sound at all.
type Sound struct {
	rate    int
	samples []float32 // interleaved, left then right
}

// NewSound checks interleaved stereo samples and keeps a copy of them.
// The rate is 44100 or 48000; every sample is a number in -1..1; there
// is at least one pair, and no half pair at the end.
func NewSound(rate int, interleaved []float32) (Sound, error) {
	if !slices.Contains(soundRates, rate) {
		return Sound{}, fmt.Errorf("movie: a sound at %d samples a "+
			"second is not offered: use one of %v", rate, soundRates)
	}
	if len(interleaved) == zero {
		return Sound{}, fmt.Errorf("movie: a sound with no samples: " +
			"leave the Sound out for a movie with no audio track")
	}
	if len(interleaved)%channels != zero {
		return Sound{}, fmt.Errorf("movie: %d samples is not whole "+
			"stereo pairs: give left then right for every instant",
			len(interleaved))
	}
	for place, sample := range interleaved {
		value := float64(sample)
		if math.IsNaN(value) || math.Abs(value) > fullScale {
			return Sound{}, fmt.Errorf("movie: sample %d is %v, outside "+
				"-1..1: scale the sound before handing it over", place,
				sample)
		}
	}
	return newSound(rate, slices.Clone(interleaved)), nil
}

// newSound is a sound from samples already checked and owned.
func newSound(rate int, interleaved []float32) Sound {
	return Sound{rate: rate, samples: interleaved}
}

// Rate is the samples a second on each channel.
func (sound Sound) Rate() int { return sound.rate }

// Length is how many instants the sound holds: pairs, not samples.
func (sound Sound) Length() int { return len(sound.samples) / channels }

// Duration is how long the sound plays.
func (sound Sound) Duration() time.Duration {
	if !positive(sound.rate) {
		return zero
	}
	seconds := float64(sound.Length()) / float64(sound.rate)
	return time.Duration(seconds * float64(time.Second))
}

// Samples is a copy of the samples, interleaved left then right.
func (sound Sound) Samples() []float32 { return slices.Clone(sound.samples) }

// present is whether there is any sound: the zero Sound has none.
func (sound Sound) present() bool { return sound.rate != zero }

// instantOf is the first instant of the sound that plays with frame
// index at that frame rate: index * soundRate / frameRate, rounded
// down, so the spans of all frames tile the picture exactly.
func (sound Sound) instantOf(index, frameRate int) int64 {
	return int64(index) * int64(sound.rate) / int64(frameRate)
}

// span is the samples that play with frames from first up to last, not
// including it: cut at the sound's end and followed by silence after.
func (sound Sound) span(first, last, frameRate int) []float32 {
	from := sound.instantOf(first, frameRate) * channels
	to := sound.instantOf(last, frameRate) * channels
	out := make([]float32, to-from)
	held := int64(len(sound.samples))
	if from < held {
		copy(out, sound.samples[from:min(to, held)])
	}
	return out
}

// stereoOf is the samples of interleaved decoded sound with that many
// channels as stereo: mono is put on both sides, stereo kept, and more
// refused, as is a sound with no whole instant in it. Samples past full
// scale are clipped to it.
func stereoOf(count int, decoded []float32) ([]float32, error) {
	if count != mono && count != channels {
		return nil, fmt.Errorf("movie: the sound has %d channels: "+
			"only mono and stereo are read", count)
	}
	if len(decoded) == zero {
		return nil, fmt.Errorf("movie: the sound holds no samples")
	}
	for place, sample := range decoded {
		if math.IsNaN(float64(sample)) {
			return nil, fmt.Errorf("movie: decoded sample %d is not "+
				"a number: the file is damaged", place)
		}
	}
	out := make([]float32, zero, len(decoded)/count*channels)
	lastChannel := count - one
	for place := zero; place+count <= len(decoded); place += count {
		left := clip(decoded[place])
		right := clip(decoded[place+lastChannel])
		out = append(out, left, right)
	}
	if len(out) == zero {
		return nil, fmt.Errorf("movie: the sound holds %d samples, fewer "+
			"than one instant of %d channels", len(decoded), count)
	}
	return out, nil
}

// clip keeps a decoded sample within full scale.
func clip(sample float32) float32 {
	value := float64(sample)
	within := math.Min(fullScale, value)
	return float32(math.Max(-fullScale, within))
}

// Slice copies a range of stereo instants without first copying the whole
// soundtrack. The returned Sound owns its samples, just like NewSound.
// Endpoints are half-open and must describe at least one complete instant.
func (sound Sound) Slice(first, last int) (Sound, error) {
	if first < zero || last <= first || last > sound.Length() {
		return Sound{}, fmt.Errorf("movie: invalid sound slice [%d,%d) "+
			"of %d instants", first, last, sound.Length())
	}
	from, to := first*channels, last*channels
	return newSound(sound.rate, slices.Clone(sound.samples[from:to])), nil
}
