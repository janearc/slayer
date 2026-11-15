package pan

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/janearc/slayer/interfaces/movie"
)

// Sound is the tune under the film: the piece from the first bar's
// start to the last bar's end, faded over its last moment.
func (made Cut) Sound() (movie.Sound, error) {
	rate := made.piece.Rate
	first := sampleAt(rate, made.barStart(made.first)-
		made.barStart(one))
	count := sampleAt(rate, made.Length())
	last := min(first+count, len(made.piece.Left))
	if first >= last {
		return movie.Sound{}, fmt.Errorf("bars %d to %d hold no sound",
			made.first, made.last)
	}
	fading := sampleAt(rate, tail)
	samples := make([]float32, nothing, (last-first)*stereo)
	for i := first; i < last; i++ {
		gain := float32(min(whole, float64(last-one-i)/float64(fading)))
		samples = append(samples, made.piece.Left[i]*gain,
			made.piece.Right[i]*gain)
	}
	return movie.NewSound(rate, samples)
}

// Write writes the film to path, an mp4, frame by frame with its sound,
// and says how far it is on said every second of film. A cancelled
// context or a failed frame throws the movie away.
func (made Cut) Write(ctx context.Context, path string,
	said io.Writer) error {
	sound, err := made.Sound()
	if err != nil {
		return err
	}
	settings, err := movie.NewSettings(movie.Request{Path: path,
		Width: made.size.X, Height: made.size.Y, Rate: made.rate,
		Frames: made.Frames(), Sound: sound, Pad: true, Replace: true})
	if err != nil {
		return err
	}
	writer, err := movie.New(ctx, settings)
	if err != nil {
		return err
	}
	for index := range made.Frames() {
		if err := made.frameInto(ctx, writer, index, said); err != nil {
			return errors.Join(err, writer.Abort())
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(said, "wrote %s: %d frames, %.2f s\n", path,
		made.Frames(), made.Length().Seconds())
	return err
}

// frameInto hands frame index to the writer, and says each second.
func (made Cut) frameInto(ctx context.Context, writer *movie.Writer,
	index int, said io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	picture, err := made.Picture(index)
	if err != nil {
		return err
	}
	if err := writer.Frame(index, picture); err != nil {
		return err
	}
	if (index+one)%made.rate != nothing {
		return nil
	}
	_, err = fmt.Fprintf(said, "frame %d of %d\n", index+one,
		made.Frames())
	return err
}

// sampleAt is the sample a duration in falls on, at rate.
func sampleAt(rate int, at interface{ Seconds() float64 }) int {
	return int(at.Seconds() * float64(rate))
}
