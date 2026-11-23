package walk

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/slayer/interfaces/movie"
)

// SongSpan is a span of a song for under a short: from a moment, so many
// seconds of it, at soundLevel, faded in and out over soundFade. A span
// past the song's end is refused by name.
func SongSpan(song movie.Sound, from, seconds float64) (movie.Sound, error) {
	rate := float64(song.Rate())
	first := int(math.Round(from * rate))
	count := int(math.Round(seconds * rate))
	if from < 0 || !(seconds > 0) || first+count > song.Length() {
		return movie.Sound{}, fmt.Errorf("%w: %.2f s of the song from "+
			"%.2f s; it is %.2f s long: start it earlier", errWalk, seconds,
			from, song.Duration().Seconds())
	}
	samples := song.Samples()[first*stereo : (first+count)*stereo]
	fade := soundFade * rate
	for i := range count {
		gain := soundLevel * math.Min(1, math.Min(float64(i)/fade,
			float64(count-1-i)/fade))
		for channel := range stereo {
			samples[i*stereo+channel] *= float32(gain)
		}
	}
	return movie.NewSound(song.Rate(), samples)
}

// Write writes a short to an mp4 at a path, with a sound under it (the
// zero Sound for none), replacing a file there. It says how many frames
// it wrote.
func (short Short) Write(ctx context.Context, path string,
	sound movie.Sound) (int, error) {
	settings, err := movie.NewSettings(movie.Request{Path: path,
		Width: frameSize.X, Height: frameSize.Y, Rate: filmRate,
		Frames: short.Frames(), Sound: sound, Replace: true})
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errWalk, err)
	}
	writer, err := movie.New(ctx, settings)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errWalk, err)
	}
	kept := NewStills()
	for i := range short.Frames() {
		frame, err := short.Frame(i, kept)
		if err == nil {
			err = writer.Frame(i, frame)
		}
		if err != nil {
			writer.Abort()
			return i, fmt.Errorf("%w: frame %d: %w", errWalk, i, err)
		}
	}
	if err := writer.Close(); err != nil {
		return short.Frames(), fmt.Errorf("%w: %w", errWalk, err)
	}
	return short.Frames(), nil
}
