// Command shorts writes the cloud sea's walkthrough shorts with package
// walk, each over a span of a cheeptune score in a voicing that starts
// on the bar the short's sound names; or one frame of a short as a PNG.
//
//	shorts -script S -pictures DIR -short NAME -score SCORE \
//		-voicing @night-pad -mp4 OUT
//	shorts -script S -pictures DIR -short NAME -frame K -png OUT
//
// The cloud sea's piece is glass hour through @night-pad, as its reel's
// is. A short with no sound in the script is written silent.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/improv/walk"
	"github.com/janearc/slayer/interfaces/movie"
)

// asked is what the command was asked for.
type asked struct {
	script, pictures, short, score, voicing, mp4, png string
	frame                                             int
}

// errAsked is a refusal of what was asked; it says what to give.
var errAsked = errors.New("shorts")

// main reads the flags and does what they ask, or says why not.
func main() {
	var given asked
	flag.StringVar(&given.script, "script", "", "the walkthrough script")
	flag.StringVar(&given.pictures, "pictures", "",
		"the folder of the shots' stills")
	flag.StringVar(&given.short, "short", "", "the short's name")
	flag.StringVar(&given.score, "score", "", "the cheeptune score")
	flag.StringVar(&given.voicing, "voicing", "",
		"the voicing, a path or a built-in named with @")
	flag.StringVar(&given.mp4, "mp4", "", "the short to write")
	flag.StringVar(&given.png, "png", "", "the frame to write")
	flag.IntVar(&given.frame, "frame", -1, "a frame of the short to write")
	flag.Parse()
	if err := run(given); err != nil {
		fmt.Fprintln(os.Stderr, "shorts:", err)
		os.Exit(1)
	}
}

// run does what was asked.
func run(given asked) error {
	script, err := walk.ReadScript(given.script, given.pictures)
	if err != nil {
		return err
	}
	short, found := script.Short(given.short)
	if !found {
		return fmt.Errorf("%w: the script has no short %q", errAsked,
			given.short)
	}
	if given.frame >= 0 {
		frame, err := short.Frame(given.frame, walk.NewStills())
		if err != nil {
			return err
		}
		return writePNG(given.png, frame)
	}
	sound, err := soundFor(short, given)
	if err != nil {
		return err
	}
	written, err := short.Write(context.Background(), given.mp4, sound)
	fmt.Printf("%s: %d frames, %.1f s\n", given.mp4, written,
		short.Seconds())
	return err
}

// soundFor is the score's span under a short, from the bar it names.
func soundFor(short walk.Short, given asked) (movie.Sound, error) {
	bar, sounded := short.Bar()
	if !sounded {
		return movie.Sound{}, nil
	}
	if given.score == "" || given.voicing == "" {
		return movie.Sound{}, fmt.Errorf("%w: short %s names bar %d; give "+
			"-score and -voicing", errAsked, short.Name(), bar)
	}
	piece, err := cheeptune.Open(given.score, given.voicing)
	if err != nil {
		return movie.Sound{}, err
	}
	from, err := barStart(piece.Score, bar)
	if err != nil {
		return movie.Sound{}, err
	}
	song, err := soundOf(piece)
	if err != nil {
		return movie.Sound{}, err
	}
	fmt.Printf("sound: bar %d, from %.3f s\n", bar, from)
	return walk.SongSpan(song, from, short.Seconds())
}

// barStart is the second a bar of a score starts at, the first bar one;
// a bar the score does not have is refused by name.
func barStart(score *cheeptune.Score, bar int) (float64, error) {
	if bar < 1 || bar > score.Bars() {
		return 0, fmt.Errorf("%w: bar %d; the score has bars 1 to %d",
			errAsked, bar, score.Bars())
	}
	return score.SecondsAt(float64((bar - 1) * score.Meter)), nil
}

// soundOf is a rendered piece as a movie's sound, its two channels
// taken in turn, as door's tuned takes them.
func soundOf(piece *cheeptune.Piece) (movie.Sound, error) {
	samples := make([]float32, 0, 2*len(piece.Left))
	for i := range piece.Left {
		samples = append(samples, piece.Left[i], piece.Right[i])
	}
	return movie.NewSound(piece.Rate, samples)
}

// writePNG writes a frame.
func writePNG(path string, frame image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, frame); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
