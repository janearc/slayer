package audition

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/janearc/slayer/interfaces/movie"
)

// Rate is the frames a second of an audition film. Picked 2026-11-12:
// twelve is small and quick, and the picture is a line
// moving across a still.
const Rate = 12

// cursorWidth is the line's width in pixels. Picked, 2026-11-12.
const cursorWidth = 3

// Film writes the tune as a film: its plate, a line moving across it, the
// part that is playing named at the foot, and the tune as the sound.
func Film(t *Tune, path string) error {
	plate, err := Plate(t)
	if err != nil {
		return err
	}
	w, err := newWriter()
	if err != nil {
		return err
	}
	both := make([]float32, 0, 2*len(t.Piece.Left))
	for i := range t.Piece.Left {
		both = append(both, t.Piece.Left[i], t.Piece.Right[i])
	}
	sound, err := movie.NewSound(t.Piece.Rate, both)
	if err != nil {
		return err
	}
	frames := int(t.seconds()*Rate) + 1
	settings, err := movie.NewSettings(movie.Request{Path: path, Width: Wide,
		Height: Tall, Rate: Rate, Frames: frames, Sound: sound,
		Replace: true})
	if err != nil {
		return err
	}
	out, err := movie.New(context.Background(), settings)
	if err != nil {
		return err
	}
	for index := range frames {
		picture := t.frame(plate, w, float64(index)/Rate)
		if err := out.Frame(index, picture); err != nil {
			out.Abort()
			return fmt.Errorf("%s: frame %d: %w", t.Name, index, err)
		}
	}
	return out.Close()
}

// frame is the plate at one moment: the line and the playing part.
func (t *Tune) frame(plate *image.RGBA, w *writer, second float64) *image.RGBA {
	picture := image.NewRGBA(plate.Bounds())
	copy(picture.Pix, plate.Pix)
	x := t.xAt(second)
	fill(picture, image.Rect(x, stripTop, x+cursorWidth, rollBottom), cursor)
	at := time.Duration(second * float64(time.Second))
	for _, span := range t.Sheet.Spans() {
		if at >= span.Start && at < span.End {
			w.text(picture, margin, footY, titleSize, color.RGBA(cursor),
				span.Name)
			w.text(picture, margin, footY+40, textSize, ink, span.Feels)
		}
	}
	return picture
}
